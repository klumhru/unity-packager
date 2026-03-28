package packager

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateGUID produces a deterministic 32-hex-char GUID from a package name and relative path.
// This avoids git churn when re-running the tool on unchanged inputs.
func GenerateGUID(packageName, relativePath string) string {
	h := md5.Sum([]byte(packageName + "/" + relativePath))
	return fmt.Sprintf("%x", h)
}

func WriteMetaFile(metaPath, guid string, isDir bool) error {
	var content string
	if isDir {
		content = fmt.Sprintf(`fileFormatVersion: 2
guid: %s
folderAsset: yes
DefaultImporter:
  externalObjects: {}
  userData:
  assetBundleName:
  assetBundleVariant:
`, guid)
	} else {
		importer := importerForFile(metaPath)
		content = fmt.Sprintf(`fileFormatVersion: 2
guid: %s
%s`, guid, importer)
	}

	return os.WriteFile(metaPath, []byte(content), 0644)
}

func importerForFile(path string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSuffix(path, ".meta")))
	switch ext {
	case ".dll":
		return `PluginImporter:
  externalObjects: {}
  serializedVersion: 2
  iconMap: {}
  executionOrder: {}
  defineConstraints: []
  isPreloaded: 0
  isOverridable: 0
  isExplicitlyReferenced: 0
  validateReferences: 1
  platformData:
  - first:
      Any:
    second:
      enabled: 1
      settings: {}
  userData:
  assetBundleName:
  assetBundleVariant:
`
	case ".cs":
		return `MonoImporter:
  externalObjects: {}
  serializedVersion: 2
  defaultReferences: []
  executionOrder: 0
  icon: {instanceID: 0}
  userData:
  assetBundleName:
  assetBundleVariant:
`
	case ".asmdef":
		return `DefaultImporter:
  externalObjects: {}
  userData:
  assetBundleName:
  assetBundleVariant:
`
	default:
		return `DefaultImporter:
  externalObjects: {}
  userData:
  assetBundleName:
  assetBundleVariant:
`
	}
}

// GenerateMetaFiles walks a directory tree and creates .meta files for every file and subdirectory.
func GenerateMetaFiles(rootDir, packageName string) error {
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .meta files themselves
		if strings.HasSuffix(path, ".meta") {
			return nil
		}

		// Skip the root directory itself
		if path == rootDir {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		metaPath := path + ".meta"
		// Skip if meta already exists
		if _, err := os.Stat(metaPath); err == nil {
			return nil
		}

		guid := GenerateGUID(packageName, relPath)
		return WriteMetaFile(metaPath, guid, info.IsDir())
	})
}
