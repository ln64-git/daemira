/**
 * SystemUpdate Feature - Automated system maintenance for Arch Linux
 *
 * Features:
 * - Periodic system updates (default: 6 hours)
 * - Comprehensive update steps (pacman, AUR, firmware, cleanup)
 * - Update history tracking
 * - .pacnew file detection
 * - Reboot requirement detection
 * - Integration with Shell utility and Logger
 */

package systemupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/ln64-git/daemira/src/utility"
)

const (
	errPasswordlessSudoNotConfigured = "passwordless sudo is not configured; system updates require sudo access without password prompts"
)

// SystemUpdateOptions configures the system update service
type SystemUpdateOptions struct {
	Interval  time.Duration // Default: 6 hours
	AutoStart bool          // Start scheduler immediately
}

// UpdateStep represents a single update step
type UpdateStep struct {
	Name     string
	Cmd      string
	Optional bool
}

// UpdateHistoryEntry tracks update execution history
type UpdateHistoryEntry struct {
	Timestamp time.Time
	Success   bool
	Duration  time.Duration
}

// SystemUpdate manages automated system updates for Arch Linux
type SystemUpdate struct {
	logger         *utility.Logger
	shell          *utility.Shell
	isRunning      bool
	updateInterval time.Duration
	lastUpdateTime *time.Time
	updateHistory  []UpdateHistoryEntry
	mu             sync.RWMutex
	stopChan       chan struct{}
	ticker         *time.Ticker
}

// NewSystemUpdate creates a new SystemUpdate instance
func NewSystemUpdate(logger *utility.Logger, options *SystemUpdateOptions) *SystemUpdate {
	interval := 6 * time.Hour
	if options != nil && options.Interval > 0 {
		interval = options.Interval
	}

	if logger == nil {
		logger = utility.GetLogger()
	}

	su := &SystemUpdate{
		logger:         logger,
		shell:          utility.NewShell(logger),
		updateInterval: interval,
		updateHistory:  make([]UpdateHistoryEntry, 0),
		stopChan:       make(chan struct{}),
	}

	if options != nil && options.AutoStart {
		su.Start()
	}

	return su
}

// Start begins the periodic update scheduler
func (su *SystemUpdate) Start() {
	su.mu.Lock()
	defer su.mu.Unlock()

	if su.isRunning {
		su.logger.Warn("SystemUpdate scheduler already running")
		return
	}

	su.isRunning = true
	su.logger.Info("Starting system update scheduler (interval: %v)", su.updateInterval)

	// Run immediately
	go su.runUpdate(context.Background())

	// Schedule periodic updates
	su.ticker = time.NewTicker(su.updateInterval)
	go func() {
		for {
			select {
			case <-su.ticker.C:
				su.runUpdate(context.Background())
			case <-su.stopChan:
				return
			}
		}
	}()
}

// Stop halts the scheduler
func (su *SystemUpdate) Stop() {
	su.mu.Lock()
	defer su.mu.Unlock()

	if !su.isRunning {
		su.logger.Warn("SystemUpdate scheduler not running")
		return
	}

	su.isRunning = false
	if su.ticker != nil {
		su.ticker.Stop()
	}
	close(su.stopChan)

	su.logger.Info("System update scheduler stopped")
}

// RunUpdate executes system update immediately
func (su *SystemUpdate) RunUpdate(ctx context.Context) error {
	return su.runUpdate(ctx)
}

