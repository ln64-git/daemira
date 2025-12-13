#!/bin/bash
# Stop Daemira daemon

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Stopping Daemira daemon...${NC}"

# Stop all daemira processes
sudo pkill -f "daemira" || true

# Wait a moment
sleep 1

# Check if still running
if pgrep -f "daemira" > /dev/null; then
    echo -e "${YELLOW}Some processes still running, force killing...${NC}"
    sudo pkill -9 -f "daemira" || true
    sleep 1
fi

if pgrep -f "daemira" > /dev/null; then
    echo -e "${YELLOW}Warning: Some daemira processes may still be running${NC}"
else
    echo -e "${GREEN}✓ Daemira daemon stopped${NC}"
fi

