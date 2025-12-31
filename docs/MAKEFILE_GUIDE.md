# Makefile Guide

This document provides detailed information about the Makefile targets available in the kfs-flow-merge project.

## Quick Reference

```bash
make help          # Display all available targets
make build         # Build the CLI tool
make test          # Run all tests
make verify        # Run all checks before committing
make ci            # Run full CI pipeline
```

## Build Targets

### `make build`
Builds the CLI tool and places the binary in `bin/kfsmerge`.

**Example:**
```bash
make build
./bin/kfsmerge -schema examples/schema.json -a examples/request.json -b examples/template.json
```

### `make install`
Installs the CLI tool to `$GOPATH/bin`, making it available globally.

**Example:**
```bash
make install
kfsmerge -schema schema.json -a request.json -b template.json
```

## Testing Targets

### `make test`
Runs all tests with race detection enabled. This is the standard test target.

**Flags:**
- `-race`: Enables race detector
- `-count=1`: Disables test caching to ensure fresh runs

**Example:**
```bash
make test
```

### `make test-verbose`
Runs tests with verbose output, showing each test as it runs.

**Example:**
```bash
make test-verbose
```

### `make test-short`
Runs only short tests, skipping long-running tests marked with `testing.Short()`.

**Example:**
```bash
make test-short
```

### `make test-coverage`
Runs tests with coverage analysis and generates an HTML coverage report.

**Output:**
- `coverage.out`: Coverage profile
- `coverage.html`: HTML coverage report (opens in browser)

**Example:**
```bash
make test-coverage
open coverage.html  # View coverage report
```

### `make bench`
Runs benchmark tests to measure performance.

**Example:**
```bash
make bench
```

## Code Quality Targets

### `make lint`
Runs static analysis tools to catch potential issues:
- `go vet`: Official Go static analyzer
- `golangci-lint`: Comprehensive linter (if installed)
- `staticcheck`: Advanced static analyzer (if installed)

**Example:**
```bash
make lint
```

**Installing optional tools:**
```bash
# golangci-lint
brew install golangci-lint  # macOS
# or
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### `make fmt`
Formats all Go code using `gofmt` and `goimports` (if available).

**Example:**
```bash
make fmt
```

### `make fmt-check`
Checks if code is properly formatted without making changes. Useful in CI pipelines.

**Example:**
```bash
make fmt-check
```

## Dependency Management

### `make deps`
Downloads and verifies all Go module dependencies.

**Example:**
```bash
make deps
```

### `make tidy`
Tidies `go.mod` and `go.sum`, removing unused dependencies and adding missing ones.

**Example:**
```bash
make tidy
```

## Cleanup Targets

### `make clean`
Removes build artifacts and clears test cache. Does NOT remove module cache.

**Removes:**
- `bin/` directory
- `coverage.out` and `coverage.html`
- Go build cache
- Go test cache

**Example:**
```bash
make clean
```

### `make clean-all`
Deep clean that also removes the module cache. Use with caution as it will require re-downloading all dependencies.

**Example:**
```bash
make clean-all
```

## Composite Targets

### `make verify`
Runs all verification checks in sequence:
1. Format check (`fmt-check`)
2. Linting (`lint`)
3. Tests (`test`)

**Use this before committing code.**

**Example:**
```bash
make verify
```

### `make all`
Complete build pipeline:
1. Clean artifacts (`clean`)
2. Download dependencies (`deps`)
3. Build CLI (`build`)
4. Run tests (`test`)

**Example:**
```bash
make all
```

### `make ci`
CI/CD pipeline optimized for continuous integration:
1. Download dependencies (`deps`)
2. Format check (`fmt-check`)
3. Linting (`lint`)
4. Tests with coverage (`test-coverage`)

**Example:**
```bash
make ci
```

## Utility Targets

### `make run-example`
Builds the CLI and runs an example merge using the files in `examples/`.

**Example:**
```bash
make run-example
```

### `make migrate-check`
Verifies that the v2.0.0 migration script is executable.

**Example:**
```bash
make migrate-check
```

### `make version`
Displays version information including:
- Version number
- Git commit hash
- Build timestamp
- Go version

**Example:**
```bash
make version
```

## Customization

### Environment Variables

You can override default variables:

```bash
# Custom binary name
make build BINARY_NAME=my-merger

# Custom version
make build VERSION=2.1.0

# Custom Go flags
make build GOFLAGS="-tags=debug"
```

### Adding Custom Targets

To add your own targets, edit the `Makefile` and add:

```makefile
## my-target: Description of what it does
my-target:
	@echo "Running my custom target"
	# Your commands here
```

Don't forget to add it to `.PHONY`:

```makefile
.PHONY: my-target
```

## Common Workflows

### Before Committing
```bash
make verify
```

### CI/CD Pipeline
```bash
make ci
```

### Local Development
```bash
# Build and test
make build test

# Or use the all target
make all
```

### Release Preparation
```bash
# Clean build
make clean

# Run full verification
make verify

# Build final binary
make build

# Check version
make version
```

## Troubleshooting

### Tests are cached
```bash
# Tests use -count=1 to disable caching, but if needed:
go clean -testcache
make test
```

### Module issues
```bash
# Clean and re-download
make clean-all
make deps
```

### Format issues
```bash
# Auto-fix formatting
make fmt

# Then verify
make fmt-check
```

### Lint failures
```bash
# Run lint to see issues
make lint

# Fix issues manually, then verify
make verify
```

