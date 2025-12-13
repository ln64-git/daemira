/**
 * Daemira - Personal System Utility Daemon
 *
 * Main daemon class that orchestrates:
 * - Google Drive bidirectional sync
 * - Automated system updates
 * - System health monitoring
 * - Desktop environment integration
 */

package daemira

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ln64-git/daemira/src/config"
	desktopmonitor "github.com/ln64-git/daemira/src/features/desktop-monitor"
	systemhealth "github.com/ln64-git/daemira/src/features/system-health"
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
	diskMonitor            *systemhealth.DiskMonitor
	performanceManager     *systemhealth.PerformanceManager
	memoryMonitor          *systemhealth.MemoryMonitor
	desktopIntegration     *desktopmonitor.DesktopIntegration
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
		logger:             logger,
		config:             cfg,
		diskMonitor:        systemhealth.GetDiskMonitor(),
		performanceManager: systemhealth.GetPerformanceManager(),
		memoryMonitor:      systemhealth.GetMemoryMonitor(),
		desktopIntegration: desktopmonitor.GetDesktopIntegration(),
	}

	logger.Info("Daemira initializing...")

	// Auto-start services in background (non-blocking)
	go d.autoStartServices()

	return d
}

// autoStartServices auto-starts services in background
func (d *Daemira) autoStartServices() {
	time.Sleep(1 * time.Second)
	d.logger.Info("autoStartServices: Starting services...")

	// Auto-start Google Drive sync (skip if running as root - rclone config is user-specific)
	if os.Geteuid() == 0 {
		d.logger.Info("autoStartServices: Skipping Google Drive sync (running as root - rclone config is user-specific)")
	} else {
		d.mu.Lock()
		if !d.googleDriveAutoStarted {
			d.mu.Unlock()
			d.logger.Info("autoStartServices: Starting Google Drive sync...")
			if _, err := d.StartGoogleDriveSync(context.Background()); err != nil {
				d.logger.Error("Failed to auto-start Google Drive sync: %v", err)
			} else {
				d.mu.Lock()
				d.googleDriveAutoStarted = true
				d.mu.Unlock()
				d.logger.Info("autoStartServices: Google Drive sync started successfully")
			}
		} else {
			d.mu.Unlock()
			d.logger.Info("autoStartServices: Google Drive sync already started")
		}
	}

	// Auto-start system update scheduler
	d.mu.Lock()
	if d.systemUpdate == nil {
		d.systemUpdate = systemupdate.NewSystemUpdate(d.logger, &systemupdate.SystemUpdateOptions{
			Interval:  6 * time.Hour,
			AutoStart: true,
		})
	}
	d.mu.Unlock()
}

// ==================== Google Drive Methods ====================

// StartGoogleDriveSync starts Google Drive sync service
func (d *Daemira) StartGoogleDriveSync(ctx context.Context) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already running
	if d.googleDrive != nil {
		status := d.googleDrive.GetStatus()
		if running, ok := status["running"].(bool); ok && running {
			return "Google Drive sync is already running.", nil
		}
	}

	remoteName := d.config.RcloneRemoteName
	if remoteName == "" {
		remoteName = "gdrive"
	}
	gd := utility.NewGoogleDrive(d.logger, remoteName)

	if err := gd.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start Google Drive sync: %w", err)
	}

	d.googleDrive = gd
	msg := "Google Drive sync started successfully"
	d.logger.Info(msg)
	fmt.Println(msg)
	return msg, nil
}

// StopGoogleDriveSync stops Google Drive sync service
func (d *Daemira) StopGoogleDriveSync(ctx context.Context) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.googleDrive == nil {
		return "Google Drive sync is not initialized.", nil
	}

	d.googleDrive.Stop()

	fmt.Println("Google Drive sync stopped")
	return "Google Drive sync stopped", nil
}

