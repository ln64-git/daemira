# Desktop Services Integration

## Overview

This document maps Quickshell DMS services to their underlying system services, explaining how desktop functionality integrates with system-level DBus services, IPC protocols, and command-line tools.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Quickshell DMS (QML UI)                  │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │SessionService│  │IdleService   │  │CompositorService │   │
│  │DisplayService│  │BatteryService│  │PowerProfile      │   │
│  └──────┬───────┘  └──────┬───────┘  └────────┬─────────┘   │
└─────────┼──────────────────┼──────────────────┼─────────────┘
          │                  │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│              System DBus Services & IPC                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │systemd-logind│  │UPower        │  │power-profiles    │  │
│  │(session mgmt)│  │(battery)     │  │-daemon           │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │Hyprland IPC  │  │Wayland       │                        │
│  │(hyprctl)     │  │Protocols     │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
          │                  │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│            Daemira Desktop Monitors (CLI/API)               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │SessionMonitor│  │Compositor    │  │DisplayMonitor    │  │
│  │              │  │Monitor       │  │                  │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Service Mapping

### Session Management

#### Quickshell: SessionService.qml
- **Purpose**: Session control, power actions, idle inhibition
- **System Service**: `org.freedesktop.login1` (systemd-logind)
- **DBus Interface**: `/org/freedesktop/login1`

**Integration Methods:**
```bash
# Query session properties
gdbus call --system \
  --dest org.freedesktop.login1 \
  --object-path /org/freedesktop/login1/session/self \
  --method org.freedesktop.DBus.Properties.GetAll \
  org.freedesktop.login1.Session

# Alternative: loginctl
loginctl show-session $XDG_SESSION_ID

# Lock session
loginctl lock-session

# Unlock session
loginctl unlock-session

# Power actions
systemctl suspend
systemctl hibernate
systemctl suspend-then-hibernate
systemctl reboot
systemctl poweroff
```

**Properties Tracked:**
- `Id` - Session identifier
- `Name` - User name
- `User` - User ID
- `Seat` - Seat identifier (usually seat0)
- `Type` - Session type (wayland/x11)
- `Active` - Whether session is active
- `IdleHint` - Whether session is idle
- `LockedHint` - Whether session is locked
- `State` - Session state (online/closing)

**Daemira Integration:**
- `SessionMonitor` queries same DBus service
- Direct access to session state
- Lock/unlock functionality via loginctl

### Idle Management

#### Quickshell: IdleService.qml
- **Purpose**: Auto-lock, monitor off, auto-suspend based on idle time
- **Wayland Protocol**: `ext-idle-notify-v1`
- **Integration**: Quickshell IdleMonitor component

**Timeout Configuration:**
- Monitor off timeout (separate for AC/battery)
- Lock timeout (separate for AC/battery)
- Suspend timeout (separate for AC/battery)
- Respects idle inhibitors

**Actions Triggered:**
- Monitor off: `CompositorService.powerOffMonitors()`
- Lock screen: `SessionService.lockRequested` signal
- Suspend: `SessionService.suspendWithBehavior()`

**Daemira Integration:**
- Cannot directly access Wayland idle protocol (client-side)
- Can query session `IdleHint` via systemd-logind DBus
- Can query lock state via loginctl

### Power Profile Management

#### Quickshell: PowerProfileWatcher.qml
- **Purpose**: Monitor power profile changes
- **System Service**: `net.hadess.PowerProfiles` (power-profiles-daemon)
- **DBus Interface**: `/net/hadess/PowerProfiles`

**Integration Methods:**
```bash
# Query current profile
powerprofilesctl get

# Set profile
powerprofilesctl set performance
powerprofilesctl set balanced
powerprofilesctl set power-saver

# List available profiles
powerprofilesctl list
```

**Profiles:**
- `performance` - Maximum performance, higher power usage
- `balanced` - Default balanced mode
- `power-saver` - Battery saving, reduced performance

**Daemira Integration:**
- `PerformanceManager` already integrates with power-profiles-daemon
- Tracks current profile, available profiles
- Can switch profiles programmatically

### Compositor Integration

#### Quickshell: CompositorService.qml
- **Purpose**: Compositor-specific functionality (Hyprland, Niri, Sway, Dwl)
- **Hyprland IPC**: Unix socket communication via `hyprctl`
- **Socket Path**: `$XDG_RUNTIME_DIR/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket.sock`

**Integration Methods:**
```bash
# Get version
hyprctl version -j

# Monitor information
hyprctl monitors -j

# Workspace information
hyprctl workspaces -j

# Active window
hyprctl activewindow -j

# All windows
hyprctl clients -j

# Execute commands
hyprctl dispatch workspace 1
hyprctl dispatch killactive
```

**JSON Output Structure:**

