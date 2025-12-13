package installer

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/ln64-git/daemira/src/utility"
)

// getArchSteps returns the installation steps for Arch Linux
func (i *Installer) getArchSteps() []*InstallStep {
	return []*InstallStep{
		i.createSystemCheckStep(),
		i.createDKMSInstallStep(),
		i.createHyprlandConfigStep(),
		i.createDMSConfigStep(),
		i.createCorePackagesStep(),
		i.createAURHelperStep(),
		i.createUserAppsStep(),
		i.createServicesStep(),
		i.createUserGroupsStep(),
		i.createShellConfigStep(),
		i.createRebootPromptStep(),
	}
}

// createSystemCheckStep creates the system check step
func (i *Installer) createSystemCheckStep() *InstallStep {
	return NewInstallStep(
		"system-check",
		"System Check",
		"Verifying system requirements",
		func(ctx context.Context, installer *Installer) error {
			// Check if running as root
			currentUser, err := user.Current()
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}

			if currentUser.Uid == "0" {
				return fmt.Errorf("this script should not be run as root")
			}

			installer.logger.Info("✓ Running as user: %s", currentUser.Username)

			// Check if on Arch Linux
			if _, err := os.Stat("/etc/arch-release"); err != nil {
				return fmt.Errorf("this installer is designed for Arch Linux only")
			}

			installer.logger.Info("✓ Arch Linux detected")

			return nil
		},
	)
}

// createDKMSInstallStep creates the DKMS installation step
func (i *Installer) createDKMSInstallStep() *InstallStep {
	step := NewInstallStep(
		"dkms-install",
		"DKMS Installation",
		"Installing DKMS (DankLinux)",
		func(ctx context.Context, installer *Installer) error {
			// Check if dms command exists
			result, err := installer.shell.Execute(ctx, "command -v dms", nil)
			if err == nil && result.ExitCode == 0 {
				installer.logger.Info("DKMS already installed, skipping...")
				return nil
			}

			installer.logger.Info("Installing DKMS from install.danklinux.com...")

			// Download and execute install script
			result, err = installer.shell.Execute(ctx, "curl -fsSL https://install.danklinux.com | sh", &utility.ExecOptions{
				Timeout: 5 * time.Minute,
			})

			if err != nil || result.ExitCode != 0 {
				return fmt.Errorf("DKMS installation failed: %v\nStderr: %s", err, result.Stderr)
			}

			installer.logger.Info("DKMS installed successfully")
			return nil
		},
	)

	step.Skip = func(installer *Installer) bool {
		result, _ := installer.shell.QuickExec("command -v dms")
		return result != nil && result.ExitCode == 0
	}

	return step
}

// createHyprlandConfigStep creates the Hyprland config step
func (i *Installer) createHyprlandConfigStep() *InstallStep {
	return NewInstallStep(
		"hyprland-config",
		"Hyprland Configuration",
		"Cloning Hyprland config from ln64-git/hypr",
		func(ctx context.Context, installer *Installer) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			hyprConfigDir := fmt.Sprintf("%s/.config/hypr", homeDir)

			// Backup existing config if it exists
			if _, err := os.Stat(hyprConfigDir); err == nil {
				timestamp := time.Now().Format("20060102_150405")
				backupDir := fmt.Sprintf("%s.backup.%s", hyprConfigDir, timestamp)
				installer.logger.Info("Backing up existing config to: %s", backupDir)

				if err := os.Rename(hyprConfigDir, backupDir); err != nil {
					return fmt.Errorf("failed to backup existing config: %w", err)
				}
			}

			// Clone the config
			installer.logger.Info("Cloning Hyprland config...")
			result, err := installer.shell.Execute(ctx, fmt.Sprintf("git clone https://github.com/ln64-git/hypr %s", hyprConfigDir), &utility.ExecOptions{
				Timeout: 2 * time.Minute,
			})

			if err != nil || result.ExitCode != 0 {
				return fmt.Errorf("failed to clone Hyprland config: %v\nStderr: %s", err, result.Stderr)
			}

			installer.logger.Info("Hyprland config installed")
			return nil
		},
	)
}

// createDMSConfigStep creates the DMS config step
func (i *Installer) createDMSConfigStep() *InstallStep {
	return NewInstallStep(
		"dms-config",
		"DMS Configuration",
		"Cloning DMS config from ln64-git/dkms-config",
		func(ctx context.Context, installer *Installer) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			dmsConfigDir := fmt.Sprintf("%s/.config/DankMaterialShell", homeDir)

			// Backup existing config if it exists
			if _, err := os.Stat(dmsConfigDir); err == nil {
				timestamp := time.Now().Format("20060102_150405")
				backupDir := fmt.Sprintf("%s.backup.%s", dmsConfigDir, timestamp)
				installer.logger.Info("Backing up existing config to: %s", backupDir)

				if err := os.Rename(dmsConfigDir, backupDir); err != nil {
					return fmt.Errorf("failed to backup existing config: %w", err)
				}
			}

			// Clone the config
			installer.logger.Info("Cloning DMS config...")
			result, err := installer.shell.Execute(ctx, fmt.Sprintf("git clone https://github.com/ln64-git/dkms-config %s", dmsConfigDir), &utility.ExecOptions{
				Timeout: 2 * time.Minute,
			})

			if err != nil || result.ExitCode != 0 {
				installer.logger.Warn("Failed to clone DMS config: %v", err)
				installer.logger.Warn("You may need to set it up manually")
				return nil // Don't fail the installation
			}

			installer.logger.Info("DMS config installed")
			return nil
		},
	)
}