// runUpdate is the internal update execution method
func (su *SystemUpdate) runUpdate(ctx context.Context) error {
	su.logger.Info("Starting system update...")
	fmt.Println("=== Starting System Update ===")
	startTime := time.Now()

	// Check if running as root - if so, no sudo needed
	if su.isRoot() {
		su.logger.Info("Running as root - sudo not required")
	} else {
		// Not running as root - check for passwordless sudo
		hasPasswordlessSudo, err := su.checkPasswordlessSudo(ctx)
		if err != nil || !hasPasswordlessSudo {
			username := os.Getenv("USER")
			if username == "" {
				if u, err := user.Current(); err == nil {
					username = u.Username
				} else {
					username = "ln64"
				}
			}

			fmt.Printf("\n✗ ERROR: system updates require root privileges\n")
			fmt.Println("\nSOLUTION 1 (EASIEST): Run daemira with sudo:")
			fmt.Println("  sudo go run main.go")
			fmt.Println("  # Or if installed:")
			fmt.Println("  sudo daemira")
			fmt.Println("\nSOLUTION 2 (RECOMMENDED): Use the setup script for passwordless sudo:")
			fmt.Println("  sudo ./scripts/setup-sudo-daemira.sh")
			fmt.Println("  # This allows: sudo daemira (without password)")
			fmt.Println("\nSOLUTION 3: Configure passwordless sudo manually:")
			fmt.Println("  sudo visudo")
			fmt.Printf("  # Add this line (replace '%s' with your username):\n", username)
			fmt.Printf("  %s ALL=(ALL) NOPASSWD: /usr/local/bin/daemira\n", username)
			fmt.Println("  # Or for development (running from source):")
			fmt.Printf("  %s ALL=(ALL) NOPASSWD: /usr/bin/go\n", username)
			fmt.Println("\nSOLUTION 4: Configure passwordless sudo for specific commands only:")
			fmt.Println("  sudo visudo")
			fmt.Printf("  # Add this line:\n")
			fmt.Printf("  %s ALL=(ALL) NOPASSWD: /usr/bin/pacman, /usr/bin/paccache, /usr/bin/pacman-optimize, /usr/bin/grub-mkconfig, /usr/bin/systemctl, /usr/bin/fwupdmgr, /usr/bin/fstrim, /usr/bin/dkms\n", username)
			su.logger.Error("%s", errPasswordlessSudoNotConfigured)
			//nolint:ST1005,SA1006 // error message is correct, linter false positive
			return errors.New(errPasswordlessSudoNotConfigured)
		}
	}

	var err error
	success := true

	// Execute update steps
	if err = su.executeUpdateSteps(ctx); err != nil {
		success = false
	}

	// Execute optimization steps
	if err2 := su.executeOptimizationSteps(ctx); err2 != nil {
		su.logger.Warn("Some optimization steps failed: %v", err2)
	}

	// Check for .pacnew files
	su.checkPacnewFiles(ctx)

	// Check if reboot required
	su.checkRebootRequired(ctx)

	// Post-update verification
	su.postUpdateVerification(ctx)

	duration := time.Since(startTime)
	su.mu.Lock()
	now := time.Now()
	su.lastUpdateTime = &now
	su.updateHistory = append(su.updateHistory, UpdateHistoryEntry{
		Timestamp: now,
		Success:   success,
		Duration:  duration,
	})
	// Keep only last 10 entries
	if len(su.updateHistory) > 10 {
		su.updateHistory = su.updateHistory[len(su.updateHistory)-10:]
	}
	su.mu.Unlock()

	if success {
		successMsg := fmt.Sprintf("System update completed successfully in %.1fs", duration.Seconds())
		su.logger.Info(successMsg)
		fmt.Printf("\n✓ %s\n", successMsg)
	} else {
		errorMsg := fmt.Sprintf("System update failed: %v", err)
		su.logger.Error(errorMsg)
		fmt.Printf("\n✗ %s\n", errorMsg)
		return err
	}

	return nil
}

// GetStatus returns the current update status
func (su *SystemUpdate) GetStatus() map[string]interface{} {
	su.mu.RLock()
	defer su.mu.RUnlock()

	status := map[string]interface{}{
		"running": su.isRunning,
		"history": su.updateHistory,
	}

	if su.lastUpdateTime != nil {
		status["lastUpdate"] = su.lastUpdateTime.Unix()
		status["nextUpdate"] = su.lastUpdateTime.Add(su.updateInterval).Unix()
	}

	return status
}

// checkPasswordlessSudo verifies if passwordless sudo is available
func (su *SystemUpdate) checkPasswordlessSudo(ctx context.Context) (bool, error) {
	result, err := su.shell.Execute(ctx, "sudo -n true", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return false, err
	}
	return result.ExitCode == 0, nil
}

// isRoot checks if running as root
func (su *SystemUpdate) isRoot() bool {
	return os.Geteuid() == 0
}

// commandExists checks if a command exists in PATH
func (su *SystemUpdate) commandExists(ctx context.Context, command string) bool {
	// Extract base command (first word before space)
	parts := strings.Fields(command)
	baseCmd := parts[0]
	if strings.HasPrefix(baseCmd, "sudo") {
		if len(parts) > 1 {
			baseCmd = parts[1]
		}
	}

	result, err := su.shell.Execute(ctx, fmt.Sprintf("command -v %s", baseCmd), &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return false
	}
	return result.ExitCode == 0
}

