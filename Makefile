# Makefile for kfs-flow-merge
# Go-based JSON Schema merge library with CLI tool

# Variables
BINARY_NAME=kfsmerge
BINARY_PATH=bin/$(BINARY_NAME)
CMD_PATH=./cmd/kfsmerge
MAIN_PACKAGE=github.com/nbcuni/kfs-flow-merge
GO=go
GOFLAGS=
LDFLAGS=
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Build information
VERSION?=2.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Colors for output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m
COLOR_BLUE=\033[34m

# Default target
.DEFAULT_GOAL := help

# Phony targets (not actual files)
.PHONY: help clean clean-all build test test-verbose test-short test-coverage install lint fmt fmt-check deps tidy verify all run-example migrate-check bench ci version

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)kfs-flow-merge Makefile$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Available targets:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)build$(COLOR_RESET)          - Build the CLI tool"
	@echo "  $(COLOR_GREEN)test$(COLOR_RESET)           - Run all tests with race detection"
	@echo "  $(COLOR_GREEN)test-verbose$(COLOR_RESET)   - Run tests with verbose output"
	@echo "  $(COLOR_GREEN)test-short$(COLOR_RESET)     - Run short tests only"
	@echo "  $(COLOR_GREEN)test-coverage$(COLOR_RESET)  - Run tests with coverage report"
	@echo "  $(COLOR_GREEN)bench$(COLOR_RESET)          - Run benchmarks"
	@echo "  $(COLOR_GREEN)install$(COLOR_RESET)        - Install CLI tool to GOPATH/bin"
	@echo "  $(COLOR_GREEN)clean$(COLOR_RESET)          - Remove build artifacts and caches"
	@echo "  $(COLOR_GREEN)lint$(COLOR_RESET)           - Run static analysis tools"
	@echo "  $(COLOR_GREEN)fmt$(COLOR_RESET)            - Format code with gofmt and goimports"
	@echo "  $(COLOR_GREEN)fmt-check$(COLOR_RESET)      - Check if code is formatted correctly"
	@echo "  $(COLOR_GREEN)deps$(COLOR_RESET)           - Download and verify dependencies"
	@echo "  $(COLOR_GREEN)tidy$(COLOR_RESET)           - Tidy and verify go.mod and go.sum"
	@echo "  $(COLOR_GREEN)verify$(COLOR_RESET)         - Run all verification checks (fmt, lint, test)"
	@echo "  $(COLOR_GREEN)all$(COLOR_RESET)            - Clean, build, and test"
	@echo "  $(COLOR_GREEN)ci$(COLOR_RESET)             - Run CI pipeline (fmt-check, lint, test-coverage)"
	@echo "  $(COLOR_GREEN)run-example$(COLOR_RESET)    - Run example merge with provided schemas"
	@echo "  $(COLOR_GREEN)migrate-check$(COLOR_RESET)  - Verify migration script is executable"
	@echo "  $(COLOR_GREEN)version$(COLOR_RESET)        - Display version information"
	@echo "  $(COLOR_GREEN)help$(COLOR_RESET)           - Display this help message"
	@echo ""
	@echo "$(COLOR_BOLD)Examples:$(COLOR_RESET)"
	@echo "  make build              # Build the CLI tool"
	@echo "  make test               # Run all tests"
	@echo "  make verify             # Run all checks before committing"
	@echo "  make ci                 # Run full CI pipeline"
	@echo "  make install            # Install CLI globally"
	@echo ""

## clean: Remove build artifacts, test cache, and temporary files
clean:
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BINARY_PATH)
	@rm -rf bin/
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@$(GO) clean -cache -testcache
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

## clean-all: Remove all build artifacts and module cache (use with caution)
clean-all:
	@echo "$(COLOR_YELLOW)Cleaning all artifacts including module cache...$(COLOR_RESET)"
	@rm -rf $(BINARY_PATH)
	@rm -rf bin/
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@$(GO) clean -cache -testcache -modcache
	@echo "$(COLOR_GREEN)✓ Deep clean complete$(COLOR_RESET)"

## build: Build the CLI tool
build:
	@echo "$(COLOR_YELLOW)Building $(BINARY_NAME)...$(COLOR_RESET)"
	@mkdir -p bin
	@$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH) $(CMD_PATH)
	@echo "$(COLOR_GREEN)✓ Build complete: $(BINARY_PATH)$(COLOR_RESET)"

## test: Run all tests with race detection
test:
	@echo "$(COLOR_YELLOW)Running tests with race detection...$(COLOR_RESET)"
	@$(GO) test -race -count=1 ./...
	@echo "$(COLOR_GREEN)✓ All tests passed$(COLOR_RESET)"

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "$(COLOR_YELLOW)Running tests (verbose)...$(COLOR_RESET)"
	@$(GO) test -v -race -count=1 ./...

