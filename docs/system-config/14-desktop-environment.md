# Desktop Environment

## Overview

This system uses **Quickshell DMS** (Desktop Management Shell) as the primary desktop shell interface, running on top of the **Hyprland** Wayland compositor.

## Compositor

### Hyprland
- **Version**: 0.52.2
- **Protocol**: Wayland
- **Configuration**: `~/.config/hypr/hyprland.conf`
- **IPC**: Hyprland socket for programmatic control via `hyprctl`
- **Features**: Dynamic tiling, animations, VRR support, multi-monitor

### Libraries
- Hyprgraphics: 0.4.0
- Hyprutils: 0.11.0
- Hyprcursor: 0.1.13
- Hyprlang: 0.6.7
- Aquamarine: 0.10.0

## Quickshell DMS

### Architecture
Quick shell DMS is a QML-based desktop shell that provides a complete desktop environment experience with panels, widgets, and system integration services.

- **Location**: `/usr/share/quickshell/dms/`
- **Configuration**: `~/.config/quickshell/`
- **Language**: QML (Qt Modeling Language)
- **Framework**: Quickshell
- **Design**: Material Design 3 dark theme

### Structure

```
/usr/share/quickshell/dms/
├── shell.qml           # Main entry point
├── Common/             # Shared resources (Theme, Settings, SessionData)
├── Services/           # System integration singletons (40 services)
├── Modules/            # UI components (TopBar, ControlCenter, Notifications, etc.)
├── Modals/             # Full-screen overlays
├── Widgets/            # Reusable UI controls
└── PLUGINS/            # Plugin system examples
```

## Quickshell Services

Quickshell DMS provides 40 system integration services located in `/usr/share/quickshell/dms/Services/`:

### Core System Services

| Service | Purpose |
|---------|---------|
| SessionService.qml | Session management, power actions (logout/suspend/hibernate/reboot/poweroff) |
| IdleService.qml | Idle monitoring, auto-lock/suspend timeouts, monitor power management |
| CompositorService.qml | Compositor integration (Niri, Hyprland, Sway, Dwl support) |
| DisplayService.qml | Brightness control, night mode/gamma control, display device management |
| PowerProfileWatcher.qml | Power profile change monitoring |

### Hardware Integration

| Service | Purpose |
|---------|---------|
| BatteryService.qml | UPower integration for battery stats, charging state, health (N/A for desktop) |
| AudioService.qml | PipeWire/WirePlumber audio management |
| BluetoothService.qml | Bluetooth device management |
| NetworkService.qml | Network connection management |
| DMSNetworkService.qml | Advanced network service |

### Desktop Features

| Service | Purpose |
|---------|---------|
| NotificationService.qml | Notification daemon integration |
| CalendarService.qml | Calendar events and scheduling |
| WeatherService.qml | Weather information integration |
| WallpaperCyclingService.qml | Automatic wallpaper rotation |
| ClipboardService | Clipboard history management |

### Compositor-Specific

| Service | Purpose |
|---------|---------|
| NiriService.qml | Niri compositor integration |
| DwlService.qml | Dwl compositor integration |
| ExtWorkspaceService.qml | External workspace management |
| WlrOutputService.qml | wlr-output-management protocol |

### Application & System

| Service | Purpose |
|---------|---------|
| AppSearchService.qml | Application launcher and search |
| DgopService.qml | Process monitoring and management |
| PluginService.qml | Plugin system for extensibility |
| BarWidgetService.qml | Top bar widget management |
| PopoutService.qml | Popout/overlay management |

### Miscellaneous

| Service | Purpose |
|---------|---------|
| DMSService.qml | Core DMS integration and IPC |
| DesktopService.qml | Desktop file and application handling |
| DSearchService.qml | Deep search functionality |
| KeybindsService.qml | Keyboard shortcut management |
| NotepadStorageService.qml | Note-taking functionality |
| PolkitService.qml | Authentication and privilege management |
| PortalService.qml | XDG Desktop Portal integration |
| PrivacyService.qml | Privacy controls |
| SystemUpdateService.qml | System update notifications |
| ToastService.qml | Toast notification system |
| UserInfoService.qml | User information |
| VPNService.qml | VPN connection management |
| CupsService.qml | Printing service |
| CavaService.qml | Audio visualizer integration |
| MprisController.qml | Media player control (MPRIS2) |
| LegacyNetworkService.qml | Legacy network support |

