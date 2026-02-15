# Release Guide

This document describes how to release sandbox0 Go SDK.

## Prerequisites

1. Git push access to the repository
2. Go 1.21+ installed
3. golangci-lint installed (or it will be installed automatically)

## Release Process

### 1. Ensure tests pass

```bash
make check
```

### 2. Release to Go Proxy

```bash
make release v=X.Y.Z
```

This command will:
1. Run build, tests, and lint checks
2. Create a git tag `vX.Y.Z`
3. Push the tag to origin
4. Go proxy will automatically index the new version

### 3. Verify the release

Visit https://pkg.go.dev/github.com/sandbox0-ai/sdk-go to confirm the new version is indexed.

You can also check:
```bash
go list -m -versions github.com/sandbox0-ai/sdk-go
```

## Available Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Verify build compiles |
| `make test` | Run unit tests |
| `make lint` | Run golangci-lint |
| `make check` | Run build + test + lint |
| `make set-version v=X.Y.Z` | Create local git tag |
| `make tag v=X.Y.Z` | Create and push git tag |
| `make publish v=X.Y.Z` | Run checks and push tag |
| `make release v=X.Y.Z` | Same as publish |

## Manual Release Steps

If you need more control:

```bash
# 1. Run checks
make check

# 2. Create tag
make set-version v=X.Y.Z

# 3. Push tag
git push origin vX.Y.Z
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible new features
- **PATCH**: Backward-compatible bug fixes

## Usage After Release

Users can install the SDK with:

```bash
go get github.com/sandbox0-ai/sdk-go@vX.Y.Z
```