## test-short: Run short tests only
test-short:
	@echo "$(COLOR_YELLOW)Running short tests...$(COLOR_RESET)"
	@$(GO) test -short -race ./...
	@echo "$(COLOR_GREEN)✓ Short tests passed$(COLOR_RESET)"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "$(COLOR_YELLOW)Running tests with coverage...$(COLOR_RESET)"
	@$(GO) test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(COLOR_GREEN)✓ Coverage report generated: $(COVERAGE_HTML)$(COLOR_RESET)"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total coverage: " $$3}'

## install: Install the CLI tool to GOPATH/bin
install:
	@echo "$(COLOR_YELLOW)Installing $(BINARY_NAME)...$(COLOR_RESET)"
	@$(GO) install $(GOFLAGS) $(CMD_PATH)
	@echo "$(COLOR_GREEN)✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)$(COLOR_RESET)"

## lint: Run static analysis tools
lint:
	@echo "$(COLOR_YELLOW)Running static analysis...$(COLOR_RESET)"
	@$(GO) vet ./...
	@echo "$(COLOR_BLUE)go vet: passed$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "$(COLOR_BLUE)golangci-lint: passed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not found, skipping (install: https://golangci-lint.run/usage/install/)$(COLOR_RESET)"; \
	fi
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
		echo "$(COLOR_BLUE)staticcheck: passed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ staticcheck not found, skipping (install: go install honnef.co/go/tools/cmd/staticcheck@latest)$(COLOR_RESET)"; \
	fi
	@echo "$(COLOR_GREEN)✓ Lint checks complete$(COLOR_RESET)"

## fmt: Format code with gofmt and goimports
fmt:
	@echo "$(COLOR_YELLOW)Formatting code...$(COLOR_RESET)"
	@gofmt -s -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
		echo "$(COLOR_BLUE)goimports: applied$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ goimports not found, using gofmt only (install: go install golang.org/x/tools/cmd/goimports@latest)$(COLOR_RESET)"; \
	fi
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

## fmt-check: Check if code is formatted correctly
fmt-check:
	@echo "$(COLOR_YELLOW)Checking code formatting...$(COLOR_RESET)"
	@test -z "$$(gofmt -l .)" || (echo "$(COLOR_YELLOW)Files need formatting:$(COLOR_RESET)" && gofmt -l . && exit 1)
	@echo "$(COLOR_GREEN)✓ Code is properly formatted$(COLOR_RESET)"

## deps: Download and verify dependencies
deps:
	@echo "$(COLOR_YELLOW)Downloading dependencies...$(COLOR_RESET)"
	@$(GO) mod download
	@$(GO) mod verify
	@echo "$(COLOR_GREEN)✓ Dependencies downloaded and verified$(COLOR_RESET)"

## tidy: Tidy and verify go.mod and go.sum
tidy:
	@echo "$(COLOR_YELLOW)Tidying go.mod...$(COLOR_RESET)"
	@$(GO) mod tidy
	@$(GO) mod verify
	@echo "$(COLOR_GREEN)✓ go.mod tidied$(COLOR_RESET)"

## verify: Run all verification checks (fmt, lint, test)
verify: fmt-check lint test
	@echo "$(COLOR_GREEN)✓ All verification checks passed$(COLOR_RESET)"

## all: Clean, build, and test
all: clean deps build test
	@echo "$(COLOR_GREEN)✓ Build and test complete$(COLOR_RESET)"

## run-example: Run example merge with provided schemas
run-example: build
	@echo "$(COLOR_YELLOW)Running example merge...$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Merging examples/request.json with examples/template.json$(COLOR_RESET)"
	@$(BINARY_PATH) -schema examples/schema.json -a examples/request.json -b examples/template.json -pretty
	@echo "$(COLOR_GREEN)✓ Example complete$(COLOR_RESET)"

## migrate-check: Verify migration script is executable
migrate-check:
	@echo "$(COLOR_YELLOW)Checking migration script...$(COLOR_RESET)"
	@if [ -x scripts/migrate-to-v2.sh ]; then \
		echo "$(COLOR_GREEN)✓ Migration script is executable$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ Migration script is not executable, fixing...$(COLOR_RESET)"; \
		chmod +x scripts/migrate-to-v2.sh; \
		echo "$(COLOR_GREEN)✓ Migration script is now executable$(COLOR_RESET)"; \
	fi

## bench: Run benchmarks
bench:
	@echo "$(COLOR_YELLOW)Running benchmarks...$(COLOR_RESET)"
	@$(GO) test -bench=. -benchmem ./...

## ci: Run CI pipeline (fmt-check, lint, test with coverage)
ci: deps fmt-check lint test-coverage
	@echo "$(COLOR_GREEN)✓ CI pipeline complete$(COLOR_RESET)"

## version: Display version information
version:
	@echo "Version:     $(VERSION)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go Version:  $(shell $(GO) version)"

