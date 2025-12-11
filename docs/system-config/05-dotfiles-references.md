# Dotfiles References

## Primary Dotfiles Repositories

### 1. Hyprland Dotfiles
- **Repository**: https://github.com/end-4/dots-hyprland
- **Description**: Comprehensive Hyprland configuration with EWW widgets
- **Status**: Active (11.4k stars, 894 forks)
- **License**: GPL-3.0
- **Components**:
  - Hyprland window manager configuration
  - EWW widget system
  - Material Design theme
  - Multiple desktop variants (m3ww, NovelKnock, Hybrid, Windoes)
  - Quickshell integration

### 2. DankMaterialShell
- **Repository**: https://github.com/AvengeMedia/DankMaterialShell
- **Description**: Material Design shell components for Hyprland
- **Status**: Active
- **Integration**: Used alongside dots-hyprland

## Configuration Locations

### Hyprland
- **Config Directory**: `~/.config/hypr/`
- **Main Config**: `~/.config/hypr/hyprland.conf`
- **Subdirectories**:
  - `hyprland/` - Core configuration files
  - `custom/` - User customizations

### Quickshell
- **Config Directory**: `~/.config/quickshell/`
- **Active Config**: `~/.config/quickshell/ii/`
- **Config Variable**: `$qsConfig = ii`

### EWW Widgets
- **Location**: Likely in `~/.config/eww/` (if using EWW from dots-hyprland)
- **Status**: May be part of dots-hyprland setup

## Customization Strategy

The configuration uses a layered approach:
1. **Base Configuration**: From `hyprland/` directory (dotfiles repository)
2. **Custom Configuration**: From `custom/` directory (user overrides)
3. **Custom files take precedence** over base files

## Key Features from Dotfiles

### Material Design Theme
- Modern Material Design 3 inspired interface
- Consistent color scheme
- Smooth animations

### Widget System
- EWW-based widgets
- Quickshell integration for dynamic content
- Clipboard history integration

### Window Management
- Gesture support (3 and 4 finger gestures)
- Workspace swipe navigation
- Window snapping with gaps

## Maintenance Notes

- **Updates**: Dotfiles should be updated via git in their respective directories
- **Custom Changes**: Always make custom changes in `custom/` directory to avoid conflicts
- **Backup**: Consider backing up `custom/` directory before updating dotfiles

## Related Packages from Dotfiles

- `illogical-impulse-hyprland` - Hyprland configuration package
- `illogical-impulse-quickshell-git` - Quickshell integration
- `illogical-impulse-fonts-themes` - Font and theme packages
- `illogical-impulse-bibata-modern-classic-bin` - Cursor theme


