/**
 * Performance manager
 * Integrates with power-profiles-daemon for CPU power management
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

// PowerProfile represents power profile types
type PowerProfile string

const (
	PowerProfilePerformance PowerProfile = "performance"
	PowerProfileBalanced    PowerProfile = "balanced"
	PowerProfilePowerSaver   PowerProfile = "power-saver"
)

// PowerProfileInfo represents power profile information
type PowerProfileInfo struct {
	Name           PowerProfile
	Active         bool
	CPUDriver      string
	PlatformDriver string
	Degraded        bool
}

// CPUStats represents CPU statistics
type CPUStats struct {
	Cores                int
	Threads              int
	CurrentFrequencyMHz  []float64
	AverageFrequencyMHz  float64
	MinFrequencyMHz      float64
	MaxFrequencyMHz      float64
	Governor             string
	PowerProfile         PowerProfile
	Utilization          float64
}

// PerformanceManager integrates with power-profiles-daemon for CPU power management
type PerformanceManager struct {
	logger *utility.Logger
	shell  *utility.Shell
	mu     sync.RWMutex
}

var (
	performanceManagerInstance *PerformanceManager
	performanceManagerOnce     sync.Once
)

// GetPerformanceManager returns the singleton PerformanceManager instance
func GetPerformanceManager() *PerformanceManager {
	performanceManagerOnce.Do(func() {
		performanceManagerInstance = &PerformanceManager{
			logger: utility.GetLogger(),
			shell:  utility.NewShell(utility.GetLogger()),
		}
	})
	return performanceManagerInstance
}

// IsPowerProfilesAvailable checks if power-profiles-daemon is available
func (pm *PerformanceManager) IsPowerProfilesAvailable(ctx context.Context) (bool, error) {
	result, err := pm.shell.Execute(ctx, "which powerprofilesctl", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return false, err
	}
	return result.ExitCode == 0, nil
}

// GetCurrentProfile gets current power profile
func (pm *PerformanceManager) GetCurrentProfile(ctx context.Context) (PowerProfile, error) {
	available, err := pm.IsPowerProfilesAvailable(ctx)
	if err != nil || !available {
		pm.logger.Warn("power-profiles-daemon not available")
		return "", fmt.Errorf("power-profiles-daemon not available")
	}

	result, err := pm.shell.Execute(ctx, "powerprofilesctl get", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		pm.logger.Error("Failed to get power profile: %v", err)
		return "", err
	}

	profile := PowerProfile(strings.TrimSpace(result.Stdout))
	return profile, nil
}

// GetAllProfiles gets all available power profiles
func (pm *PerformanceManager) GetAllProfiles(ctx context.Context) ([]PowerProfileInfo, error) {
	available, err := pm.IsPowerProfilesAvailable(ctx)
	if err != nil || !available {
		pm.logger.Warn("power-profiles-daemon not available")
		return []PowerProfileInfo{}, nil
	}

	result, err := pm.shell.Execute(ctx, "powerprofilesctl list", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		pm.logger.Error("Failed to list power profiles: %v", err)
		return []PowerProfileInfo{}, err
	}

	var profiles []PowerProfileInfo
	lines := strings.Split(result.Stdout, "\n")

	var currentProfile *PowerProfileInfo
	profileRegex := regexp.MustCompile(`^(\*\s+)?(\w+(-\w+)?):$`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for profile name (starts with * for active, or just profile name)
		if matches := profileRegex.FindStringSubmatch(trimmed); len(matches) > 0 {
			// Save previous profile if exists
			if currentProfile != nil {
				profiles = append(profiles, *currentProfile)
			}

			// Start new profile
			active := strings.HasPrefix(trimmed, "*")
			nameStr := trimmed
			if active {
				nameStr = strings.TrimPrefix(nameStr, "*")
			}
			nameStr = strings.TrimSpace(nameStr)
			nameStr = strings.TrimSuffix(nameStr, ":")

			currentProfile = &PowerProfileInfo{
				Name:   PowerProfile(nameStr),
				Active: active,
			}
		} else if currentProfile != nil {
			// Parse profile properties
			if strings.HasPrefix(trimmed, "CpuDriver:") {
				currentProfile.CPUDriver = strings.TrimSpace(strings.TrimPrefix(trimmed, "CpuDriver:"))
			} else if strings.HasPrefix(trimmed, "PlatformDriver:") {
				currentProfile.PlatformDriver = strings.TrimSpace(strings.TrimPrefix(trimmed, "PlatformDriver:"))
			} else if strings.HasPrefix(trimmed, "Degraded:") {
				currentProfile.Degraded = strings.Contains(trimmed, "yes")
			}
		}
	}

	// Add last profile
	if currentProfile != nil {
		profiles = append(profiles, *currentProfile)
	}

	return profiles, nil
}

// SetProfile sets power profile
func (pm *PerformanceManager) SetProfile(ctx context.Context, profile PowerProfile) error {
	available, err := pm.IsPowerProfilesAvailable(ctx)
	if err != nil || !available {
		pm.logger.Error("power-profiles-daemon not available")
		return fmt.Errorf("power-profiles-daemon not available")
	}

	result, err := pm.shell.Execute(ctx, fmt.Sprintf("powerprofilesctl set %s", profile), &utility.ExecOptions{
		Timeout: 10 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		pm.logger.Error("Failed to set power profile to %s: %v", profile, err)
		return err
	}

	pm.logger.Info("Power profile set to: %s", profile)
	return nil
}

// GetCPUFrequencies gets CPU frequency for all cores
func (pm *PerformanceManager) GetCPUFrequencies(ctx context.Context) ([]float64, error) {
	result, err := pm.shell.Execute(ctx, "grep MHz /proc/cpuinfo | awk '{print $4}'", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		pm.logger.Error("Failed to get CPU frequencies: %v", err)
		return []float64{}, err
	}

	var frequencies []float64
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, line := range lines {
		if freq, err := strconv.ParseFloat(strings.TrimSpace(line), 64); err == nil {
			frequencies = append(frequencies, freq)
		}
	}

	return frequencies, nil
}

// GetCPUGovernor gets CPU governor
func (pm *PerformanceManager) GetCPUGovernor(ctx context.Context) (string, error) {
	result, err := pm.shell.Execute(ctx, "cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return "", err
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GetCPUStats gets comprehensive CPU statistics
func (pm *PerformanceManager) GetCPUStats(ctx context.Context) (*CPUStats, error) {
	// Get CPU info
	cpuInfoResult, err := pm.shell.Execute(ctx, "lscpu", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	cpuInfo := cpuInfoResult.Stdout

	// Extract cores and threads
	coresRegex := regexp.MustCompile(`Core\(s\) per socket:\s*(\d+)`)
	threadsRegex := regexp.MustCompile(`Thread\(s\) per core:\s*(\d+)`)
	socketsRegex := regexp.MustCompile(`Socket\(s\):\s*(\d+)`)

	sockets := 1
	if matches := socketsRegex.FindStringSubmatch(cpuInfo); len(matches) > 1 {
		if s, err := strconv.Atoi(matches[1]); err == nil {
			sockets = s
		}
	}

	coresPerSocket := 1
	if matches := coresRegex.FindStringSubmatch(cpuInfo); len(matches) > 1 {
		if c, err := strconv.Atoi(matches[1]); err == nil {
			coresPerSocket = c
		}
	}

	threadsPerCore := 1
	if matches := threadsRegex.FindStringSubmatch(cpuInfo); len(matches) > 1 {
		if t, err := strconv.Atoi(matches[1]); err == nil {
			threadsPerCore = t
		}
	}

	cores := sockets * coresPerSocket
	threads := cores * threadsPerCore

	// Get frequencies
	frequencies, err := pm.GetCPUFrequencies(ctx)
	if err != nil {
		return nil, err
	}

	averageFrequency := float64(0)
	minFrequency := float64(0)
	maxFrequency := float64(0)

	if len(frequencies) > 0 {
		sum := float64(0)
		for _, freq := range frequencies {
			sum += freq
		}
		averageFrequency = sum / float64(len(frequencies))

		minFrequency = frequencies[0]
		maxFrequency = frequencies[0]
		for _, freq := range frequencies {
			if freq < minFrequency {
				minFrequency = freq
			}
			if freq > maxFrequency {
				maxFrequency = freq
			}
		}
	}

	// Get governor
	governor, _ := pm.GetCPUGovernor(ctx)

	// Get power profile
	powerProfile, _ := pm.GetCurrentProfile(ctx)

	// Get CPU utilization (simple average from uptime)
	var utilization float64
	uptimeResult, err := pm.shell.Execute(ctx, "cat /proc/loadavg", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err == nil && uptimeResult.ExitCode == 0 {
		parts := strings.Fields(uptimeResult.Stdout)
		if len(parts) > 0 {
			if loadAvg, err := strconv.ParseFloat(parts[0], 64); err == nil {
				utilization = (loadAvg / float64(threads)) * 100
				if utilization > 100 {
					utilization = 100
				}
			}
		}
	}

	return &CPUStats{
		Cores:               cores,
		Threads:             threads,
		CurrentFrequencyMHz: frequencies,
		AverageFrequencyMHz:  averageFrequency,
		MinFrequencyMHz:     minFrequency,
		MaxFrequencyMHz:     maxFrequency,
		Governor:            governor,
		PowerProfile:        powerProfile,
		Utilization:         utilization,
	}, nil
}

// SuggestProfile suggests optimal power profile based on CPU utilization
func (pm *PerformanceManager) SuggestProfile(ctx context.Context) (PowerProfile, error) {
	stats, err := pm.GetCPUStats(ctx)
	if err != nil {
		return PowerProfileBalanced, err
	}

	if stats.Utilization == 0 {
		// Default to balanced if can't determine
		return PowerProfileBalanced, nil
	}

	// Suggest based on utilization
	if stats.Utilization > 70 {
		return PowerProfilePerformance, nil
	} else if stats.Utilization < 30 {
		return PowerProfilePowerSaver, nil
	} else {
		return PowerProfileBalanced, nil
	}
}

// FormatCPUStats formats CPU stats for display
func (pm *PerformanceManager) FormatCPUStats(stats *CPUStats) string {
	output := "=== CPU Statistics ===\n\n"
	output += fmt.Sprintf("Cores: %d (%d threads)\n", stats.Cores, stats.Threads)
	output += fmt.Sprintf("Average Frequency: %.0f MHz\n", stats.AverageFrequencyMHz)
	output += fmt.Sprintf("Frequency Range: %.0f - %.0f MHz\n", stats.MinFrequencyMHz, stats.MaxFrequencyMHz)

	if stats.Governor != "" {
		output += fmt.Sprintf("Governor: %s\n", stats.Governor)
	}

	if stats.PowerProfile != "" {
		output += fmt.Sprintf("Power Profile: %s\n", stats.PowerProfile)
	}

	if stats.Utilization > 0 {
		output += fmt.Sprintf("CPU Utilization: %.1f%%\n", stats.Utilization)
	}

	return output
}