// GetGoogleDriveSyncStatus gets Google Drive sync status
func (d *Daemira) GetGoogleDriveSyncStatus() string {
	d.mu.RLock()
	gd := d.googleDrive
	d.mu.RUnlock()

	if gd == nil {
		return "Google Drive sync is not initialized yet (may be starting in background)."
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	status := d.googleDrive.GetStatus()

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
	if dirs, ok := status["directories"].([]string); ok {
		directories = len(dirs)
	}
	output += fmt.Sprintf("  Directories: %d\n", directories)

	queueSize := 0
	if q, ok := status["queueSize"].(int); ok {
		queueSize = q
	}
	output += fmt.Sprintf("  Queue Size: %d\n\n", queueSize)

	if syncStates, ok := status["syncStates"].(map[string]interface{}); ok && len(syncStates) > 0 {
		output += "  Directory States:\n"
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
					output += fmt.Sprintf("       Last sync: %s\n", lastSync.Format(time.RFC1123))
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

// SyncAllGoogleDrive forces sync all directories immediately
func (d *Daemira) SyncAllGoogleDrive(ctx context.Context) (string, error) {
	d.mu.RLock()
	gd := d.googleDrive
	d.mu.RUnlock()

	// Auto-start if not initialized
	if gd == nil {
		d.logger.Info("Google Drive not initialized, starting now...")
		if _, err := d.StartGoogleDriveSync(ctx); err != nil {
			return "", fmt.Errorf("failed to start Google Drive sync: %w", err)
		}
		// Wait a moment for initialization
		time.Sleep(1 * time.Second)
		d.mu.RLock()
		gd = d.googleDrive
		d.mu.RUnlock()
	}

	if gd == nil {
		return "", fmt.Errorf("google Drive sync failed to initialize")
	}

	result := gd.SyncAll()
	fmt.Println(result)
	return result, nil
}

// SyncDirectoryGoogleDrive forces sync a specific directory immediately
func (d *Daemira) SyncDirectoryGoogleDrive(ctx context.Context, directoryPath string) (string, error) {
	d.mu.RLock()
	gd := d.googleDrive
	d.mu.RUnlock()

	// Auto-start if not initialized
	if gd == nil {
		d.logger.Info("Google Drive not initialized, starting now...")
		if _, err := d.StartGoogleDriveSync(ctx); err != nil {
			return "", fmt.Errorf("failed to start Google Drive sync: %w", err)
		}
		// Wait a moment for initialization
		time.Sleep(1 * time.Second)
		d.mu.RLock()
		gd = d.googleDrive
		d.mu.RUnlock()
	}

	if gd == nil {
		return "", fmt.Errorf("google Drive sync failed to initialize")
	}

	result := gd.SyncDirectory(directoryPath)
	fmt.Println(result)
	return result, nil
}

// ResyncDirectoryGoogleDrive forces a resync of a specific directory (rebuilds cache and syncs deletions)
func (d *Daemira) ResyncDirectoryGoogleDrive(ctx context.Context, directoryPath string) (string, error) {
	d.mu.RLock()
	gd := d.googleDrive
	d.mu.RUnlock()

	// Auto-start if not initialized
	if gd == nil {
		d.logger.Info("Google Drive not initialized, starting now...")
		if _, err := d.StartGoogleDriveSync(ctx); err != nil {
			return "", fmt.Errorf("failed to start Google Drive sync: %w", err)
		}
		// Wait a moment for initialization
		time.Sleep(1 * time.Second)
		d.mu.RLock()
		gd = d.googleDrive
		d.mu.RUnlock()
	}

	if gd == nil {
		return "", fmt.Errorf("google Drive sync failed to initialize")
	}

	if err := gd.ResyncDirectory(ctx, directoryPath); err != nil {
		return "", fmt.Errorf("resync failed: %w", err)
	}

	result := fmt.Sprintf("Resync completed for %s. Cache rebuilt and deletions synced.", directoryPath)
	fmt.Println(result)
	return result, nil
}

// GetGoogleDriveExcludePatterns gets Google Drive exclude patterns
func (d *Daemira) GetGoogleDriveExcludePatterns() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.googleDrive == nil {
		return "Google Drive sync is not initialized."
	}

	patterns := d.googleDrive.GetExcludePatterns()
	output := fmt.Sprintf("Google Drive Exclude Patterns (%d total):\n\n", len(patterns))
	output += "These files/folders will NOT be synced:\n"
	for i, pattern := range patterns {
		output += fmt.Sprintf("  %d. %s\n", i+1, pattern)
	}

	return output
}

// AddGoogleDriveExcludePattern adds custom exclude pattern
func (d *Daemira) AddGoogleDriveExcludePattern(pattern string) string {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.googleDrive == nil {
		return "Google Drive sync is not initialized."
	}

	d.googleDrive.AddExcludePattern(pattern)
	return fmt.Sprintf("Added exclude pattern: %s", pattern)
}

// ==================== System Update Methods ====================

// GetSystemUpdateStatus gets system update status
func (d *Daemira) GetSystemUpdateStatus() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.systemUpdate == nil {
		return "System update scheduler is not initialized."
	}

	status := d.systemUpdate.GetStatus()
	output := "System Update Status:\n"
	output += fmt.Sprintf("  Running: %s\n", boolToYesNo(status["running"].(bool)))

	if lastUpdate, ok := status["lastUpdate"].(int64); ok && lastUpdate > 0 {
		output += fmt.Sprintf("  Last Update: %s\n", time.Unix(lastUpdate, 0).Format(time.RFC1123))
	}

	if nextUpdate, ok := status["nextUpdate"].(int64); ok && nextUpdate > 0 {
		output += fmt.Sprintf("  Next Update: %s\n", time.Unix(nextUpdate, 0).Format(time.RFC1123))
	}

	if history, ok := status["history"].([]systemupdate.UpdateHistoryEntry); ok && len(history) > 0 {
		output += "\n  Recent Updates:\n"
		// Show last 5 entries
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
			duration := entry.Duration.Seconds()
			output += fmt.Sprintf("    %s %s (%.1fs)\n", success, entry.Timestamp.Format(time.RFC1123), duration)
		}
	}

	return output
}

