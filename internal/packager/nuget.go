package packager

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/klumhru/unity-packager/internal/config"
	"github.com/klumhru/unity-packager/internal/unity"
)

func (p *Packager) processNuGet(spec config.PackageSpec) error {
	framework := spec.NuGetFramework
	if framework == "" {
		framework = "netstandard2.0"
	}

	nupkgPath, err := p.downloadOrCache(spec.NuGetID, spec.NuGetVersion)
	if err != nil {
		return fmt.Errorf("downloading NuGet package %q: %w", spec.NuGetID, err)
	}

	destDir := filepath.Join(p.packagesDir, spec.Name)
	pluginsDir := filepath.Join(destDir, "Plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return err
	}

	// Extract DLLs from the nupkg
	if err := extractDLLs(nupkgPath, framework, pluginsDir, spec.Exclude); err != nil {
		return fmt.Errorf("extracting NuGet package %q: %w", spec.NuGetID, err)
	}

	// Extract license/readme from nupkg root
	if err := ExtractLegalFilesFromZip(nupkgPath, destDir); err != nil {
		return fmt.Errorf("extracting legal files for %q: %w", spec.Name, err)
	}

	// Generate package.json
	version := spec.NuGetVersion
	description := fmt.Sprintf("NuGet package %s v%s", spec.NuGetID, spec.NuGetVersion)
	pkg := unity.NewPackageJSON(spec.Name, version, description)
	if err := unity.WritePackageJSON(destDir, pkg); err != nil {
		return fmt.Errorf("writing package.json for %q: %w", spec.Name, err)
	}

	// Generate meta files
	if err := GenerateMetaFiles(destDir, spec.Name); err != nil {
		return fmt.Errorf("generating meta files for %q: %w", spec.Name, err)
	}

	return nil
}

func (p *Packager) downloadOrCache(id, version string) (string, error) {
	// Check cache
	if cached := p.cache.NuGetPath(id, version); cached != "" {
		p.logf("  using cached nupkg for %s@%s", id, version)
		return cached, nil
	}

	// Download from NuGet API
	lowerID := strings.ToLower(id)
	lowerVersion := strings.ToLower(version)
	url := fmt.Sprintf("https://api.nuget.org/v3-flatcontainer/%s/%s/%s.%s.nupkg",
		lowerID, lowerVersion, lowerID, lowerVersion)

	p.logf("  downloading %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP GET %s: status %d", url, resp.StatusCode)
	}

	// Write to cache or temp file
	var destPath string
	if storePath := p.cache.NuGetStorePath(id, version); storePath != "" {
		destPath = storePath
	} else {
		tmpFile, err := os.CreateTemp("", "unity-packager-nupkg-*")
		if err != nil {
			return "", err
		}
		destPath = tmpFile.Name()
		tmpFile.Close()
		p.tmpFiles = append(p.tmpFiles, destPath)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}

	return destPath, nil
}

func extractDLLs(nupkgPath, framework, destDir string, excludePatterns []string) error {
	r, err := zip.OpenReader(nupkgPath)
	if err != nil {
		return fmt.Errorf("opening nupkg: %w", err)
	}
	defer r.Close()

	prefix := fmt.Sprintf("lib/%s/", framework)
	found := false

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := strings.ToLower(f.Name)
		if !strings.HasPrefix(name, strings.ToLower(prefix)) {
			continue
		}

		// Get just the filename relative to the framework dir
		relName := f.Name[len(prefix):]

		if ShouldExclude(relName, excludePatterns) {
			continue
		}

		found = true
		destPath := filepath.Join(destDir, filepath.FromSlash(relName))

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
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

	if !found {
		// List available frameworks for a helpful error
		frameworks := availableFrameworks(r)
		return fmt.Errorf("no files found for framework %q in nupkg (available: %s)",
			framework, strings.Join(frameworks, ", "))
	}

	return nil
}

func availableFrameworks(r *zip.ReadCloser) []string {
	seen := make(map[string]bool)
	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if strings.HasPrefix(name, "lib/") {
			parts := strings.SplitN(name[4:], "/", 2)
			if len(parts) >= 1 && parts[0] != "" {
				seen[parts[0]] = true
			}
		}
	}
	var frameworks []string
	for fw := range seen {
		frameworks = append(frameworks, fw)
	}
	return frameworks
}
