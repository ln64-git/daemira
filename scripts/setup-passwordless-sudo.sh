#!/bin/bash
# Helper script to configure passwordless sudo for Daemira system updates
# This script creates a sudoers.d file with the correct configuration

USERNAME="${1:-${USER}}"
SUDOERS_FILE="/etc/sudoers.d/daemira-updates"

echo "Configuring passwordless sudo for user: $USERNAME"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Create the sudoers.d entry
cat > "$SUDOERS_FILE" << EOF
# Daemira system update commands - passwordless sudo
# This file allows $USERNAME to run system update commands without password
$USERNAME ALL=(ALL) NOPASSWD: /usr/bin/pacman, /usr/bin/paccache, /usr/bin/pacman-optimize, /usr/bin/grub-mkconfig, /usr/bin/systemctl, /usr/bin/fwupdmgr, /usr/bin/fstrim, /usr/bin/dkms
EOF

# Set correct permissions
chmod 0440 "$SUDOERS_FILE"

# Validate the sudoers file
if visudo -c -f "$SUDOERS_FILE" 2>/dev/null; then
    echo "✓ Successfully configured passwordless sudo for Daemira updates"
    echo "  File created: $SUDOERS_FILE"
    echo ""
    echo "You can now run system updates without entering a password."
else
    echo "✗ Error: Invalid sudoers syntax. Removing file..."
    rm -f "$SUDOERS_FILE"
    exit 1
fi
