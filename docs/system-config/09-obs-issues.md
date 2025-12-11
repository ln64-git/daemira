# OBS Studio Issues

## Issue #1: obs-webrtc Plugin Failing to Load ✅ FIXED

**Status**: Resolved (user confirmed)

### Problem
OBS Studio shows error: "The following OBS plugins failed to load: obs-webrtc"

### Root Cause
**ABI Incompatibility After System Update**

- **obs-webrtc plugin** was compiled against `libdatachannel.so.0.23`
- **System update** upgraded `libdatachannel` to version `0.24.0-1.1`
- Plugin cannot find the old library version it was linked against
- Error: `libdatachannel.so.0.23 => not found`

### Installed Versions
- **OBS Studio**: `obs-studio-git 32.0.1.r0.g0b12296-2`
- **libdatachannel**: `0.24.0-1.1` (installed)
- **obs-webrtc plugin**: Located at `/usr/lib/obs-plugins/obs-webrtc.so`

## Solutions

### Solution 1: Reinstall OBS Studio (Recommended)
This will rebuild OBS against the new libdatachannel version:

```bash
# Reinstall obs-studio-git to rebuild against new libdatachannel
sudo pacman -S obs-studio-git

# If that doesn't work, force reinstall
sudo pacman -S --force obs-studio-git
```

### Solution 2: Create Library Symlink (Quick Fix, Less Safe)
Create a symlink from the new version to the old version name:

```bash
# Create symlink (requires root)
sudo ln -s /usr/lib/libdatachannel.so.0.24 /usr/lib/libdatachannel.so.0.23

# Verify it works
ldd /usr/lib/obs-plugins/obs-webrtc.so | grep datachannel
```

**Warning**: This is a workaround. The proper fix is Solution 1.

### Solution 3: Check for Updates
Sometimes the AUR package needs to be updated:

```bash
# If using yay/paru
yay -S obs-studio-git

# Or check AUR for updates
# https://aur.archlinux.org/packages/obs-studio-git
```

### Solution 4: Temporarily Disable Plugin
If you don't need WebRTC functionality:

1. Open OBS Studio
2. Go to Tools > Plugins
3. Disable obs-webrtc plugin
4. Restart OBS

## Verification

After applying a fix, verify the plugin loads:

```bash
# Check if library dependency is resolved
ldd /usr/lib/obs-plugins/obs-webrtc.so | grep datachannel

# Should show:
# libdatachannel.so.0.23 => /usr/lib/libdatachannel.so.0.23 (or similar)
```

Or simply restart OBS Studio and check if the error is gone.

## Prevention

After system updates that upgrade libraries:
1. Check OBS for plugin errors
2. Reinstall OBS if plugins fail to load
3. Check AUR package updates if using -git packages

## Related Packages

- `obs-studio-git` - OBS Studio (git version)
- `obs-backgroundremoval 1.3.5-1` - Background removal plugin
- `libdatachannel 0.24.0-1.1` - WebRTC library (dependency)

## Notes

- This is a common issue with AUR -git packages after system updates
- The obs-webrtc plugin is built into obs-studio-git, not a separate package
- Regular `obs-studio` package (non-git) may have better compatibility
- Consider switching to stable `obs-studio` if -git version causes frequent issues

---

## Issue #2: Virtual Camera Freezes OBS ⚠️ CRITICAL

### Problem
When pressing "Start Virtual Camera" in OBS Studio, the application freezes completely.

### Root Cause
**Kernel Crash in v4l2loopback Module**

- v4l2loopback module is crashing the kernel when OBS queries virtual camera capabilities
- Kernel error: `RIP: 0010:vidioc_querycap+0xa5/0x100 [v4l2loopback]`
- Module version: `v4l2loopback-dkms 0.15.2-1`
- Kernel: `6.18.0-1-cachyos-bmq-lto`
- Module was built on Dec 5, but kernel was updated, causing incompatibility

### Evidence
- Multiple kernel panics in logs when accessing virtual camera
- Module loads successfully but crashes on query
- `/dev/video0`, `/dev/video1`, `/dev/video2` devices exist
- Module is loaded: `v4l2loopback 81920 4`
- **OBS Logs Show**: `info: Attempting to reset output capability of '/dev/video0'` - then freeze
- Module crashes when OBS tries to reset output capabilities for virtual camera

### Solutions

#### Solution 1: Rebuild v4l2loopback Module (Recommended)
Rebuild the DKMS module for your current kernel:

```bash
# Rebuild v4l2loopback for current kernel
sudo dkms install v4l2loopback/0.15.2 -k $(uname -r)

# Or reinstall the package to trigger rebuild
sudo pacman -S v4l2loopback-dkms
```

#### Solution 2: Unload and Reload Module
Try unloading and reloading the module:

```bash
# Unload the module
sudo modprobe -r v4l2loopback

# Wait a moment
sleep 2

# Reload with default parameters
sudo modprobe v4l2loopback

# Or with specific device count
sudo modprobe v4l2loopback devices=1
```

#### Solution 3: Check for Module Updates
Check if there's an updated version available:

```bash
# Check available versions
pacman -Ss v4l2loopback

# Update if available
sudo pacman -Syu v4l2loopback-dkms
```

#### Solution 4: Disable Virtual Camera (RECOMMENDED - WORKAROUND)
Since v4l2loopback is incompatible, disable it to prevent crashes:

```bash
# Run the disable script
cd /home/ln64/Source/daemira/docs/system-config
./disable-v4l2loopback.sh
```

