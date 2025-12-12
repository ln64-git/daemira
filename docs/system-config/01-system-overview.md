# System Overview

## Operating System

- **Distribution**: CachyOS Linux
- **Base**: Arch Linux (rolling release)
- **Build ID**: rolling
- **Kernel**: 6.18.0-1-cachyos-bmq-lto
- **Kernel Package**: linux-cachyos-bmq-lto 6.18.0-1
- **Architecture**: x86_64

## System Information

- **Hostname**: archDuke
- **User**: ln64
- **Shell**: /usr/bin/fish
- **Total Installed Packages**: 2211

## Storage

- **Root Partition**: /dev/nvme0n1p2 (931.2GB)
  - **Used**: 458GB (53%)
  - **Available**: 411GB
  - **Filesystem**: ext4
- **Boot Partition**: /dev/nvme0n1p1 (300MB, mounted at /boot/efi)
- **Additional Disks**:
  - sda: 3.6TB
  - sdb: 3.6TB
  - sdc: 931.5GB
  - sdd: 1.8TB
- **Swap**: 31GB zram (0B used)

## Memory

- **Total RAM**: 32GB (DDR4 @ 3600 MT/s)
- **Used**: ~14GB
- **Free**: ~3.4GB
- **Available**: ~18GB
- **Buff/Cache**: ~14GB
- **Configuration**: 4x 8GB modules (G Skill Intl / SK Hynix)

## System Update Status

Last system update: Recent (kernel errors suggest potential post-update issues)

## Desktop Environment

- **Desktop Shell**: Quickshell DMS
  - Location: `/usr/share/quickshell/dms/`
  - Configuration: `~/.config/quickshell/`
  - Theme: Material Design 3 (dark)
  - Services: 40 QML-based system integration services

- **Compositor**: Hyprland 0.52.2
  - Display Server: Wayland
  - Configuration: `~/.config/hypr/hyprland.conf`
  - IPC: `hyprctl` command-line interface
  - Features: Dynamic tiling, VRR, multi-monitor support

- **Session Management**:
  - Display Manager: SDDM
  - Session Type: Wayland
  - Session Manager: systemd-logind
  - Power Management: power-profiles-daemon

## Key System Services

- **PipeWire**: Active (user service)
- **WirePlumber**: Active (user service)
- **Hyprland**: Running (compositor)
- **Quickshell**: Running (desktop shell)
- **systemd-logind**: Active (session management)
- **power-profiles-daemon**: Active (power profiles)
