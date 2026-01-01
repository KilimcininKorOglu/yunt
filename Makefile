# Yunt Makefile
# Build automation for the Yunt mail server

# Variables
BINARY_NAME := yunt
BUILD_DIR := bin
DIST_DIR := dist
MAIN_PATH := ./cmd/yunt

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS := -ldflags "-s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(BUILD_DATE)"

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOVET := $(GO) vet
GOFMT := gofmt
GOIMPORTS := goimports

# Cross-compilation targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Default target
.DEFAULT_GOAL := build

# Phony targets
.PHONY: all build build-all build-full test test-coverage test-race lint lint-fix fmt vet tidy clean run help
.PHONY: release release-linux release-darwin release-windows install dev deps check
.PHONY: web-install web-dev web-build web-lint web-check web-clean

## all: Build, lint, and test
all: lint test build

## build: Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-full: Build web UI and Go binary together
build-full: web-build build
	@echo "Full build complete: Web UI embedded in $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Build for all target platforms
build-all: clean
	@echo "Building $(BINARY_NAME) $(VERSION) for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output=$(DIST_DIR)/$(BINARY_NAME)-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then output=$$output.exe; fi; \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $$output $(MAIN_PATH); \
	done
	@echo "All binaries built in $(DIST_DIR)/"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -v -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated: $(BUILD_DIR)/coverage.html"

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -v -race ./...

## lint: Run static analysis with golangci-lint
lint: vet
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "Skipping lint..."; \
	fi

## lint-fix: Run linter and auto-fix issues
lint-fix:
	@echo "Running linter with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

## fmt: Format code with gofmt and goimports
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@if command -v goimports >/dev/null 2>&1; then \
		$(GOIMPORTS) -w .; \
	fi

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## tidy: Tidy and verify go modules
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy
	$(GO) mod verify

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## run: Build and run the server
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) serve

## dev: Run with live reload (requires air)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Install with:"; \
		echo "  go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) $(MAIN_PATH)

## check: Run all checks (vet, lint, test)
check: vet lint test

## release: Build versioned release binaries
release:
	@echo "Building release $(VERSION)..."
	@./scripts/release.sh $(VERSION)

## release-linux: Build release for Linux platforms
release-linux:
	@echo "Building Linux releases..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

## release-darwin: Build release for macOS platforms
release-darwin:
	@echo "Building macOS releases..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

## release-windows: Build release for Windows platforms
release-windows:
	@echo "Building Windows releases..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

## help: Show this help message
help:
	@echo "Yunt - Development Mail Server"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'

# Web UI Targets

## web-install: Install web dependencies
web-install:
	@echo "Installing web dependencies..."
	cd web && npm install

## web-dev: Start web development server
web-dev:
	@echo "Starting web development server..."
	cd web && npm run dev

## web-build: Build web for production (outputs to webui/dist for go:embed)
web-build: web-clean-dist
	@echo "Building web for production..."
	cd web && npm run build
	@echo "Web UI built to webui/dist/"

## web-clean-dist: Clean webui/dist directory (preserving .gitkeep)
web-clean-dist:
	@echo "Cleaning webui/dist..."
	@find webui/dist -type f ! -name '.gitkeep' -delete 2>/dev/null || true
	@find webui/dist -type d -empty -delete 2>/dev/null || true

## web-lint: Lint web code
web-lint:
	@echo "Linting web code..."
	cd web && npm run lint && npm run format:check

## web-check: Run web type checking
web-check:
	@echo "Running web type checking..."
	cd web && npm run check

## web-clean: Clean web build artifacts
web-clean:
	@echo "Cleaning web build artifacts..."
	rm -rf web/build web/.svelte-kit web/node_modules
	@find webui/dist -type f ! -name '.gitkeep' -delete 2>/dev/null || true
	@find webui/dist -type d -empty -delete 2>/dev/null || true
