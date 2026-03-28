package unity

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PackageJSON struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	DisplayName  string `json:"displayName"`
	Description  string `json:"description"`
	Unity        string `json:"unity"`
}

func NewPackageJSON(name, version, description string) PackageJSON {
	if version == "" {
		version = "0.0.0"
	}
	displayName := name
	return PackageJSON{
		Name:        name,
		Version:     version,
		DisplayName: displayName,
		Description: description,
		Unity:       "2021.3",
	}
}

func WritePackageJSON(dir string, pkg PackageJSON) error {
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)
}

// ReadPackageJSON reads and parses an existing Unity package.json.
func ReadPackageJSON(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
