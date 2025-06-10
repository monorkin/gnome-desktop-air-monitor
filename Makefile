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
RELEASE_FILES=$(shell find $(BUILD_DIR) -type f -name "$(BINARY_NAME)-linux-*" -print) \
							$(shell find $(BUILD_DIR) -type f -name "$(BINARY_NAME)-*.png" -print) \
							icon.svg

# Go build flags
LDFLAGS=-ldflags "-X github.com/monorkin/gnome-desktop-air-monitor/internal/version.Version=$(VERSION)"
BUILD_FLAGS=-trimpath

# Default target
.DEFAULT_GOAL := build

# Targets
.PHONY: help build build-debug run clean test test-verbose test-race test-coverage \
        fmt vet lint deps tidy check install uninstall dev all debug-info \
        install-extension uninstall-extension reload-extension restart-gnome-shell \
        convert-icon multiarch-build release pre-release-check

## install: Installs the app
install: build convert-icon
	@echo "Installing $(BINARY_NAME)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installing icon..."
	sudo mkdir -p /usr/share/icons/hicolor/scalable/apps
	sudo cp icon.svg /usr/share/icons/hicolor/scalable/apps/$(BINARY_NAME).svg
	sudo mkdir -p /usr/share/icons/hicolor/48x48/apps
	sudo cp $(BUILD_DIR)/$(BINARY_NAME)-48.png /usr/share/icons/hicolor/48x48/apps/$(BINARY_NAME).png
	sudo mkdir -p /usr/share/icons/hicolor/64x64/apps
	sudo cp $(BUILD_DIR)/$(BINARY_NAME)-64.png /usr/share/icons/hicolor/64x64/apps/$(BINARY_NAME).png
	sudo mkdir -p /usr/share/icons/hicolor/128x128/apps
	sudo cp $(BUILD_DIR)/$(BINARY_NAME)-128.png /usr/share/icons/hicolor/128x128/apps/$(BINARY_NAME).png
	sudo mkdir -p /usr/share/icons/hicolor/256x256/apps
	sudo cp $(BUILD_DIR)/$(BINARY_NAME)-256.png /usr/share/icons/hicolor/256x256/apps/$(BINARY_NAME).png
	sudo gtk-update-icon-cache /usr/share/icons/hicolor/ 2>/dev/null || true
	@echo "Installing desktop file..."
	sudo cp $(BINARY_NAME).desktop /usr/share/applications/$(BINARY_NAME).desktop
	sudo update-desktop-database
	@$(MAKE) install-extension

## uninstall: Uninstall the app
uninstall: uninstall-extension
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Removing icons..."
	sudo rm -f /usr/share/icons/hicolor/scalable/apps/$(BINARY_NAME).svg
	sudo rm -f /usr/share/icons/hicolor/48x48/apps/$(BINARY_NAME).png
	sudo rm -f /usr/share/icons/hicolor/64x64/apps/$(BINARY_NAME).png
	sudo rm -f /usr/share/icons/hicolor/128x128/apps/$(BINARY_NAME).png
	sudo rm -f /usr/share/icons/hicolor/256x256/apps/$(BINARY_NAME).png
	sudo gtk-update-icon-cache /usr/share/icons/hicolor/ 2>/dev/null || true
	@echo "Removing desktop file..."
	sudo rm -f /usr/share/applications/$(BINARY_NAME).desktop
	sudo update-desktop-database

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

## multiarch-build: Build the application for multiple architectures (ARM64 and x86_64)
multiarch-build: deps internal/licenses/THIRD_PARTY_LICENSES internal/licenses/LICENSE
	@echo "Building $(BINARY_NAME) for multiple architectures..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux AMD64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "Building for Linux ARM64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "Building for Linux ARM (32-bit)..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7 $(MAIN_PATH)
	@echo "Multi-architecture build complete:"
	@ls -la $(BUILD_DIR)/$(BINARY_NAME)-linux-*

internal/licenses/THIRD_PARTY_LICENSES: THIRD_PARTY_LICENSES
	cp THIRD_PARTY_LICENSES internal/licenses/THIRD_PARTY_LICENSES

internal/licenses/LICENSE:
	cp LICENSE internal/licenses/LICENSE

THIRD_PARTY_LICENSES:
	go install github.com/google/go-licenses@latest
	go run ./util/bundle_licenses.go

## convert-icon: Convert SVG icon to different PNG sizes
convert-icon:
	@echo "Converting icon from SVG to PNG..."
	@mkdir -p $(BUILD_DIR)
	@if [ ! -f icon.svg ]; then echo "Warning: icon.svg not found, skipping icon conversion"; exit 0; fi
	@if ! command -v convert >/dev/null 2>&1; then echo "Warning: ImageMagick (convert) not found, skipping icon conversion"; exit 0; fi
	convert icon.svg -resize 48x48 $(BUILD_DIR)/$(BINARY_NAME)-48.png
	convert icon.svg -resize 64x64 $(BUILD_DIR)/$(BINARY_NAME)-64.png
	convert icon.svg -resize 128x128 $(BUILD_DIR)/$(BINARY_NAME)-128.png
	convert icon.svg -resize 256x256 $(BUILD_DIR)/$(BINARY_NAME)-256.png

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
	@echo "Go Version: $(shell go version)"
	@echo "Build Dir: $(BUILD_DIR)"
	@echo "Extension UUID: $(EXTENSION_UUID)"
	@echo "Extension Dir: $(EXTENSION_DIR)"
	@echo "Release Files: $(RELEASE_FILES)"

## release: Create and publish a new release with multi-architecture binaries
release: convert-icon multiarch-build
	@echo "Checking if gh is installed and configured..."
	@if ! command -v gh >/dev/null 2>&1; then \
		echo "❌ GitHub CLI (gh) is required for releases."; \
		echo "Install with: sudo apt install gh  # or  brew install gh"; \
		exit 1; \
	fi
	@if ! gh auth status >/dev/null 2>&1; then \
		echo "❌ GitHub CLI not authenticated."; \
		echo "Run: gh auth login"; \
		exit 1; \
	fi
	@echo "Checking for uncommited changes..."
	@if git status --porcelain | grep -q .; then \
		echo "❌ Repository has uncommitted changes:"; \
		git status --porcelain; \
		echo "Please commit or stash changes before releasing."; \
		exit 1; \
	fi
	@echo "Checking repository status for release..."
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "❌ No valid version found. Please create a tag first with: git tag v1.0.0"; \
		exit 1; \
	fi
	@echo "Checking if all release files are present..."
	@ for file in $(RELEASE_FILES); do \
		if [ ! -f "$$file" ]; then \
			echo "❌ Missing release file: $$file"; \
			exit 1; \
		fi; \
	done
	@echo "✅ Repository is ready for release $(RELEASE_TAG)"
	@echo "Pushing tag to GitHub..."
	git push origin $(VERSION)
	@echo "Extracting release notes for $(VERSION)..."
	@awk '/^## \[$(VERSION)\]/{flag=1;next}/^## \[/{flag=0}flag' CHANGELOG.md > /tmp/release_notes_$(VERSION).md
	@echo "Creating GitHub release..."
	gh release create $(VERSION) \
		--title "$(VERSION)" \
		--notes-file /tmp/release_notes_$(VERSION).md \
		--verify-tag \
		$(RELEASE_FILES)
	@rm -f /tmp/release_notes_$(VERSION).md
	@echo "✅ Release $(VERSION) created successfully!"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
