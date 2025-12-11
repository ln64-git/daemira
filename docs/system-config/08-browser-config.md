# Browser Configuration

## Primary Browser: Zen Browser

### Installation
- **Package**: `zen-browser-bin 1.17.12b-1`
- **Installation Path**: `/opt/zen-browser-bin/`
- **Binary**: `/opt/zen-browser-bin/zen-bin`
- **Build ID**: 20251130082909

### Current Status
- **Issue**: Running slowly after system update
- **CPU Usage**: ~13% (main process)
- **Memory Usage**: ~2.5% (846MB main process)
- **Process Count**: Multiple content processes (normal for multi-process architecture)

### Process Architecture
Zen Browser uses a multi-process architecture:
- **Main Process**: Parent process (PID varies)
- **Content Processes**: Multiple child processes for tabs
- **RDD Process**: Remote Data Decoder
- **Utility Process**: Background utilities
- **Socket Process**: IPC communication

### Configuration Location
- **User Config**: `~/.config/zen-browser/` (if exists)
- **Profile**: Uses "Personal" profile by default
- **Cache/Data**: Typically in `~/.cache/zen-browser/` and `~/.local/share/zen-browser/`

## Other Installed Browsers

### Cachy Browser
- **Package**: `cachy-browser 137.0.2-5`
- **Description**: CachyOS optimized browser

### Firefox
- **Package**: `firefox 146.0-1.1`
- **PWA Support**: `firefoxpwa 2.17.2-1.2`
- **KDE Integration**: `plasma-browser-integration 6.5.4-1.1`

## Browser Performance Issues

### Symptoms
- Slow page loading
- High CPU usage
- Sluggish UI responsiveness
- Multiple processes consuming resources

### Potential Causes

1. **Post-Update Issues**:
   - Incompatibility with new kernel/drivers
   - Wayland compositor changes
   - Library updates

2. **GPU Acceleration**:
   - AMD GPU driver issues
   - Vulkan/OpenGL problems
   - Hardware acceleration disabled

3. **Memory/Resource Pressure**:
   - Too many tabs/extensions
   - Memory leaks
   - Background processes

4. **Profile Issues**:
   - Corrupted profile
   - Extension conflicts
   - Cache corruption

### Troubleshooting Steps

#### 1. Check GPU Acceleration
```bash
# In browser, navigate to:
about:support

# Check Graphics section for:
# - WebGL status
# - Hardware acceleration status
# - GPU process count
```

#### 2. Clear Browser Data
- Settings > Privacy & Security > Clear Data
- Clear cache, cookies, and site data
- Restart browser

#### 3. Test with New Profile
```bash
# Launch with new profile
/opt/zen-browser-bin/zen-bin -P TestProfile

# Or create via Settings > Profiles
```

#### 4. Disable Extensions
- Settings > Extensions
- Disable all extensions
- Test performance
- Re-enable one by one

#### 5. Check System Resources
```bash
# Check memory
free -h

# Check CPU
top -p $(pgrep -f zen-bin | head -1)

# Check disk I/O
iostat -x 1
```

#### 6. Compare with Other Browsers
```bash
# Test Firefox
firefox

# Test Cachy Browser
cachy-browser

# Compare performance
```

#### 7. Browser-Specific Debugging
```bash
# Launch with debugging
/opt/zen-browser-bin/zen-bin --safe-mode

# Check browser logs
# (Location depends on Zen Browser implementation)
```

### Performance Optimization

#### Enable Hardware Acceleration
1. Navigate to `about:config`
2. Search for `gfx.webrender.all`
3. Set to `true` if available
4. Search for `layers.acceleration.force-enabled`
5. Set to `true`

#### Reduce Resource Usage
- Limit number of open tabs
- Use tab suspension extensions
- Disable unnecessary extensions
- Reduce background processes

#### Wayland-Specific Optimizations
- Ensure running under Wayland (not X11)
- Check compositor performance
- Verify GPU drivers are up to date

### Alternative Solutions

If Zen Browser continues to have issues:

1. **Use Firefox temporarily**:
   ```bash
   firefox
   ```

2. **Use Cachy Browser**:
   ```bash
   cachy-browser
   ```

3. **Reinstall Zen Browser**:
   ```bash
   sudo pacman -Rns zen-browser-bin
   sudo pacman -S zen-browser-bin
   ```

4. **Check for Updates**:
   ```bash
   sudo pacman -Sy zen-browser-bin
   ```

## Browser Integration

### Hyprland Integration
- Browser should work with Hyprland window manager
- Wayland native support (if available)
- Gesture support via Hyprland gestures

### Audio Integration
- Browser audio should route through PipeWire
- If audio not working system-wide, browser audio will also fail
- Fix system audio first (see [Audio Configuration](./03-audio-config.md))

## Configuration Files

### Zen Browser Config Locations
- **Config**: `~/.config/zen-browser/` (if exists)
- **Cache**: `~/.cache/zen-browser/`
- **Data**: `~/.local/share/zen-browser/`
- **Profiles**: Within data directory

### Firefox Config Locations
- **Config**: `~/.mozilla/firefox/`
- **Cache**: `~/.cache/mozilla/firefox/`

## Notes

- Zen Browser is Firefox-based, so many Firefox troubleshooting steps apply
- Multi-process architecture is normal and expected
- High process count doesn't necessarily indicate a problem
- Performance issues may resolve after system restart or service restarts


