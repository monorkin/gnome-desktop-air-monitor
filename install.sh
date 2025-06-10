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
  local deps=("curl" "tar")
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

# Get latest release tag from GitHub API (without jq)
get_latest_release_tag() {
  log_info "Fetching latest release information..."
  local response
  response=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest")

  # Extract tag_name using grep and sed (no jq required)
  local tag_name
  tag_name=$(echo "$response" | grep '"tag_name":' | head -n1 | sed 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/')

  if [ -z "$tag_name" ]; then
    log_error "Failed to fetch release tag"
    log_info "Please check your internet connection and try again"
    exit 1
  fi

  echo "$tag_name"
}

# Download and install binary
install_binary() {
  local tag_name="$1"
  local arch="$2"
  local binary_name="${BINARY_NAME}-linux-${arch}"
  local download_url="https://github.com/$REPO/releases/download/$tag_name/$binary_name"

  log_info "Downloading ${BINARY_NAME} ${tag_name} for ${arch}..."

  # Create temporary directory
  local temp_dir
  temp_dir=$(mktemp -d)
  trap "rm -rf $temp_dir" EXIT

  # Download binary
  if ! curl -sSL "$download_url" -o "$temp_dir/$binary_name"; then
    log_error "Failed to download binary from: $download_url"
    log_info "Please check if the release exists for your architecture"
    exit 1
  fi

  # Make executable
  chmod +x "$temp_dir/$binary_name"

  # Install binary
  log_info "Installing binary to $INSTALL_DIR..."
  sudo cp "$temp_dir/$binary_name" "$INSTALL_DIR/$BINARY_NAME"

  log_success "Binary installed successfully"
}

# Download and install icons from release
install_icons() {
  local tag_name="$1"
  local temp_dir
  temp_dir=$(mktemp -d)
  trap "rm -rf $temp_dir" EXIT

  log_info "Downloading icons from release..."

  # Download SVG icon from release
  local svg_url="https://github.com/$REPO/releases/download/$tag_name/icon.svg"
  if curl -sSL "$svg_url" -o "$temp_dir/icon.svg"; then
    sudo mkdir -p "$ICON_DIR/scalable/apps"
    sudo cp "$temp_dir/icon.svg" "$ICON_DIR/scalable/apps/$BINARY_NAME.svg"
    log_success "SVG icon installed"
  else
    log_warning "Failed to download SVG icon from release"
  fi

  # Download PNG icons from release
  for size in 48 64 128 256; do
    local icon_name="${BINARY_NAME}-${size}.png"
    local png_url="https://github.com/$REPO/releases/download/$tag_name/$icon_name"

    if curl -sSL "$png_url" -o "$temp_dir/$icon_name"; then
      sudo mkdir -p "$ICON_DIR/${size}x${size}/apps"
      sudo cp "$temp_dir/$icon_name" "$ICON_DIR/${size}x${size}/apps/$BINARY_NAME.png"
      log_info "Downloaded ${size}x${size} PNG icon"
    else
      log_warning "Failed to download ${size}x${size} PNG icon"
    fi
  done

  # Update icon cache
  if command -v gtk-update-icon-cache &>/dev/null; then
    sudo gtk-update-icon-cache "$ICON_DIR" 2>/dev/null || true
  fi
}

# Download and install desktop file
install_desktop_file() {
  local tag_name="$1"
  local temp_dir
  temp_dir=$(mktemp -d)
  trap "rm -rf $temp_dir" EXIT

  log_info "Downloading desktop file..."

  # Download desktop file from release
  local desktop_url="https://github.com/$REPO/releases/download/$tag_name/${BINARY_NAME}.desktop"
  if curl -sSL "$desktop_url" -o "$temp_dir/${BINARY_NAME}.desktop"; then
    sudo cp "$temp_dir/${BINARY_NAME}.desktop" "$DESKTOP_DIR/"
    log_success "Desktop file installed"
  else
    log_warning "Failed to download desktop file from release"
  fi

  # Update desktop database
  if command -v update-desktop-database &>/dev/null; then
    sudo update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
  fi
}

# Install GNOME Shell extension
install_extension() {
  local tag_name="$1"
  local temp_dir extension_uuid extension_dir
  temp_dir=$(mktemp -d)
  trap "rm -rf $temp_dir" EXIT

  log_info "Downloading GNOME Shell extension..."

  # Download extension archive from release
  local extension_url="https://github.com/$REPO/releases/download/$tag_name/${BINARY_NAME}-shell-extension.tar.gz"
  if curl -sSL "$extension_url" -o "$temp_dir/extension.tar.gz"; then
    # Extract the archive
    tar -xzf "$temp_dir/extension.tar.gz" -C "$temp_dir/"
    
    # Extract UUID from metadata.json (without jq)
    if [ -f "$temp_dir/metadata.json" ]; then
      extension_uuid=$(grep '"uuid":' "$temp_dir/metadata.json" | sed 's/.*"uuid":[[:space:]]*"\([^"]*\)".*/\1/')

      if [ -n "$extension_uuid" ]; then
        extension_dir="$HOME/.local/share/gnome-shell/extensions/$extension_uuid"
        mkdir -p "$extension_dir"
        cp "$temp_dir/"*.js "$temp_dir/"*.json "$extension_dir/" 2>/dev/null || true
        log_success "GNOME Shell extension installed"
        log_info "Enable it with: gnome-extensions enable $extension_uuid"
        log_info "Or through GNOME Extensions app"
      else
        log_warning "Failed to parse extension UUID"
      fi
    else
      log_warning "Extension metadata.json not found in archive"
    fi
  else
    log_warning "Failed to download GNOME Shell extension from release"
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

  # Get latest release tag
  local tag_name
  tag_name=$(get_latest_release_tag)
  log_info "Latest version: $tag_name"

  # Install components
  install_binary "$tag_name" "$arch"
  install_icons "$tag_name"
  install_desktop_file "$tag_name"
  install_extension "$tag_name"

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
