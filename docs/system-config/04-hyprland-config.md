# Hyprland Configuration

## Hyprland Version

- **Installed Version**: hyprland 0.52.2-2

## Configuration Structure

Configuration is located at: `~/.config/hypr/`

### Main Configuration File

- **Primary Config**: `~/.config/hypr/hyprland.conf`
- **Structure**: Sources multiple files from `hyprland/` and `custom/` directories

### Configuration Files

#### Core Configuration (from `hyprland/` directory)
- `env.conf` - Environment variables
- `execs.conf` - Startup executables
- `general.conf` - General settings (gaps, borders, gestures)
- `rules.conf` - Window rules
- `colors.conf` - Color scheme
- `keybinds.conf` - Key bindings

#### Custom Configuration (from `custom/` directory)
- `env.conf` - Custom environment variables
- `execs.conf` - Custom startup executables
- `general.conf` - Custom general settings
- `rules.conf` - Custom window rules
- `keybinds.conf` - Custom key bindings

#### Additional Files
- `monitors.conf` - Monitor configuration (managed by nwg-displays)
- `workspaces.conf` - Workspace configuration

## Startup Executables

From `hyprland/execs.conf`:

1. **Authentication**:
   - `gnome-keyring-daemon --start --components=secrets`

2. **System Services**:
   - `hypridle` - Idle management
   - `dbus-update-activation-environment --all`
   - `dbus-update-activation-environment --systemd WAYLAND_DISPLAY XDG_CURRENT_DESKTOP`

3. **Audio**:
   - `easyeffects --gapplication-service`

4. **Clipboard**:
   - `wl-paste --type text --watch bash -c 'cliphist store && qs -c $qsConfig ipc call cliphistService update'`
   - `wl-paste --type image --watch bash -c 'cliphist store && qs -c $qsConfig ipc call cliphistService update'`

5. **Cursor**:
   - `hyprctl setcursor Bibata-Modern-Classic 24`

6. **Hyprland Plugin Manager**:
   - `hyprpm reload`

## General Settings

From `hyprland/general.conf`:

### Monitor Configuration
- `monitor=,preferred,auto,1` - Auto-detect preferred monitor

### Gestures
- **3-finger swipe**: Move windows
- **4-finger horizontal**: Switch workspace
- **4-finger pinch**: Float window
- **4-finger up**: Toggle quickshell overview
- **4-finger down**: Close quickshell overview

### Workspace Swipe Settings
- `workspace_swipe_distance = 700`
- `workspace_swipe_cancel_ratio = 0.2`
- `workspace_swipe_min_speed_to_force = 5`
- `workspace_swipe_direction_lock = true`
- `workspace_swipe_direction_lock_threshold = 10`
- `workspace_swipe_create_new = true`

### Window Appearance
- `gaps_in = 5`
- `gaps_out = 10`
- `border_size = 0`
- `col.active_border = rgba(0DB7D4FF)`
- `col.inactive_border = rgba(31313600)`
- `resize_on_border = true`
- `allow_tearing = true`

### Window Snapping
- Enabled with 4px window gap and 5px monitor gap
- Respects gaps

## Quickshell Integration

- **Config Variable**: `$qsConfig = ii`
- **Config Location**: `~/.config/quickshell/ii/`
- **Integration**: Used for clipboard service and overview toggle

## Related Packages

- `illogical-impulse-hyprland 1.0-4` - Hyprland configuration package
- `illogical-impulse-quickshell-git 0.2.0.r34.gdb1777c-1` - Quickshell integration
- `hypridle` - Idle daemon
- `hyprpm` - Plugin manager

## Monitor Management

- Uses `nwg-displays` for graphical monitor management
- Monitor configuration can be overwritten by nwg-displays
- Installation: `sudo pacman -S nwg-displays`

## Window Rules

Window rules are defined in:
- `hyprland/rules.conf`
- `custom/rules.conf`

## Key Bindings

Key bindings are defined in:
- `hyprland/keybinds.conf`
- `custom/keybinds.conf`

## Submap System

- Uses global submap for catchall keybinds
- **Important**: `exec = hyprctl dispatch submap global` must not be removed
- Required for catchall functionality