// createCorePackagesStep creates the core packages installation step
func (i *Installer) createCorePackagesStep() *InstallStep {
	return NewInstallStep(
		"core-packages",
		"Core Packages",
		"Installing core system packages",
		func(ctx context.Context, installer *Installer) error {
			corePackages := []string{
				"base-devel", "git", "curl", "wget",
				"hyprland", "xdg-desktop-portal-hyprland", "qt5-wayland", "qt6-wayland",
				"pipewire", "pipewire-alsa", "pipewire-pulse", "pipewire-jack", "wireplumber", "alsa-utils",
				"bluez", "bluez-utils", "blueman",
				"networkmanager", "nm-connection-editor",
				"foot", "fish", "starship", "btop", "fastfetch",
				"ttf-dejavu", "ttf-liberation", "noto-fonts", "noto-fonts-emoji",
				"adobe-source-han-sans-cn-fonts", "adobe-source-han-sans-jp-fonts", "adobe-source-han-sans-kr-fonts",
				"nautilus", "thunar",
				"p7zip", "unrar", "unzip", "zip",
			}

			installer.logger.Info("Installing %d core packages...", len(corePackages))

			for _, pkg := range corePackages {
				// Check if already installed
				result, _ := installer.shell.QuickExec(fmt.Sprintf("pacman -Q %s", pkg))
				if result != nil && result.ExitCode == 0 {
					installer.logger.Debug("%s already installed", pkg)
					continue
				}

				installer.logger.Info("Installing %s...", pkg)
				result, err := installer.shell.ExecWithSudo(fmt.Sprintf("pacman -S --noconfirm %s", pkg))
				if err != nil || result.ExitCode != 0 {
					installer.logger.Warn("Failed to install %s: %v", pkg, err)
					// Continue with other packages
				}
			}

			installer.logger.Info("Core packages installation complete")
			return nil
		},
	)
}

// createAURHelperStep creates the AUR helper (yay) installation step
func (i *Installer) createAURHelperStep() *InstallStep {
	step := NewInstallStep(
		"aur-helper",
		"AUR Helper (yay)",
		"Installing yay AUR helper",
		func(ctx context.Context, installer *Installer) error {
			installer.logger.Info("Installing yay AUR helper...")

			// Clone yay repository
			result, err := installer.shell.Execute(ctx, "cd /tmp && git clone https://aur.archlinux.org/yay.git && cd yay && makepkg -si --noconfirm", &utility.ExecOptions{
				Timeout: 5 * time.Minute,
			})

			if err != nil || result.ExitCode != 0 {
				return fmt.Errorf("failed to install yay: %v\nStderr: %s", err, result.Stderr)
			}

			// Cleanup
			installer.shell.QuickExec("rm -rf /tmp/yay")

			installer.logger.Info("yay installed successfully")
			return nil
		},
	)

	step.Skip = func(installer *Installer) bool {
		result, _ := installer.shell.QuickExec("command -v yay")
		return result != nil && result.ExitCode == 0
	}

	return step
}

// createUserAppsStep creates the user applications installation step
func (i *Installer) createUserAppsStep() *InstallStep {
	return NewInstallStep(
		"user-apps",
		"User Applications",
		"Installing user applications",
		func(ctx context.Context, installer *Installer) error {
			userApps := []string{
				"discord", "firefox", "google-chrome",
				"spotify", "obs-studio", "steam",
				"obsidian", "vscode",
				"github-cli", "docker", "docker-compose",
				"gparted", "baobab",
			}

			installer.logger.Info("Installing %d user applications...", len(userApps))

			for _, app := range userApps {
				installer.logger.Info("Installing %s...", app)
				result, err := installer.shell.Execute(ctx, fmt.Sprintf("yay -S --noconfirm %s", app), &utility.ExecOptions{
					Timeout: 10 * time.Minute,
				})

				if err != nil || result.ExitCode != 0 {
					installer.logger.Warn("Failed to install %s, skipping...", app)
					continue
				}
			}

			installer.logger.Info("User applications installation complete")
			return nil
		},
	)
}

// createServicesStep creates the services enablement step
func (i *Installer) createServicesStep() *InstallStep {
	return NewInstallStep(
		"enable-services",
		"Enable Services",
		"Enabling system services",
		func(ctx context.Context, installer *Installer) error {
			services := []string{"NetworkManager", "bluetooth", "docker"}

			for _, service := range services {
				installer.logger.Info("Enabling %s...", service)
				result, err := installer.shell.ExecWithSudo(fmt.Sprintf("systemctl enable %s", service))
				if err != nil || result.ExitCode != 0 {
					installer.logger.Warn("Failed to enable %s", service)
				}
			}

			installer.logger.Info("Services enabled")
			return nil
		},
	)
}

