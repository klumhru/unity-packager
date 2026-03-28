# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

unity-packager is a Go CLI tool that downloads upstream packages (git repos, NuGet packages, HTTP archives) and packages them into a Unity project's `Packages/` directory. It handles four package types:

- **git-unity**: Repos with an existing Unity `package.json` (clone + copy)
- **git-raw**: Non-Unity repos (clone into `Runtime/`, generate `package.json` + `.asmdef`)
- **nuget**: NuGet packages (download `.nupkg`, extract DLLs into `Plugins/`)
- **archive**: HTTP zip/tar.gz/tgz archives (e.g., Firebase Unity SDK); auto-detects Unity vs raw

## Build and Test

```bash
go build ./...              # build
go test ./...               # unit tests
go test -tags=integration ./...  # integration tests (downloads real packages)
go run . -project <path>    # run directly
```

Integration tests use real upstreams (VContainer, protobuf, Grpc.Core) and require network access. They're behind the `integration` build tag to avoid running on every `go test`.

## Architecture

```
main.go                         # CLI entry point (flag parsing only)
internal/
  config/config.go              # Config types, Load(), Validate()
  packager/
    packager.go                 # Orchestrator — iterates packages, dispatches by type
    git.go                      # git-unity and git-raw handlers (shells out to git)
    nuget.go                    # NuGet handler (HTTP download + zip extract)
    archive.go                  # HTTP archive handler (zip/tar.gz/tgz)
    meta.go                     # Unity .meta file generation (deterministic GUIDs via MD5)
    filter.go                   # Glob-based file exclusion + filtered directory copy
    cache.go                    # Download cache (~/.cache/unity-packager/)
  unity/
    packagejson.go              # Unity package.json types + generation
    asmdef.go                   # .asmdef file types + generation
    namespace.go                # Infer root namespace from .cs files
```

All packages are under `internal/` — no exported library API.

## Key Design Decisions

- **Git via `os/exec`**, not go-git — lighter binary, transparent auth handling
- **Deterministic .meta GUIDs**: `md5(packageName + "/" + relativePath)` — avoids git churn on re-runs
- **`doublestar` library** for `**` glob patterns in exclude filters (stdlib `filepath.Match` doesn't support `**`)
- **Fail fast**: any package failure stops the whole run
- **Cache**: `~/.cache/unity-packager/`, keyed by url+ref for git, id+version for nuget, url for archives. `-no-cache` flag to bypass.

## Config File

The tool reads `Packages/upstream-packages.json`. See `testdata/upstream-packages.json` for a complete example with all four package types.

Package types produce different folder layouts:
- `git-raw` → `Runtime/` folder with `.asmdef` (rootNamespace inferred from `.cs` files)
- `nuget` → `Plugins/` folder with extracted DLLs
- `git-unity` → direct copy of upstream structure
- `archive` → auto-detects: if archive contains `package.json`, copies directly (like git-unity); otherwise uses `Runtime/` + asmdef (like git-raw). Single top-level dirs are auto-unwrapped.
