package packager

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// legalFilePatterns are base names (case-insensitive) of files that should always
// be included in the package root, regardless of exclude filters.
// legalFileBaseNames are the base names (without extension) of files that should
// always be included. legalFileExtensions are the extensions to match. The full
// pattern list is generated as the cross product of these two sets, plus bare names.
var legalFileBaseNames = []string{
	"license",
	"licence",
	"readme",
	"notice",
	"third-party-notices",
	"thirdpartynotices",
}

var legalFileExtensions = []string{"", ".md", ".txt", ".rst"}

var legalFilePatterns = generateLegalFilePatterns()

func generateLegalFilePatterns() []string {
	var patterns []string
	for _, base := range legalFileBaseNames {
		for _, ext := range legalFileExtensions {
			patterns = append(patterns, base+ext)
		}
	}
	return patterns
}

// isLegalFile returns true if the filename (base name only) matches a known
// license/readme/notice pattern.
func isLegalFile(name string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range legalFilePatterns {
		if lower == pattern {
			return true
		}
	}
	return false
}

// CopyLegalFiles copies license, readme, and notice files from srcDir (top-level only)
// to destDir, regardless of any exclude patterns. Files already present in destDir are
// not overwritten.
func CopyLegalFiles(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isLegalFile(entry.Name()) {
			continue
		}

		destPath := filepath.Join(destDir, entry.Name())
		// Don't overwrite if already present (e.g., copied by CopyFiltered)
		if _, err := os.Stat(destPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := copyFile(srcPath, destPath, info.Mode()); err != nil {
			return err
		}
	}

	return nil
}

// CopyLegalFilesSearchingUp copies legal files from srcDir to destDir, searching
// upward through parent directories up to (and including) rootDir. This ensures
// license files at a repository root are found even when a subpath is used.
// Files found closer to srcDir take precedence over those found higher up.
func CopyLegalFilesSearchingUp(srcDir, rootDir, destDir string) error {
	srcAbs, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return err
	}

	// Collect directories from srcDir up to rootDir
	var dirs []string
	current := srcAbs
	for {
		dirs = append(dirs, current)
		if current == rootAbs {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	// Search from closest to furthest — CopyLegalFiles won't overwrite existing files
	for _, dir := range dirs {
		if err := CopyLegalFiles(dir, destDir); err != nil {
			return err
		}
	}

	return nil
}

// ExtractLegalFilesFromZip extracts license, readme, and notice files from a zip
// archive's root to destDir. Used for nupkg files where these are at the archive root.
func ExtractLegalFilesFromZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Only look at root-level files in the archive
		if strings.Contains(f.Name, "/") {
			continue
		}

		if !isLegalFile(f.Name) {
			continue
		}

		destPath := filepath.Join(destDir, filepath.Base(filepath.Clean(f.Name)))
		// Guard against path traversal
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)+string(os.PathSeparator)) &&
			filepath.Clean(destPath) != filepath.Clean(destDir) {
			continue
		}

		// Don't overwrite
		if _, err := os.Stat(destPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
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

		_, copyErr := io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if copyErr != nil {
			return copyErr
		}
	}

	return nil
}
