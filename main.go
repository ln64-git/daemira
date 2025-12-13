package main

import (
	"context"
	"fmt"
	"os"
	"time"

	daemira "github.com/ln64-git/daemira/internal"
	"github.com/ln64-git/daemira/src/config"
	"github.com/ln64-git/daemira/src/features/installer"
	systemhealth "github.com/ln64-git/daemira/src/features/system-health"
	"github.com/ln64-git/daemira/src/utility"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	logger  *utility.Logger
	daemon  *daemira.Daemira
)

func main() {
	// Check if running as root
	if os.Geteuid() == 0 {
		logger = utility.NewLogger("cli", utility.INFO)
		logger.Info("Running with root privileges")
	} else {
		logger = utility.NewLogger("cli", utility.INFO)
		logger.Info("Running as user (system updates will require sudo)")
	}
	defer logger.Close()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.Warn("Failed to load config: %v, using defaults", err)
		cfg = &config.Config{
			RcloneRemoteName: "gdrive",
		}
	}

	// Initialize daemon
	daemon = daemira.NewDaemira(logger, cfg)

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "daemira",
		Short: "Daemira - Personal System Daemon",
		Long:  `Daemira is a comprehensive personal system daemon for Linux with Google Drive sync, system updates, health monitoring, and more.`,
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("Daemira v%s", version)
			logger.Info("Starting daemon services...")

			// Wait for autoStartServices to initialize (runs in background)
			time.Sleep(2 * time.Second)

			// Note: System update is already started by autoStartServices
			// No need to run it again here to avoid duplicates

			// 2. Google Drive sync is started by autoStartServices in background
			// It will automatically queue all directories for sync
			// No need to manually trigger - the background workers handle it
			logger.Info("Google Drive sync will start automatically via autoStartServices")
			logger.Info("Initial syncs will begin in background...")

			// 3. Schedule updates every 6 hours (already set up in autoStartServices)
			logger.Info("System update scheduler: Running every 6 hours")
			logger.Info("Daemon is running. Press Ctrl+C to stop.")
			logger.Info("")
			logger.Info("To check status, run in another terminal: ./bin/daemira status")
			logger.Info("Or: ./bin/daemira gdrive status")
			logger.Info("")

			// Periodic status updates every 30 seconds
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			// Initial status check after 5 seconds
			go func() {
				time.Sleep(5 * time.Second)
				logger.Info("=== Initial Status Check ===")
				logger.Info("Getting Google Drive status...")
				status := daemon.GetGoogleDriveSyncStatus()
				logger.Info("Status length: %d", len(status))
				if status == "" {
					logger.Warn("Google Drive status is empty - may not be initialized yet")
				} else {
					logger.Info("Google Drive Status:")
					fmt.Println(status)
				}

				// Also check full system status
				logger.Info("Getting full system status...")
				ctx := context.Background()
				fullStatus, err := daemon.GetSystemStatus(ctx)
				if err != nil {
					logger.Error("Failed to get system status: %v", err)
				} else {
					logger.Info("Full System Status:")
					fmt.Println(fullStatus)
				}
				logger.Info("=== End Status Check ===")
			}()

			// Keep process alive and show periodic status
			for range ticker.C {
				logger.Info("=== Status Update ===")
				status := daemon.GetGoogleDriveSyncStatus()
				if status != "" {
					fmt.Println(status)
				} else {
					logger.Warn("Status is empty")
				}
				logger.Info("")
			}
		},
	}

	// Add version flag
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	// Add subcommands
	rootCmd.AddCommand(createStatusCmd())
	rootCmd.AddCommand(createDaemonCmd())
	rootCmd.AddCommand(createInstallCmd())
	rootCmd.AddCommand(createGDriveCmd())
	rootCmd.AddCommand(createSystemCmd())
	rootCmd.AddCommand(createStorageCmd())
	rootCmd.AddCommand(createPerformanceCmd())
	rootCmd.AddCommand(createMemoryCmd())
	rootCmd.AddCommand(createDesktopCmd())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Error: %v", err)
		os.Exit(1)
	}
}

func createStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show comprehensive system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			status, err := daemon.GetSystemStatus(ctx)
			if err != nil {
				return err
			}
			fmt.Println(status)
			return nil
		},
	}
}

func createDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Daemon management commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Starting Daemira daemon...")
			logger.Info("Daemon mode: Running in foreground")
			logger.Info("Press Ctrl+C to stop")
			// Keep process alive
			select {}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			_, err := daemon.StopGoogleDriveSync(ctx)
			if err != nil {
				return err
			}
			logger.Info("Daemon stopped")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		Run: func(cmd *cobra.Command, args []string) {
			status := daemon.GetGoogleDriveSyncStatus()
			fmt.Println(status)
		},
	})

	return cmd
}

func createInstallCmd() *cobra.Command {
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
			inst, err := installer.NewInstaller(logger, useTUI)
			if err != nil {
				logger.Error("Failed to create installer: %v", err)
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
			defer cancel()

			if stepID != "" {
				logger.Info("Running specific step: %s", stepID)
				return inst.RunStep(ctx, stepID)
			}

			return inst.Run(ctx)
		},
	}

	cmd.Flags().BoolVar(&noTUI, "no-tui", false, "Run installer in headless mode (no TUI)")
	cmd.Flags().StringVar(&stepID, "step", "", "Run a specific installation step by ID")

	return cmd
}

func createGDriveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gdrive",
		Short: "Google Drive sync commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start Google Drive sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.StartGoogleDriveSync(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			fmt.Println("\nPress Ctrl+C to stop")
			select {}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop Google Drive sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.StopGoogleDriveSync(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show Google Drive sync status",
		Run: func(cmd *cobra.Command, args []string) {
			status := daemon.GetGoogleDriveSyncStatus()
			fmt.Println(status)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Force sync all directories immediately",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.SyncAllGoogleDrive(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "sync-dir",
		Short: "Force sync a specific directory immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			directoryPath := args[0]
			result, err := daemon.SyncDirectoryGoogleDrive(ctx, directoryPath)
			if err != nil {
				return err
			}
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
			ctx := context.Background()
			directoryPath := args[0]
			result, err := daemon.ResyncDirectoryGoogleDrive(ctx, directoryPath)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "patterns",
		Short: "List exclude patterns",
		Run: func(cmd *cobra.Command, args []string) {
			patterns := daemon.GetGoogleDriveExcludePatterns()
			fmt.Println(patterns)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "exclude",
		Short: "Add exclude pattern",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result := daemon.AddGoogleDriveExcludePattern(args[0])
			fmt.Println(result)
		},
	})

	return cmd
}

func createSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System update commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Run system update immediately",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.RunSystemUpdate(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show system update status",
		Run: func(cmd *cobra.Command, args []string) {
			status := daemon.GetSystemUpdateStatus()
			fmt.Println(status)
		},
	})

	return cmd
}

func createStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Storage monitoring commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show disk usage summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			status, err := daemon.GetDiskStatus(ctx)
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
			result, err := daemon.CheckDiskSpace(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "health",
		Short: "Show disk health (SMART) status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.GetDiskHealth(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	return cmd
}

func createPerformanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "performance",
		Short: "Performance management commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get current power profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.GetPowerProfile(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all available power profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.ListPowerProfiles(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "suggest",
		Short: "Suggest optimal power profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.SuggestPowerProfile(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set",
		Short: "Set power profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			profile := systemhealth.PowerProfile(args[0])
			result, err := daemon.SetPowerProfile(ctx, profile)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "cpu",
		Short: "Show CPU statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.GetCPUStats(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	return cmd
}

func createMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Memory monitoring commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "stats",
		Short: "Show memory statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.GetMemoryStats(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "swappiness",
		Short: "Check swappiness configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.CheckSwappiness(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	return cmd
}

func createDesktopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "desktop",
		Short: "Desktop environment commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show desktop environment status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := daemon.GetDesktopStatus(ctx)
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
			result, err := daemon.GetSessionInfo(ctx)
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
			result, err := daemon.GetCompositorInfo(ctx)
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
			result, err := daemon.GetDisplayInfo(ctx)
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
			result, err := daemon.LockSession(ctx)
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
			result, err := daemon.UnlockSession(ctx)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	return cmd
}