// RunSystemUpdate runs system update immediately
func (d *Daemira) RunSystemUpdate(ctx context.Context) (string, error) {
	d.mu.Lock()
	if d.systemUpdate == nil {
		d.systemUpdate = systemupdate.NewSystemUpdate(d.logger, nil)
	}
	su := d.systemUpdate
	d.mu.Unlock()

	if err := su.RunUpdate(ctx); err != nil {
		return "", err
	}

	return "System update completed. Check logs for details.", nil
}

// ==================== Storage Monitoring Methods ====================

// GetDiskStatus gets disk usage summary
func (d *Daemira) GetDiskStatus(ctx context.Context) (string, error) {
	return d.diskMonitor.GetDiskSummary(ctx)
}

// CheckDiskSpace checks for low disk space warnings
func (d *Daemira) CheckDiskSpace(ctx context.Context) (string, error) {
	warnings, err := d.diskMonitor.CheckLowSpace(ctx)
	if err != nil {
		return "", err
	}

	if len(warnings) == 0 {
		return "All disks have sufficient space.", nil
	}

	output := "⚠️  DISK SPACE WARNINGS:\n\n"
	for _, warning := range warnings {
		icon := "🟡"
		if warning.Level == "critical" {
			icon = "🔴"
		}
		output += fmt.Sprintf("%s %s\n", icon, warning.Message)
	}

	return output, nil
}

