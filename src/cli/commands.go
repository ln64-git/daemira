package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	daemira "github.com/ln64-git/daemira/internal"
	desktopmonitor "github.com/ln64-git/daemira/src/features/desktop-monitor"
	"github.com/ln64-git/daemira/src/features/installer"
	systemhealth "github.com/ln64-git/daemira/src/features/system-health"
	systemupdate "github.com/ln64-git/daemira/src/features/system-update"
	"github.com/ln64-git/daemira/src/utility"
	"github.com/spf13/cobra"
)

// CLI holds references to daemon and logger for command handlers
type CLI struct {
	daemon *daemira.Daemira
	logger *utility.Logger
}

// NewCLI creates a new CLI instance
func NewCLI(daemon *daemira.Daemira, logger *utility.Logger) *CLI {
	return &CLI{
		daemon: daemon,
		logger: logger,
	}
}

// CreateCommands creates all CLI commands
func (c *CLI) CreateCommands() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "daemira",
		Short: "Daemira - Personal System Daemon",
		Long:  `Daemira is a comprehensive personal system daemon for Linux with Google Drive sync, system updates, health monitoring, and more.`,
		Run: func(cmd *cobra.Command, args []string) {
			c.logger.Info("Daemira v%s", "0.1.0")
			c.logger.Info("Starting daemon services...")

			if err := c.daemon.Start(); err != nil {
				c.logger.Error("Failed to start daemon: %v", err)
				os.Exit(1)
			}

			c.logger.Info("Daemon is running. Press Ctrl+C to stop.")
			c.logger.Info("")
			c.logger.Info("To check status, run in another terminal: ./bin/daemira status")
			c.logger.Info("Or: ./bin/daemira gdrive status")
			c.logger.Info("")

			// Keep process alive
			select {}
		},
	}

	// Add subcommands
	rootCmd.AddCommand(c.createStatusCmd())
	rootCmd.AddCommand(c.createDaemonCmd())
	rootCmd.AddCommand(c.createInstallCmd())
	rootCmd.AddCommand(c.createGDriveCmd())
	rootCmd.AddCommand(c.createSystemCmd())
	rootCmd.AddCommand(c.createStorageCmd())
	rootCmd.AddCommand(c.createPerformanceCmd())
	rootCmd.AddCommand(c.createMemoryCmd())
	rootCmd.AddCommand(c.createDesktopCmd())

	return rootCmd
}

func (c *CLI) createStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show comprehensive system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			status, err := c.getSystemStatus(ctx)
			if err != nil {
				return err
			}
			fmt.Println(status)
			return nil
		},
	}
}

func (c *CLI) createDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Daemon management commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			c.logger.Info("Starting Daemira daemon...")
			if err := c.daemon.Start(); err != nil {
				return err
			}
			c.logger.Info("Daemon mode: Running in foreground")
			c.logger.Info("Press Ctrl+C to stop")
			select {}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			gd := c.daemon.GetGoogleDrive()
			if gd != nil {
				if err := gd.Stop(); err != nil {
					return err
				}
			}
			c.logger.Info("Daemon stopped")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		Run: func(cmd *cobra.Command, args []string) {
			status := c.getGoogleDriveSyncStatus()
			fmt.Println(status)
		},
	})

	return cmd
}

func (c *CLI) createInstallCmd() *cobra.Command {
	var noTUI bool
	var stepID string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Run system installer",
		Long: `Run the Daemira system installer.

This will install:
  - DKMS (DankLinux)
  - Hyprland config
  - DMS config
  - Core packages
  - User applications
  - System services`,
		RunE: func(cmd *cobra.Command, args []string) error {
			useTUI := !noTUI
			inst, err := installer.NewInstaller(c.logger, useTUI)
			if err != nil {
				c.logger.Error("Failed to create installer: %v", err)
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
			defer cancel()

			if stepID != "" {
				c.logger.Info("Running specific step: %s", stepID)
				return inst.RunStep(ctx, stepID)
			}

			return inst.Run(ctx)
		},
	}

	cmd.Flags().BoolVar(&noTUI, "no-tui", false, "Run installer in headless mode (no TUI)")
	cmd.Flags().StringVar(&stepID, "step", "", "Run a specific installation step by ID")

	return cmd
}

