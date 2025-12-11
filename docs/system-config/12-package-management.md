# Package Management

## Package Manager

- **Primary**: pacman (Arch Linux package manager)
- **AUR Helpers**: Both `yay` and `paru` installed
- **Total Packages**: 2211 installed packages

## Repository Configuration

### Enabled Repositories (from `/etc/pacman.conf`)

- **cachyos-v3** - CachyOS v3 repository
- **cachyos-core-v3** - CachyOS core packages
- **cachyos-extra-v3** - CachyOS extra packages
- **cachyos** - CachyOS repository
- **core** - Arch Linux core repository
- **extra** - Arch Linux extra repository (likely)
- **community** - Arch Linux community repository (likely)
- **multilib** - 32-bit compatibility (if enabled)

### Mirror Configuration

- CachyOS mirrors configured via:
  - `/etc/pacman.d/cachyos-v3-mirrorlist`
  - `/etc/pacman.d/cachyos-mirrorlist`

## AUR Helpers

### yay
- **Location**: `/usr/bin/yay`
- **Status**: Installed and available

### paru
- **Location**: `/usr/bin/paru`
- **Status**: Installed and available

## Package Categories

### Development Tools
- **Count**: ~171 packages related to development languages
- **Languages**: Python, Node.js, Rust, Go, Java (and related tools)

### Desktop Environment
- **Window Manager**: Hyprland
- **Widget System**: Quickshell
- **Audio**: PipeWire, WirePlumber
- **Browsers**: Zen Browser, Firefox, Cachy Browser

## Common Package Management Commands

```bash
# Update system
sudo pacman -Syu

# Search packages
pacman -Ss <package>

# Install package
sudo pacman -S <package>

# Install from AUR
yay -S <package>
# or
paru -S <package>

# Remove package
sudo pacman -R <package>

# List installed packages
pacman -Q

# Query package info
pacman -Qi <package>

# List files in package
pacman -Ql <package>

# Find package owning file
pacman -Qo <file>

# Check for orphaned packages
pacman -Qdt

# Clean package cache
sudo pacman -Sc
```

## Maintenance

### Regular Maintenance Tasks

```bash
# Update system
sudo pacman -Syu

# Check for orphaned packages
pacman -Qdt

# Clean package cache (keep 2 most recent)
sudo pacman -Sc

# Clean all cache (aggressive)
sudo pacman -Scc

# Rebuild package database (if corrupted)
sudo pacman-db-upgrade
```

### After System Updates

1. Check for broken packages: `pacman -Qkk`
2. Check for orphaned packages: `pacman -Qdt`
3. Rebuild DKMS modules: `sudo dkms autoinstall`
4. Restart services if needed
5. Check system logs: `journalctl -p 3 -xb`

## Package Sources

- **Official Repos**: Arch Linux + CachyOS repositories
- **AUR**: Arch User Repository (via yay/paru)
- **Custom**: `/opt/antigravity` (custom software location)

