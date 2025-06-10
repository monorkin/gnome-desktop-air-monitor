#!/bin/bash

# GNOME Desktop Air Monitor - Uninstallation Script
# Usage: curl -sSL https://raw.githubusercontent.com/monorkin/gnome-desktop-air-monitor/main/uninstall.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
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

# Remove binary
remove_binary() {
  if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    log_info "Removing binary from $INSTALL_DIR..."
    sudo rm -f "$INSTALL_DIR/$BINARY_NAME"
    log_success "Binary removed"
  else
    log_info "Binary not found in $INSTALL_DIR"
  fi
}

# Remove desktop file and icons
remove_desktop_files() {
  # Remove desktop file
  if [ -f "$DESKTOP_DIR/$BINARY_NAME.desktop" ]; then
    log_info "Removing desktop file..."
    sudo rm -f "$DESKTOP_DIR/$BINARY_NAME.desktop"
    log_success "Desktop file removed"
  else
    log_info "Desktop file not found"
  fi

  # Remove icons
  log_info "Removing icons..."
  sudo rm -f "$ICON_DIR/scalable/apps/$BINARY_NAME.svg"
  sudo rm -f "$ICON_DIR/48x48/apps/$BINARY_NAME.png"
  sudo rm -f "$ICON_DIR/64x64/apps/$BINARY_NAME.png"
  sudo rm -f "$ICON_DIR/128x128/apps/$BINARY_NAME.png"
  sudo rm -f "$ICON_DIR/256x256/apps/$BINARY_NAME.png"

  # Update desktop database
  if command -v update-desktop-database &>/dev/null; then
    sudo update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
  fi

  # Update icon cache
  if command -v gtk-update-icon-cache &>/dev/null; then
    sudo gtk-update-icon-cache "$ICON_DIR" 2>/dev/null || true
  fi

  log_success "Icons removed"
}

# Remove GNOME Shell extension
remove_extension() {
  local extension_dir_pattern="$HOME/.local/share/gnome-shell/extensions/*air-monitor*"
  local found_extension=false

  # Look for extension directories that might contain our extension
  for ext_dir in $extension_dir_pattern; do
    if [ -d "$ext_dir" ] && [ -f "$ext_dir/metadata.json" ]; then
      # Check if this is our extension by looking for our binary name in metadata
      if grep -q "air.*monitor\|gnome.*desktop.*air" "$ext_dir/metadata.json" 2>/dev/null; then
        local uuid
        uuid=$(basename "$ext_dir")
        log_info "Found extension: $uuid"

        # Disable extension first
        if command -v gnome-extensions &>/dev/null; then
          gnome-extensions disable "$uuid" 2>/dev/null || true
        fi

        # Remove extension directory
        rm -rf "$ext_dir"
        log_success "Extension removed: $uuid"
        found_extension=true
      fi
    fi
  done

  if [ "$found_extension" = false ]; then
    log_info "No GNOME Shell extension found"
  fi
}

# Remove configuration and data (optional)
remove_user_data() {
  local config_dirs=(
    "$HOME/.config/$BINARY_NAME"
    "$HOME/.local/share/$BINARY_NAME"
    "$HOME/.cache/$BINARY_NAME"
  )

  local found_data=false
  for dir in "${config_dirs[@]}"; do
    if [ -d "$dir" ]; then
      found_data=true
      break
    fi
  done

  if [ "$found_data" = true ]; then
    echo
    log_info "User data and configuration directories found:"
    for dir in "${config_dirs[@]}"; do
      if [ -d "$dir" ]; then
        echo "  - $dir"
      fi
    done
    echo
    read -p "Do you want to remove user data and configuration? [y/N]: " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
      for dir in "${config_dirs[@]}"; do
        if [ -d "$dir" ]; then
          rm -rf "$dir"
          log_success "Removed: $dir"
        fi
      done
    else
      log_info "User data preserved"
    fi
  else
    log_info "No user data found"
  fi
}

# Main uninstallation function
main() {
  echo
  log_info "GNOME Desktop Air Monitor Uninstallation Script"
  echo

  # Pre-flight checks
  check_root

  # Check if application is installed
  if [ ! -f "$INSTALL_DIR/$BINARY_NAME" ] && [ ! -f "$DESKTOP_DIR/$BINARY_NAME.desktop" ]; then
    log_warning "GNOME Desktop Air Monitor doesn't appear to be installed"
    log_info "Nothing to uninstall"
    exit 0
  fi

  # Confirm uninstallation
  read -p "Are you sure you want to uninstall GNOME Desktop Air Monitor? [y/N]: " -n 1 -r
  echo

  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "Uninstallation cancelled"
    exit 0
  fi

  # Remove components
  remove_binary
  remove_desktop_files
  remove_extension
  remove_user_data

  echo
  log_success "Uninstallation completed successfully!"
  echo
  log_info "Thank you for using GNOME Desktop Air Monitor"
  echo
}

# Run main function
main "$@"