// executeUpdateSteps runs all update steps
func (su *SystemUpdate) executeUpdateSteps(ctx context.Context) error {
	fmt.Println("\n=== Executing Update Steps ===")

	// Determine command prefix based on whether we're root
	cmdPrefix := ""
	if !su.isRoot() {
		cmdPrefix = "sudo -n "
	}

	steps := []UpdateStep{
		{
			Name:     "Refreshing mirrorlist",
			Cmd:      cmdPrefix + "pacman-mirrors --fasttrack",
			Optional: true,
		},
		{
			Name: "Updating keyrings",
			Cmd:  cmdPrefix + "pacman -Sy --needed --noconfirm archlinux-keyring cachyos-keyring",
		},
		{
			Name: "Updating package databases",
			Cmd:  cmdPrefix + "pacman -Syy --noconfirm",
		},
		{
			Name: "Upgrading packages",
			Cmd:  cmdPrefix + "pacman -Syu --noconfirm",
		},
		{
			Name: "Updating AUR packages",
			Cmd:  "yay -Sua --noconfirm --answerclean All --answerdiff None --answeredit None --removemake --cleanafter",
		},
		{
			Name:     "Updating firmware",
			Cmd:      cmdPrefix + "fwupdmgr refresh --force && " + cmdPrefix + "fwupdmgr update -y",
			Optional: true,
		},
		{
			Name: "Removing orphaned packages",
			Cmd:  `orphans=$(pacman -Qdtq 2>/dev/null); [ -z "$orphans" ] || ` + cmdPrefix + `pacman -Rns --noconfirm $orphans`,
		},
		{
			Name: "Cleaning package cache",
			Cmd:  cmdPrefix + "paccache -rk2",
		},
		{
			Name: "Cleaning uninstalled cache",
			Cmd:  cmdPrefix + "paccache -ruk0",
		},
		{
			Name: "Cleaning yay cache",
			Cmd:  "yes | yay -Sc --noconfirm --answerclean All --answerdiff None --answeredit None --removemake",
		},
		{
			Name:     "Optimizing pacman database",
			Cmd:      cmdPrefix + "pacman-optimize",
			Optional: true,
		},
		{
			Name: "Updating GRUB",
			Cmd:  cmdPrefix + "grub-mkconfig -o /boot/grub/grub.cfg",
		},
		{
			Name: "Reloading systemd daemon",
			Cmd:  cmdPrefix + "systemctl daemon-reload",
		},
	}

	for i, step := range steps {
		stepNum := i + 1
		su.logger.Info("Step %d/%d: %s", stepNum, len(steps), step.Name)
		fmt.Printf("\n[%d/%d] %s...\n", stepNum, len(steps), step.Name)

		// For optional steps, check if command exists first
		if step.Optional {
			if !su.commandExists(ctx, step.Cmd) {
				skipMsg := fmt.Sprintf("Skipped (optional): %s - command not available on this system", step.Name)
				su.logger.Info(skipMsg)
				fmt.Printf("  ⚠ %s\n", skipMsg)
				continue
			}
		}

		// Use shorter timeout for first few commands
		timeout := 30 * time.Second
		if i >= 3 {
			timeout = 10 * time.Minute
		}

		passwordDetected := false
		var stdoutLines []string
		var stderrLines []string

		result, err := su.shell.Execute(ctx, step.Cmd, &utility.ExecOptions{
			Timeout: timeout,
			StdoutCallback: func(line string) {
				stdoutLines = append(stdoutLines, line)
				su.logger.Debug("  %s", line)
				if strings.TrimSpace(line) != "" {
					fmt.Printf("  %s\n", line)
				}
			},
			StderrCallback: func(line string) {
				stderrLines = append(stderrLines, line)
				lowerLine := strings.ToLower(line)
				if strings.Contains(lowerLine, "password") ||
					strings.Contains(lowerLine, "sudo: a password is required") {
					passwordDetected = true
				}

				if strings.TrimSpace(line) != "" && !passwordDetected {
					lowerLine := strings.ToLower(line)
					isNormalWarning := strings.Contains(lowerLine, "warning:") &&
						(strings.Contains(lowerLine, "is newer than") ||
							strings.Contains(lowerLine, "is up to date") ||
							strings.Contains(lowerLine, "-- skipping"))
					if !isNormalWarning {
						fmt.Printf("  [stderr] %s\n", line)
					}
				}
			},
		})

		// Check for password requirement
		if passwordDetected || (result != nil && result.Stderr != "" &&
			(strings.Contains(strings.ToLower(result.Stderr), "password") ||
				strings.Contains(strings.ToLower(result.Stderr), "sudo: a password is required"))) {
			errorMsg := fmt.Sprintf("sudo password required for: %s", step.Name)
			fmt.Printf("\n✗ ERROR: %s\n", errorMsg)
			fmt.Printf("  Command: %s\n", step.Cmd)
			fmt.Println("  Solutions:")
			fmt.Println("  1. Configure passwordless sudo for this command")
			fmt.Printf("  2. Run manually: %s\n", step.Cmd)
			fmt.Println("  3. Run entire update with sudo: sudo daemira system:update")
			//nolint:SA1006 // fmt.Errorf is correct here with format string and argument
			return fmt.Errorf("sudo password required for: %s", step.Name)
		}

		if err != nil {
			if step.Optional {
				su.logger.Warn("Skipped (optional): %s - %v", step.Name, err)
				fmt.Printf("  ⚠ Skipped (optional): %s\n", step.Name)
				continue
			}
			return fmt.Errorf("step failed: %s - %w", step.Name, err)
		}

		if result.TimedOut {
			errorMsg := fmt.Sprintf("Command timed out: %s", step.Name)
			su.logger.Error(errorMsg)
			fmt.Printf("  ✗ %s\n", errorMsg)
			if step.Optional {
				su.logger.Warn("Skipping optional step due to timeout")
				fmt.Println("  ⚠ Skipping optional step")
				continue
			}
			return fmt.Errorf("step timed out: %s", step.Name)
		}

		if result.ExitCode == 0 {
			su.logger.Info("Completed: %s", step.Name)
			fmt.Printf("  ✓ %s\n", step.Name)
		} else {
			isCommandNotFound := result.Stderr != "" &&
				(strings.Contains(strings.ToLower(result.Stderr), "command not found") ||
					strings.Contains(strings.ToLower(result.Stderr), "no such file or directory"))

			if step.Optional {
				if isCommandNotFound {
					skipMsg := fmt.Sprintf("Skipped (optional): %s - command not available on this system", step.Name)
					su.logger.Info(skipMsg)
					fmt.Printf("  ⚠ %s\n", skipMsg)
				} else {
					warnMsg := fmt.Sprintf("Skipped (optional): %s (exit code %d)", step.Name, result.ExitCode)
					su.logger.Warn(warnMsg)
					fmt.Printf("  ⚠ %s\n", warnMsg)
				}
			} else {
				warnMsg := fmt.Sprintf("Warning: %s exited with code %d", step.Name, result.ExitCode)
				su.logger.Warn(warnMsg)
				fmt.Printf("  ⚠ %s\n", warnMsg)
			}

			if result.Stderr != "" && !isCommandNotFound {
				if strings.Contains(strings.ToLower(result.Stderr), "password") ||
					strings.Contains(strings.ToLower(result.Stderr), "sudo: a password is required") {
					return fmt.Errorf("sudo password required for: %s. Configure passwordless sudo", step.Name)
				}
				errorPreview := result.Stderr
				if len(errorPreview) > 200 {
					errorPreview = errorPreview[:200]
				}
				fmt.Printf("  Error output: %s\n", errorPreview)
			}
		}
	}

	return nil
}

