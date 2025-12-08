# ✨ Daemira

**Daemira** — a personal system daemon for automated Google Drive sync and system maintenance on Arch Linux.

A contemporary, type-safe TypeScript daemon that handles background tasks so you don't have to. Built with Bun for speed and simplicity.

## Features

- 🔄 **Google Drive Sync** - Bidirectional sync using rclone bisync with intelligent exclude patterns
- 🔧 **System Updates** - Automated Arch Linux maintenance (pacman, AUR, firmware, cleanup)
- 📝 **Smart Logging** - File-based logging with rotation and configurable log levels
- 🗂️ **Notion Integration** - Ready-to-use Notion API client (CRUD, file sync)
- 🛡️ **Type Safety** - Full TypeScript with Zod validation

## Quick Start

### Prerequisites

```bash
# Install rclone
sudo pacman -S rclone

# Configure Google Drive remote
rclone config
# Name it 'gdrive' or customize with RCLONE_REMOTE_NAME in .env
```

### Installation

```bash
# Install dependencies
bun install

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Make entry point executable
chmod +x src/main.ts
```

### Usage

**Via bun scripts:**
```bash
bun start                  # Start all services
bun gdrive:status         # Check Google Drive sync status
bun gdrive:sync           # Force sync now
bun system:update         # Run system update
bun system:status         # Check update schedule
```

**Direct execution:**
```bash
./src/main.ts                     # Start daemon
./src/main.ts gdrive:start        # Start Google Drive sync
./src/main.ts gdrive:status       # Show sync status
./src/main.ts gdrive:patterns     # List exclude patterns
./src/main.ts gdrive:exclude "*.tmp"  # Add exclude pattern
./src/main.ts system:update       # Run system update
./src/main.ts system:status       # Show update history
```

## Configuration

Create `.env` from template:

```bash
# Environment
NODE_ENV=development
LOG_LEVEL=info              # debug, info, warn, error

# Google Drive
RCLONE_REMOTE_NAME=gdrive

# Notion (optional)
NOTION_TOKEN=your_token
NOTION_DATABASE_ID=your_db_id

# AI Providers (optional)
OPENAI_API_KEY=your_key
GEMINI_API_KEY=your_key
GROK_API_KEY=your_key
```

## Architecture

```
daemira/
├── src/
│   ├── main.ts             # Entry point
│   ├── Daemira.ts          # Main orchestrator
│   ├── config/
│   │   └── index.ts        # Zod-validated configuration
│   ├── utility/
│   │   ├── Logger.ts       # File logger with rotation
│   │   ├── Shell.ts        # Command executor
│   │   ├── GoogleDrive.ts  # Google Drive sync
│   │   └── Notion.ts       # Notion API client
│   └── features/
│       └── system-update/
│           ├── SystemUpdate.ts
│           └── index.ts
└── log/                    # Log files (auto-created)
```

## Logs

Logs are stored in `./log/`:
- `current.log` - Current session
- `archive/bot-1.log` to `bot-7.log` - Last 7 sessions

Set log level via `LOG_LEVEL` environment variable: `debug`, `info`, `warn`, `error`

## Google Drive Sync

### Default Directories Synced

- Documents
- Downloads
- Pictures
- Desktop
- Music
- Source
- .config

### Intelligent Exclusions

Automatically excludes 60+ patterns:
- Build artifacts (`node_modules`, `dist`, `build`, `.next`, `target`, etc.)
- Version control (`.git`, `.svn`, `.hg`)
- IDE files (`.vscode`, `.idea`, `*.swp`)
- OS files (`.DS_Store`, `Thumbs.db`)
- Environment files (`.env`, `.env.local`)
- Caches and temporary files
- Large media caches (Steam, browser caches)

### Sync Features

- **Bidirectional** - Changes sync both ways using rclone bisync
- **Conflict Resolution** - "Newer wins" strategy
- **Queue-Based** - One sync at a time to avoid overwhelming the system
- **Periodic Sync** - Every 30 seconds (configurable)
- **Resilient** - Auto-recovery from interrupted syncs
- **Lock Management** - Automatic cleanup of stale lock files

## System Updates

Comprehensive Arch Linux maintenance workflow:

1. Refresh mirrorlist (optional)
2. Update keyrings (archlinux-keyring, cachyos-keyring)
3. Update package databases
4. Upgrade packages (pacman)
5. Update AUR packages (yay)
6. Update firmware (fwupd)
7. Remove orphaned packages
8. Clean package cache (paccache)
9. Clean AUR cache
10. Optimize pacman database (optional)
11. Update GRUB configuration
12. Reload systemd daemon
13. Check for .pacnew configuration files
14. Check if reboot required (kernel updates)

**Default Schedule:** Every 6 hours
**Timeout:** 10 minutes per step
**Logging:** All output captured to log files

## Development

```bash
# Watch mode (auto-restart on changes)
bun dev

# Type checking
tsc --noEmit

# Run specific command
bun start gdrive:status
```

## Utilities

### Logger

Singleton logger with file rotation and level filtering:

```typescript
import { Logger } from "./utility/Logger";
const logger = Logger.getInstance();

logger.debug("Debug info");
logger.info("General info");
logger.warn("Warning message");
logger.error("Error occurred");
```

### Shell

Execute system commands with proper error handling:

```typescript
import { Shell } from "./utility/Shell";

const result = await Shell.execute("ls -la", {
  timeout: 5000,
  onStdout: (line) => console.log(line),
  onStderr: (line) => console.error(line),
});

console.log(result.exitCode, result.stdout, result.stderr);
```

### Notion

Notion API operations with retry logic:

```typescript
import { Notion } from "./utility/Notion";
import { config } from "./config";

const notion = new Notion(config.notionToken);

// Query database
const pages = await notion.queryDatabase(config.notionDatabaseId);

// Create page
const page = await notion.createPage(databaseId, properties, content);

// Update page
await notion.updatePage(pageId, properties);

// Sync file to Notion
await notion.syncFileToPage(pageId, "./README.md");
```

## License

Private use only.

---

Built with ❤️ using [Bun](https://bun.sh), TypeScript, and modern async patterns.