Monitors:
```json
[{
  "id": 0,
  "name": "DP-1",
  "description": "Dell Inc. DELL U3425WE ...",
  "make": "Dell Inc.",
  "model": "DELL U3425WE",
  "serial": "...",
  "width": 3440,
  "height": 1440,
  "refreshRate": 144.00,
  "x": 0,
  "y": 0,
  "activeWorkspace": {"id": 1, "name": "1"},
  "scale": 1.00,
  "transform": 0,
  "vrr": true,
  "activelyTearing": false,
  "dpmsStatus": true
}]
```

Workspaces:
```json
[{
  "id": 1,
  "name": "1",
  "monitor": "DP-1",
  "monitorID": 0,
  "windows": 3,
  "hasfullscreen": false,
  "lastwindow": "0x...",
  "lastwindowtitle": "Window Title"
}]
```

**Daemira Integration:**
- `CompositorMonitor` uses `hyprctl -j` for JSON output
- Tracks workspaces, windows, monitor configuration
- Compositor detection via environment variables

### Display Management

#### Quickshell: DisplayService.qml
- **Purpose**: Brightness control, night mode, gamma control
- **Brightness**: `/sys/class/backlight/` (N/A for desktop monitors)
- **Gamma**: Custom gamma control implementation
- **Monitor Config**: Via Hyprland IPC

**Desktop Monitor Notes:**
- External monitors don't have backlight control via `/sys/class/backlight/`
- Brightness adjusted via monitor OSD (hardware controls)
- Software gamma/temperature control possible
- Monitor configuration (resolution, refresh, VRR) via compositor

**Daemira Integration:**
- `DisplayMonitor` tracks monitor configuration only
- No brightness control (desktop monitors)
- Focuses on resolution, refresh rate, scale, VRR status

### Battery Management (N/A for Desktop)

#### Quickshell: BatteryService.qml
- **Purpose**: Battery monitoring
- **System Service**: `org.freedesktop.UPower`
- **DBus Interface**: `/org/freedesktop/UPower`

**Not applicable for desktop system** - no battery hardware.

Quickshell gracefully handles absence:
```qml
readonly property bool batteryAvailable: batteries.length > 0
// UI elements check batteryAvailable before displaying
```

## Power Management Behavior

### AC vs Battery Power Profiles

Quickshell can auto-switch power profiles based on power source:

**BatteryService.qml** monitors plugged state:
```qml
onIsPluggedInChanged: {
    if (isPluggedIn) {
        // Switch to AC profile
        PowerProfiles.profile = SettingsData.acProfileName;
    } else {
        // Switch to battery profile
        PowerProfiles.profile = SettingsData.batteryProfileName;
    }
}
```

**For desktop systems:** Always considered "plugged in", uses AC profile settings.

### Idle Timeout System

**Quickshell IdleService** manages three independent timers:

1. **Monitor Off Timer**
   - Timeout: `SettingsData.acMonitorTimeout` (desktop always uses AC timeout)
   - Action: `CompositorService.powerOffMonitors()` → DPMS off
   - Restore: Any input event powers monitors back on

2. **Lock Timer**
   - Timeout: `SettingsData.acLockTimeout`
   - Action: `SessionService.lockRequested` signal → Lock screen displayed
   - Optional fade-to-lock animation

3. **Suspend Timer**
   - Timeout: `SettingsData.acSuspendTimeout`
   - Action: `SessionService.suspend()` → `systemctl suspend`
   - Behavior options: suspend, hibernate, suspend-then-hibernate

**Idle Inhibitors:**
- Media playback inhibits idle timers
- Fullscreen applications can inhibit
- Manual inhibit toggle available
- Native Wayland `IdleInhibitor` protocol support

## DBus Service Endpoints

### systemd-logind

**Service:** `org.freedesktop.login1`
**System Bus**

**Key Interfaces:**
- `/org/freedesktop/login1` - Manager
- `/org/freedesktop/login1/session/{id}` - Session
- `/org/freedesktop/login1/user/{uid}` - User

**Methods Used:**
- `Session.Lock()` - Lock session
- `Session.Unlock()` - Unlock session
- `Manager.Suspend(bool)` - Suspend system
- `Manager.Hibernate(bool)` - Hibernate system
- `Manager.Reboot(bool)` - Reboot system
- `Manager.PowerOff(bool)` - Power off system

**Properties Monitored:**
- `Session.Active` - Is session active
- `Session.IdleHint` - Is session idle
- `Session.LockedHint` - Is session locked
- `Manager.PreparingForSleep` - System preparing to sleep

### power-profiles-daemon

**Service:** `net.hadess.PowerProfiles`
**System Bus**

**Key Interfaces:**
- `/net/hadess/PowerProfiles` - Main interface

**Properties:**
- `ActiveProfile` - Current active profile (string)
- `Profiles` - Array of available profile objects
- `Actions` - Array of actions (profile switching)

**Values:**
- `"performance"` - Performance mode
- `"balanced"` - Balanced mode
- `"power-saver"` - Power saver mode

### UPower (Not Used on Desktop)

**Service:** `org.freedesktop.UPower`
**System Bus**

Not applicable for desktop systems without battery.

## Hyprland IPC Protocol

### Socket Communication

