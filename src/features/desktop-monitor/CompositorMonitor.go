/**
 * Compositor monitor - monitors Hyprland compositor state
 */

package desktopmonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ln64-git/daemira/src/utility"
)

// CompositorMonitor monitors Hyprland compositor state
type CompositorMonitor struct {
	logger *utility.Logger
	shell  *utility.Shell
	mu     sync.RWMutex
}

var (
	compositorMonitorInstance *CompositorMonitor
	compositorMonitorOnce     sync.Once
)

// GetCompositorMonitor returns the singleton CompositorMonitor instance
func GetCompositorMonitor() *CompositorMonitor {
	compositorMonitorOnce.Do(func() {
		compositorMonitorInstance = &CompositorMonitor{
			logger: utility.GetLogger(),
			shell:  utility.NewShell(utility.GetLogger()),
		}
	})
	return compositorMonitorInstance
}

// IsAvailable checks if Hyprland is available
func (cm *CompositorMonitor) IsAvailable() bool {
	return os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != ""
}

// GetCompositorInfo gets compositor information
func (cm *CompositorMonitor) GetCompositorInfo(ctx context.Context) (*CompositorInfo, error) {
	if !cm.IsAvailable() {
		return &CompositorInfo{
			Name:      "unknown",
			Version:   "unknown",
			Available: false,
		}, nil
	}

	result, err := cm.shell.Execute(ctx, "hyprctl version -j", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		cm.logger.Error("hyprctl version failed: %v", err)
		return &CompositorInfo{
			Name:      "Hyprland",
			Version:   "unknown",
			Available: false,
		}, nil
	}

	var versionData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &versionData); err != nil {
		cm.logger.Error("Error parsing version JSON: %v", err)
		return &CompositorInfo{
			Name:      "Hyprland",
			Version:   "unknown",
			Available: false,
		}, nil
	}

	version := "unknown"
	if tag, ok := versionData["tag"].(string); ok && tag != "" {
		version = tag
	} else if commit, ok := versionData["commit"].(string); ok && commit != "" {
		if len(commit) > 7 {
			version = commit[:7]
		} else {
			version = commit
		}
	}

	branch := ""
	if b, ok := versionData["branch"].(string); ok {
		branch = b
	}

	commit := ""
	if c, ok := versionData["commit"].(string); ok {
		commit = c
	}

	buildDate := ""
	if d, ok := versionData["date"].(string); ok {
		buildDate = d
	}

	return &CompositorInfo{
		Name:      "Hyprland",
		Version:   version,
		Available: true,
		Branch:    branch,
		Commit:    commit,
		BuildDate: buildDate,
	}, nil
}

// GetWorkspaces gets all workspaces
func (cm *CompositorMonitor) GetWorkspaces(ctx context.Context) ([]WorkspaceInfo, error) {
	if !cm.IsAvailable() {
		return []WorkspaceInfo{}, nil
	}

	result, err := cm.shell.Execute(ctx, "hyprctl workspaces -j", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		cm.logger.Error("hyprctl workspaces failed: %v", err)
		return []WorkspaceInfo{}, err
	}

	var workspaces []WorkspaceInfo
	if err := json.Unmarshal([]byte(result.Stdout), &workspaces); err != nil {
		cm.logger.Error("Error parsing workspaces JSON: %v", err)
		return []WorkspaceInfo{}, err
	}

	return workspaces, nil
}

// GetActiveWindow gets the active window
func (cm *CompositorMonitor) GetActiveWindow(ctx context.Context) (*WindowInfo, error) {
	if !cm.IsAvailable() {
		return nil, nil
	}

	result, err := cm.shell.Execute(ctx, "hyprctl activewindow -j", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, nil
	}

	var window WindowInfo
	if err := json.Unmarshal([]byte(result.Stdout), &window); err != nil {
		return nil, nil
	}

	if window.Address == "" || window.Address == "0x" {
		return nil, nil
	}

	return &window, nil
}

// GetWindows gets all windows
func (cm *CompositorMonitor) GetWindows(ctx context.Context) ([]WindowInfo, error) {
	if !cm.IsAvailable() {
		return []WindowInfo{}, nil
	}

	result, err := cm.shell.Execute(ctx, "hyprctl clients -j", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		cm.logger.Error("hyprctl clients failed: %v", err)
		return []WindowInfo{}, err
	}

	var windows []WindowInfo
	if err := json.Unmarshal([]byte(result.Stdout), &windows); err != nil {
		cm.logger.Error("Error parsing windows JSON: %v", err)
		return []WindowInfo{}, err
	}

	return windows, nil
}

// GetWindowCount gets the number of windows
func (cm *CompositorMonitor) GetWindowCount(ctx context.Context) (int, error) {
	windows, err := cm.GetWindows(ctx)
	if err != nil {
		return 0, err
	}
	return len(windows), nil
}

// FormatCompositorInfo formats compositor info for display
func (cm *CompositorMonitor) FormatCompositorInfo(info *CompositorInfo, workspaces []WorkspaceInfo, activeWindow *WindowInfo, windowCount int) string {
	lines := []string{
		"Compositor Information:",
		fmt.Sprintf("  Name: %s", info.Name),
		fmt.Sprintf("  Version: %s", info.Version),
		fmt.Sprintf("  Available: %s", boolToYesNo(info.Available)),
	}

	if info.Branch != "" {
		lines = append(lines, fmt.Sprintf("  Branch: %s", info.Branch))
	}

	if info.Commit != "" {
		commitShort := info.Commit
		if len(commitShort) > 7 {
			commitShort = commitShort[:7]
		}
		lines = append(lines, fmt.Sprintf("  Commit: %s", commitShort))
	}

	if info.Available {
		lines = append(lines, "")
		lines = append(lines, "Workspaces:")

		if len(workspaces) == 0 {
			lines = append(lines, "  No workspaces found")
		} else {
			// Sort workspaces by ID
			sortedWorkspaces := make([]WorkspaceInfo, len(workspaces))
			copy(sortedWorkspaces, workspaces)
			sort.Slice(sortedWorkspaces, func(i, j int) bool {
				return sortedWorkspaces[i].ID < sortedWorkspaces[j].ID
			})

			for _, ws := range sortedWorkspaces {
				lines = append(lines, fmt.Sprintf("  %d (%s): %d windows on %s", ws.ID, ws.Name, ws.Windows, ws.Monitor))
			}
		}

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Total Windows: %d", windowCount))

		if activeWindow != nil {
			lines = append(lines, "")
			lines = append(lines, "Active Window:")
			lines = append(lines, fmt.Sprintf("  Title: %s", activeWindow.Title))
			lines = append(lines, fmt.Sprintf("  Class: %s", activeWindow.Class))
			lines = append(lines, fmt.Sprintf("  Workspace: %s", activeWindow.Workspace.Name))
		} else {
			lines = append(lines, "")
			lines = append(lines, "Active Window: None")
		}
	}

	return strings.Join(lines, "\n")
}
