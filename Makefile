# go-remove Makefile

# =============================================================================
# Variables
# =============================================================================

# Binary configuration
BINARY_NAME := go-remove
BINARY_PATH := ./bin/$(BINARY_NAME)
MAIN_PACKAGE := .

# Go configuration
GO := go
GO_FLAGS := -v
GO_BUILD_FLAGS := -ldflags="-s -w"
GOPATH := $(shell $(GO) env GOPATH)
GOBIN := $(GOPATH)/bin

# Testing configuration
TEST_FLAGS := -v -race
TEST_TIMEOUT := 2m
COVERAGE_PROFILE := coverage.out
COVERAGE_HTML := coverage.html

# Directory configuration
INTEGRATION_TEST_DIR := ./testing/integration
E2E_TEST_DIR := ./testing/e2e

# Tool configuration
GOLANGCI_LINT := golangci-lint
GOLANGCI_LINT_FLAGS := run ./...
MOCKERY := mockery
GORELEASER := goreleaser

# =============================================================================
# Default Target
# =============================================================================

# Default target runs comprehensive verification
.PHONY: all
all: verify

# =============================================================================
# Code Quality Targets
# =============================================================================

# Run golangci-lint for comprehensive static analysis
# Code must pass without --fix flag per project rules
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) $(GOLANGCI_LINT_FLAGS)

# Run go vet for additional static analysis
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# =============================================================================
# Testing Targets
# =============================================================================

# Run all tests with race detection
.PHONY: test
test:
	@echo "Running all tests with race detection..."
	$(GO) test $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) ./...

# Run unit tests only (white-box tests in package directories)
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	$(GO) test $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) ./cmd/... ./internal/...

# Run integration tests in /testing/integration
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@if [ -d $(INTEGRATION_TEST_DIR) ]; then \
		$(GO) test $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) $(INTEGRATION_TEST_DIR)/...; \
	else \
		echo "No integration tests found in $(INTEGRATION_TEST_DIR)"; \
	fi

# Run E2E tests in /testing/e2e
.PHONY: test-e2e
test-e2e:
	@echo "Running E2E tests..."
	@if [ -d $(E2E_TEST_DIR) ]; then \
		$(GO) test $(TEST_FLAGS) -timeout $(TEST_TIMEOUT) $(E2E_TEST_DIR)/...; \
	else \
		echo "No E2E tests found in $(E2E_TEST_DIR)"; \
	fi

# Generate test coverage report
.PHONY: coverage
coverage:
	@echo "Generating test coverage report..."
	$(GO) test -race -coverprofile=$(COVERAGE_PROFILE) -covermode=atomic ./...
	$(GO) tool cover -func=$(COVERAGE_PROFILE)

# Generate and open HTML coverage report
.PHONY: coverage-html
coverage-html: coverage
	@echo "Generating HTML coverage report..."
	$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Run benchmark tests
.PHONY: benchmark
benchmark:
	@echo "Running benchmark tests..."
	$(GO) test -bench=. -benchmem ./...

# =============================================================================
# Build Targets
# =============================================================================

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(dir $(BINARY_PATH))
	$(GO) build $(GO_BUILD_FLAGS) -o $(BINARY_PATH) $(MAIN_PACKAGE)

# Install the binary to $GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to $(GOBIN)..."
	cp $(BINARY_PATH) $(GOBIN)/$(BINARY_NAME)
	@echo "Installed: $(GOBIN)/$(BINARY_NAME)"

# =============================================================================
# Maintenance Targets
# =============================================================================

# Clean build artifacts and coverage files
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_PATH)
	rm -f $(COVERAGE_PROFILE)
	rm -f $(COVERAGE_HTML)
	@echo "Clean complete"

# Format code using golangci-lint
.PHONY: fmt
fmt:
	@echo "Formatting code with golangci-lint..."
	$(GOLANGCI_LINT) fmt ./...

# Generate mocks using mockery (based on .mockery.yaml)
.PHONY: mocks
mocks:
	@echo "Generating mocks with mockery..."
	$(MOCKERY)

# Download and verify dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify

# Run go mod tidy to clean up dependencies
.PHONY: tidy
tidy:
	@echo "Tidying go modules..."
	$(GO) mod tidy

# =============================================================================
# Verification Targets
# =============================================================================

# Comprehensive verification: lint, vet, and test
.PHONY: verify
verify: lint vet test
	@echo "All verification checks passed!"

# =============================================================================
# Release Targets
# =============================================================================

# Build release binaries using goreleaser (snapshot mode for testing)
.PHONY: release-snapshot
release-snapshot:
	@echo "Building release snapshot..."
	$(GORELEASER) release --snapshot --clean

# Build release binaries using goreleaser (full release)
.PHONY: release
release:
	@echo "Building release..."
	$(GORELEASER) release --clean

# =============================================================================
# Development Helpers
# =============================================================================

# Run the application (build and execute)
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH)

# Check for common issues
.PHONY: check
check: fmt lint vet
	@echo "All checks passed!"

# Display help information
.PHONY: help
help:
	@echo "go-remove Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all               - Default target, runs comprehensive verification"
	@echo "  lint              - Run golangci-lint (must pass without --fix flag)"
	@echo "  vet               - Run go vet for additional static analysis"
	@echo "  test              - Run all tests with race detection"
	@echo "  test-unit         - Run unit tests only (white-box tests)"
	@echo "  test-integration  - Run integration tests in /testing/integration"
	@echo "  test-e2e          - Run E2E tests in /testing/e2e"
	@echo "  coverage          - Generate test coverage report"
	@echo "  coverage-html     - Generate and open HTML coverage report"
	@echo "  benchmark         - Run benchmark tests"
	@echo "  build             - Build the binary"
	@echo "  install           - Install the binary to \$$GOPATH/bin"
	@echo "  clean             - Remove build artifacts and coverage files"
	@echo "  fmt               - Format code using golangci-lint"
	@echo "  mocks             - Generate mocks using mockery"
	@echo "  deps              - Download and verify dependencies"
	@echo "  tidy              - Run go mod tidy"
	@echo "  verify            - Run lint, vet, and test (comprehensive verification)"
	@echo "  release-snapshot  - Build release snapshot with goreleaser"
	@echo "  release           - Build full release with goreleaser"
	@echo "  run               - Build and run the application"
	@echo "  check             - Run fmt, lint, and vet"
	@echo "  help              - Display this help information"