func (c *CLI) createGDriveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gdrive",
		Short: "Google Drive sync commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start Google Drive sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.daemon.SyncGoogleDrive(); err != nil {
				return err
			}
			fmt.Println("Google Drive sync started")
			fmt.Println("\nPress Ctrl+C to stop")
			select {}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop Google Drive sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			gd := c.daemon.GetGoogleDrive()
			if gd == nil {
				return fmt.Errorf("Google Drive sync is not initialized")
			}
			if err := gd.Stop(); err != nil {
				return err
			}
			fmt.Println("Google Drive sync stopped")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show Google Drive sync status",
		Run: func(cmd *cobra.Command, args []string) {
			status := c.getGoogleDriveSyncStatus()
			fmt.Println(status)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Force sync all directories immediately",
		RunE: func(cmd *cobra.Command, args []string) error {
			gd := c.daemon.GetGoogleDrive()
			if gd == nil {
				return fmt.Errorf("Google Drive sync is not running. Start it first with: daemira gdrive start")
			}
			result := gd.SyncAll()
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "sync-dir",
		Short: "Force sync a specific directory immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gd := c.daemon.GetGoogleDrive()
			if gd == nil {
				return fmt.Errorf("Google Drive sync is not running. Start it first with: daemira gdrive start")
			}
			result := gd.SyncDirectory(args[0])
			fmt.Println(result)
			fmt.Println("\nThe sync will begin shortly. Check status with: daemira gdrive status")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "resync-dir",
		Short: "Force resync a specific directory (rebuilds cache and syncs deletions)",
		Long:  "Use this when files were deleted locally and need to be deleted from Google Drive. This rebuilds the bisync cache and ensures deletions are synced.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gd := c.daemon.GetGoogleDrive()
			if gd == nil {
				return fmt.Errorf("Google Drive sync is not running. Start it first with: daemira gdrive start")
			}
			ctx := context.Background()
			if err := gd.ResyncDirectory(ctx, args[0]); err != nil {
				return fmt.Errorf("resync failed: %w", err)
			}
			result := fmt.Sprintf("Resync completed for %s. Cache rebuilt and deletions synced.", args[0])
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "patterns",
		Short: "List exclude patterns",
		Run: func(cmd *cobra.Command, args []string) {
			gd := c.daemon.GetGoogleDrive()
			if gd == nil {
				fmt.Println("Google Drive sync is not initialized.")
				return
			}
			patterns := gd.GetExcludePatterns()
			output := fmt.Sprintf("Google Drive Exclude Patterns (%d total):\n\n", len(patterns))
			output += "These files/folders will NOT be synced:\n"
			for i, pattern := range patterns {
				output += fmt.Sprintf("  %d. %s\n", i+1, pattern)
			}
			fmt.Println(output)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "exclude",
		Short: "Add exclude pattern",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			gd := c.daemon.GetGoogleDrive()
			if gd == nil {
				fmt.Println("Google Drive sync is not initialized.")
				return
			}
			gd.AddExcludePattern(args[0])
			fmt.Printf("Added exclude pattern: %s\n", args[0])
		},
	})

	return cmd
}

func (c *CLI) createSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System update commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Run system update immediately",
		RunE: func(cmd *cobra.Command, args []string) error {
			su := c.daemon.GetSystemUpdate()
			if su == nil {
				if err := c.daemon.KeepSystemUpdated(); err != nil {
					return err
				}
				su = c.daemon.GetSystemUpdate()
			}
			ctx := context.Background()
			if err := su.RunUpdate(ctx); err != nil {
				return err
			}
			fmt.Println("System update completed. Check logs for details.")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show system update status",
		Run: func(cmd *cobra.Command, args []string) {
			status := c.getSystemUpdateStatus()
			fmt.Println(status)
		},
	})

	return cmd
}

