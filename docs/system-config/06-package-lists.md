# Package Lists

## System Summary

- **Total Installed Packages**: 2211
- **Package Manager**: pacman (Arch Linux)

## Core System Packages

### Kernel
- `linux-cachyos-bmq-lto 6.18.0-1` - CachyOS optimized kernel with BMQ scheduler and LTO

### Audio Stack (See [Audio Configuration](./03-audio-config.md))
- `pipewire 1:1.4.9-2.1`
- `wireplumber 0.5.12-1.1`
- `pipewire-pulse 1:1.4.9-2.1`
- `pipewire-alsa 1:1.4.9-2.1`
- `pipewire-audio 1:1.4.9-2.1`
- `pipewire-jack 1:1.4.9-2.1`

### Window Manager
- `hyprland 0.52.2-2`

### Graphics
- `mesa 1:25.3.1-3`
- `vulkan-radeon 1:25.3.1-3`
- `vulkan-mesa-implicit-layers 1:25.3.1-3`
- `opencl-mesa 1:25.3.1-3`
- `xf86-video-amdgpu 25.0.0-1.1`

## Browsers

- `zen-browser-bin 1.17.12b-1` - Primary browser (Firefox-based)
- `cachy-browser 137.0.2-5` - CachyOS browser
- `firefox 146.0-1.1` - Standard Firefox
- `firefoxpwa 2.17.2-1.2` - Firefox PWA support
- `plasma-browser-integration 6.5.4-1.1` - KDE browser integration

## Desktop Environment Components

### Illogical Impulse Packages (Dotfiles Related)
- `illogical-impulse-hyprland 1.0-4`
- `illogical-impulse-quickshell-git 0.2.0.r34.gdb1777c-1`
- `illogical-impulse-fonts-themes 1.0-3`
- `illogical-impulse-bibata-modern-classic-bin 2.0.6-1`
- `illogical-impulse-audio 1.0-2`
- `illogical-impulse-basic 1.0-2`
- `illogical-impulse-portal 1.0-2`
- `illogical-impulse-python 1.1-4`
- `illogical-impulse-kde 1.0-2`
- `illogical-impulse-microtex-git r494.0e3707f-2`

### Widgets and UI
- `power-profiles-daemon 0.30-2`

## Development Tools

### Runtime
- `bun` - JavaScript runtime (used by daemira project)
- `nodejs` - Node.js (likely installed)

### Build Tools
- Various compiler and build tools (part of base-devel)

## GUI Toolkits

- **Qt5/Qt6 packages**: ~105 packages installed
- **GTK packages**: Various (count not specified)

## System Utilities

- `alsa-utils 1.2.14-1.1`
- `mesa-utils 9.0.0-7.1`
- `vulkan-tools 1.4.328.1-1.1`

## Firmware

- `linux-firmware-amdgpu 1:20251125-2`
- `linux-firmware-nvidia 1:20251125-2` (installed but not in use)

## Package Management Notes

- Using standard Arch Linux `pacman` package manager
- CachyOS repositories provide optimized packages
- Custom kernel with BMQ scheduler and LTO optimizations

## Package Categories Breakdown

### Audio: ~20 packages
### Graphics/GPU: ~15 packages  
### Browsers: 5 packages
### Desktop/WM: ~15 packages (illogical-impulse + hyprland)
### Qt/GTK: ~105+ packages
### Development: Various (TypeScript, Node.js ecosystem)

## Missing Packages (Potential Issues)

- `pipewire-v4l2` - For camera support (WirePlumber reports missing libcamera plugin)
- `upower` - For battery monitoring (WirePlumber reports UPower errors)