// executeOptimizationSteps runs post-update optimization
func (su *SystemUpdate) executeOptimizationSteps(ctx context.Context) error {
	su.logger.Info("Running post-update optimization...")
	fmt.Println("\n=== Running Post-Update Optimization ===")

	// Step 14: Run TRIM on SSD
	su.runTrimOperation(ctx, 14)

	// Step 15: Check I/O scheduler
	su.checkIOScheduler(ctx, 15)

	// Step 16: Check SMART health
	su.checkSmartHealth(ctx, 16)

	// Step 17: Verify power profile
	su.checkPowerProfile(ctx, 17)

	// Step 18: Check memory swappiness
	su.checkSwappiness(ctx, 18)

	// Step 19: Check disk space
	su.checkDiskSpace(ctx, 19)

	// Step 20: Rebuild DKMS modules if needed
	su.rebuildDKMSModules(ctx, 20)

	return nil
}

// runTrimOperation runs TRIM on SSD
func (su *SystemUpdate) runTrimOperation(ctx context.Context, stepNum int) {
	su.logger.Info("Step %d/20: Running TRIM on SSD", stepNum)
	fmt.Printf("  [%d/20] Running TRIM on SSD...\n", stepNum)

	passwordDetected := false
	result, err := su.shell.Execute(ctx, "sudo -n fstrim -v /", &utility.ExecOptions{
		Timeout: 30 * time.Second,
		StderrCallback: func(line string) {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "password") ||
				strings.Contains(lowerLine, "sudo: a password is required") {
				passwordDetected = true
			}
		},
	})

	if passwordDetected || (result != nil && result.Stderr != "" &&
		(strings.Contains(strings.ToLower(result.Stderr), "password") ||
			strings.Contains(strings.ToLower(result.Stderr), "sudo: a password is required"))) {
		warnMsg := "TRIM skipped: sudo password required (run manually: sudo fstrim -v /)"
		su.logger.Warn(warnMsg)
		fmt.Printf("    ⚠ %s\n", warnMsg)
		return
	}

	if err != nil {
		warnMsg := fmt.Sprintf("TRIM operation failed: %v", err)
		su.logger.Warn(warnMsg)
		fmt.Printf("    ⚠ %s\n", warnMsg)
		return
	}

	if result.ExitCode == 0 {
		msg := fmt.Sprintf("TRIM completed: %s", strings.TrimSpace(result.Stdout))
		su.logger.Info(msg)
		fmt.Printf("    ✓ %s\n", msg)
	} else if result.TimedOut {
		warnMsg := "TRIM operation timed out"
		su.logger.Warn(warnMsg)
		fmt.Printf("    ⚠ %s\n", warnMsg)
	} else {
		warnMsg := fmt.Sprintf("TRIM operation returned exit code %d", result.ExitCode)
		su.logger.Warn(warnMsg)
		fmt.Printf("    ⚠ %s\n", warnMsg)
	}
}

