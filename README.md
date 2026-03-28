# unity-packager

A CLI tool that downloads upstream packages from git repositories and NuGet and packages them for use in Unity projects.

## Install

```bash
go install github.com/klumhru/unity-packager@latest
```

Or build from source:

```bash
git clone https://github.com/klumhru/unity-packager.git
cd unity-packager
go build -o unity-packager .
```

Copy the binary to a location on your `PATH` (e.g., `~/.local/bin/`).

## Usage

```bash
# Run from your Unity project root
unity-packager

# Specify a project path
unity-packager -project /path/to/MyUnityProject

# Verbose output
unity-packager -verbose

# Skip cleaning existing packages (incremental update)
unity-packager -clean=false

# Force re-download, ignore cache
unity-packager -no-cache
```

The tool reads `Packages/upstream-packages.json` in your Unity project and downloads each package into the `Packages/` directory.

## Configuration

Create `Packages/upstream-packages.json` in your Unity project:

```json
{
  "packages": [
    {
      "name": "jp.hadashikick.vcontainer",
      "type": "git-unity",
      "url": "https://github.com/hadashiA/VContainer.git",
      "ref": "1.16.7",
      "path": "VContainer/Assets/VContainer",
      "exclude": ["Tests~/**"]
    },
    {
      "name": "com.google.protobuf",
      "type": "git-raw",
      "url": "https://github.com/protocolbuffers/protobuf.git",
      "ref": "v3.27.1",
      "path": "csharp/src/Google.Protobuf",
      "version": "3.27.1",
      "description": "Google Protocol Buffers for C#",
      "exclude": ["**/*Test*.cs", "**/*.csproj", "**/*.sln"]
    },
    {
      "name": "com.grpc.core",
      "type": "nuget",
      "nugetId": "Grpc.Core",
      "nugetVersion": "2.46.6",
      "nugetFramework": "netstandard2.0",
      "dependencies": ["com.google.protobuf"]
    }
  ]
}
```

## Package Types

### `git-unity`

For git repos that are already structured as Unity packages (contain a `package.json`). The tool clones the repo and copies the package contents directly.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unity package name, used as the folder name under `Packages/` |
| `url` | yes | Git clone URL |
| `ref` | yes | Git ref — tag, branch, or commit SHA |
| `path` | no | Subdirectory within the repo to use as the package root |
| `exclude` | no | Glob patterns for files/dirs to exclude |

### `git-raw`

For git repos that are not designed for Unity. The tool clones the repo, copies source files into a `Runtime/` folder, generates a `package.json`, and creates an `.asmdef` file with the root namespace inferred from the C# source.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unity package name |
| `url` | yes | Git clone URL |
| `ref` | yes | Git ref |
| `path` | no | Subdirectory within the repo |
| `version` | no | Version for generated `package.json` (default: `0.0.0`) |
| `description` | no | Description for generated `package.json` |
| `dependencies` | no | List of other package names — maps to asmdef references |
| `exclude` | no | Glob patterns to exclude |

Output structure:

```
Packages/com.example.raw-lib/
├── package.json
├── Runtime/
│   ├── com.example.raw-lib.asmdef
│   └── ... (source files)
```

### `nuget`

For NuGet packages. The tool downloads the `.nupkg`, extracts DLLs for the target framework into a `Plugins/` folder, and generates a `package.json`.

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unity package name |
| `nugetId` | yes | NuGet package ID |
| `nugetVersion` | yes | NuGet package version |
| `nugetFramework` | no | Target framework to extract (default: `netstandard2.0`) |
| `dependencies` | no | List of other package names |
| `exclude` | no | Glob patterns to exclude |

Output structure:

```
Packages/com.example.nuget-lib/
├── package.json
├── Plugins/
│   ├── Example.dll
│   └── ...
```

## Features

- **Meta file generation** — creates Unity `.meta` files with deterministic GUIDs (based on package name + relative path), so re-running the tool doesn't cause unnecessary git changes
- **Download caching** — upstream packages are cached in `~/.cache/unity-packager/` to avoid re-downloading on subsequent runs
- **File exclusion** — glob patterns with `**` support for filtering out tests, docs, or other unwanted files
- **Asmdef generation** — `git-raw` packages get an `.asmdef` with the root namespace inferred from the C# source files, and references populated from the `dependencies` list

## Requirements

- Go 1.21+ (for building)
- `git` on PATH (for git package types)
- Network access to github.com and nuget.org

## License

MIT
