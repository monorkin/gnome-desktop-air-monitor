# GNOME Desktop Air Monitor - Claude Context

## Project Overview
This is a GNOME desktop application for monitoring air quality data from Awair devices. It includes:
- A GTK4/Adwaita desktop application built in Go
- A GNOME Shell extension for system tray integration
- A CLI interface for device and measurement management
- Database storage for measurements using SQLite
- Network device discovery using Zeroconf/mDNS

## Technology Stack
- **Language**: Go 1.23+
- **GUI Framework**: GTK4 with gotk4 bindings and Adwaita styling
- **Database**: SQLite with GORM ORM
- **Network**: Zeroconf for device discovery, HTTP client for Awair API
- **CLI**: Cobra framework
- **Build System**: Make
- **Packaging**: Manual installation with install.sh script

## Project Structure
```
├── cmd/gnome-desktop-air-monitor/  # Main application entry point
├── internal/
│   ├── app/                        # GTK4 application logic
│   ├── cli/                        # CLI commands (device, measurement, root)
│   ├── config/                     # Configuration and storage
│   ├── database/                   # Database setup and migrations
│   ├── globals/                    # Global initialization
│   ├── licenses/                   # Bundled license files
│   └── models/                     # Data models (device, measurement)
├── awair/api/                      # Awair device API client
├── shell_extension/                # GNOME Shell extension
└── util/                          # Build utilities
```

## Development Commands
```bash
# Build and run with debug symbols
make dev

# Run with CLI arguments
make dev ARGS="device ls"

# Build for production
make build

# Build with debug symbols (for testing compilation)
make build-debug

# Run tests with coverage
make test

# Format and check code
make fmt
make vet
make check

# Install locally
make install

# Uninstall
make uninstall

# Test shell extension
make shell-extension-dev

# Multi-architecture build
make multiarch-build

# Create release
make release
```

## Key Components

### Database
- Uses SQLite with GORM
- Migrations in `internal/database/migrations/`
- Models: Device, Measurement
- Auto-migration on startup

### API Integration
- Awair Element device support (tested)
- REST API client in `awair/api/`
- Device discovery via mDNS/Zeroconf
- Measurement collection and storage

### GUI Application
- GTK4 with Adwaita design
- Device management pages
- Settings configuration
- D-Bus service for shell extension communication

### Shell Extension
- JavaScript-based GNOME Shell extension
- Shows air quality in top bar
- Communicates with main app via D-Bus

## Installation Methods
1. **Pre-compiled binary**: `curl -sSL https://raw.githubusercontent.com/monorkin/gnome-desktop-air-monitor/main/install.sh | bash`
2. **From source**: `make install`

## Dependencies
- Go 1.23+
- GTK4/GDK 4.0+
- ImageMagick (for icon conversion)
- GNOME Shell (for extension)

## Testing
- Run tests: `make test`
- Coverage report generated as `coverage.html`
- Race condition detection enabled

## Build Information
- Version from git tags
- Build-time metadata embedded in binary
- Cross-platform builds for Linux (amd64, arm64, armv7)

## License Management
- MIT License
- Third-party licenses bundled automatically via `go-licenses`
- License files embedded in binary during build

## Configuration
- Settings stored in user config directories
- Database in user data directories
- Platform-specific storage paths

## Debugging
- Debug build: `make build-debug`
- Verbose logging: `--verbose` flag
- Extension development: `make shell-extension-dev`
