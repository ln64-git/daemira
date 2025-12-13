# Daemira - Personal System Daemon

A comprehensive personal system daemon for Linux with Google Drive sync, system updates, health monitoring, and more.

## Quick Start

### Keep System Updated

Run with `sudo` to enable system updates:

```bash
sudo go run main.go
```

Or if installed:
```bash
sudo daemira
```

This will:
- Run system update immediately
- Schedule automatic updates every 6 hours
- Keep running in the background

### Keep Google Drive Synced

Run as your regular user (rclone config is user-specific):

```bash
go run main.go
```

Or if installed:
```bash
daemira
```

This will:
- Start Google Drive sync automatically
- Sync all configured directories every 30 seconds
- Keep running in the background

### Run Both Services

**Option 1: Use the start script (recommended)**
```bash
make start
# or
./scripts/start-daemira.sh
```

This starts both:
- System update service (as root)
- Google Drive sync service (as user)

**Option 2: Manual (two terminals)**
```bash
# Terminal 1: System updates
sudo go run main.go

# Terminal 2: Google Drive sync
go run main.go
```

**Option 3: Background processes**
```bash
# System updates (as root)
sudo go run main.go > /tmp/daemira-updates.log 2>&1 &

# Google Drive sync (as user)
go run main.go > /tmp/daemira-gdrive.log 2>&1 &
```

### Stop Services

```bash
make stop
# or
./scripts/stop-daemira.sh
# or manually
sudo pkill -f daemira
```

## Commands

- `daemira status` - Show comprehensive system status
- `daemira gdrive status` - Show Google Drive sync status
- `daemira gdrive sync` - Force sync all directories immediately
- `daemira system update` - Run system update manually
- `daemira install` - Run system installer

## Configuration

Configuration is loaded from `.env` file in the project root. See `src/config/config.go` for available options.

## Logs

- Console output: Colored logs to stdout
- File logs: `log/current.log` (rotates automatically)

## Development

```bash
# Run in development mode
make dev

# Run with specific command
make dev ARGS="status"

# Build binary
make build

# Install to system
make install
```

## Notes

- **System updates require root** - Run with `sudo` or configure passwordless sudo
- **Google Drive sync requires user config** - Run as your regular user (not root)
- **Both can run simultaneously** - Use the start script or run in separate terminals
