# Audio Configuration

## Audio Stack

- **Primary Audio System**: PipeWire
- **PipeWire Version**: 1:1.4.9-2.1
- **WirePlumber Version**: 0.5.12-1.1 (Session Manager)
- **PulseAudio Compatibility**: pipewire-pulse 1:1.4.9-2.1

## Installed Audio Packages

### Core Audio
- pipewire 1:1.4.9-2.1
- pipewire-pulse 1:1.4.9-2.1
- pipewire-alsa 1:1.4.9-2.1
- pipewire-audio 1:1.4.9-2.1
- pipewire-jack 1:1.4.9-2.1
- wireplumber 0.5.12-1.1
- libwireplumber 0.5.12-1.1

### ALSA Support
- alsa-lib 1.2.14-2.1
- alsa-utils 1.2.14-1.1
- alsa-plugins 1:1.2.12-5.1
- alsa-firmware 1.2.4-4
- alsa-ucm-conf 1.2.14-2
- alsa-topology-conf 1.2.5.1-4
- alsa-card-profiles 1:1.4.9-2.1

### Additional Audio Tools
- gst-plugin-pipewire 1:1.4.9-2.1
- callaudiod 0.1.99-2.1
- easyeffects (executed on startup)

## Service Status

### Current Status (as of last check)

- **pipewire.service**: Active (running) - user service
- **pipewire-pulse.service**: Active (running) - user service  
- **wireplumber.service**: Active (running) - user service

### Service Issues Detected

1. **WirePlumber Warnings**:
   - `wp-event-dispatcher: wp_event_dispatcher_unregister_hook: assertion 'already_registered_dispatcher == self' failed` (multiple instances)
   - `wp-event-dispatcher: <WpAsyncEventHook:0x555ad8109eb0> failed: failed to activate item: Object activation aborted: proxy destroyed`
   - `default: Failed to get percentage from UPower: org.freedesktop.DBus.Error.NameHasNoOwner`
   - `wp-device: SPA handle 'api.libcamera.enum.manager' could not be loaded; is it installed?`
   - `s-monitors-libcamera: PipeWire's libcamera SPA plugin is missing or broken. Some camera types may not be supported.`

2. **PulseAudio Connection Issues**:
   - `pactl` commands timing out with "Connection failure: Timeout"
   - This suggests PipeWire-Pulse socket may not be properly accessible

## Configuration Files

- **PipeWire Config**: `~/.config/pipewire.conf.d/` (directory exists)
- **Hyprland Audio Exec**: `exec-once = easyeffects --gapplication-service`

## Audio Hardware

- **Intel Audio**: Comet Lake PCH cAVS (00:1f.3)
- **AMD HDMI Audio**: Navi 10 HDMI Audio (03:00.1)

## Troubleshooting Steps

### If Audio Not Working:

1. **Check service status**:
   ```bash
   systemctl --user status pipewire pipewire-pulse wireplumber
   ```

2. **Restart audio services**:
   ```bash
   systemctl --user restart pipewire pipewire-pulse wireplumber
   ```

3. **Check PipeWire sockets**:
   ```bash
   ls -la /run/user/$(id -u)/pipewire*
   ```

4. **Verify PulseAudio compatibility**:
   ```bash
   pactl info
   ```

5. **Check audio devices**:
   ```bash
   pactl list short sinks
   pactl list short sources
   ```

6. **Check WirePlumber logs**:
   ```bash
   journalctl --user -u wireplumber -n 50
   ```

7. **Verify audio hardware detection**:
   ```bash
   lspci | grep -i audio
   aplay -l
   ```

### Common Fixes

- **Socket permissions**: Ensure `/run/user/$(id -u)/pipewire*` sockets are accessible
- **DBus issues**: Restart user session or log out/in
- **Missing libcamera**: Install `pipewire-v4l2` if camera support needed
- **UPower errors**: Usually harmless, but can install `upower` if battery monitoring needed

## Post-Update Issues

After recent system update, audio connection issues may be related to:
- PipeWire socket permissions
- WirePlumber configuration changes
- Service startup order
- Missing dependencies


