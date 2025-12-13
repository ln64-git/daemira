#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

prompt() {
    echo -e "${YELLOW}[PROMPT]${NC} $1"
}

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   error "This script should not be run as root. Run as your user account."
fi

# Ensure we're on Arch Linux
if [[ ! -f /etc/arch-release ]]; then
    error "This script is designed for Arch Linux only."
fi

info "Starting Arch Linux installation script..."

# ==================
# DKMS INSTALLATION
# ==================
info "Installing DKMS (DankLinux)..."
if command -v dms &> /dev/null; then
    warn "DKMS already installed, skipping..."
else
    prompt "Installing DKMS from install.danklinux.com..."
    curl -fsSL https://install.danklinux.com | sh
    success "DKMS installed successfully"
fi

# ==================
# HYPRLAND CONFIG
# ==================
info "Setting up Hyprland configuration..."

# Backup existing config if it exists
if [[ -d ~/.config/hypr ]]; then
    warn "Existing Hyprland config found. Creating backup..."
    mv ~/.config/hypr ~/.config/hypr.backup.$(date +%Y%m%d_%H%M%S)
    success "Backup created"
fi

# Clone hypr config
info "Cloning Hyprland config from ln64-git/hypr..."
mkdir -p ~/.config
git clone https://github.com/ln64-git/hypr ~/.config/hypr
success "Hyprland config installed"

# ==================
# DMS (DANKMATERIALSHELL) CONFIG
# ==================
info "Setting up DankMaterialShell configuration..."

# Backup existing DMS config if it exists
if [[ -d ~/.config/DankMaterialShell ]]; then
    warn "Existing DMS config found. Creating backup..."
    mv ~/.config/DankMaterialShell ~/.config/DankMaterialShell.backup.$(date +%Y%m%d_%H%M%S)
    success "Backup created"
fi

# Clone DMS config
info "Cloning DMS config from ln64-git/dkms-config..."
if git clone https://github.com/ln64-git/dkms-config ~/.config/DankMaterialShell 2>/dev/null; then
    success "DMS config installed"
else
    warn "Failed to clone DMS config from ln64-git/dkms-config"
    warn "You may need to set it up manually or check your internet connection"
fi

# ==================
# INSTALL CORE PACKAGES
# ==================
info "Installing core packages..."

CORE_PACKAGES=(
    # Base system
    "base-devel"
    "git"
    "curl"
    "wget"

    # Desktop environment dependencies
    "hyprland"
    "xdg-desktop-portal-hyprland"
    "qt5-wayland"
    "qt6-wayland"

    # Audio
    "pipewire"
    "pipewire-alsa"
    "pipewire-pulse"
    "pipewire-jack"
    "wireplumber"
    "alsa-utils"

    # Bluetooth
    "bluez"
    "bluez-utils"
    "blueman"

    # Network
    "networkmanager"
    "nm-connection-editor"

    # Terminal & utilities
    "alacritty"
    "btop"
    "fastfetch"
    "fish"
    "foot"
    "starship"
    "neofetch"

    # Fonts
    "ttf-dejavu"
    "ttf-liberation"
    "noto-fonts"
    "noto-fonts-emoji"
    "adobe-source-han-sans-cn-fonts"
    "adobe-source-han-sans-jp-fonts"
    "adobe-source-han-sans-kr-fonts"

    # File management
    "nautilus"
    "thunar"

    # Archive support
    "p7zip"
    "unrar"
    "unzip"
    "zip"
)

for package in "${CORE_PACKAGES[@]}"; do
    if ! pacman -Q "$package" &> /dev/null; then
        info "Installing $package..."
        sudo pacman -S --noconfirm "$package"
    else
        info "$package already installed"
    fi
done

success "Core packages installed"

# ==================
# INSTALL AUR HELPER (YAY)
# ==================
if ! command -v yay &> /dev/null; then
    info "Installing yay AUR helper..."
    cd /tmp
    git clone https://aur.archlinux.org/yay.git
    cd yay
    makepkg -si --noconfirm
    cd ~
    rm -rf /tmp/yay
    success "yay installed"
