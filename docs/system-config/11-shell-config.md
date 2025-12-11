# Shell Configuration

## Shell Environment

- **Default Shell**: Fish (`/usr/bin/fish`)
- **Shell Version**: Check with `fish --version`
- **Config Location**: `~/.config/fish/config.fish`

## Fish Shell Configuration

### Prompt
- **Prompt System**: Starship (`starship init fish`)
- **Custom Prompt Function**: `fish_prompt` defined in config
- **Greeting**: Disabled (`set fish_greeting`)

### PATH Configuration
- **Custom PATH**: `/opt/antigravity` added to PATH
- **Location**: Set in `~/.config/fish/config.fish`

### Quickshell Integration
- Background theme sequences disabled (commented out)
- Quickshell terminal sequences file: `~/.local/state/quickshell/user/generated/terminal/sequences.txt`

## Configuration File Contents

```fish
function fish_prompt -d "Write out the prompt"
    printf '%s@%s %s%s%s > ' $USER $hostname \
        (set_color $fish_color_cwd) (prompt_pwd) (set_color normal)
end

set -gx PATH /opt/antigravity $PATH

if status is-interactive
    set fish_greeting
    starship init fish | source
end
```

## Related Tools

- **Starship**: Cross-shell prompt (installed and configured)
- **Quickshell**: Terminal integration (part of Hyprland setup)

## User Groups

User `ln64` is a member of:
- `sys` - System administration
- `network` - Network configuration
- `docker` - Docker access
- `realtime` - Real-time scheduling
- `libvirt` - Virtualization
- `i2c` - I2C device access
- `rfkill` - RF kill switch control
- `users` - Standard users
- `video` - Video device access
- `storage` - Storage device access
- `lp` - Printer access
- `input` - Input device access
- `audio` - Audio device access
- `wheel` - Sudo access

## Environment Variables

Check current environment:
```bash
env | sort
```

Key variables likely set:
- `PATH` - Includes `/opt/antigravity`
- `SHELL` - `/usr/bin/fish`
- `USER` - `ln64`
- `HOME` - `/home/ln64`