// checkIOScheduler checks I/O scheduler for NVMe
func (su *SystemUpdate) checkIOScheduler(ctx context.Context, stepNum int) {
	su.logger.Info("Step %d/20: Checking I/O scheduler", stepNum)
	fmt.Printf("  [%d/20] Checking I/O scheduler...\n", stepNum)

	result, err := su.shell.Execute(ctx, "cat /sys/block/nvme0n1/queue/scheduler 2>/dev/null", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		fmt.Println("    ⚠ Could not check I/O scheduler (NVMe device may not exist)")
		return
	}

	scheduler := strings.TrimSpace(result.Stdout)
	su.logger.Info("I/O Scheduler: %s", scheduler)

	if strings.Contains(scheduler, "[none]") || strings.Contains(scheduler, "[mq-deadline]") {
		msg := fmt.Sprintf("I/O scheduler is optimal: %s", scheduler)
		su.logger.Info(msg)
		fmt.Printf("    ✓ %s\n", msg)
	} else {
		msg := fmt.Sprintf("I/O scheduler: %s (consider 'none' or 'mq-deadline' for NVMe)", scheduler)
		su.logger.Warn(msg)
		fmt.Printf("    ⚠ %s\n", msg)
	}
}

// checkSmartHealth checks SMART health for all disks (simplified implementation)
func (su *SystemUpdate) checkSmartHealth(ctx context.Context, stepNum int) {
	su.logger.Info("Step %d/20: Checking SMART disk health", stepNum)
	fmt.Printf("  [%d/20] Checking SMART disk health...\n", stepNum)

	// Get list of disk devices
	result, err := su.shell.Execute(ctx, "lsblk -d -n -o NAME | grep -E '^[sv]d[a-z]|^nvme'", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		su.logger.Debug("Could not list disk devices")
		return
	}

	devices := strings.Fields(result.Stdout)
	if len(devices) == 0 {
		su.logger.Debug("No disk devices found")
		return
	}

	healthyCount := 0
	failedCount := 0
	var failedDisks []string

	for _, device := range devices {
		devicePath := "/dev/" + device
		smartResult, err := su.shell.Execute(ctx, fmt.Sprintf("sudo -n smartctl -H %s 2>/dev/null", devicePath), &utility.ExecOptions{
			Timeout: 10 * time.Second,
		})

		if err != nil || smartResult.ExitCode != 0 {
			// Skip if smartctl not available or device doesn't support SMART
			continue
		}

		output := strings.ToLower(smartResult.Stdout)
		if strings.Contains(output, "passed") || strings.Contains(output, "ok") {
			healthyCount++
			su.logger.Info("%s: SMART health PASSED", devicePath)
		} else {
			failedCount++
			failedDisks = append(failedDisks, devicePath)
			su.logger.Error("%s: SMART health FAILED", devicePath)
		}
	}

	if healthyCount > 0 && failedCount == 0 {
		fmt.Printf("    ✓ All %d disk(s) passed SMART health check\n", healthyCount)
	} else if failedCount > 0 {
		fmt.Printf("    ⚠ %d disk(s) failed SMART check: %s\n", failedCount, strings.Join(failedDisks, ", "))
		if healthyCount > 0 {
			fmt.Printf("    ✓ %d disk(s) passed\n", healthyCount)
		}
	}
}