// GetDiskHealth gets SMART health status for all disks
func (d *Daemira) GetDiskHealth(ctx context.Context) (string, error) {
	statuses, err := d.diskMonitor.GetAllSmartStatus(ctx)
	if err != nil {
		return "", err
	}

	if len(statuses) == 0 {
		return "No SMART status available. Install smartmontools or run with sudo.", nil
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

	return output, nil
}

// ==================== Performance Management Methods ====================

// GetPowerProfile gets current power profile
func (d *Daemira) GetPowerProfile(ctx context.Context) (string, error) {
	profile, err := d.performanceManager.GetCurrentProfile(ctx)
	if err != nil {
		return "Power profiles not available (power-profiles-daemon not running)", nil
	}

	return fmt.Sprintf("Current power profile: %s", profile), nil
}

// SetPowerProfile sets power profile
func (d *Daemira) SetPowerProfile(ctx context.Context, profile systemhealth.PowerProfile) (string, error) {
	if err := d.performanceManager.SetProfile(ctx, profile); err != nil {
		return "", fmt.Errorf("failed to set power profile to %s: %w", profile, err)
	}

	return fmt.Sprintf("Power profile set to: %s", profile), nil
}

// ListPowerProfiles lists all available power profiles
func (d *Daemira) ListPowerProfiles(ctx context.Context) (string, error) {
	profiles, err := d.performanceManager.GetAllProfiles(ctx)
	if err != nil {
		return "", err
	}

	if len(profiles) == 0 {
		return "No power profiles available (power-profiles-daemon not running)", nil
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

	return output, nil
}

// GetCPUStats gets CPU statistics
func (d *Daemira) GetCPUStats(ctx context.Context) (string, error) {
	stats, err := d.performanceManager.GetCPUStats(ctx)
	if err != nil {
		return "", err
	}
	return d.performanceManager.FormatCPUStats(stats), nil
}

// SuggestPowerProfile auto-suggests optimal power profile
func (d *Daemira) SuggestPowerProfile(ctx context.Context) (string, error) {
	suggested, err := d.performanceManager.SuggestProfile(ctx)
	if err != nil {
		return "", err
	}

	current, _ := d.performanceManager.GetCurrentProfile(ctx)

	output := fmt.Sprintf("Suggested power profile: %s\n", suggested)
	if current != "" {
		output += fmt.Sprintf("Current power profile: %s\n", current)

		if current != suggested {
			output += fmt.Sprintf("\nRecommendation: Switch to %s for better performance/efficiency", suggested)
		} else {
			output += "\n✓ Current profile is optimal"
		}
	}

	return output, nil
}

// ==================== Memory Monitoring Methods ====================

// GetMemoryStats gets memory statistics
func (d *Daemira) GetMemoryStats(ctx context.Context) (string, error) {
	stats, err := d.memoryMonitor.GetMemoryStats(ctx)
	if err != nil {
		return "", err
	}
	return d.memoryMonitor.FormatMemoryStats(stats), nil
}

// CheckSwappiness checks swappiness configuration
func (d *Daemira) CheckSwappiness(ctx context.Context) (string, error) {
	check, err := d.memoryMonitor.CheckSwappiness(ctx)
	if err != nil {
		return "", err
	}
	return check["message"].(string), nil
}

// ==================== Desktop Environment Methods ====================

// GetDesktopStatus gets desktop status
func (d *Daemira) GetDesktopStatus(ctx context.Context) (string, error) {
	return d.desktopIntegration.GetFormattedStatus(ctx)
}

// GetSessionInfo gets session info
func (d *Daemira) GetSessionInfo(ctx context.Context) (string, error) {
	return d.desktopIntegration.GetSessionStatus(ctx)
}

// GetCompositorInfo gets compositor info
func (d *Daemira) GetCompositorInfo(ctx context.Context) (string, error) {
	return d.desktopIntegration.GetCompositorStatus(ctx)
}

// GetDisplayInfo gets display info
func (d *Daemira) GetDisplayInfo(ctx context.Context) (string, error) {
	return d.desktopIntegration.GetDisplayStatus(ctx)
}

// LockSession locks the session
func (d *Daemira) LockSession(ctx context.Context) (string, error) {
	return d.desktopIntegration.LockSession(ctx)
}

// UnlockSession unlocks the session
func (d *Daemira) UnlockSession(ctx context.Context) (string, error) {
	return d.desktopIntegration.UnlockSession(ctx)
}

// ==================== System Health Overview ====================

// GetSystemStatus gets comprehensive system status
func (d *Daemira) GetSystemStatus(ctx context.Context) (string, error) {
	output := "=== Daemira System Status ===\n\n"

	// CPU & Performance
	if stats, err := d.performanceManager.GetCPUStats(ctx); err == nil {
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
	if memStats, err := d.memoryMonitor.GetMemoryStats(ctx); err == nil {
		output += fmt.Sprintf("Memory: %.1fGB / %.1fGB (%.1f%%)", memStats.UsedGB, memStats.TotalGB, memStats.PercentUsed)
		if memStats.Swap.UsedGB > 0 {
			output += fmt.Sprintf(" + %.1fGB swap", memStats.Swap.UsedGB)
		}
		output += "\n"
	} else {
		output += "Memory: Unable to read stats\n"
	}

	// Disk space warnings
	if warnings, err := d.diskMonitor.CheckLowSpace(ctx); err == nil {
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
	d.mu.RLock()
	if d.googleDrive != nil {
		gdStatus := d.googleDrive.GetStatus()
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
	d.mu.RUnlock()

	// System Update status
	d.mu.RLock()
	if d.systemUpdate != nil {
		suStatus := d.systemUpdate.GetStatus()
		if lastUpdate, ok := suStatus["lastUpdate"].(int64); ok && lastUpdate > 0 {
			hoursSince := time.Since(time.Unix(lastUpdate, 0)).Hours()
			output += fmt.Sprintf("System Update: Last %.1fh ago\n", hoursSince)
		} else {
			output += "System Update: Never run\n"
		}
	} else {
		output += "System Update: Not initialized\n"
	}
	d.mu.RUnlock()

	// Desktop Environment
	if desktopSummary, err := d.desktopIntegration.GetDesktopSummary(ctx); err == nil {
		output += fmt.Sprintf("\nDesktop Environment:\n  %s\n", desktopSummary)
	} else {
		output += "\nDesktop Environment: Unable to query\n"
	}

	return output, nil
}

// Helper functions

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func boolToPassedFailed(b bool) string {
	if b {
		return "PASSED"
	}
	return "FAILED"
}

func boolToRunningStopped(b bool) string {
	if b {
		return "Running"
	}
	return "Stopped"
}
