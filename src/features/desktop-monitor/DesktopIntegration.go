/**
 * Desktop integration - orchestrates all desktop monitoring components
 */

package desktopmonitor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ln64-git/daemira/src/utility"
)

// DesktopIntegration orchestrates all desktop monitoring components
type DesktopIntegration struct {
	logger           *utility.Logger
	sessionMonitor   *SessionMonitor
	compositorMonitor *CompositorMonitor
	displayMonitor   *DisplayMonitor
	mu               sync.RWMutex
}

var (
	desktopIntegrationInstance *DesktopIntegration
	desktopIntegrationOnce     sync.Once
)

// GetDesktopIntegration returns the singleton DesktopIntegration instance
func GetDesktopIntegration() *DesktopIntegration {
	desktopIntegrationOnce.Do(func() {
		desktopIntegrationInstance = &DesktopIntegration{
			logger:            utility.GetLogger(),
			sessionMonitor:    GetSessionMonitor(),
			compositorMonitor: GetCompositorMonitor(),
			displayMonitor:    GetDisplayMonitor(),
		}
	})
	return desktopIntegrationInstance
}

// DetectCompositor detects the compositor type
func (di *DesktopIntegration) DetectCompositor() CompositorType {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return CompositorTypeHyprland
	}
	if os.Getenv("NIRI_SOCKET") != "" {
		return CompositorTypeNiri
	}
	if os.Getenv("SWAYSOCK") != "" {
		return CompositorTypeSway
	}
	if os.Getenv("I3SOCK") != "" {
		return CompositorTypeI3
	}
	return CompositorTypeUnknown
}

// IsDesktopMonitoringAvailable checks if desktop monitoring is available
func (di *DesktopIntegration) IsDesktopMonitoringAvailable() bool {
	return di.compositorMonitor.IsAvailable() || os.Getenv("XDG_SESSION_ID") != ""
}

// GetDesktopStatus gets complete desktop status
func (di *DesktopIntegration) GetDesktopStatus(ctx context.Context) (*DesktopStatus, error) {
	session, err := di.sessionMonitor.GetSessionInfo(ctx)
	if err != nil {
		return nil, err
	}

	compositor, err := di.compositorMonitor.GetCompositorInfo(ctx)
	if err != nil {
		return nil, err
	}

	workspaces, err := di.compositorMonitor.GetWorkspaces(ctx)
	if err != nil {
		return nil, err
	}

	windows, err := di.compositorMonitor.GetWindows(ctx)
	if err != nil {
		return nil, err
	}

	monitors, err := di.displayMonitor.GetMonitors(ctx)
	if err != nil {
		return nil, err
	}

	return &DesktopStatus{
		Session:    *session,
		Compositor: *compositor,
		Workspaces: workspaces,
		Windows:    windows,
		Monitors:   monitors,
	}, nil
}

// GetFormattedStatus gets formatted desktop status
func (di *DesktopIntegration) GetFormattedStatus(ctx context.Context) (string, error) {
	status, err := di.GetDesktopStatus(ctx)
	if err != nil {
		return "", err
	}

	lines := []string{
		"Desktop Environment Status",
		strings.Repeat("=", 50),
		"",
	}

	lines = append(lines, di.sessionMonitor.FormatSessionInfo(&status.Session))
	lines = append(lines, "")

	activeWindow, _ := di.compositorMonitor.GetActiveWindow(ctx)
	windowCount := len(status.Windows)
	lines = append(lines, di.compositorMonitor.FormatCompositorInfo(&status.Compositor, status.Workspaces, activeWindow, windowCount))
	lines = append(lines, "")

	lines = append(lines, di.displayMonitor.FormatMonitorInfo(status.Monitors))

	return strings.Join(lines, "\n"), nil
}

// GetSessionStatus gets session status
func (di *DesktopIntegration) GetSessionStatus(ctx context.Context) (string, error) {
	session, err := di.sessionMonitor.GetSessionInfo(ctx)
	if err != nil {
		return "", err
	}
	return di.sessionMonitor.FormatSessionInfo(session), nil
}

// GetCompositorStatus gets compositor status
func (di *DesktopIntegration) GetCompositorStatus(ctx context.Context) (string, error) {
	compositor, err := di.compositorMonitor.GetCompositorInfo(ctx)
	if err != nil {
		return "", err
	}

	workspaces, err := di.compositorMonitor.GetWorkspaces(ctx)
	if err != nil {
		return "", err
	}

	activeWindow, _ := di.compositorMonitor.GetActiveWindow(ctx)
	windowCount, _ := di.compositorMonitor.GetWindowCount(ctx)

	return di.compositorMonitor.FormatCompositorInfo(compositor, workspaces, activeWindow, windowCount), nil
}

// GetDisplayStatus gets display status
func (di *DesktopIntegration) GetDisplayStatus(ctx context.Context) (string, error) {
	monitors, err := di.displayMonitor.GetMonitors(ctx)
	if err != nil {
		return "", err
	}
	return di.displayMonitor.FormatMonitorInfo(monitors), nil
}

// LockSession locks the session
func (di *DesktopIntegration) LockSession(ctx context.Context) (string, error) {
	if err := di.sessionMonitor.LockSession(ctx); err != nil {
		return fmt.Sprintf("Failed to lock session: %v", err), err
	}
	return "Session locked successfully", nil
}

// UnlockSession unlocks the session
func (di *DesktopIntegration) UnlockSession(ctx context.Context) (string, error) {
	if err := di.sessionMonitor.UnlockSession(ctx); err != nil {
		return fmt.Sprintf("Failed to unlock session: %v", err), err
	}
	return "Session unlocked successfully", nil
}

// GetDesktopSummary gets a summary of desktop status
func (di *DesktopIntegration) GetDesktopSummary(ctx context.Context) (string, error) {
	status, err := di.GetDesktopStatus(ctx)
	if err != nil {
		return "", err
	}

	lines := []string{}

	lines = append(lines, fmt.Sprintf("Compositor: %s %s", status.Compositor.Name, status.Compositor.Version))
	lines = append(lines, fmt.Sprintf("Session: %s (%s)", status.Session.Type, status.Session.Seat))

	if len(status.Workspaces) > 0 {
		lines = append(lines, fmt.Sprintf("Workspaces: %d active", len(status.Workspaces)))
	}

	if len(status.Windows) > 0 {
		lines = append(lines, fmt.Sprintf("Windows: %d open", len(status.Windows)))
	}

	if len(status.Monitors) > 0 {
		lines = append(lines, fmt.Sprintf("Displays: %s", di.displayMonitor.FormatMonitorSummary(status.Monitors)))
	}

	lockState := "unlocked"
	if status.Session.Locked {
		lockState = "locked"
	}
	lines = append(lines, fmt.Sprintf("Lock State: %s", lockState))

	return strings.Join(lines, "\n  "), nil
}