func (c *CLI) createStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Storage monitoring commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show disk usage summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			dm := systemhealth.GetDiskMonitor()
			status, err := dm.GetDiskSummary(ctx)
			if err != nil {
				return err
			}
			fmt.Println(status)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Check for low disk space warnings",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			dm := systemhealth.GetDiskMonitor()
			warnings, err := dm.CheckLowSpace(ctx)
			if err != nil {
				return err
			}
			if len(warnings) == 0 {
				fmt.Println("All disks have sufficient space.")
				return nil
			}
			output := "⚠️  DISK SPACE WARNINGS:\n\n"
			for _, warning := range warnings {
				icon := "🟡"
				if warning.Level == "critical" {
					icon = "🔴"
				}
				output += fmt.Sprintf("%s %s\n", icon, warning.Message)
			}
			fmt.Println(output)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "health",
		Short: "Show disk health (SMART) status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			dm := systemhealth.GetDiskMonitor()
			statuses, err := dm.GetAllSmartStatus(ctx)
			if err != nil {
				return err
			}
			if len(statuses) == 0 {
				fmt.Println("No SMART status available. Install smartmontools or run with sudo.")
				return nil
			}
			output := "=== Disk Health (SMART) ===\n\n"
			for _, status := range statuses {
				healthIcon := "✓"
				if !status.Passed {
					healthIcon = "✗"
				}
				output += fmt.Sprintf("%s %s: %s\n", healthIcon, status.Device, boolToPassedFailed(status.Passed))
				if status.Temperature != nil {
					output += fmt.Sprintf("  Temperature: %d°C\n", *status.Temperature)
				}
				if status.PowerOnHours != nil {
					output += fmt.Sprintf("  Power On Hours: %d\n", *status.PowerOnHours)
				}
				if len(status.Errors) > 0 {
					output += fmt.Sprintf("  Errors: %s\n", strings.Join(status.Errors, ", "))
				}
				output += "\n"
			}
			fmt.Println(output)
			return nil
		},
	})

	return cmd
}

func (c *CLI) createPerformanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "performance",
		Short: "Performance management commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get current power profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pm := systemhealth.GetPerformanceManager()
			profile, err := pm.GetCurrentProfile(ctx)
			if err != nil {
				fmt.Println("Power profiles not available (power-profiles-daemon not running)")
				return nil
			}
			fmt.Printf("Current power profile: %s\n", profile)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all available power profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pm := systemhealth.GetPerformanceManager()
			profiles, err := pm.GetAllProfiles(ctx)
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				fmt.Println("No power profiles available (power-profiles-daemon not running)")
				return nil
			}
			output := "=== Available Power Profiles ===\n\n"
			for _, profile := range profiles {
				activeIcon := "○"
				if profile.Active {
					activeIcon = "●"
				}
				output += fmt.Sprintf("%s %s\n", activeIcon, profile.Name)
				if profile.CPUDriver != "" {
					output += fmt.Sprintf("  CPU Driver: %s\n", profile.CPUDriver)
				}
				if profile.PlatformDriver != "" {
					output += fmt.Sprintf("  Platform Driver: %s\n", profile.PlatformDriver)
				}
				if profile.Degraded {
					output += "  Status: Degraded\n"
				}
				output += "\n"
			}
			fmt.Println(output)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "suggest",
		Short: "Suggest optimal power profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pm := systemhealth.GetPerformanceManager()
			suggested, err := pm.SuggestProfile(ctx)
			if err != nil {
				return err
			}
			current, _ := pm.GetCurrentProfile(ctx)
			output := fmt.Sprintf("Suggested power profile: %s\n", suggested)
			if current != "" {
				output += fmt.Sprintf("Current power profile: %s\n", current)
				if current != suggested {
					output += fmt.Sprintf("\nRecommendation: Switch to %s for better performance/efficiency", suggested)
				} else {
					output += "\n✓ Current profile is optimal"
				}
			}
			fmt.Println(output)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set",
		Short: "Set power profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pm := systemhealth.GetPerformanceManager()
			profile := systemhealth.PowerProfile(args[0])
			if err := pm.SetProfile(ctx, profile); err != nil {
				return fmt.Errorf("failed to set power profile to %s: %w", profile, err)
			}
			fmt.Printf("Power profile set to: %s\n", profile)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "cpu",
		Short: "Show CPU statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pm := systemhealth.GetPerformanceManager()
			stats, err := pm.GetCPUStats(ctx)
			if err != nil {
				return err
			}
			fmt.Println(pm.FormatCPUStats(stats))
			return nil
		},
	})

	return cmd
}

