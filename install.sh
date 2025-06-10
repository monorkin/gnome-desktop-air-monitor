#!/bin/bash

# GNOME Desktop Air Monitor - Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/monorkin/gnome-desktop-air-monitor/main/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="monorkin/gnome-desktop-air-monitor"
BINARY_NAME="gnome-desktop-air-monitor"
INSTALL_DIR="/usr/local/bin"
DESKTOP_DIR="/usr/share/applications"
ICON_DIR="/usr/share/icons/hicolor"

# Functions
log_info() {
  echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
  echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
  echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
  if [[ $EUID -eq 0 ]]; then
    log_error "This script should not be run as root. Please run as a regular user."
    log_info "The script will prompt for sudo when needed."
    exit 1
  fi
}

# Check for required commands
check_dependencies() {
  local deps=("curl" "jq" "tar")
  local missing=()

  for dep in "${deps[@]}"; do
    if ! command -v "$dep" &>/dev/null; then
      missing+=("$dep")
    fi
  done

  if [ ${#missing[@]} -ne 0 ]; then
    log_error "Missing required dependencies: ${missing[*]}"
    log_info "Please install them and run the script again."
    log_info "On Ubuntu/Debian: sudo apt update && sudo apt install ${missing[*]}"
    log_info "On Fedora: sudo dnf install ${missing[*]}"
    log_info "On Arch: sudo pacman -S ${missing[*]}"
    exit 1
  fi
}

# Detect architecture
detect_arch() {
  local arch
  arch=$(uname -m)

  case $arch in
  x86_64)
    echo "amd64"
    ;;
  aarch64)
    echo "arm64"
    ;;
  armv7l)
    echo "armv7"
    ;;
  armv6l)
    echo "armv7" # Use armv7 for armv6 as fallback
    ;;
  *)
    log_error "Unsupported architecture: $arch"
    log_info "Supported architectures: x86_64, aarch64, armv7l"
    exit 1
    ;;
  esac
}

# Get latest release info from GitHub
get_latest_release() {
  log_info "Fetching latest release information..."
  local release_info
  release_info=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest")

  if ! echo "$release_info" | jq -e '.tag_name' &>/dev/null; then
    log_error "Failed to fetch release information"
    log_info "Please check your internet connection and try again"
    exit 1
  fi

  echo "$release_info"
}

