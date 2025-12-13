/**
 * Disk monitoring utility
 * Monitors disk space, health (SMART), and provides alerts
 */

package systemhealth

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ln64-git/daemira/src/utility"
)

// DiskUsage represents disk usage information
type DiskUsage struct {
	Device      string
	MountPoint  string
	Filesystem  string
	TotalBytes  int64
	UsedBytes   int64
	FreeBytes   int64
	PercentUsed float64
	TotalGB     float64
	UsedGB      float64
	FreeGB      float64
	Status      string // "healthy", "warning", "critical"
}

// DiskWarning represents a disk health warning
type DiskWarning struct {
	Device      string
	MountPoint  string
	Level       string // "warning", "critical"
	Message     string
	FreeGB      float64
	PercentUsed float64
}

// SmartStatus represents SMART health status
type SmartStatus struct {
	Device       string
	Passed       bool
	Temperature  *int
	PowerOnHours *int
	PowerCycles  *int
	Errors       []string
	RawOutput    string
}

// Protected disks that should never be mounted or modified
var protectedDisks = []string{"sdc"} // Windows partition

// DiskMonitor monitors disk space, health (SMART), and provides alerts
type DiskMonitor struct {
	logger *utility.Logger
	shell  *utility.Shell
	mu     sync.RWMutex
}

var (
	diskMonitorInstance *DiskMonitor
	diskMonitorOnce     sync.Once
)

// GetDiskMonitor returns the singleton DiskMonitor instance
func GetDiskMonitor() *DiskMonitor {
	diskMonitorOnce.Do(func() {
		diskMonitorInstance = &DiskMonitor{
			logger: utility.GetLogger(),
			shell:  utility.NewShell(utility.GetLogger()),
		}
	})
	return diskMonitorInstance
}

// IsProtectedDisk checks if a disk is protected (e.g., Windows partition)
func (dm *DiskMonitor) IsProtectedDisk(device string) bool {
	for _, protected := range protectedDisks {
		if strings.Contains(device, protected) {
			return true
		}
	}
	return false
}

// GetAllDiskUsage gets all mounted disk usage information
func (dm *DiskMonitor) GetAllDiskUsage(ctx context.Context) ([]DiskUsage, error) {
	result, err := dm.shell.Execute(ctx,
		`df -B1 --output=source,target,fstype,size,used,avail,pcent | grep -E "^/dev/"`,
		&utility.ExecOptions{
			Timeout: 10 * time.Second,
		})

	if err != nil || result.ExitCode != 0 {
		dm.logger.Error("Failed to get disk usage: %v", err)
		return []DiskUsage{}, err
	}

	var disks []DiskUsage
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")

	for _, line := range lines {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) < 7 {
			continue
		}

		device := parts[0]
		mountPoint := parts[1]
		filesystem := parts[2]
		totalStr := parts[3]
		usedStr := parts[4]
		freeStr := parts[5]
		percentStr := parts[6]

		// Validate all required fields exist
		if device == "" || mountPoint == "" || filesystem == "" ||
			totalStr == "" || usedStr == "" || freeStr == "" || percentStr == "" {
			continue
		}

		percentUsed, err := strconv.ParseFloat(strings.TrimSuffix(percentStr, "%"), 64)
		if err != nil {
			continue
		}

		totalBytes, err := strconv.ParseInt(totalStr, 10, 64)
		if err != nil {
			continue
		}

		usedBytes, err := strconv.ParseInt(usedStr, 10, 64)
		if err != nil {
			continue
		}

		freeBytes, err := strconv.ParseInt(freeStr, 10, 64)
		if err != nil {
			continue
		}

		// Determine status based on thresholds
		status := "healthy"
		if percentUsed >= 95 || freeBytes < 100*1024*1024*1024 {
			status = "critical"
		} else if percentUsed >= 90 || freeBytes < 200*1024*1024*1024 {
			status = "warning"
		}

		disks = append(disks, DiskUsage{
			Device:      device,
			MountPoint:  mountPoint,
			Filesystem:  filesystem,
			TotalBytes:  totalBytes,
			UsedBytes:   usedBytes,
			FreeBytes:   freeBytes,
			PercentUsed: percentUsed,
			TotalGB:     float64(totalBytes) / 1024 / 1024 / 1024,
			UsedGB:      float64(usedBytes) / 1024 / 1024 / 1024,
			FreeGB:      float64(freeBytes) / 1024 / 1024 / 1024,
			Status:      status,
		})
	}

	return disks, nil
}

// CheckLowSpace checks for low disk space warnings
func (dm *DiskMonitor) CheckLowSpace(ctx context.Context) ([]DiskWarning, error) {
	disks, err := dm.GetAllDiskUsage(ctx)
	if err != nil {
		return nil, err
	}

	var warnings []DiskWarning
	for _, disk := range disks {
		if disk.Status == "critical" {
			warnings = append(warnings, DiskWarning{
				Device:      disk.Device,
				MountPoint:  disk.MountPoint,
				Level:       "critical",
				Message:     fmt.Sprintf("CRITICAL: %s has only %.1fGB free (%.1f%% used)", disk.MountPoint, disk.FreeGB, disk.PercentUsed),
				FreeGB:      disk.FreeGB,
				PercentUsed: disk.PercentUsed,
			})
		} else if disk.Status == "warning" {
			warnings = append(warnings, DiskWarning{
				Device:      disk.Device,
				MountPoint:  disk.MountPoint,
				Level:       "warning",
				Message:     fmt.Sprintf("WARNING: %s has %.1fGB free (%.1f%% used)", disk.MountPoint, disk.FreeGB, disk.PercentUsed),
				FreeGB:      disk.FreeGB,
				PercentUsed: disk.PercentUsed,
			})
		}
	}

	return warnings, nil
}

