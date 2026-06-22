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
	// Editor-only assemblies go under Editor/ (a Unity special folder), everything
	// else under Plugins/.
	libDirName := "Plugins"
	if spec.EditorOnly {
		libDirName = "Editor"
	}
	pluginsDir := filepath.Join(destDir, libDirName)
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return err
	}

	// Extract DLLs from the nupkg
	if err := extractDLLs(nupkgPath, framework, pluginsDir, spec.Exclude); err != nil {
		return fmt.Errorf("extracting NuGet package %q: %w", spec.NuGetID, err)
	}

	// Recursively resolve transitive NuGet dependencies into the same Plugins/ dir.
	if spec.NuGetResolveDependencies {
		visited := map[string]bool{strings.ToLower(spec.NuGetID): true}
		if err := p.resolveNuGetDeps(nupkgPath, framework, pluginsDir, spec.Exclude, visited); err != nil {
			return fmt.Errorf("resolving dependencies for %q: %w", spec.NuGetID, err)
		}
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

	found, err := extractLibFolder(r, framework, destDir, excludePatterns)
	if err != nil {
		return err
	}
	if !found {
		// List available frameworks for a helpful error
		frameworks := availableFrameworks(r)
		return fmt.Errorf("no files found for framework %q in nupkg (available: %s)",
			framework, strings.Join(frameworks, ", "))
	}
	return nil
}

// extractLibFolder copies every non-excluded file under lib/<framework>/ into
// destDir. It returns whether any file was extracted.
func extractLibFolder(r *zip.ReadCloser, framework, destDir string, excludePatterns []string) (bool, error) {
	prefix := fmt.Sprintf("lib/%s/", strings.ToLower(framework))
	found := false

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := strings.ToLower(f.Name)
		if !strings.HasPrefix(name, prefix) {
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
			return found, err
		}

		rc, err := f.Open()
		if err != nil {
			return found, err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return found, err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return found, err
		}
	}

	return found, nil
}

// resolveNuGetDeps reads the .nuspec at nupkgPath, then downloads and extracts
// each applicable transitive dependency's DLLs into pluginsDir, recursing into
// their dependencies. Framework/runtime meta-packages and already-visited
// packages are skipped. A dependency with no compatible lib folder (e.g. a
// meta-package) is not an error: its DLLs are skipped but its own dependencies
// are still resolved.
func (p *Packager) resolveNuGetDeps(nupkgPath, framework, pluginsDir string, excludePatterns []string, visited map[string]bool) error {
	ns, err := parseNuspec(nupkgPath)
	if err != nil {
		return err
	}

	for _, dep := range ns.dependenciesForFramework(framework) {
		lid := strings.ToLower(dep.ID)
		if lid == "" || visited[lid] || skipNuGetDependency(dep.ID) {
			continue
		}
		visited[lid] = true

		version := resolveVersion(dep.Version)
		if version == "" {
			p.logf("  skipping dependency %s (unresolvable version %q)", dep.ID, dep.Version)
			continue
		}

		p.logf("  resolving dependency %s@%s", dep.ID, version)
		depPath, err := p.downloadOrCache(dep.ID, version)
		if err != nil {
			return fmt.Errorf("downloading dependency %s@%s: %w", dep.ID, version, err)
		}

		if err := extractDepDLLs(depPath, framework, pluginsDir, excludePatterns); err != nil {
			return fmt.Errorf("extracting dependency %s@%s: %w", dep.ID, version, err)
		}

		if err := p.resolveNuGetDeps(depPath, framework, pluginsDir, excludePatterns, visited); err != nil {
			return err
		}
	}

	return nil
}

// extractDepDLLs extracts a transitive dependency's DLLs, choosing the best
// compatible lib folder for the target framework. Missing lib folders are
// tolerated (meta-packages carry no assemblies).
func extractDepDLLs(nupkgPath, framework, destDir string, excludePatterns []string) error {
	r, err := zip.OpenReader(nupkgPath)
	if err != nil {
		return fmt.Errorf("opening nupkg: %w", err)
	}
	defer r.Close()

	folder := bestLibFramework(r, framework)
	if folder == "" {
		return nil
	}
	_, err = extractLibFolder(r, folder, destDir, excludePatterns)
	return err
}

// bestLibFramework picks the lib/ framework folder most compatible with the
// target: an exact (normalized) match if present, otherwise the highest
// netstandard version not exceeding the target. Returns "" if none qualify.
func bestLibFramework(r *zip.ReadCloser, target string) string {
	available := availableFrameworks(r)
	normTarget := normalizeFramework(target)

	for _, fw := range available {
		if normalizeFramework(fw) == normTarget {
			return fw
		}
	}

	targetVer, targetIsNS := netstandardVersion(normTarget)
	if !targetIsNS {
		return ""
	}
	best := ""
	bestVer := -1.0
	for _, fw := range available {
		v, ok := netstandardVersion(normalizeFramework(fw))
		if ok && v <= targetVer && v > bestVer {
			bestVer = v
			best = fw
		}
	}
	return best
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