// checkPowerProfile verifies power profile configuration
func (su *SystemUpdate) checkPowerProfile(ctx context.Context, stepNum int) {
	su.logger.Info("Step %d/20: Checking power profile", stepNum)
	fmt.Printf("  [%d/20] Checking power profile...\n", stepNum)

	result, err := su.shell.Execute(ctx, "powerprofilesctl get 2>/dev/null", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		msg := "power-profiles-daemon not available"
		su.logger.Debug(msg)
		fmt.Printf("    ⚠ %s\n", msg)
		return
	}

	profile := strings.TrimSpace(result.Stdout)
	msg := fmt.Sprintf("Current power profile: %s", profile)
	su.logger.Info(msg)
	fmt.Printf("    ✓ %s\n", msg)
}

// checkSwappiness checks memory swappiness configuration
func (su *SystemUpdate) checkSwappiness(ctx context.Context, stepNum int) {
	su.logger.Info("Step %d/20: Checking memory swappiness", stepNum)
	fmt.Printf("  [%d/20] Checking memory swappiness...\n", stepNum)

	result, err := su.shell.Execute(ctx, "cat /proc/sys/vm/swappiness", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		su.logger.Debug("Could not check swappiness")
		return
	}

	swappiness := strings.TrimSpace(result.Stdout)
	// Optimal swappiness is typically 10-60 for desktop systems
	msg := fmt.Sprintf("Swappiness: %s", swappiness)
	su.logger.Info(msg)
	fmt.Printf("    ✓ %s\n", msg)
}

// checkDiskSpace checks disk space for low space warnings
func (su *SystemUpdate) checkDiskSpace(ctx context.Context, stepNum int) {
	su.logger.Info("Step %d/20: Checking disk space", stepNum)
	fmt.Printf("  [%d/20] Checking disk space...\n", stepNum)

	result, err := su.shell.Execute(ctx, `df -h | awk 'NR>1 {print $5 " " $6}' | sed 's/%//'`, &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		su.logger.Debug("Could not check disk space")
		return
	}

	lines := strings.Split(result.Stdout, "\n")
	warnings := 0

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		var percent int
		fmt.Sscanf(fields[0], "%d", &percent)
		mountPoint := fields[1]

		if percent >= 90 {
			warnings++
			icon := "🔴"
			level := "CRITICAL"
			if percent < 95 {
				icon = "🟡"
				level = "WARNING"
			}
			msg := fmt.Sprintf("%s %s: %s is %d%% full", icon, level, mountPoint, percent)
			su.logger.Warn(msg)
			fmt.Printf("      %s\n", msg)
		}
	}

	if warnings == 0 {
		msg := "All disks have sufficient space"
		su.logger.Info(msg)
		fmt.Printf("    ✓ %s\n", msg)
	} else {
		msg := fmt.Sprintf("Found %d disk space warning(s)", warnings)
		su.logger.Warn(msg)
		fmt.Printf("    ⚠ %s\n", msg)
	}
}

