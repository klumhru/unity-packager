package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type PackageType string

const (
	GitUnity PackageType = "git-unity"
	GitRaw   PackageType = "git-raw"
	NuGet    PackageType = "nuget"
	Archive  PackageType = "archive"
)

type PackageSpec struct {
	Name           string      `json:"name"`
	Type           PackageType `json:"type"`
	URL            string      `json:"url,omitempty"`
	Ref            string      `json:"ref,omitempty"`
	Path           string      `json:"path,omitempty"`
	Version        string      `json:"version,omitempty"`
	Description    string      `json:"description,omitempty"`
	Dependencies   []string    `json:"dependencies,omitempty"`
	NuGetID        string      `json:"nugetId,omitempty"`
	NuGetVersion   string      `json:"nugetVersion,omitempty"`
	NuGetFramework string      `json:"nugetFramework,omitempty"`
	Exclude        []string    `json:"exclude,omitempty"`
}

type Config struct {
	Packages []PackageSpec `json:"packages"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Packages) == 0 {
		return fmt.Errorf("no packages defined")
	}

	names := make(map[string]bool)
	for i, pkg := range c.Packages {
		if pkg.Name == "" {
			return fmt.Errorf("package %d: name is required", i)
		}
		if names[pkg.Name] {
			return fmt.Errorf("package %d: duplicate name %q", i, pkg.Name)
		}
		names[pkg.Name] = true

		switch pkg.Type {
		case GitUnity, GitRaw:
			if pkg.URL == "" {
				return fmt.Errorf("package %q: url is required for type %s", pkg.Name, pkg.Type)
			}
			if pkg.Ref == "" {
				return fmt.Errorf("package %q: ref is required for type %s", pkg.Name, pkg.Type)
			}
		case Archive:
			if pkg.URL == "" {
				return fmt.Errorf("package %q: url is required for type archive", pkg.Name)
			}
		case NuGet:
			if pkg.NuGetID == "" {
				return fmt.Errorf("package %q: nugetId is required for type nuget", pkg.Name)
			}
			if pkg.NuGetVersion == "" {
				return fmt.Errorf("package %q: nugetVersion is required for type nuget", pkg.Name)
			}
		default:
			return fmt.Errorf("package %q: unknown type %q (must be git-unity, git-raw, nuget, or archive)", pkg.Name, pkg.Type)
		}

		// Validate dependencies reference known package names
		for _, dep := range pkg.Dependencies {
			if !names[dep] {
				// Check if it's defined later in the list
				found := false
				for _, other := range c.Packages {
					if other.Name == dep {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("package %q: dependency %q not found in packages list", pkg.Name, dep)
				}
			}
		}
	}

	return nil
}
