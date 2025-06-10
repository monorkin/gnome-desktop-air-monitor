# Changelog

All notable changes to GNOME Desktop Air Monitor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.1] - 2025-06-10

### Added

- **Desktop GTK4/Libadwaita application** for monitoring Awair air quality devices
- **Device discovery and management** via mDNS/Bonjour auto-discovery
- **Real-time air quality monitoring** with automatic data collection every 10 seconds:
  - Temperature (°C)
  - Humidity (%)
  - CO₂ levels (ppm)
  - VOC levels (ppb)
  - PM2.5 particulate matter (μg/m³)
  - Overall air quality score
- **Multi-device support** with individual device pages and management
- GNOME Shell extension for system tray integration
- **Multi-architecture support**:
  - Linux x86_64 (AMD64)
  - Linux ARM64 (aarch64)
  - Linux ARMv7 (32-bit ARM)
- CLI interface
  - Query devices
  - Query measurements
