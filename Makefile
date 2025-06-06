# Variables
BINARY_NAME=gnome-desktop-air-monitor
MAIN_PATH=./cmd/gnome-desktop-air-monitor
BUILD_DIR=./bin
PKG=github.com/monorkin/gnome-desktop-air-monitor

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
        fmt vet lint deps tidy check install install-tools dev all debug-info

## install: Installs the app
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

## build: Build the application for production
build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## build-debug: Build the application with debug symbols
build-debug: deps
	@echo "Building $(BINARY_NAME) with debug symbols..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_PATH)

## dev: Build and run the application
dev: build-debug
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)-debug

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

## debug-info: Show build information
debug-info:
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(shell go version)"
	@echo "Build Dir: $(BUILD_DIR)"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
