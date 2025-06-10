# Variables
BINARY_NAME=gnome-desktop-air-monitor
MAIN_PATH=./cmd/gnome-desktop-air-monitor
BUILD_DIR=./bin
PKG=github.com/monorkin/gnome-desktop-air-monitor

# Extension variables
EXTENSION_UUID=$(shell if command -v jq >/dev/null 2>&1; then \
	jq -r '.uuid' shell_extension/metadata.json; \
else \
	grep -o '"uuid"[[:space:]]*:[[:space:]]*"[^"]*"' shell_extension/metadata.json | cut -d'"' -f4; \
fi)
EXTENSION_DIR=~/.local/share/gnome-shell/extensions/$(EXTENSION_UUID)

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
BUILD_FLAGS=-trimpath

# Default target
.DEFAULT_GOAL := build

# Targets
.PHONY: help build build-debug run clean test test-verbose test-race test-coverage \
        fmt vet lint deps tidy check install uninstall dev all debug-info \
        install-extension uninstall-extension reload-extension restart-gnome-shell

## install: Installs the app
install: build
	@echo "Installing $(BINARY_NAME)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@$(MAKE) install-extension

## uninstall: Uninstall the app
uninstall: uninstall-extension
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

## install-extension: Install GNOME shell extension
install-extension:
	@echo "Installing GNOME shell extension ($(EXTENSION_UUID))..."
	@mkdir -p $(EXTENSION_DIR)
	@cp -r shell_extension/* $(EXTENSION_DIR)/

## uninstall-extension: Uninstall GNOME shell extension
uninstall-extension:
	@echo "Uninstalling GNOME shell extension ($(EXTENSION_UUID))..."
	gnome-extensions disable $(EXTENSION_UUID) 2>/dev/null || true
	@rm -rf $(EXTENSION_DIR)
	@echo "Extension uninstalled."

## shell-extension-dev: Start a GNOME shell session for extension development
shell-extension-dev:
	dbus-run-session -- gnome-shell --nested --wayland

## build: Build the application for production
build: deps internal/licenses/THIRD_PARTY_LICENSES internal/licenses/LICENSE
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## build-debug: Build the application with debug symbols
build-debug: deps internal/licenses/THIRD_PARTY_LICENSES internal/licenses/LICENSE
	@echo "Building $(BINARY_NAME) with debug symbols..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_PATH)

internal/licenses/THIRD_PARTY_LICENSES: THIRD_PARTY_LICENSES
	cp THIRD_PARTY_LICENSES internal/licenses/THIRD_PARTY_LICENSES

internal/licenses/LICENSE:
	cp LICENSE internal/licenses/LICENSE

THIRD_PARTY_LICENSES:
	go install github.com/google/go-licenses@latest
	go run ./util/bundle_licenses.go

## dev: Build and run the application (pass args with ARGS="...")
dev: build-debug
	@$(MAKE) install-extension
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)-debug $(ARGS)

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/
	go clean -cache -testcache -modcache

## test: Run tests
test:
	@echo "Running tests..."
	go test -coverprofile=coverage.out -race -v ./...
	go tool cover -html=coverage.out -o coverage.html

## fmt: Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## deps: Download dependencies
deps:
	@echo "Tidying dependencies..."
	go mod tidy
	@echo "Downloading dependencies..."
	go mod download

## check: Run all checks (format, vet, lint, test)
check: fmt vet test

## bundle-licenses: Bundle licenses of dependencies
bundle-licenses: deps
	go install github.com/google/go-licenses@latest
	go run ./util/bundle_licenses.go
	cp LICENSE internal/licenses/LICENSE
	cp THIRD_PARTY_LICENSES internal/licenses/THIRD_PARTY_LICENSES

## debug-info: Show build information
debug-info:
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(shell go version)"
	@echo "Build Dir: $(BUILD_DIR)"
	@echo "Extension UUID: $(EXTENSION_UUID)"
	@echo "Extension Dir: $(EXTENSION_DIR)"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
