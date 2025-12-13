/**
 * Memory monitor
 * Tracks memory usage, swap, and zram statistics
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

// MemoryStats represents memory statistics
type MemoryStats struct {
	TotalBytes     int64
	UsedBytes      int64
	FreeBytes      int64
	AvailableBytes int64
	BuffersBytes   int64
	CachedBytes    int64
	TotalGB        float64
	UsedGB         float64
	FreeGB         float64
	AvailableGB    float64
	PercentUsed    float64
	Swap           SwapStats
	Zram           *ZramStats
}

// SwapStats represents swap statistics
type SwapStats struct {
	TotalBytes  int64
	UsedBytes   int64
	FreeBytes   int64
	TotalGB     float64
	UsedGB      float64
	FreeGB      float64
	PercentUsed float64
}

// ZramStats represents zram statistics
type ZramStats struct {
	Device           string
	TotalBytes       int64
	UsedBytes        int64
	CompressedBytes  int64
	CompressionRatio float64
	PercentUsed      float64
}

// Optimal swappiness for zram
const optimalSwappinessZram = 180

// MemoryMonitor tracks memory usage, swap, and zram statistics
type MemoryMonitor struct {
	logger *utility.Logger
	shell  *utility.Shell
	mu     sync.RWMutex
}

var (
	memoryMonitorInstance *MemoryMonitor
	memoryMonitorOnce     sync.Once
)

// GetMemoryMonitor returns the singleton MemoryMonitor instance
func GetMemoryMonitor() *MemoryMonitor {
	memoryMonitorOnce.Do(func() {
		memoryMonitorInstance = &MemoryMonitor{
			logger: utility.GetLogger(),
			shell:  utility.NewShell(utility.GetLogger()),
		}
	})
	return memoryMonitorInstance
}

// GetSwappiness gets current swappiness value
func (mm *MemoryMonitor) GetSwappiness(ctx context.Context) (int, error) {
	result, err := mm.shell.Execute(ctx, "cat /proc/sys/vm/swappiness", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		mm.logger.Error("Failed to get swappiness: %v", err)
		return 0, err
	}

	swappiness, err := strconv.Atoi(strings.TrimSpace(result.Stdout))
	if err != nil {
		return 0, err
	}

	return swappiness, nil
}

// GetRecommendedSwappiness gets recommended swappiness value
func (mm *MemoryMonitor) GetRecommendedSwappiness() int {
	// For zram, recommended is 180
	// For regular swap, recommended is 60
	return optimalSwappinessZram
}

// GetMemoryStats gets memory statistics from /proc/meminfo
func (mm *MemoryMonitor) GetMemoryStats(ctx context.Context) (*MemoryStats, error) {
	result, err := mm.shell.Execute(ctx, "cat /proc/meminfo", &utility.ExecOptions{
		Timeout: 5 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}

	lines := strings.Split(result.Stdout, "\n")
	memInfo := make(map[string]int64)

	// Parse meminfo
	re := regexp.MustCompile(`^(\w+):\s+(\d+)`)
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			key := matches[1]
			value, err := strconv.ParseInt(matches[2], 10, 64)
			if err == nil {
				memInfo[key] = value * 1024 // Convert kB to bytes
			}
		}
	}

	totalBytes := memInfo["MemTotal"]
	freeBytes := memInfo["MemFree"]
	availableBytes := memInfo["MemAvailable"]
	buffersBytes := memInfo["Buffers"]
	cachedBytes := memInfo["Cached"]
	usedBytes := totalBytes - freeBytes - buffersBytes - cachedBytes

	swapTotalBytes := memInfo["SwapTotal"]
	swapFreeBytes := memInfo["SwapFree"]
	swapUsedBytes := swapTotalBytes - swapFreeBytes

	// Get zram stats
	zram, _ := mm.GetZramStats(ctx)

	percentUsed := float64(0)
	if totalBytes > 0 {
		percentUsed = (float64(usedBytes) / float64(totalBytes)) * 100
	}

	swapPercentUsed := float64(0)
	if swapTotalBytes > 0 {
		swapPercentUsed = (float64(swapUsedBytes) / float64(swapTotalBytes)) * 100
	}

	return &MemoryStats{
		TotalBytes:     totalBytes,
		UsedBytes:      usedBytes,
		FreeBytes:      freeBytes,
		AvailableBytes: availableBytes,
		BuffersBytes:   buffersBytes,
		CachedBytes:    cachedBytes,
		TotalGB:        float64(totalBytes) / 1024 / 1024 / 1024,
		UsedGB:         float64(usedBytes) / 1024 / 1024 / 1024,
		FreeGB:         float64(freeBytes) / 1024 / 1024 / 1024,
		AvailableGB:    float64(availableBytes) / 1024 / 1024 / 1024,
		PercentUsed:    percentUsed,
		Swap: SwapStats{
			TotalBytes:  swapTotalBytes,
			UsedBytes:   swapUsedBytes,
			FreeBytes:   swapFreeBytes,
			TotalGB:     float64(swapTotalBytes) / 1024 / 1024 / 1024,
			UsedGB:      float64(swapUsedBytes) / 1024 / 1024 / 1024,
			FreeGB:      float64(swapFreeBytes) / 1024 / 1024 / 1024,
			PercentUsed: swapPercentUsed,
		},
		Zram: zram,
	}, nil
}

// GetZramStats gets zram statistics if available
func (mm *MemoryMonitor) GetZramStats(ctx context.Context) (*ZramStats, error) {
	// Check if zram0 exists
	checkResult, err := mm.shell.Execute(ctx, "test -e /sys/block/zram0 && echo exists", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err != nil || checkResult.ExitCode != 0 || !strings.Contains(checkResult.Stdout, "exists") {
		return nil, nil
	}

	// Get zram stats from sysfs
	diskSizeResult, err := mm.shell.Execute(ctx, "cat /sys/block/zram0/disksize 2>/dev/null", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	memUsedResult, err := mm.shell.Execute(ctx, "cat /sys/block/zram0/mem_used_total 2>/dev/null", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	origDataSizeResult, err := mm.shell.Execute(ctx, "cat /sys/block/zram0/orig_data_size 2>/dev/null", &utility.ExecOptions{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	if diskSizeResult.ExitCode != 0 || memUsedResult.ExitCode != 0 || origDataSizeResult.ExitCode != 0 {
		return nil, nil
	}

	totalBytes, err := strconv.ParseInt(strings.TrimSpace(diskSizeResult.Stdout), 10, 64)
	if err != nil {
		return nil, err
	}

	compressedBytes, err := strconv.ParseInt(strings.TrimSpace(memUsedResult.Stdout), 10, 64)
	if err != nil {
		return nil, err
	}

	usedBytes, err := strconv.ParseInt(strings.TrimSpace(origDataSizeResult.Stdout), 10, 64)
	if err != nil {
		return nil, err
	}

	compressionRatio := float64(1)
	if compressedBytes > 0 {
		compressionRatio = float64(usedBytes) / float64(compressedBytes)
	}

	percentUsed := float64(0)
	if totalBytes > 0 {
		percentUsed = (float64(usedBytes) / float64(totalBytes)) * 100
	}

	return &ZramStats{
		Device:           "/dev/zram0",
		TotalBytes:       totalBytes,
		UsedBytes:        usedBytes,
		CompressedBytes:  compressedBytes,
		CompressionRatio: compressionRatio,
		PercentUsed:      percentUsed,
	}, nil
}

// CheckSwappiness checks if swappiness is optimal
func (mm *MemoryMonitor) CheckSwappiness(ctx context.Context) (map[string]interface{}, error) {
	current, err := mm.GetSwappiness(ctx)
	if err != nil {
		return map[string]interface{}{
			"current":     -1,
			"recommended": optimalSwappinessZram,
			"optimal":     false,
			"message":     "Unable to read swappiness value",
		}, nil
	}

	recommended := mm.GetRecommendedSwappiness()
	optimal := current == recommended

	var message string
	if optimal {
		message = fmt.Sprintf("Swappiness is optimal for zram (%d)", current)
	} else {
		message = fmt.Sprintf("Swappiness is %d, recommended %d for zram. Run: sudo sysctl vm.swappiness=%d", current, recommended, recommended)
	}

	return map[string]interface{}{
		"current":     current,
		"recommended": recommended,
		"optimal":     optimal,
		"message":     message,
	}, nil
}

// FormatMemoryStats formats memory stats for display
func (mm *MemoryMonitor) FormatMemoryStats(stats *MemoryStats) string {
	output := "=== Memory Statistics ===\n\n"

	output += fmt.Sprintf("Total: %.1f GB\n", stats.TotalGB)
	output += fmt.Sprintf("Used: %.1f GB (%.1f%%)\n", stats.UsedGB, stats.PercentUsed)
	output += fmt.Sprintf("Free: %.1f GB\n", stats.FreeGB)
	output += fmt.Sprintf("Available: %.1f GB\n", stats.AvailableGB)
	output += fmt.Sprintf("Buffers/Cache: %.1f GB\n\n", float64(stats.BuffersBytes+stats.CachedBytes)/1024/1024/1024)

	output += "Swap:\n"
	output += fmt.Sprintf("  Total: %.1f GB\n", stats.Swap.TotalGB)
	output += fmt.Sprintf("  Used: %.1f GB (%.1f%%)\n", stats.Swap.UsedGB, stats.Swap.PercentUsed)
	output += fmt.Sprintf("  Free: %.1f GB\n", stats.Swap.FreeGB)

	if stats.Zram != nil {
		output += fmt.Sprintf("\nzram (%s):\n", stats.Zram.Device)
		output += fmt.Sprintf("  Size: %.1f GB\n", float64(stats.Zram.TotalBytes)/1024/1024/1024)
		output += fmt.Sprintf("  Used (uncompressed): %.1f GB\n", float64(stats.Zram.UsedBytes)/1024/1024/1024)
		output += fmt.Sprintf("  Used (compressed): %.1f GB\n", float64(stats.Zram.CompressedBytes)/1024/1024/1024)
		output += fmt.Sprintf("  Compression Ratio: %.2fx\n", stats.Zram.CompressionRatio)
		output += fmt.Sprintf("  Usage: %.1f%%\n", stats.Zram.PercentUsed)
	}

	return output
}
