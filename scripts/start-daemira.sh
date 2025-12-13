#!/bin/bash
# Start Daemira daemon (both system updates and Google Drive sync)
# This script runs both services in the background

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DAEMIRA_BIN="$PROJECT_ROOT/bin/daemira"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Daemira daemon...${NC}"

# Check if daemira binary exists, if not build it
if [ ! -f "$DAEMIRA_BIN" ]; then
    echo -e "${YELLOW}Binary not found, building...${NC}"
    cd "$PROJECT_ROOT"
    go build -o bin/daemira main.go
fi

# Check if already running
if pgrep -f "daemira" > /dev/null; then
    echo -e "${YELLOW}Daemira is already running!${NC}"
    echo "To stop it, run: pkill -f daemira"
    exit 1
fi

# Start system update service (as root)
echo -e "${GREEN}Starting system update service (as root)...${NC}"
sudo "$DAEMIRA_BIN" > /tmp/daemira-system-updates.log 2>&1 &
SYSTEM_UPDATE_PID=$!
echo "System update service PID: $SYSTEM_UPDATE_PID"

# Wait a moment for root service to start
sleep 2

# Start Google Drive sync service (as user)
echo -e "${GREEN}Starting Google Drive sync service (as user)...${NC}"
"$DAEMIRA_BIN" > /tmp/daemira-gdrive-sync.log 2>&1 &
GDRIVE_SYNC_PID=$!
echo "Google Drive sync service PID: $GDRIVE_SYNC_PID"

echo ""
echo -e "${GREEN}✓ Daemira daemon started!${NC}"
echo ""
echo "Services running:"
echo "  - System Updates: PID $SYSTEM_UPDATE_PID (running as root)"
echo "  - Google Drive Sync: PID $GDRIVE_SYNC_PID (running as user)"
echo ""
echo "Logs:"
echo "  - System updates: /tmp/daemira-system-updates.log"
echo "  - Google Drive sync: /tmp/daemira-gdrive-sync.log"
echo ""
echo "To stop:"
echo "  sudo pkill -f 'daemira'"
echo ""
echo "To check status:"
echo "  $DAEMIRA_BIN status"
echo "  $DAEMIRA_BIN gdrive status"

