# System Configuration Documentation

This directory contains comprehensive documentation of the system configuration for future reference and troubleshooting.

## Documentation Index

- [System Overview](./01-system-overview.md) - Basic system information, OS, kernel, hardware
- [Hardware Specifications](./02-hardware-specs.md) - Detailed hardware components (CPU, GPU, RAM, Audio)
- [Audio Configuration](./03-audio-config.md) - PipeWire, WirePlumber, and audio troubleshooting
- [Hyprland Configuration](./04-hyprland-config.md) - Window manager setup and dotfiles
- [Dotfiles References](./05-dotfiles-references.md) - External dotfiles repositories in use
- [Package Lists](./06-package-lists.md) - Key installed packages by category
- [Known Issues & Troubleshooting](./07-known-issues.md) - Current issues and solutions
- [Browser Configuration](./08-browser-config.md) - Zen browser and other browsers
- [OBS Studio Issues](./09-obs-issues.md) - OBS configuration and virtual camera problems
- [Network Configuration](./10-network-config.md) - Network interfaces and Docker
- [Shell Configuration](./11-shell-config.md) - Fish shell setup and user groups
- [Package Management](./12-package-management.md) - pacman, AUR helpers, repositories
- [Filesystem Configuration](./13-filesystem-config.md) - Disk layout and filesystem details

## Quick Reference

- **OS**: CachyOS (Arch-based)
- **Kernel**: 6.18.0-1-cachyos-bmq-lto
- **Window Manager**: Hyprland 0.52.2-2
- **Audio System**: PipeWire 1.4.9-2.1 with WirePlumber 0.5.12-1.1
- **Primary Browser**: Zen Browser 1.17.12b-1
- **Shell**: Fish (with Starship prompt)
- **CPU**: Intel Core i7-10700K (16 threads @ 4.7GHz)
- **GPU**: AMD Radeon RX 5700 XT
- **RAM**: 32GB DDR4 @ 3600 MT/s (4x 8GB G Skill/SK Hynix)
- **Storage**: 931.5GB NVMe SSD (ext4, 53% used)
- **Packages**: 2211 installed

## Diagnostic Scripts

- **`disable-v4l2loopback.sh`** - Disable v4l2loopback module to prevent OBS crashes
- **`fix-obs-virtual-camera.sh`** - Attempt to fix OBS virtual camera issues
- **`fix-v4l2loopback-in-use.sh`** - Fix "module in use" errors
- **`unload-v4l2loopback.sh`** - Unload v4l2loopback module

## Current Known Issues

1. **Audio Not Working** - Zombie WirePlumber process blocking audio (see [Known Issues](./07-known-issues.md))
2. **OBS Virtual Camera Freezes** - v4l2loopback incompatible with kernel 6.18 (see [OBS Issues](./09-obs-issues.md))
3. **Zen Browser Slowness** - Post-update performance issues (see [Browser Config](./08-browser-config.md))

## Last Updated

2025-12-10