// createUserGroupsStep creates the user groups step
func (i *Installer) createUserGroupsStep() *InstallStep {
	return NewInstallStep(
		"user-groups",
		"User Groups",
		"Adding user to required groups",
		func(ctx context.Context, installer *Installer) error {
			currentUser, _ := user.Current()
			groups := []string{"docker", "audio", "video", "input"}

			for _, group := range groups {
				// Check if user is already in group
				result, _ := installer.shell.QuickExec("groups")
				if result != nil && strings.Contains(result.Stdout, group) {
					installer.logger.Debug("User already in %s group", group)
					continue
				}

				installer.logger.Info("Adding user to %s group...", group)
				result, err := installer.shell.ExecWithSudo(fmt.Sprintf("usermod -aG %s %s", group, currentUser.Username))
				if err != nil || result.ExitCode != 0 {
					installer.logger.Warn("Failed to add user to %s group", group)
				}
			}

			installer.logger.Info("User groups configured")
			return nil
		},
	)
}

// createShellConfigStep creates the shell configuration step
func (i *Installer) createShellConfigStep() *InstallStep {
	return NewInstallStep(
		"shell-config",
		"Shell Configuration",
		"Configuring Fish shell with Starship",
		func(ctx context.Context, installer *Installer) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			// Set fish as default shell
			currentShell := os.Getenv("SHELL")
			if !strings.Contains(currentShell, "fish") {
				installer.logger.Info("Setting fish as default shell...")
				result, err := installer.shell.QuickExec("chsh -s /usr/bin/fish")
				if err != nil || result.ExitCode != 0 {
					installer.logger.Warn("Failed to set fish as default shell")
				}
			}

			// Configure starship with pure preset
			starshipConfig := fmt.Sprintf("%s/.config/starship.toml", homeDir)
			if _, err := os.Stat(starshipConfig); os.IsNotExist(err) {
				installer.logger.Info("Setting up Starship with Pure preset...")
				result, err := installer.shell.Execute(ctx, fmt.Sprintf("starship preset pure-preset > %s", starshipConfig), nil)
				if err != nil || result.ExitCode != 0 {
					installer.logger.Warn("Failed to configure Starship")
				}
			}

			// Configure fish to use starship
			fishConfig := fmt.Sprintf("%s/.config/fish/config.fish", homeDir)
			os.MkdirAll(fmt.Sprintf("%s/.config/fish", homeDir), 0755)

			// Check if starship is already configured
			if content, err := os.ReadFile(fishConfig); err == nil {
				if strings.Contains(string(content), "starship init fish") {
					installer.logger.Info("Starship already configured in Fish")
					return nil
				}
			}

			installer.logger.Info("Adding Starship to Fish config...")
			f, err := os.OpenFile(fishConfig, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open fish config: %w", err)
			}
			defer f.Close()

			f.WriteString("\n# Initialize Starship prompt\n")
			f.WriteString("starship init fish | source\n")

			installer.logger.Info("Shell configuration complete")
			return nil
		},
	)
}

// createRebootPromptStep creates the reboot prompt step
func (i *Installer) createRebootPromptStep() *InstallStep {
	return NewInstallStep(
		"reboot-prompt",
		"Reboot Prompt",
		"Prompting for system reboot",
		func(ctx context.Context, installer *Installer) error {
			installer.logger.Info("")
			installer.logger.Info("Note: You may need to log out and back in for group changes to take effect")
			installer.logger.Info("")
			installer.logger.Warn("Would you like to reboot now? (y/N)")

			// In headless mode, skip the reboot
			// In TUI mode, this would be interactive
			installer.logger.Info("Skipping automatic reboot in headless mode")
			installer.logger.Info("Please reboot manually when ready: sudo systemctl reboot")

			return nil
		},
	)
}

// getFedoraSteps returns the installation steps for Fedora (placeholder)
func (i *Installer) getFedoraSteps() []*InstallStep {
	return []*InstallStep{
		NewInstallStep(
			"fedora-placeholder",
			"Fedora Support",
			"Fedora installation not implemented yet",
			func(ctx context.Context, installer *Installer) error {
				return fmt.Errorf("Fedora support is planned but not yet implemented")
			},
		),
	}
}

// getDebianSteps returns the installation steps for Debian/Ubuntu (placeholder)
func (i *Installer) getDebianSteps() []*InstallStep {
	return []*InstallStep{
		NewInstallStep(
			"debian-placeholder",
			"Debian/Ubuntu Support",
			"Debian/Ubuntu installation not implemented yet",
			func(ctx context.Context, installer *Installer) error {
				return fmt.Errorf("Debian/Ubuntu support is planned but not yet implemented")
			},
		),
	}
}
