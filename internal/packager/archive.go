package packager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/klumhru/unity-packager/internal/config"
	"github.com/klumhru/unity-packager/internal/unity"
)

func (p *Packager) processArchive(spec config.PackageSpec) error {
	extractDir, err := p.downloadAndExtractArchive(spec.URL)
	if err != nil {
		return err
	}

	// If a subpath is specified, use it directly; otherwise unwrap single top-level dirs
	// (common in archives like "firebase-unity-sdk-12.0.0/...")
	srcDir := extractDir
	if spec.Path != "" {
		srcDir = filepath.Join(extractDir, spec.Path)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			return fmt.Errorf("path %q not found in archive", spec.Path)
		}
	} else {
		srcDir = unwrapSingleDir(extractDir)
	}

	destDir := filepath.Join(p.packagesDir, spec.Name)

	// Check if this looks like a Unity package
	pkgJSONPath := filepath.Join(srcDir, "package.json")
	if _, err := os.Stat(pkgJSONPath); err == nil {
		// Unity package — copy directly (like git-unity)
		if err := CopyFiltered(srcDir, destDir, spec.Exclude); err != nil {
			return fmt.Errorf("copying archive package %q: %w", spec.Name, err)
		}
	} else {
		// Raw package — copy into Runtime/, generate package.json + asmdef (like git-raw)
		runtimeDir := filepath.Join(destDir, "Runtime")
		if err := os.MkdirAll(runtimeDir, 0755); err != nil {
			return err
		}

		if err := CopyFiltered(srcDir, runtimeDir, spec.Exclude); err != nil {
			return fmt.Errorf("copying archive package %q: %w", spec.Name, err)
		}

		pkg := unity.NewPackageJSON(spec.Name, spec.Version, spec.Description)
		if err := unity.WritePackageJSON(destDir, pkg); err != nil {
			return fmt.Errorf("writing package.json for %q: %w", spec.Name, err)
		}

		rootNamespace := unity.InferRootNamespace(runtimeDir)
		if rootNamespace == "" {
			rootNamespace = spec.Name
		}

		asmdef := unity.NewAsmDef(spec.Name, rootNamespace, spec.Dependencies)
		if err := unity.WriteAsmDef(runtimeDir, spec.Name+".asmdef", asmdef); err != nil {
			return fmt.Errorf("writing asmdef for %q: %w", spec.Name, err)
		}
	}

	if err := GenerateMetaFiles(destDir, spec.Name); err != nil {
		return fmt.Errorf("generating meta files for %q: %w", spec.Name, err)
	}

	return nil
}

func (p *Packager) downloadAndExtractArchive(url string) (string, error) {
	// Check cache
	if cached := p.cache.ArchiveDir(url); cached != "" {
		p.logf("  using cached archive for %s", url)
		return cached, nil
	}

	p.logf("  downloading %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP GET %s: status %d", url, resp.StatusCode)
	}

	// Write to a temp file first so we can detect format
	tmpFile, err := os.CreateTemp("", "unity-packager-archive-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	// Determine extract directory
	var extractDir string
	if storeDir := p.cache.ArchiveStoreDir(url); storeDir != "" {
		extractDir = storeDir
	} else {
		dir, err := os.MkdirTemp("", "unity-packager-extract-*")
		if err != nil {
			return "", err
		}
		p.tmpDirs = append(p.tmpDirs, dir)
		extractDir = dir
	}

	// Detect format from URL and extract
	lowerURL := strings.ToLower(url)
	switch {
	case strings.HasSuffix(lowerURL, ".zip"):
		err = extractZip(tmpPath, extractDir)
	case strings.HasSuffix(lowerURL, ".tar.gz") || strings.HasSuffix(lowerURL, ".tgz"):
		err = extractTarGz(tmpPath, extractDir)
	default:
		// Try to detect from content: check if it's a zip (starts with PK)
		err = tryExtract(tmpPath, extractDir)
	}

	if err != nil {
		return "", fmt.Errorf("extracting archive from %s: %w", url, err)
	}

	return extractDir, nil
}

// unwrapSingleDir checks if a directory contains exactly one subdirectory and nothing else.
// If so, returns the path to that subdirectory. This handles archives that wrap everything
// in a top-level folder (e.g., "firebase-unity-sdk-12.0.0/").
func unwrapSingleDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(dir, entries[0].Name())
	}
	return dir
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		destPath := filepath.Join(destDir, f.Name)

		// Guard against zip slip
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)+string(os.PathSeparator)) && filepath.Clean(destPath) != filepath.Clean(destDir) {
			continue
		}

		if f.FileInfo().IsDir() {
			mode := f.Mode()
			if mode == 0 {
				mode = 0755
			}
			os.MkdirAll(destPath, mode)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		mode := f.Mode()
		if mode == 0 {
			mode = 0644
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	return extractTar(gz, destDir)
}

func extractTar(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, header.Name)

		// Guard against path traversal
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)+string(os.PathSeparator)) && filepath.Clean(destPath) != filepath.Clean(destDir) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			mode := os.FileMode(header.Mode)
			if mode == 0 {
				mode = 0755
			}
			os.MkdirAll(destPath, mode)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			mode := os.FileMode(header.Mode)
			if mode == 0 {
				mode = 0644
			}
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// tryExtract attempts to detect the archive format by content and extract.
func tryExtract(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}

	// Read magic bytes
	magic := make([]byte, 4)
	n, _ := f.Read(magic)
	f.Close()

	if n >= 2 && magic[0] == 'P' && magic[1] == 'K' {
		return extractZip(archivePath, destDir)
	}
	if n >= 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		return extractTarGz(archivePath, destDir)
	}

	return fmt.Errorf("unsupported archive format (expected zip, tar.gz, or tgz)")
}
