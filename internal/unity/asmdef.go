package unity

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AsmDef struct {
	Name               string   `json:"name"`
	RootNamespace      string   `json:"rootNamespace"`
	References         []string `json:"references"`
	IncludePlatforms   []string `json:"includePlatforms"`
	ExcludePlatforms   []string `json:"excludePlatforms"`
	AllowUnsafeCode    bool     `json:"allowUnsafeCode"`
	OverrideReferences bool     `json:"overrideReferences"`
	PrecompiledReferences []string `json:"precompiledReferences"`
	AutoReferenced     bool     `json:"autoReferenced"`
	DefineConstraints  []string `json:"defineConstraints"`
	VersionDefines     []string `json:"versionDefines"`
	NoEngineReferences bool     `json:"noEngineReferences"`
}

func NewAsmDef(name, rootNamespace string, dependencies []string) AsmDef {
	return AsmDef{
		Name:                  name,
		RootNamespace:         rootNamespace,
		References:            dependencies,
		IncludePlatforms:      []string{},
		ExcludePlatforms:      []string{},
		AllowUnsafeCode:       false,
		OverrideReferences:    false,
		PrecompiledReferences: []string{},
		AutoReferenced:        true,
		DefineConstraints:     []string{},
		VersionDefines:        []string{},
		NoEngineReferences:    false,
	}
}

func WriteAsmDef(dir, filename string, asmdef AsmDef) error {
	data, err := json.MarshalIndent(asmdef, "", "    ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, filename), data, 0644)
}