## Session Management

### Display Manager
- **SDDM** (Simple Desktop Display Manager)
- Wayland session type
- Auto-login configured for user `ln64`

### Session

Control
- **systemd-logind**: Session tracking and power management
- **XDG_SESSION_TYPE**: wayland
- **Session ID**: Retrieved via `$XDG_SESSION_ID`

### Power Management
- **Power Profiles Daemon**: System-wide power profile control
- **Profiles**: performance, balanced, power-saver
- **Integration**: Quickshell PowerProfileWatcher monitors changes
- **AC/Battery**: Can auto-switch profiles based on power source

### Idle Management
- **Quickshell IdleService**: Configurable idle timeouts
- **Monitor Off Timeout**: Separate AC/battery timeouts
- **Lock Timeout**: Auto-lock after idle period
- **Suspend Timeout**: Auto-suspend with configurable behavior
- **Inhibitors**: Respect idle inhibitors (e.g., media playback)

## Desktop Environment Startup Flow

1. **SDDM** starts and displays login screen
2. User logs in → SDDM starts Wayland session
3. **Hyprland** compositor launches
4. **Hyprland** executes `~/.config/hypr/hyprland.conf`
5. **Quickshell** launches from Hyprland config
6. Quickshell loads `shell.qml` and initializes all services
7. Desktop shell fully loaded with:
   - Top bar(s) on each monitor
   - System tray
   - Workspaces
   - Background services (IdleService, NotificationService, etc.)

## Key Configuration Locations

| Component | Configuration Path |
|-----------|-------------------|
| Hyprland | `~/.config/hypr/hyprland.conf` |
| Quickshell | `~/.config/quickshell/` |
| Quickshell DMS | `/usr/share/quickshell/dms/` (system-wide) |
| DMS Settings | `~/.config/DankMaterialShell/settings.json` |
| DMS Plugins | `~/.config/DankMaterialShell/plugins/` |
| SDDM | `/etc/sddm.conf`, `/etc/sddm.conf.d/` |

## IPC and Control

### Hyprland IPC
- **Socket**: `$XDG_RUNTIME_DIR/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket.sock`
- **Command Tool**: `hyprctl`
- **JSON Output**: Most commands support `-j` flag for JSON output
- **Examples**:
  - `hyprctl monitors -j` - List monitors
  - `hyprctl workspaces -j` - List workspaces
  - `hyprctl clients -j` - List windows
  - `hyprctl activewindow -j` - Get active window

### Quickshell DMS IPC
- **Socket**: `$DMS_SOCKET` environment variable
- **Protocol**: Custom JSON-RPC over Unix socket
- **Capabilities**: loginctl integration, state queries
- **Used by**: SessionService, IdleService

### systemd-logind DBus
- **Service**: `org.freedesktop.login1`
- **Interface**: `/org/freedesktop/login1`
- **Methods**: Lock/unlock session, power actions, session properties
- **Used by**: Quickshell SessionService, IdleService

## Features

### Multi-Monitor Support
- Per-monitor top bars
- Independent workspace sets per monitor
- VRR (Variable Refresh Rate) support
- Mixed refresh rates supported

### Window Management
- Dynamic tiling via Hyprland
- Workspace switching
- Window rules and automation
- Floating windows support
- Special workspaces (scratchpad)

### System Integration
- Automatic theme generation via matugen (wallpaper-based colors)
- GTK and Qt application theming
- Notification system
- Media controls (MPRIS2)
- Clipboard history
- Application launcher with search

### Power Management
- Idle detection and auto-lock
- Monitor power management (DPMS)
- Suspend/hibernate support
- Power profile switching
- Lid close handling

## Desktop Environment Version Info

- **Desktop Shell**: Quickshell DMS (Material Design 3 theme)
- **Compositor**: Hyprland 0.52.2
- **Display Server**: Wayland
- **Session Manager**: systemd-logind
- **Display Manager**: SDDM
- **Services**: 40 QML-based system integration services
