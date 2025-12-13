/**
 * Daemira - Personal System Utility Daemon
 *
 * Core orchestrator that launches internal features:
 * - Google Drive bidirectional sync
 * - Automated system updates
 */

package daemira

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ln64-git/daemira/src/config"
	systemupdate "github.com/ln64-git/daemira/src/features/system-update"
	"github.com/ln64-git/daemira/src/utility"
)

// Daemira is the main orchestrator for all daemon services
type Daemira struct {
	logger                 *utility.Logger
	config                 *config.Config
	googleDrive            *utility.GoogleDrive
	googleDriveAutoStarted bool
	systemUpdate           *systemupdate.SystemUpdate
	mu                     sync.RWMutex
}

// NewDaemira creates a new Daemira instance
func NewDaemira(logger *utility.Logger, cfg *config.Config) *Daemira {
	if logger == nil {
		logger = utility.GetLogger()
	}

	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			logger.Warn("Failed to load config: %v, using defaults", err)
			cfg = &config.Config{
				RcloneRemoteName: "gdrive",
			}
		}
	}

	d := &Daemira{
		logger: logger,
		config: cfg,
	}

	logger.Info("Daemira initializing...")

	return d
}

// Start is the default function that chains KeepSystemUpdated and SyncGoogleDrive together
func (d *Daemira) Start() error {
	d.logger.Info("Starting Daemira services...")

	// Start system updates
	if err := d.KeepSystemUpdated(); err != nil {
		return fmt.Errorf("failed to start system updates: %w", err)
	}

	// Start Google Drive sync
	if err := d.SyncGoogleDrive(); err != nil {
		return fmt.Errorf("failed to start Google Drive sync: %w", err)
	}

	d.logger.Info("Daemira services started successfully")
	return nil
}

// KeepSystemUpdated starts the system update scheduler
func (d *Daemira) KeepSystemUpdated() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.systemUpdate == nil {
		d.systemUpdate = systemupdate.NewSystemUpdate(d.logger, &systemupdate.SystemUpdateOptions{
			Interval:  6 * time.Hour,
			AutoStart: true,
		})
		d.logger.Info("System update scheduler started (interval: 6 hours)")
	} else {
		d.logger.Info("System update scheduler already running")
	}

	return nil
}

// SyncGoogleDrive starts Google Drive sync service
func (d *Daemira) SyncGoogleDrive() error {
	// Skip if running as root - rclone config is user-specific
	if os.Geteuid() == 0 {
		d.logger.Info("Skipping Google Drive sync (running as root - rclone config is user-specific)")
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already running
	if d.googleDrive != nil {
		status := d.googleDrive.GetStatus()
		if running, ok := status["running"].(bool); ok && running {
			d.logger.Info("Google Drive sync is already running")
			return nil
		}
	}

	remoteName := d.config.RcloneRemoteName
	if remoteName == "" {
		remoteName = "gdrive"
	}
	gd := utility.NewGoogleDrive(d.logger, remoteName)

	ctx := context.Background()
	if err := gd.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Google Drive sync: %w", err)
	}

	d.googleDrive = gd
	d.googleDriveAutoStarted = true
	d.logger.Info("Google Drive sync started successfully")
	return nil
}

// GetGoogleDrive returns the GoogleDrive instance (for CLI access)
func (d *Daemira) GetGoogleDrive() *utility.GoogleDrive {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.googleDrive
}

// GetSystemUpdate returns the SystemUpdate instance (for CLI access)
func (d *Daemira) GetSystemUpdate() *systemupdate.SystemUpdate {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.systemUpdate
}