// GetSmartStatus gets SMART health status for a disk
// Requires smartmontools (smartctl)
func (dm *DiskMonitor) GetSmartStatus(ctx context.Context, device string) (*SmartStatus, error) {
	// Check if smartctl is available
	checkResult, err := dm.shell.Execute(ctx, "which smartctl", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err != nil || checkResult.ExitCode != 0 {
		dm.logger.Warn("smartctl not found - install smartmontools package")
		return nil, fmt.Errorf("smartctl not available")
	}

	// Get SMART health
	result, err := dm.shell.Execute(ctx, fmt.Sprintf("sudo smartctl -H %s", device), &utility.ExecOptions{
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	passed := strings.Contains(result.Stdout, "PASSED")
	status := &SmartStatus{
		Device:    device,
		Passed:    passed,
		RawOutput: result.Stdout,
	}

	// Get detailed SMART data
	detailResult, err := dm.shell.Execute(ctx, fmt.Sprintf("sudo smartctl -a %s", device), &utility.ExecOptions{
		Timeout: 30 * time.Second,
	})
	if err == nil && detailResult.ExitCode == 0 {
		// Extract temperature
		tempRegex := regexp.MustCompile(`Temperature.*?(\d+)\s*Celsius`)
		if matches := tempRegex.FindStringSubmatch(detailResult.Stdout); len(matches) > 1 {
			if temp, err := strconv.Atoi(matches[1]); err == nil {
				status.Temperature = &temp
			}
		}

		// Extract power on hours
		hoursRegex := regexp.MustCompile(`Power_On_Hours.*?(\d+)`)
		if matches := hoursRegex.FindStringSubmatch(detailResult.Stdout); len(matches) > 1 {
			if hours, err := strconv.Atoi(matches[1]); err == nil {
				status.PowerOnHours = &hours
			}
		}

		// Extract power cycles
		cyclesRegex := regexp.MustCompile(`Power_Cycle_Count.*?(\d+)`)
		if matches := cyclesRegex.FindStringSubmatch(detailResult.Stdout); len(matches) > 1 {
			if cycles, err := strconv.Atoi(matches[1]); err == nil {
				status.PowerCycles = &cycles
			}
		}

		// Check for errors
		var errors []string
		if strings.Contains(detailResult.Stdout, "FAILING_NOW") {
			errors = append(errors, "Disk has attributes FAILING NOW")
		}

		reallocRegex := regexp.MustCompile(`Reallocated_Sector_Ct\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+(\d+)`)
		if matches := reallocRegex.FindStringSubmatch(detailResult.Stdout); len(matches) > 1 {
			if count, err := strconv.Atoi(matches[1]); err == nil && count > 0 {
				errors = append(errors, fmt.Sprintf("Reallocated sectors: %d", count))
			}
		}

		if len(errors) > 0 {
			status.Errors = errors
		}
	}

	return status, nil
}

// GetAllSmartStatus gets SMART status for all physical disks
func (dm *DiskMonitor) GetAllSmartStatus(ctx context.Context) ([]SmartStatus, error) {
	result, err := dm.shell.Execute(ctx,
		`lsblk -d -n -o NAME,TYPE | grep disk | awk '{print "/dev/"$1}'`,
		&utility.ExecOptions{
			Timeout: 10 * time.Second,
		})

	if err != nil || result.ExitCode != 0 {
		dm.logger.Error("Failed to list disks: %v", err)
		return []SmartStatus{}, err
	}

	disks := strings.Fields(strings.TrimSpace(result.Stdout))
	var statuses []SmartStatus

	for _, disk := range disks {
		// Skip protected disks
		if dm.IsProtectedDisk(disk) {
			dm.logger.Info("Skipping protected disk: %s", disk)
			continue
		}

		status, err := dm.GetSmartStatus(ctx, disk)
		if err != nil {
			continue
		}
		if status != nil {
			statuses = append(statuses, *status)
		}
	}

	return statuses, nil
}

// FormatDiskUsage formats disk usage for display
func (dm *DiskMonitor) FormatDiskUsage(disk DiskUsage) string {
	var statusIcon string
	switch disk.Status {
	case "critical":
		statusIcon = "🔴"
	case "warning":
		statusIcon = "🟡"
	default:
		statusIcon = "🟢"
	}
	return fmt.Sprintf("%s %s (%s): %.1fGB / %.1fGB (%.1f%%) - %.1fGB free",
		statusIcon, disk.MountPoint, disk.Device, disk.UsedGB, disk.TotalGB, disk.PercentUsed, disk.FreeGB)
}

// GetDiskSummary gets a summary of all disk usage
func (dm *DiskMonitor) GetDiskSummary(ctx context.Context) (string, error) {
	disks, err := dm.GetAllDiskUsage(ctx)
	if err != nil {
		return "", err
	}

	warnings, err := dm.CheckLowSpace(ctx)
	if err != nil {
		return "", err
	}

	summary := "=== Disk Usage Summary ===\n\n"

	// Add warnings first
	if len(warnings) > 0 {
		summary += "⚠️  WARNINGS:\n"
		for _, warning := range warnings {
			icon := "🟡"
			if warning.Level == "critical" {
				icon = "🔴"
			}
			summary += fmt.Sprintf("  %s %s\n", icon, warning.Message)
		}
		summary += "\n"
	}

	// Add all disks
	summary += "All Disks:\n"
	for _, disk := range disks {
		summary += fmt.Sprintf("  %s\n", dm.FormatDiskUsage(disk))
	}

	return summary, nil
}
