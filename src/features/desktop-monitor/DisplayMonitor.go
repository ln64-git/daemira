/**
 * Display monitor - monitors display/monitor information
 */

package desktopmonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ln64-git/daemira/src/utility"
)

// DisplayMonitor monitors display/monitor information
type DisplayMonitor struct {
	logger *utility.Logger
	shell  *utility.Shell
	mu     sync.RWMutex
}

var (
	displayMonitorInstance *DisplayMonitor
	displayMonitorOnce     sync.Once
)

// GetDisplayMonitor returns the singleton DisplayMonitor instance
func GetDisplayMonitor() *DisplayMonitor {
	displayMonitorOnce.Do(func() {
		displayMonitorInstance = &DisplayMonitor{
			logger: utility.GetLogger(),
			shell:  utility.NewShell(utility.GetLogger()),
		}
	})
	return displayMonitorInstance
}

// IsAvailable checks if Hyprland is available
func (dm *DisplayMonitor) IsAvailable() bool {
	return os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != ""
}

// GetMonitors gets all monitors
func (dm *DisplayMonitor) GetMonitors(ctx context.Context) ([]MonitorInfo, error) {
	if !dm.IsAvailable() {
		return []MonitorInfo{}, nil
	}

	result, err := dm.shell.Execute(ctx, "hyprctl monitors -j", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		dm.logger.Error("hyprctl monitors failed: %v", err)
		return []MonitorInfo{}, err
	}

	var monitors []MonitorInfo
	if err := json.Unmarshal([]byte(result.Stdout), &monitors); err != nil {
		dm.logger.Error("Error parsing monitors JSON: %v", err)
		return []MonitorInfo{}, err
	}

	return monitors, nil
}

// GetPrimaryMonitor gets the primary/active monitor
func (dm *DisplayMonitor) GetPrimaryMonitor(ctx context.Context) (*MonitorInfo, error) {
	monitors, err := dm.GetMonitors(ctx)
	if err != nil {
		return nil, err
	}

	if len(monitors) == 0 {
		return nil, nil
	}

	// Find active monitor (with active workspace ID > 0)
	for i := range monitors {
		if monitors[i].ActiveWorkspace.ID > 0 {
			return &monitors[i], nil
		}
	}

	// Return first monitor if no active found
	return &monitors[0], nil
}

// GetMonitorCount gets the number of monitors
func (dm *DisplayMonitor) GetMonitorCount(ctx context.Context) (int, error) {
	monitors, err := dm.GetMonitors(ctx)
	if err != nil {
		return 0, err
	}
	return len(monitors), nil
}

// FormatMonitorInfo formats monitor info for display
func (dm *DisplayMonitor) FormatMonitorInfo(monitors []MonitorInfo) string {
	if len(monitors) == 0 {
		return "Display Information:\n  No monitors detected"
	}

	lines := []string{"Display Information:"}

	for _, monitor := range monitors {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s:", monitor.Name))

		if monitor.Description != "" && monitor.Description != monitor.Name {
			lines = append(lines, fmt.Sprintf("    Description: %s", monitor.Description))
		}

		if monitor.Make != "" && monitor.Model != "" {
			lines = append(lines, fmt.Sprintf("    Make/Model: %s %s", monitor.Make, monitor.Model))
		}

		lines = append(lines, fmt.Sprintf("    Resolution: %dx%d@%.2fHz", monitor.Width, monitor.Height, monitor.RefreshRate))
		lines = append(lines, fmt.Sprintf("    Position: %d,%d", monitor.X, monitor.Y))
		lines = append(lines, fmt.Sprintf("    Scale: %.2f", monitor.Scale))
		lines = append(lines, fmt.Sprintf("    VRR: %s", boolToEnabled(monitor.VRR)))
		lines = append(lines, fmt.Sprintf("    DPMS: %s", boolToOnOff(monitor.DPMSStatus)))
		lines = append(lines, fmt.Sprintf("    Active Workspace: %s", monitor.ActiveWorkspace.Name))

		if monitor.Transform != 0 {
			lines = append(lines, fmt.Sprintf("    Transform: %d", monitor.Transform))
		}
	}

	return strings.Join(lines, "\n")
}

// FormatMonitorSummary formats a summary of monitors
func (dm *DisplayMonitor) FormatMonitorSummary(monitors []MonitorInfo) string {
	if len(monitors) == 0 {
		return "No monitors"
	}

	var summaries []string
	for _, m := range monitors {
		vrrStr := ""
		if m.VRR {
			vrrStr = " VRR"
		}
		summaries = append(summaries, fmt.Sprintf("%s (%dx%d@%.0fHz%s)", m.Name, m.Width, m.Height, m.RefreshRate, vrrStr))
	}

	return strings.Join(summaries, ", ")
}

// boolToEnabled converts bool to "enabled"/"disabled"
func boolToEnabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

// boolToOnOff converts bool to "on"/"off"
func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
