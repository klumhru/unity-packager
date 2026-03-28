package packager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/klumhru/unity-packager/internal/config"
)

type Options struct {
	Verbose bool
	Clean   bool
	NoCache bool
}

type Packager struct {
	projectRoot string
	packagesDir string
	cache       *Cache
	verbose     bool
	clean       bool
	noCache     bool
	tmpDirs     []string
	tmpFiles    []string
}

func New(projectRoot string, opts Options) *Packager {
	return &Packager{
		projectRoot: projectRoot,
		packagesDir: filepath.Join(projectRoot, "Packages"),
		verbose:     opts.Verbose,
		clean:       opts.Clean,
		noCache:     opts.NoCache,
	}
}

func (p *Packager) Run(cfg *config.Config) error {
	// Initialize cache
	cache, err := NewCache(!p.noCache)
	if err != nil {
		log.Printf("warning: cache disabled: %v", err)
		cache = &Cache{enabled: false}
	}
	p.cache = cache

	defer p.cleanup()

	// Ensure Packages directory exists
	if err := os.MkdirAll(p.packagesDir, 0755); err != nil {
		return fmt.Errorf("creating Packages directory: %w", err)
	}

	for _, spec := range cfg.Packages {
		p.logf("processing %s (%s)", spec.Name, spec.Type)

		if p.clean {
			destDir := filepath.Join(p.packagesDir, spec.Name)
			if _, err := os.Stat(destDir); err == nil {
				p.logf("  cleaning %s", destDir)
				if err := os.RemoveAll(destDir); err != nil {
					return fmt.Errorf("cleaning %s: %w", destDir, err)
				}
			}
		}

		if err := p.processPackage(spec); err != nil {
			return fmt.Errorf("package %q: %w", spec.Name, err)
		}

		p.logf("  done")
	}

	return nil
}

func (p *Packager) processPackage(spec config.PackageSpec) error {
	switch spec.Type {
	case config.GitUnity:
		return p.processGitUnity(spec)
	case config.GitRaw:
		return p.processGitRaw(spec)
	case config.NuGet:
		return p.processNuGet(spec)
	default:
		return fmt.Errorf("unknown package type %q", spec.Type)
	}
}

func (p *Packager) logf(format string, args ...interface{}) {
	if p.verbose {
		log.Printf(format, args...)
	}
}

func (p *Packager) cleanup() {
	for _, dir := range p.tmpDirs {
		os.RemoveAll(dir)
	}
	for _, file := range p.tmpFiles {
		os.Remove(file)
	}
}
