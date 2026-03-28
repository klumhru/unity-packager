package packager

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ShouldExclude returns true if relativePath matches any of the glob patterns.
func ShouldExclude(relativePath string, patterns []string) bool {
	// Normalize to forward slashes for matching
	normalized := filepath.ToSlash(relativePath)
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, normalized)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// CopyFiltered copies files from srcDir to destDir, excluding paths that match any pattern.
// It preserves directory structure and file permissions.
func CopyFiltered(srcDir, destDir string, patterns []string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Skip .git directory
		if relPath == ".git" || strings.HasPrefix(relPath, ".git"+string(filepath.Separator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip Unity-ignored directories (names ending with ~, e.g., Tests~, Documentation~)
		if info.IsDir() && strings.HasSuffix(info.Name(), "~") {
			return filepath.SkipDir
		}

		if ShouldExclude(relPath, patterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