func (c *CLI) createMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Memory monitoring commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "stats",
		Short: "Show memory statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			mm := systemhealth.GetMemoryMonitor()
			stats, err := mm.GetMemoryStats(ctx)
			if err != nil {
				return err
			}
			fmt.Println(mm.FormatMemoryStats(stats))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "swappiness",
		Short: "Check swappiness configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			mm := systemhealth.GetMemoryMonitor()
			check, err := mm.CheckSwappiness(ctx)
			if err != nil {
				return err
			}
			fmt.Println(check["message"].(string))
			return nil
		},
	})

	return cmd
}

func (c *CLI) createDesktopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "desktop",
		Short: "Desktop environment commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show desktop environment status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			di := desktopmonitor.GetDesktopIntegration()
			result, err := di.GetFormattedStatus(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "session",
		Short: "Show session information",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			di := desktopmonitor.GetDesktopIntegration()
			result, err := di.GetSessionStatus(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "compositor",
		Short: "Show compositor information",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			di := desktopmonitor.GetDesktopIntegration()
			result, err := di.GetCompositorStatus(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "displays",
		Short: "Show display information",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			di := desktopmonitor.GetDesktopIntegration()
			result, err := di.GetDisplayStatus(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "lock",
		Short: "Lock the session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			di := desktopmonitor.GetDesktopIntegration()
			result, err := di.LockSession(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "unlock",
		Short: "Unlock the session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			di := desktopmonitor.GetDesktopIntegration()
			result, err := di.UnlockSession(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	return cmd
}

// Status formatting methods

func (c *CLI) getGoogleDriveSyncStatus() string {
	gd := c.daemon.GetGoogleDrive()
	if gd == nil {
		return "Google Drive sync is not initialized yet (may be starting in background)."
	}

	status := gd.GetStatus()
	output := "Google Drive Sync Status:\n"

	running := false
	if r, ok := status["running"].(bool); ok {
		running = r
	}
	output += fmt.Sprintf("  Running: %s\n", boolToYesNo(running))

	syncMode := "periodic"
	if m, ok := status["syncMode"].(string); ok {
		syncMode = m
	}
	syncInterval := 30
	if i, ok := status["syncInterval"].(int); ok {
		syncInterval = i
	}
	output += fmt.Sprintf("  Mode: %s (every %ds)\n", syncMode, syncInterval)

	directories := 0
	if dirs, ok := status["directories"].(int); ok {
		directories = dirs
	}
	output += fmt.Sprintf("  Directories: %d\n", directories)

	queueSize := 0
	if q, ok := status["queueSize"].(int); ok {
		queueSize = q
	}
	output += fmt.Sprintf("  Queue Size: %d\n\n", queueSize)

	if syncStates, ok := status["syncStates"].(map[string]interface{}); ok && len(syncStates) > 0 {
		output += "  Directory States:\n"
		// Note: syncStates is a complex nested structure, simplified display
		for path, stateData := range syncStates {
			if state, ok := stateData.(map[string]interface{}); ok {
				stateIcon := "✓"
				stateStatus := "idle"
				if s, ok := state["status"].(string); ok {
					stateStatus = s
					if s == "syncing" {
						stateIcon = "↻"
					} else if s == "error" {
						stateIcon = "✗"
					}
				}
				output += fmt.Sprintf("    %s %s\n", stateIcon, path)
				output += fmt.Sprintf("       Status: %s\n", stateStatus)

				if lastSync, ok := state["lastSyncTime"].(time.Time); ok && !lastSync.IsZero() {
					output += fmt.Sprintf("       Last sync: %s\n", formatTime(lastSync))
				} else {
					output += "       Last sync: Never\n"
				}

				if errMsg, ok := state["errorMessage"].(string); ok && errMsg != "" {
					output += fmt.Sprintf("       Error: %s\n", errMsg)
				}
			}
		}
	}

	return output
}

func (c *CLI) getSystemUpdateStatus() string {
	su := c.daemon.GetSystemUpdate()
	if su == nil {
		return "System update scheduler is not initialized."
	}

	status := su.GetStatus()
	output := "System Update Status:\n"
	output += fmt.Sprintf("  Running: %s\n", boolToYesNo(status["running"].(bool)))

	if lastUpdate, ok := status["lastUpdate"].(int64); ok && lastUpdate > 0 {
		output += fmt.Sprintf("  Last Update: %s\n", formatTime(time.Unix(lastUpdate, 0)))
	}

	if nextUpdate, ok := status["nextUpdate"].(int64); ok && nextUpdate > 0 {
		output += fmt.Sprintf("  Next Update: %s\n", formatTime(time.Unix(nextUpdate, 0)))
	}

	if history, ok := status["history"].([]systemupdate.UpdateHistoryEntry); ok && len(history) > 0 {
		output += "\n  Recent Updates:\n"
		start := len(history) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(history); i++ {
			entry := history[i]
			success := "✓"
			if !entry.Success {
				success = "✗"
			}
			output += fmt.Sprintf("    %s %s (%.1fs)\n", success, formatTime(entry.Timestamp), entry.Duration.Seconds())
		}
	}

	return output
}

func (c *CLI) getSystemStatus(ctx context.Context) (string, error) {
	output := "=== Daemira System Status ===\n\n"

	// CPU & Performance
	pm := systemhealth.GetPerformanceManager()
	if stats, err := pm.GetCPUStats(ctx); err == nil {
		output += fmt.Sprintf("CPU: %dC/%dT @ %.0fMHz", stats.Cores, stats.Threads, stats.AverageFrequencyMHz)
		if stats.PowerProfile != "" {
			output += fmt.Sprintf(" (%s)", stats.PowerProfile)
		}
		if stats.Utilization > 0 {
			output += fmt.Sprintf(" - %.1f%% utilized", stats.Utilization)
		}
		output += "\n"
	} else {
		output += "CPU: Unable to read stats\n"
	}

	// Memory
	mm := systemhealth.GetMemoryMonitor()
	if memStats, err := mm.GetMemoryStats(ctx); err == nil {
		output += fmt.Sprintf("Memory: %.1fGB / %.1fGB (%.1f%%)", memStats.UsedGB, memStats.TotalGB, memStats.PercentUsed)
		if memStats.Swap.UsedGB > 0 {
			output += fmt.Sprintf(" + %.1fGB swap", memStats.Swap.UsedGB)
		}
		output += "\n"
	} else {
		output += "Memory: Unable to read stats\n"
	}

	// Disk space warnings
	dm := systemhealth.GetDiskMonitor()
	if warnings, err := dm.CheckLowSpace(ctx); err == nil {
		if len(warnings) > 0 {
			output += fmt.Sprintf("\n⚠️  Disk Warnings: %d\n", len(warnings))
			for _, warning := range warnings {
				icon := "🟡"
				if warning.Level == "critical" {
					icon = "🔴"
				}
				output += fmt.Sprintf("  %s %s: %.1fGB free\n", icon, warning.MountPoint, warning.FreeGB)
			}
		} else {
			output += "Disk Space: All healthy\n"
		}
	} else {
		output += "Disk Space: Unable to check\n"
	}

	// Google Drive status
	output += "\n"
	gd := c.daemon.GetGoogleDrive()
	if gd != nil {
		gdStatus := gd.GetStatus()
		running := false
		if r, ok := gdStatus["running"].(bool); ok {
			running = r
		}
		queueSize := 0
		if q, ok := gdStatus["queueSize"].(int); ok {
			queueSize = q
		}
		output += fmt.Sprintf("Google Drive: %s (%d queued)\n", boolToRunningStopped(running), queueSize)
	} else {
		output += "Google Drive: Not initialized\n"
	}

	// System Update status
	su := c.daemon.GetSystemUpdate()
	if su != nil {
		suStatus := su.GetStatus()
		if lastUpdate, ok := suStatus["lastUpdate"].(int64); ok && lastUpdate > 0 {
			hoursSince := time.Since(time.Unix(lastUpdate, 0)).Hours()
			output += fmt.Sprintf("System Update: Last %.1fh ago\n", hoursSince)
		} else {
			output += "System Update: Never run\n"
		}
	} else {
		output += "System Update: Not initialized\n"
	}

	// Desktop Environment
	di := desktopmonitor.GetDesktopIntegration()
	if desktopSummary, err := di.GetDesktopSummary(ctx); err == nil {
		output += fmt.Sprintf("\nDesktop Environment:\n  %s\n", desktopSummary)
	} else {
		output += "\nDesktop Environment: Unable to query\n"
	}

	return output, nil
}