else
    info "yay already installed"
fi

# ==================
# INSTALL USER APPLICATIONS
# ==================
info "Installing user applications..."

USER_APPS=(
    # Communication
    "discord"

    # Browsers
    "firefox"
    "google-chrome"

    # Media
    "spotify"
    "obs-studio"

    # Gaming
    "steam"

    # Productivity
    "obsidian"
    "vscode"

    # Development tools
    "github-cli"
    "docker"
    "docker-compose"

    # System utilities
    "gparted"
    "baobab"
)

for app in "${USER_APPS[@]}"; do
    if ! pacman -Q "$app" &> /dev/null && ! yay -Q "$app" &> /dev/null; then
        info "Installing $app..."
        yay -S --noconfirm "$app" || warn "Failed to install $app, skipping..."
    else
        info "$app already installed"
    fi
done

success "User applications installed"

# ==================
# ENABLE SERVICES
# ==================
info "Enabling system services..."

SERVICES=(
    "NetworkManager"
    "bluetooth"
    "docker"
)

for service in "${SERVICES[@]}"; do
    info "Enabling $service..."
    sudo systemctl enable "$service" || warn "Failed to enable $service"
done

success "System services enabled"

# ==================
# ADD USER TO GROUPS
# ==================
info "Adding user to required groups..."

GROUPS=(
    "docker"
    "audio"
    "video"
    "input"
)

for group in "${GROUPS[@]}"; do
    if ! groups | grep -q "$group"; then
        info "Adding user to $group group..."
        sudo usermod -aG "$group" "$USER"
    fi
done

success "User added to groups"

# ==================
# CONFIGURE SHELL (FISH + STARSHIP)
# ==================
info "Configuring Fish shell with Starship prompt..."

# Set fish as default shell
if [[ "$SHELL" != *"fish"* ]]; then
    info "Setting fish as default shell..."
    chsh -s /usr/bin/fish
    success "Fish shell set as default"
else
    info "Fish is already the default shell"
fi

# Configure starship with pure preset
info "Setting up Starship with Pure preset..."
mkdir -p ~/.config
if [[ ! -f ~/.config/starship.toml ]]; then
    starship preset pure-preset > ~/.config/starship.toml
    success "Starship configured with Pure preset"
else
    warn "Starship config already exists, skipping..."
fi

# Configure fish to use starship
mkdir -p ~/.config/fish
if ! grep -q "starship init fish" ~/.config/fish/config.fish 2>/dev/null; then
    info "Adding Starship to Fish config..."
    echo "" >> ~/.config/fish/config.fish
    echo "# Initialize Starship prompt" >> ~/.config/fish/config.fish
    echo "starship init fish | source" >> ~/.config/fish/config.fish
    success "Starship added to Fish config"
else
    info "Starship already configured in Fish"
fi

# Set foot as default terminal in Hyprland config (if not already set)
if [[ -f ~/.config/hypr/hyprland.conf ]]; then
    if ! grep -q "foot" ~/.config/hypr/hyprland.conf; then
        info "Setting foot as default terminal in Hyprland..."
        echo "" >> ~/.config/hypr/hyprland.conf
        echo "# Set default terminal" >> ~/.config/hypr/hyprland.conf
        echo '$term = foot' >> ~/.config/hypr/hyprland.conf
        success "Foot set as default terminal"
    fi
fi

success "Shell configuration complete"

# ==================
# FINAL STEPS
# ==================
info "Installation complete!"
echo ""
success "Next steps:"
echo "  1. Reboot your system to apply all changes"
echo "  2. Log in to Hyprland"
echo "  3. Configure your applications as needed"
echo ""
warn "Note: You may need to log out and back in for group changes to take effect"
echo ""
prompt "Would you like to reboot now? (y/N)"
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    info "Rebooting in 5 seconds..."
    sleep 5
    systemctl reboot
fi
