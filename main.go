package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/klumhru/unity-packager/internal/config"
	"github.com/klumhru/unity-packager/internal/packager"
)

func main() {
	project := flag.String("project", ".", "path to Unity project root")
	configPath := flag.String("config", "", "override config file path (default: Packages/upstream-packages.json)")
	clean := flag.Bool("clean", true, "remove existing package dirs before re-packaging")
	noCache := flag.Bool("no-cache", false, "force re-download, ignore cache")
	verbose := flag.Bool("verbose", false, "verbose logging")
	flag.Parse()

	projectRoot, err := filepath.Abs(*project)
	if err != nil {
		log.Fatalf("invalid project path: %v", err)
	}

	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = filepath.Join(projectRoot, "Packages", "upstream-packages.json")
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	p := packager.New(projectRoot, packager.Options{
		Verbose: *verbose,
		Clean:   *clean,
		NoCache: *noCache,
	})

	if err := p.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