# Download and install binary
install_binary() {
  local release_info="$1"
  local arch="$2"
  local version tag_name download_url binary_name

  tag_name=$(echo "$release_info" | jq -r '.tag_name')
  version=${tag_name#v} # Remove 'v' prefix if present
  binary_name="${BINARY_NAME}-linux-${arch}"

  # Try to find the asset with the correct architecture
  download_url=$(echo "$release_info" | jq -r --arg binary "$binary_name" '.assets[] | select(.name == $binary) | .browser_download_url')

  if [ "$download_url" = "null" ] || [ -z "$download_url" ]; then
    log_error "No binary found for architecture: $arch"
    log_info "Available assets:"
    echo "$release_info" | jq -r '.assets[].name' | sed 's/^/  - /'
    exit 1
  fi

  log_info "Downloading ${BINARY_NAME} ${version} for ${arch}..."
  log_info "Download URL: $download_url"

  # Create temporary directory
  local temp_dir
  temp_dir=$(mktemp -d)
  trap "rm -rf $temp_dir" EXIT

  # Download binary
  if ! curl -sSL "$download_url" -o "$temp_dir/$binary_name"; then
    log_error "Failed to download binary"
    exit 1
  fi

  # Make executable
  chmod +x "$temp_dir/$binary_name"

  # Install binary
  log_info "Installing binary to $INSTALL_DIR..."
  sudo cp "$temp_dir/$binary_name" "$INSTALL_DIR/$BINARY_NAME"

  log_success "Binary installed successfully"
}

# Download and install desktop file and icon
install_desktop_files() {
  local temp_dir
  temp_dir=$(mktemp -d)
  trap "rm -rf $temp_dir" EXIT

  log_info "Downloading desktop file and icon..."

  # Download desktop file
  if curl -sSL "https://raw.githubusercontent.com/$REPO/main/${BINARY_NAME}.desktop" -o "$temp_dir/${BINARY_NAME}.desktop"; then
    sudo cp "$temp_dir/${BINARY_NAME}.desktop" "$DESKTOP_DIR/"
    log_success "Desktop file installed"
  else
    log_warning "Failed to download desktop file"
  fi

  # Download icon
  if curl -sSL "https://raw.githubusercontent.com/$REPO/main/icon.svg" -o "$temp_dir/icon.svg"; then
    # Install SVG icon
    sudo mkdir -p "$ICON_DIR/scalable/apps"
    sudo cp "$temp_dir/icon.svg" "$ICON_DIR/scalable/apps/$BINARY_NAME.svg"

    # Convert to PNG if ImageMagick is available
    if command -v convert &>/dev/null; then
      log_info "Converting icon to PNG formats..."
      for size in 48 64 128 256; do
        sudo mkdir -p "$ICON_DIR/${size}x${size}/apps"
        convert "$temp_dir/icon.svg" -resize "${size}x${size}" "$temp_dir/icon-${size}.png" 2>/dev/null || true
        if [ -f "$temp_dir/icon-${size}.png" ]; then
          sudo cp "$temp_dir/icon-${size}.png" "$ICON_DIR/${size}x${size}/apps/$BINARY_NAME.png"
        fi
      done
    else
      log_info "ImageMagick not found, skipping PNG icon generation"
      log_info "Install ImageMagick for better icon support: sudo apt install imagemagick"
    fi

    log_success "Icon installed"
  else
    log_warning "Failed to download icon"
  fi

  # Update desktop database
  if command -v update-desktop-database &>/dev/null; then
    sudo update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
  fi

  # Update icon cache
  if command -v gtk-update-icon-cache &>/dev/null; then
    sudo gtk-update-icon-cache "$ICON_DIR" 2>/dev/null || true
  fi
}

# Install GNOME Shell extension
install_extension() {
  local temp_dir extension_uuid extension_dir
  temp_dir=$(mktemp -d)
  trap "rm -rf $temp_dir" EXIT

  log_info "Downloading GNOME Shell extension..."

  # Download extension files
  if curl -sSL "https://raw.githubusercontent.com/$REPO/main/shell_extension/extension.js" -o "$temp_dir/extension.js" &&
    curl -sSL "https://raw.githubusercontent.com/$REPO/main/shell_extension/metadata.json" -o "$temp_dir/metadata.json"; then

    # Get extension UUID from metadata
    extension_uuid=$(jq -r '.uuid' "$temp_dir/metadata.json" 2>/dev/null || echo "")

    if [ -n "$extension_uuid" ]; then
      extension_dir="$HOME/.local/share/gnome-shell/extensions/$extension_uuid"
      mkdir -p "$extension_dir"
      cp "$temp_dir/"* "$extension_dir/"
      log_success "GNOME Shell extension installed"
      log_info "Enable it with: gnome-extensions enable $extension_uuid"
      log_info "Or through GNOME Extensions app"
    else
      log_warning "Failed to parse extension UUID"
    fi
  else
    log_warning "Failed to download GNOME Shell extension"
  fi
}

# Main installation function
main() {
  echo
  log_info "GNOME Desktop Air Monitor Installation Script"
  echo

  # Pre-flight checks
  check_root
  check_dependencies

  # Detect system
  local arch
  arch=$(detect_arch)
  log_info "Detected architecture: $arch"

  # Get release info
  local release_info
  release_info=$(get_latest_release)
  local version
  version=$(echo "$release_info" | jq -r '.tag_name')
  log_info "Latest version: $version"

  # Install components
  install_binary "$release_info" "$arch"
  install_desktop_files
  install_extension

  echo
  log_success "Installation completed successfully!"
  echo
  log_info "You can now:"
  log_info "  • Run from terminal: $BINARY_NAME"
  log_info "  • Launch from applications menu"
  log_info "  • Enable GNOME Shell extension for system tray integration"
  echo
  log_info "For uninstallation, run:"
  log_info "  curl -sSL https://raw.githubusercontent.com/$REPO/main/uninstall.sh | bash"
  echo
}

# Run main function
main "$@"

