# Contributing to unity-packager

Thanks for your interest in contributing! Here's how to get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/unity-packager.git`
3. Create a branch: `git checkout -b my-feature`
4. Make your changes
5. Run tests: `go test ./...`
6. Push and open a pull request

## Development

Build and test:

```bash
go build ./...           # build
go test ./...            # unit tests
go test -tags=integration -timeout 5m ./...  # integration tests (requires network)
```

Integration tests download real packages from GitHub and NuGet, so they need network access and take longer to run. They're behind the `integration` build tag.

## Pull Requests

- Keep PRs focused on a single change
- Include tests for new functionality
- Make sure `go test ./...` passes before submitting
- Update documentation if you're adding or changing config options

## Adding a New Package Type

If you want to add a new package source type:

1. Add the type constant to `internal/config/config.go` and update `Validate()`
2. Create a handler in `internal/packager/` (see `git.go` or `nuget.go` for examples)
3. Wire it into `processPackage()` in `internal/packager/packager.go`
4. Add cache support in `internal/packager/cache.go` if the source is downloadable
5. Add unit tests and an integration test

## Reporting Bugs

Open an issue with:

- What you expected to happen
- What actually happened
- Your `upstream-packages.json` config (redact private URLs if needed)
- Output from running with `-verbose`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
