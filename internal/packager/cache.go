package packager

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type Cache struct {
	baseDir string
	enabled bool
}

func NewCache(enabled bool) (*Cache, error) {
	if !enabled {
		return &Cache{enabled: false}, nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("determining cache directory: %w", err)
	}

	baseDir := filepath.Join(cacheDir, "unity-packager")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	return &Cache{baseDir: baseDir, enabled: true}, nil
}

// GitDir returns the cached directory for a git repo at a given ref.
// Returns empty string if not cached.
func (c *Cache) GitDir(url, ref string) string {
	if !c.enabled {
		return ""
	}
	dir := filepath.Join(c.baseDir, "git", cacheKey(url, ref))
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	return ""
}

// GitStoreDir returns the directory where a git clone should be stored in the cache.
func (c *Cache) GitStoreDir(url, ref string) string {
	if !c.enabled {
		return ""
	}
	dir := filepath.Join(c.baseDir, "git", cacheKey(url, ref))
	os.MkdirAll(dir, 0755)
	return dir
}

// NuGetPath returns the cached .nupkg file path.
// Returns empty string if not cached.
func (c *Cache) NuGetPath(id, version string) string {
	if !c.enabled {
		return ""
	}
	path := filepath.Join(c.baseDir, "nuget", fmt.Sprintf("%s.%s.nupkg", id, version))
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// NuGetStorePath returns the path where a .nupkg should be stored in the cache.
func (c *Cache) NuGetStorePath(id, version string) string {
	if !c.enabled {
		return ""
	}
	dir := filepath.Join(c.baseDir, "nuget")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, fmt.Sprintf("%s.%s.nupkg", id, version))
}

// ArchiveDir returns the cached extracted directory for an archive URL.
// Returns empty string if not cached.
func (c *Cache) ArchiveDir(url string) string {
	if !c.enabled {
		return ""
	}
	dir := filepath.Join(c.baseDir, "archive", cacheKey(url))
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	return ""
}

// ArchiveStoreDir returns the directory where an extracted archive should be stored in the cache.
func (c *Cache) ArchiveStoreDir(url string) string {
	if !c.enabled {
		return ""
	}
	dir := filepath.Join(c.baseDir, "archive", cacheKey(url))
	os.MkdirAll(dir, 0755)
	return dir
}

func cacheKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