// rebuildDKMSModules rebuilds DKMS modules after kernel update
func (su *SystemUpdate) rebuildDKMSModules(ctx context.Context, stepNum int) {
	su.logger.Info("Step %d/20: Checking DKMS modules", stepNum)
	fmt.Printf("  [%d/20] Checking DKMS modules...\n", stepNum)

	statusResult, err := su.shell.Execute(ctx, "dkms status", &utility.ExecOptions{
		Timeout: 10 * time.Second,
	})

	if err != nil || statusResult.ExitCode != 0 || strings.TrimSpace(statusResult.Stdout) == "" {
		msg := "No DKMS modules installed"
		su.logger.Debug(msg)
		fmt.Printf("    ✓ %s\n", msg)
		return
	}

	su.logger.Info("DKMS modules present, verifying installation")

	// Determine command prefix based on whether we're root
	dkmsCmd := "dkms autoinstall"
	if !su.isRoot() {
		dkmsCmd = "sudo -n " + dkmsCmd
	}

	passwordDetected := false
	result, err := su.shell.Execute(ctx, dkmsCmd, &utility.ExecOptions{
		Timeout: 2 * time.Minute,
		StdoutCallback: func(line string) {
			su.logger.Debug("  %s", line)
		},
		StderrCallback: func(line string) {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "password") ||
				strings.Contains(lowerLine, "sudo: a password is required") {
				passwordDetected = true
			}
		},
	})

	if passwordDetected || (result != nil && result.Stderr != "" &&
		(strings.Contains(strings.ToLower(result.Stderr), "password") ||
			strings.Contains(strings.ToLower(result.Stderr), "sudo: a password is required"))) {
		msg := "DKMS check skipped: sudo password required"
		su.logger.Warn(msg)
		fmt.Printf("    ⚠ %s\n", msg)
	} else if err == nil && result.ExitCode == 0 {
		msg := "DKMS modules verified/rebuilt successfully"
		su.logger.Info(msg)
		fmt.Printf("    ✓ %s\n", msg)
	} else {
		msg := fmt.Sprintf("DKMS autoinstall exited with code %d", result.ExitCode)
		su.logger.Warn(msg)
		fmt.Printf("    ⚠ %s\n", msg)
	}
}

// checkPacnewFiles checks for .pacnew configuration files
func (su *SystemUpdate) checkPacnewFiles(ctx context.Context) {
	result, err := su.shell.Execute(ctx, "find /etc -name '*.pacnew' 2>/dev/null", &utility.ExecOptions{
		Timeout: 10 * time.Second,
	})

	if err != nil {
		su.logger.Debug("Could not check for .pacnew files")
		return
	}

	files := strings.Fields(result.Stdout)
	if len(files) > 0 {
		su.logger.Warn("Found %d .pacnew file(s) that may need manual merging:", len(files))
		for _, file := range files {
			su.logger.Warn("  %s", file)
		}
		su.logger.Info("Consider using 'pacdiff' to merge configuration changes.")
	}
}

// checkRebootRequired checks if reboot is required after kernel update
func (su *SystemUpdate) checkRebootRequired(ctx context.Context) {
	// Check if current kernel matches running kernel
	result, err := su.shell.Execute(ctx, "[ -f /usr/lib/modules/$(uname -r)/modules.dep ]", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil {
		su.logger.Debug("Could not check reboot status")
		return
	}

	needsReboot := result.ExitCode != 0
	if needsReboot {
		su.logger.Warn("Kernel update detected - reboot recommended for changes to take effect")
	}
}

// postUpdateVerification runs post-update system verification
func (su *SystemUpdate) postUpdateVerification(ctx context.Context) {
	su.logger.Info("Running post-update verification...")

	// Check for any systemd service failures
	result, err := su.shell.Execute(ctx, "systemctl --failed --no-legend --no-pager", &utility.ExecOptions{
		Timeout: 10 * time.Second,
	})

	if err == nil && strings.TrimSpace(result.Stdout) != "" {
		lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		var failedServices []string
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				failedServices = append(failedServices, fields[0])
			}
		}
		if len(failedServices) > 0 {
			su.logger.Warn("Found %d failed service(s): %s", len(failedServices), strings.Join(failedServices, ", "))
		}
	} else {
		su.logger.Info("No failed system services detected")
	}

	su.logger.Info("System update verification complete")
}
