/**
 * Session monitor - monitors systemd-logind session state
 */

package desktopmonitor

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ln64-git/daemira/src/utility"
)

// SessionMonitor monitors systemd-logind session state
type SessionMonitor struct {
	logger *utility.Logger
	shell  *utility.Shell
	mu     sync.RWMutex
}

var (
	sessionMonitorInstance *SessionMonitor
	sessionMonitorOnce     sync.Once
)

// GetSessionMonitor returns the singleton SessionMonitor instance
func GetSessionMonitor() *SessionMonitor {
	sessionMonitorOnce.Do(func() {
		sessionMonitorInstance = &SessionMonitor{
			logger: utility.GetLogger(),
			shell:  utility.NewShell(utility.GetLogger()),
		}
	})
	return sessionMonitorInstance
}

// GetSessionInfo gets current session information
func (sm *SessionMonitor) GetSessionInfo(ctx context.Context) (*SessionInfo, error) {
	sessionID := os.Getenv("XDG_SESSION_ID")
	if sessionID == "" {
		sm.logger.Warn("XDG_SESSION_ID not set, session monitoring unavailable")
		return sm.getDefaultSessionInfo(), nil
	}

	result, err := sm.shell.Execute(ctx, fmt.Sprintf("loginctl show-session %s", sessionID), &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		sm.logger.Error("loginctl failed: %v", err)
		return sm.getDefaultSessionInfo(), nil
	}

	return sm.parseLoginctlOutput(result.Stdout), nil
}

// parseLoginctlOutput parses loginctl output into SessionInfo
func (sm *SessionMonitor) parseLoginctlOutput(output string) *SessionInfo {
	lines := strings.Split(output, "\n")
	props := make(map[string]string)

	re := regexp.MustCompile(`^([^=]+)=(.*)$`)
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			props[strings.TrimSpace(matches[1])] = strings.TrimSpace(matches[2])
		}
	}

	sessionID := props["Id"]
	if sessionID == "" {
		sessionID = os.Getenv("XDG_SESSION_ID")
	}
	if sessionID == "" {
		sessionID = "unknown"
	}

	user := props["Name"]
	if user == "" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "unknown"
	}

	seat := props["Seat"]
	if seat == "" {
		seat = "seat0"
	}

	sessionType := strings.ToLower(props["Type"])
	if sessionType == "" {
		sessionType = strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	}
	if sessionType == "" {
		sessionType = "unknown"
	}

	state := props["State"]
	if state == "" {
		state = "unknown"
	}

	active := props["Active"] == "yes"
	idle := props["IdleHint"] == "yes"
	locked := props["LockedHint"] == "yes"

	vt := 0
	if vtStr := props["VTNr"]; vtStr != "" {
		if v, err := strconv.Atoi(vtStr); err == nil {
			vt = v
		}
	}

	display := props["Display"]
	if display == "" {
		display = os.Getenv("DISPLAY")
	}

	return &SessionInfo{
		SessionID: sessionID,
		User:      user,
		Seat:      seat,
		Type:      sessionType,
		State:     state,
		Active:    active,
		Idle:      idle,
		Locked:    locked,
		VT:        vt,
		Display:   display,
	}
}

// getDefaultSessionInfo returns default session info when loginctl fails
func (sm *SessionMonitor) getDefaultSessionInfo() *SessionInfo {
	sessionID := os.Getenv("XDG_SESSION_ID")
	if sessionID == "" {
		sessionID = "unknown"
	}

	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}

	sessionType := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	if sessionType == "" {
		sessionType = "unknown"
	}

	display := os.Getenv("DISPLAY")

	return &SessionInfo{
		SessionID: sessionID,
		User:      user,
		Seat:      "seat0",
		Type:      sessionType,
		State:     "unknown",
		Active:    false,
		Idle:      false,
		Locked:    false,
		VT:        0,
		Display:   display,
	}
}

// IsSessionLocked checks if session is locked
func (sm *SessionMonitor) IsSessionLocked(ctx context.Context) (bool, error) {
	info, err := sm.GetSessionInfo(ctx)
	if err != nil {
		return false, err
	}
	return info.Locked, nil
}

// GetIdleStatus checks if session is idle
func (sm *SessionMonitor) GetIdleStatus(ctx context.Context) (bool, error) {
	info, err := sm.GetSessionInfo(ctx)
	if err != nil {
		return false, err
	}
	return info.Idle, nil
}

// LockSession locks the current session
func (sm *SessionMonitor) LockSession(ctx context.Context) error {
	sessionID := os.Getenv("XDG_SESSION_ID")
	if sessionID == "" {
		return fmt.Errorf("XDG_SESSION_ID not set")
	}

	result, err := sm.shell.Execute(ctx, fmt.Sprintf("loginctl lock-session %s", sessionID), &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("loginctl lock-session failed: %v", err)
	}

	sm.logger.Info("Session locked successfully")
	return nil
}

// UnlockSession unlocks the current session
func (sm *SessionMonitor) UnlockSession(ctx context.Context) error {
	sessionID := os.Getenv("XDG_SESSION_ID")
	if sessionID == "" {
		return fmt.Errorf("XDG_SESSION_ID not set")
	}

	result, err := sm.shell.Execute(ctx, fmt.Sprintf("loginctl unlock-session %s", sessionID), &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("loginctl unlock-session failed: %v", err)
	}

	sm.logger.Info("Session unlocked successfully")
	return nil
}

// FormatSessionInfo formats session info for display
func (sm *SessionMonitor) FormatSessionInfo(info *SessionInfo) string {
	lines := []string{
		"Session Information:",
		fmt.Sprintf("  Session ID: %s", info.SessionID),
		fmt.Sprintf("  User: %s", info.User),
		fmt.Sprintf("  Seat: %s", info.Seat),
		fmt.Sprintf("  Type: %s", info.Type),
		fmt.Sprintf("  State: %s", info.State),
		fmt.Sprintf("  Active: %s", boolToYesNo(info.Active)),
		fmt.Sprintf("  Idle: %s", boolToYesNo(info.Idle)),
		fmt.Sprintf("  Locked: %s", boolToYesNo(info.Locked)),
	}

	if info.VT > 0 {
		lines = append(lines, fmt.Sprintf("  VT: %d", info.VT))
	}

	if info.Display != "" {
		lines = append(lines, fmt.Sprintf("  Display: %s", info.Display))
	}

	return strings.Join(lines, "\n")
}

// boolToYesNo converts bool to "yes"/"no"
func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
