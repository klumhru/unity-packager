package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/klumhru/unity-packager/internal/config"
	"github.com/klumhru/unity-packager/internal/unity"
)

func (p *Packager) processGitUnity(spec config.PackageSpec) error {
	srcDir, err := p.cloneOrCache(spec.URL, spec.Ref)
	if err != nil {
		return err
	}

	// If a subpath is specified, use it
	if spec.Path != "" {
		srcDir = filepath.Join(srcDir, spec.Path)
	}

	// Verify it's a Unity package
	pkgJSONPath := filepath.Join(srcDir, "package.json")
	if _, err := os.Stat(pkgJSONPath); os.IsNotExist(err) {
		return fmt.Errorf("git-unity package %q: no package.json found at %s (use git-raw for non-Unity repos)", spec.Name, pkgJSONPath)
	}

	destDir := filepath.Join(p.packagesDir, spec.Name)
	if err := CopyFiltered(srcDir, destDir, spec.Exclude); err != nil {
		return fmt.Errorf("copying git-unity package %q: %w", spec.Name, err)
	}

	if err := GenerateMetaFiles(destDir, spec.Name); err != nil {
		return fmt.Errorf("generating meta files for %q: %w", spec.Name, err)
	}

	return nil
}

func (p *Packager) processGitRaw(spec config.PackageSpec) error {
	srcDir, err := p.cloneOrCache(spec.URL, spec.Ref)
	if err != nil {
		return err
	}

	// If a subpath is specified, use it
	if spec.Path != "" {
		srcDir = filepath.Join(srcDir, spec.Path)
	}

	destDir := filepath.Join(p.packagesDir, spec.Name)
	runtimeDir := filepath.Join(destDir, "Runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}

	// Copy source files into Runtime/
	if err := CopyFiltered(srcDir, runtimeDir, spec.Exclude); err != nil {
		return fmt.Errorf("copying git-raw package %q: %w", spec.Name, err)
	}

	// Generate package.json
	pkg := unity.NewPackageJSON(spec.Name, spec.Version, spec.Description)
	if err := unity.WritePackageJSON(destDir, pkg); err != nil {
		return fmt.Errorf("writing package.json for %q: %w", spec.Name, err)
	}

	// Infer root namespace from .cs files
	rootNamespace := unity.InferRootNamespace(runtimeDir)
	if rootNamespace == "" {
		rootNamespace = spec.Name
	}

	// Generate .asmdef
	asmdef := unity.NewAsmDef(spec.Name, rootNamespace, spec.Dependencies)
	asmdefFilename := spec.Name + ".asmdef"
	if err := unity.WriteAsmDef(runtimeDir, asmdefFilename, asmdef); err != nil {
		return fmt.Errorf("writing asmdef for %q: %w", spec.Name, err)
	}

	// Generate meta files for everything
	if err := GenerateMetaFiles(destDir, spec.Name); err != nil {
		return fmt.Errorf("generating meta files for %q: %w", spec.Name, err)
	}

	return nil
}

func (p *Packager) cloneOrCache(url, ref string) (string, error) {
	// Check cache first
	if cached := p.cache.GitDir(url, ref); cached != "" {
		p.logf("  using cached clone for %s@%s", url, ref)
		return cached, nil
	}

	// Determine clone destination
	var cloneDir string
	if storeDir := p.cache.GitStoreDir(url, ref); storeDir != "" {
		cloneDir = storeDir
	} else {
		tmpDir, err := os.MkdirTemp("", "unity-packager-git-*")
		if err != nil {
			return "", err
		}
		p.tmpDirs = append(p.tmpDirs, tmpDir)
		cloneDir = tmpDir
	}

	if err := cloneRepo(url, ref, cloneDir); err != nil {
		return "", err
	}

	return cloneDir, nil
}

func cloneRepo(url, ref, destDir string) error {
	// Try shallow clone with --branch (works for tags and branches)
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, url, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Fallback: full clone + checkout (needed for commit SHAs)
		os.RemoveAll(destDir)
		os.MkdirAll(destDir, 0755)

		cmd = exec.Command("git", "clone", url, destDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone %s: %w", url, err)
		}

		cmd = exec.Command("git", "-C", destDir, "checkout", ref)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git checkout %s: %w", ref, err)
		}
	}

	return nil
}
