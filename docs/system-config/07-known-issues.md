# Known Issues & Troubleshooting

## Current Issues (2025-12-10)

### 1. Audio Not Working ⚠️ CRITICAL

**Symptoms**:
- `pactl` commands timing out with "Connection failure: Timeout"
- Audio services appear to be running but not accessible
- No audio output
- WirePlumber service timing out on stop/restart

**Diagnosis** (from diagnostic output):
- **ROOT CAUSE IDENTIFIED**: Defunct (zombie) WirePlumber process blocking audio
- Process: `[wireplumber] <defunct>` (PID 38535)
- WirePlumber service failing with timeout errors
- Service timeout: 1min 30s (default)
- Multiple `wp_event_dispatcher_unregister_hook` assertion failures
- PipeWire and PipeWire-Pulse services running but can't communicate properly

**Possible Causes**:
1. **Zombie WirePlumber process** (PRIMARY ISSUE)
2. Post-update service restart needed
3. WirePlumber hanging during shutdown
4. DBus communication problem
5. Service dependency issue

**Troubleshooting Steps** (in order):

```bash
# STEP 1: Kill defunct/zombie WirePlumber processes
ps aux | grep wireplumber | grep -E "(defunct|zombie|<defunct>)"
# If found, note the PID and kill the parent process or restart user session

# STEP 2: Force kill all WirePlumber processes
pkill -9 wireplumber
# Wait a few seconds
sleep 3

# STEP 3: Clean up sockets (optional, be careful)
# rm -f /run/user/$(id -u)/pipewire*  # Only if absolutely necessary

# STEP 4: Restart services in correct order
systemctl --user stop wireplumber pipewire-pulse pipewire
sleep 2
systemctl --user start pipewire
sleep 2
systemctl --user start pipewire-pulse
sleep 2
systemctl --user start wireplumber
sleep 3

# STEP 5: Verify services are running
systemctl --user status pipewire pipewire-pulse wireplumber

# STEP 6: Test audio connection
pactl info
pactl list short sinks
```

**If above doesn't work**:
```bash
# Option 1: Reload systemd user daemon
systemctl --user daemon-reload
systemctl --user restart pipewire pipewire-pulse wireplumber

# Option 2: Log out and log back in (most reliable for full session restart)
# This will restart all user services properly
```

**Workaround**: Restart user session or log out/in (most reliable fix)

### 2. Zen Browser Running Slowly

**Symptoms**:
- Browser is slow/responsive
- High CPU usage (13% reported)
- Multiple content processes running

**Diagnosis**:
- Zen Browser 1.17.12b-1 installed
- Multiple content processes active (normal for multi-process architecture)
- Recent system update may have affected performance

**Possible Causes**:
1. Post-update compatibility issues
2. GPU acceleration problems
3. Memory pressure (though 19GB available)
4. Extension or profile issues
5. Wayland compositor performance

**Troubleshooting Steps**:
```bash
# 1. Check browser processes
ps aux | grep zen-bin

# 2. Check GPU acceleration
# In browser: about:support (check Graphics section)

# 3. Clear browser cache
# Settings > Privacy > Clear Data

# 4. Check for problematic extensions
# Disable extensions one by one

# 5. Create new profile to test
# Settings > Profiles > Create New Profile
```

**Workaround**: 
- Restart browser
- Try different browser (firefox, cachy-browser) to isolate issue
- Check if issue persists after system restart

### 3. WirePlumber Warnings

**Symptoms** (non-critical but logged):
- Multiple `wp-event-dispatcher` assertion failures
- UPower errors (battery monitoring)
- Missing libcamera plugin warnings

**Impact**: 
- Generally non-critical
- Camera support may be limited
- Battery monitoring unavailable

**Solutions**:
```bash
# Install missing camera support
sudo pacman -S pipewire-v4l2

# Install UPower for battery monitoring (if on laptop)
sudo pacman -S upower
```

### 4. Kernel Errors (v4l2loopback)

**Symptoms**:
- Kernel errors in journal related to v4l2loopback
- `vidioc_querycap` errors in kernel log

**Impact**:
- May affect virtual camera functionality
- Not critical for basic system operation

**Note**: v4l2loopback is a kernel module for creating virtual video devices. Errors may indicate:
- Module incompatibility with current kernel
- Configuration issues
- Not needed if virtual cameras aren't used

**Solution**:
```bash
# If not needed, can be ignored or module can be blacklisted
# Check if module is loaded
lsmod | grep v4l2loopback

# If causing issues and not needed:
sudo modprobe -r v4l2loopback
```

## Post-Update Issues

### System Update Context
- Recent system update performed
- Kernel: 6.18.0-1-cachyos-bmq-lto
- Multiple services may need restart

### Recommended Post-Update Steps

1. **Restart audio services**:
   ```bash
   systemctl --user restart pipewire pipewire-pulse wireplumber
   ```

2. **Restart Hyprland** (if needed):
   ```bash
   hyprctl reload
   ```

3. **Check for orphaned packages**:
   ```bash
   pacman -Qdt
   ```

4. **Update system database**:
   ```bash
   sudo pacman -Sy
   ```

5. **Check for broken packages**:
   ```bash
   pacman -Qkk
   ```

## System Health Checks

### Quick Health Check Script
```bash
#!/bin/bash
# System health check

echo "=== Service Status ==="
systemctl --user status pipewire pipewire-pulse wireplumber --no-pager -l

echo -e "\n=== Audio Devices ==="
pactl list short sinks 2>&1 || echo "Audio not accessible"

echo -e "\n=== Disk Usage ==="
df -h /

echo -e "\n=== Memory Usage ==="
free -h

echo -e "\n=== Recent Errors ==="
journalctl -p 3 -xb --no-pager | tail -20
```

## Prevention

### Before System Updates
1. Check current working state
2. Note any custom configurations
3. Backup important configs

### After System Updates
1. Restart affected services
2. Test critical functionality (audio, display, network)
3. Check logs for errors
4. Verify dotfiles compatibility

## Getting Help

### Useful Commands for Diagnostics
```bash
# System information
uname -a
cat /etc/os-release
pacman -Q | wc -l

# Service status
systemctl --user list-units --type=service --state=running

# Recent errors
journalctl -p 3 -xb --no-pager | tail -50

# Audio diagnostics
pactl info
aplay -l
alsamixer

# GPU information
glxinfo | grep -E "OpenGL"
lspci | grep -i vga
```

### Log Locations
- System logs: `journalctl`
- User service logs: `journalctl --user`
- Audio logs: `journalctl --user -u pipewire -u wireplumber`
- Kernel logs: `dmesg` (requires root)