Or manually:
```bash
# Close all apps using video devices (Discord, OBS, etc.)
# Then:
sudo modprobe -r v4l2loopback
echo "blacklist v4l2loopback" | sudo tee /etc/modprobe.d/blacklist-v4l2loopback.conf
```

This will prevent OBS from freezing, but virtual camera won't be available. Use OBS streaming/recording instead.

#### Solution 5: Try OBS-Specific v4l2loopback Version (May Still Crash)
There's an OBS-compatible version in AUR that may work better:

```bash
# First disable current module
sudo modprobe -r v4l2loopback
echo "blacklist v4l2loopback" | sudo tee /etc/modprobe.d/blacklist-v4l2loopback.conf

# Remove blacklist temporarily
sudo rm /etc/modprobe.d/blacklist-v4l2loopback.conf

# Install OBS-specific version from AUR
yay -S v4l2loopback-obs-dkms

# Load the new module
sudo modprobe v4l2loopback
```

This version (0.13.4) is specifically backported for OBS compatibility, but may still crash with kernel 6.18.

#### Solution 5: Unload Module and Use OBS Built-in Virtual Camera
OBS 32+ has a built-in virtual camera that may work without v4l2loopback:

```bash
# Unload the problematic module
sudo modprobe -r v4l2loopback

# Blacklist it to prevent auto-loading
echo "blacklist v4l2loopback" | sudo tee /etc/modprobe.d/blacklist-v4l2loopback.conf

# Restart OBS and try virtual camera
# OBS may use its built-in implementation
```

#### Solution 6: Use Alternative Virtual Camera Methods
If v4l2loopback continues to crash, consider alternatives:

1. **OBS Virtual Camera (built-in)** - May work without v4l2loopback on newer OBS
2. **v4l2loopback-dkms-git from AUR** - Try git version (0.12.5)
3. **Stream directly** - Use OBS streaming instead of virtual camera
4. **Alternative software** - Use different streaming/recording software

#### Solution 7: Temporarily Disable Virtual Camera
If you don't need virtual camera functionality right now:

1. Don't use "Start Virtual Camera" in OBS
2. Use OBS streaming/recording features instead
3. Wait for v4l2loopback update or kernel fix

#### Solution 8: Downgrade Kernel (Last Resort)
If the module is incompatible with kernel 6.18, temporarily use an older kernel:

```bash
# Boot from previous kernel version
# Or install older kernel version
```

### Verification

After applying a fix:

```bash
# Check if module loads without errors
sudo modprobe -r v4l2loopback
sudo modprobe v4l2loopback
dmesg | tail -20  # Check for errors

# Test in OBS
# Try starting virtual camera again
```

### Prevention

- After kernel updates, rebuild DKMS modules: `sudo dkms autoinstall`
- Check kernel compatibility before major updates
- Monitor kernel logs after updates: `journalctl -k --since "1 hour ago"`

### Related Information

- **Module Location**: `/lib/modules/6.18.0-1-cachyos-bmq-lto/updates/dkms/v4l2loopback.ko.zst`
- **Package**: `v4l2loopback-dkms 0.15.2-1`
- **Kernel**: `6.18.0-1-cachyos-bmq-lto`
- **Device Files**: `/dev/video0`, `/dev/video1`, `/dev/video2`

### Confirmed Incompatibility

**Status**: CONFIRMED - v4l2loopback 0.15.2-1 is incompatible with kernel 6.18.0-1-cachyos-bmq-lto

**Error Pattern**:
- OBS logs show: `info: Attempting to reset output capability of '/dev/video0'`
- Then: `info: Attempting to reset output capability of '/dev/video1'`
- Then: **FREEZE** (kernel crash in v4l2loopback module)

**Root Cause**: The module crashes when OBS tries to reset output capabilities for virtual camera setup. This is a kernel-level bug, not an OBS issue.

### Workaround: Disable Virtual Camera Feature

Since the module is incompatible, disable virtual camera functionality:

#### Option 1: Blacklist Module (Recommended)
Prevent v4l2loopback from loading:

```bash
# Create blacklist file
echo "blacklist v4l2loopback" | sudo tee /etc/modprobe.d/blacklist-v4l2loopback.conf

# Unload current module (after closing apps using it)
sudo modprobe -r v4l2loopback

# Rebuild initramfs (optional, for boot-time blacklist)
sudo mkinitcpio -P
```

Then restart OBS. The virtual camera option may be disabled or unavailable.

#### Option 2: Remove Module Package
If you don't need virtual camera at all:

```bash
# Remove the package
sudo pacman -Rns v4l2loopback-dkms v4l2loopback-utils

# Unload module
sudo modprobe -r v4l2loopback
```

#### Option 3: Use OBS Streaming Instead
Instead of virtual camera, use OBS's streaming/recording features:
- Stream directly to platforms
- Record to file
- Use OBS's built-in sharing features

### Long-term Solutions

1. **Wait for v4l2loopback update** - Report bug to maintainers
2. **Try different kernel** - Standard kernel (not BMQ-LTO) may work
3. **Use OBS-specific version** - `v4l2loopback-obs-dkms` from AUR (may still have issues)
4. **Switch to stable OBS** - `obs-studio` (non-git) may have better compatibility

### Notes

- This is a kernel module crash, not an OBS bug
- The crash happens in kernel space, which can freeze the entire system
- DKMS modules need to be rebuilt after kernel updates
- **CONFIRMED**: v4l2loopback 0.15.2-1 crashes with kernel 6.18.0-1-cachyos-bmq-lto
- Consider reporting this to v4l2loopback maintainers: https://github.com/umlaeute/v4l2loopback/issues