**Socket Locations:**
- Command socket: `$XDG_RUNTIME_DIR/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket.sock`
- Event socket: `$XDG_RUNTIME_DIR/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket2.sock`

**Environment Variable:**
- `$HYPRLAND_INSTANCE_SIGNATURE` - Unique instance identifier

### Command Interface

**Format:** `hyprctl [command] [args] [flags]`

**Common Commands:**
- `version` - Version information
- `monitors` - Monitor list and configuration
- `workspaces` - Workspace information
- `clients` - Window (client) list
- `activewindow` - Currently focused window
- `dispatch [command] [args]` - Execute Hyprland command

**Flags:**
- `-j` - JSON output (recommended for parsing)
- `-r` - Suppress header

**Examples:**
```bash
# Get monitor info in JSON
hyprctl monitors -j

# Get workspace info
hyprctl workspaces -j

# Get active window
hyprctl activewindow -j

# Switch workspace
hyprctl dispatch workspace 1

# Move window to workspace
hyprctl dispatch movetoworkspace 2
```

### Event Socket

**Real-time Events:**
- Workspace changes
- Window focus changes
- Monitor connect/disconnect
- Keyboard layout changes

**Usage:**
```bash
# Listen to events
socat - UNIX-CONNECT:$XDG_RUNTIME_DIR/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket2.sock
```

**Event Format:** `EVENT>>DATA`

Examples:
- `workspace>>1` - Switched to workspace 1
- `activewindow>>class,title` - Active window changed
- `focusedmon>>monitor,workspace` - Focused monitor changed

## Integration Comparison

| Feature | Quickshell DMS | Daemira |
|---------|---------------|---------|
| **Purpose** | Desktop UI shell | System monitoring daemon |
| **Interface** | Graphical (QML) | CLI/programmatic |
| **Session Data** | Via DMSService IPC + DBus | Direct DBus/loginctl |
| **Compositor** | Via CompositorService | Direct hyprctl |
| **Power Profiles** | Via PowerProfileWatcher | Via PerformanceManager |
| **Idle Detection** | Wayland IdleMonitor | Session IdleHint |
| **Lock Screen** | Built-in Lock module | Via loginctl |
| **Use Case** | User interaction | Automation/scripting |

**Both tools:**
- Query the same underlying system services
- Use standard DBus interfaces
- Compatible and complementary
- No conflicts or interference

## Daemira Desktop Monitoring

### Data Sources

**SessionMonitor:**
- `loginctl show-session $XDG_SESSION_ID`
- systemd-logind DBus interface
- Environment variables: `$XDG_SESSION_ID`, `$XDG_SESSION_TYPE`

**CompositorMonitor:**
- `hyprctl version -j`
- `hyprctl monitors -j`
- `hyprctl workspaces -j`
- `hyprctl activewindow -j`
- `hyprctl clients -j`
- Environment: `$HYPRLAND_INSTANCE_SIGNATURE`

**DisplayMonitor:**
- `hyprctl monitors -j`
- Monitor configuration from Hyprland

### CLI Access

```bash
# Desktop environment status
./daemira.ts desktop:status

# Session information
./daemira.ts desktop:session

# Compositor details
./daemira.ts desktop:compositor

# Display configuration
./daemira.ts desktop:displays

# Lock session
./daemira.ts desktop:lock

# Comprehensive system status (includes desktop)
./daemira.ts status
```

## Service Dependencies

### Required for Desktop Monitoring

- **systemd** - systemd-logind for session management
- **Hyprland** - Compositor (or other supported compositor)
- **loginctl** - Session control tool
- **hyprctl** - Hyprland control tool

### Optional

- **power-profiles-daemon** - Power profile management (already integrated)
- **UPower** - Battery monitoring (N/A for desktop)

### Not Required

- **Quickshell** - Independent of Daemira desktop monitoring
- Desktop monitoring works whether or not Quickshell is running

## Graceful Degradation

Daemira desktop monitors handle missing services:

```typescript
// Compositor detection
if (!process.env.HYPRLAND_INSTANCE_SIGNATURE) {
    return { compositor: 'unknown', available: false };
}

// Session monitoring
try {
    const session = await querySession();
} catch (error) {
    logger.warn('Session monitoring unavailable');
    return fallbackSessionInfo();
}
```

**Behavior:**
- Missing compositor → Reports "unknown", skips compositor queries
- Missing loginctl → Session monitoring unavailable
- No Hyprland → Compositor info shows "not available"
- Never crashes, always returns usable data

## Security Considerations

### DBus Permissions

- **systemd-logind**: Requires session user access (granted by default)
- **Power actions**: May require PolicyKit authentication
- **Lock/Unlock**: Session user can lock/unlock own session

### Hyprland IPC

- **Socket Permissions**: User-only access (`0700`)
- **No Authentication**: Trust-based (local user socket)
- **Safe Commands**: Read-only queries safe, dispatch commands modify state

### Best Practices

- Use read-only queries for monitoring
- Require user confirmation for power actions
- Log all privileged operations
- Handle permission errors gracefully
