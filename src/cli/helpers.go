package cli

import (
	"fmt"
	"time"
)

// Helper functions for formatting output

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

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format(time.RFC1123)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

