# Comprehensive System Optimization Plan for Daemira

**System**: CachyOS Linux (archDuke)
**Daemon**: Daemira v1.0
**Date**: 2025-12-11
**Status**: Phase 1 Complete (Audio ✓, Browser ✓)

---

## Executive Summary

This plan enhances the Daemira daemon to provide comprehensive system optimization, monitoring, and automation capabilities for a high-performance CachyOS workstation. With critical issues (audio, browser) already resolved, we focus on performance optimization, proactive monitoring, and intelligent automation.

**System Profile**:
- **CPU**: Intel Core i7-10700K (8C/16T @ 4.7GHz) - Currently at 87% scaling
- **GPU**: AMD Radeon RX 5700 XT (Mesa 25.3.1)
- **RAM**: 32GB DDR4 @ 3600 MT/s
- **Storage**: 931.5GB NVMe (53% used) + 4 unmounted HDDs (3.6TB, 3.6TB, 931.5GB, 1.8TB)
- **Audio**: PipeWire 1.4.9 + WirePlumber 0.5.12 ✓ Fixed
- **Browser**: Zen Browser 1.17.12b ✓ Fixed
- **WM**: Hyprland 0.52.2 with Quickshell DMS desktop

---

## Current Status & Architecture

### ✅ Resolved Issues
1. **Audio System** - Working (zombie WirePlumber fixed)
2. **Browser Performance** - Restored to normal
3. **Power Profiles** - Quickshell DMS interface available (`/usr/share/quickshell/dms`)

### 🔧 Deferred Issues
1. **OBS Virtual Camera** - Waiting for kernel update (v4l2loopback incompatible with 6.18)

### 📦 Existing Daemira Features
- **Google Drive Sync**: Bidirectional rclone bisync (30s interval, 60+ exclude patterns)
- **System Update**: 13-step Arch maintenance (6h interval)
- **Logging**: Rotating logs with environment-based filtering
- **Architecture**: Bun runtime, TypeScript, modular design

---

## Optimization Phases

## Phase 1: Performance Monitoring & Optimization (Priority: HIGH)

### 1.1 CPU Performance Manager

**Objective**: Integrate with power-profiles-daemon for intelligent CPU scaling

**Current State**:
- Power profiles managed by `power-profiles-daemon` (running)
- Current profile: **balanced** (detected via `powerprofilesctl`)
- Quickshell DMS provides UI controls
- Profiles available: performance, balanced, power-saver
- Uses `intel_pstate` CPU driver

**Implementation**:

Create `src/utility/PerformanceManager.ts`:
```typescript
interface PowerProfile {
  name: 'performance' | 'balanced' | 'power-saver';
  cpuDriver: string;
  platformDriver?: string;
  degraded: boolean;
}

class PerformanceManager {
  // Get current power profile
  async getCurrentProfile(): Promise<PowerProfile>

  // Set power profile
  async setProfile(profile: 'performance' | 'balanced' | 'power-saver'): Promise<void>

  // Auto-detect optimal profile based on workload
  async suggestProfile(): Promise<'performance' | 'balanced' | 'power-saver'>

  // Monitor CPU frequency and utilization
  async getCPUStats(): Promise<CPUStats>
}
```

**Integration with Daemira**:
- Add `system:performance <profile>` command
- Add `system:performance-auto` for intelligent switching
- Monitor CPU utilization and suggest profile changes
- Log profile changes for historical analysis

**Files**:
- New: `src/utility/PerformanceManager.ts`
- Modify: `src/Daemira.ts` (add performance management)
- Modify: `src/main.ts` (add CLI commands)

### 1.2 Memory Monitor & Optimizer

**Objective**: Track memory usage and optimize zram configuration

**Current State**:
- 32GB RAM: 14GB used, 18GB available
- 31GB zram swap: 0GB used (excellent)
- Need to check swappiness value

**Implementation**:

Create `src/utility/MemoryMonitor.ts`:
```typescript
interface MemoryStats {
  total: number;
  used: number;
  free: number;
  available: number;
  buffers: number;
  cached: number;
  swapTotal: number;
  swapUsed: number;
  swapFree: number;
  zramStats?: ZramStats;
}

class MemoryMonitor {
  // Read /proc/meminfo
  async getMemoryStats(): Promise<MemoryStats>

  // Read /sys/block/zram0/mm_stat
  async getZramStats(): Promise<ZramStats>

  // Check current swappiness
  async getSwappiness(): Promise<number>

  // Recommend optimal swappiness for zram
  recommendSwappiness(): number // Returns 180 for zram

  // Track memory trends
  async recordMemoryUsage(): Promise<void>
}
```

**Features**:
- Check swappiness value (optimal for zram: 180)
- Monitor memory pressure
- Detect memory leaks in long-running processes
- Track swap usage patterns

**Files**:
- New: `src/utility/MemoryMonitor.ts`
- Modify: `src/features/system-update/SystemUpdate.ts` (add swappiness check)

### 1.3 Disk I/O Optimization

**Objective**: Optimize NVMe performance and disk health

**Current State**:
- NVMe: ext4, 53% used (458GB/931GB)
- I/O scheduler: Unknown (should be `none` or `mq-deadline` for NVMe)
- TRIM status: Unknown
- 4 unmounted HDDs available

**Implementation**:

Enhance `src/features/system-update/SystemUpdate.ts`:
```typescript
// Add these steps to SystemUpdate workflow:

Step 14: Check and enable TRIM
- Run: fstrim -v /
- Verify TRIM support
- Add to weekly maintenance

Step 15: Check I/O scheduler
- Read: /sys/block/nvme0n1/queue/scheduler
- Verify optimal setting for NVMe
- Log if suboptimal

Step 16: SMART disk health check
- Run: smartctl -H /dev/nvme0n1
- Check all disks (including unmounted HDDs)
- Log health status and warnings
- Alert on SMART errors
```

**Features**:
- Weekly TRIM execution
- I/O scheduler verification
- SMART health monitoring
- Disk temperature tracking

**Files**:
- Modify: `src/features/system-update/SystemUpdate.ts`
- New: `src/utility/DiskMonitor.ts` (optional, for detailed monitoring)

### 1.4 GPU Monitor

**Objective**: Track AMD GPU performance and health

**Current State**:
- AMD RX 5700 XT with Mesa 25.3.1
- Vulkan and OpenCL support
- No active monitoring

**Implementation**:

Create `src/utility/GPUMonitor.ts`:
```typescript
interface GPUStats {
  temperature?: number;
  powerState?: string;
  clockSpeed?: number;
  memoryUsed?: number;
  memoryTotal?: number;
  utilization?: number;
}

class GPUMonitor {
  // Read from /sys/class/drm/card0/
  async getGPUStats(): Promise<GPUStats>

  // Check Vulkan functionality
  async verifyVulkan(): Promise<boolean>

  // Monitor GPU temperature
  async getTemperature(): Promise<number | null>
}
```

**Features**:
- GPU temperature monitoring
- Power state tracking
- Memory usage
- Integration with system status

**Files**:
- New: `src/utility/GPUMonitor.ts`
- Modify: `src/Daemira.ts` (add GPU monitoring to status)

---

## Phase 2: System Health Monitoring (Priority: HIGH)

### 2.1 Comprehensive Health Checks

**Objective**: Auto-detect and fix common system issues

**Implementation**:

Create `src/features/system-monitor/SystemMonitor.ts`:
```typescript
interface SystemHealth {
  overall: 'healthy' | 'degraded' | 'critical';
  checks: {
    audio: HealthCheck;
    cpu: HealthCheck;
    memory: HealthCheck;
    disk: HealthCheck;
    gpu: HealthCheck;
    services: HealthCheck;
  };
  issues: SystemIssue[];
  recommendations: string[];
}

class SystemMonitor {
  // Run all health checks
  async checkSystemHealth(): Promise<SystemHealth>

  // Auto-fix known issues
  async autoFix(issue: SystemIssue): Promise<boolean>

  // Monitor zombie processes
  async detectZombieProcesses(): Promise<Process[]>

  // Check service status
  async checkCriticalServices(): Promise<ServiceStatus[]>
}
```

**Health Checks**:
1. **Audio System**: Verify PipeWire/WirePlumber, detect zombies
2. **CPU**: Check frequency scaling, temperature
3. **Memory**: Check usage, swap, pressure
4. **Disk**: Check space, SMART, I/O errors
5. **GPU**: Check temperature, driver status
6. **Services**: Check systemd service failures

**Auto-Recovery**:
- Audio zombie process cleanup
- Service restarts
- Cache cleanup
- Log rotation

**Files**:
- New: `src/features/system-monitor/SystemMonitor.ts`
- New: `src/features/system-monitor/HealthCheck.ts`
- New: `src/features/system-monitor/index.ts`

### 2.2 Proactive Issue Detection

**Implementation**:

Create `src/features/system-monitor/IssueDetector.ts`:
```typescript
class IssueDetector {
  // Detect performance anomalies
  async detectAnomalies(): Promise<Anomaly[]>

  // Check for common post-update issues
  async checkPostUpdateHealth(): Promise<Issue[]>

  // Monitor for kernel module issues
  async checkKernelModules(): Promise<ModuleIssue[]>

  // Detect package conflicts
  async detectPackageIssues(): Promise<PackageIssue[]>
}
```

**Features**:
- Performance regression detection
- Post-update verification
- Kernel module incompatibility detection (like v4l2loopback)
- Package conflict detection

**Integration**:
- Run after SystemUpdate completes
- Periodic checks (every 6 hours)
- Alert on critical issues
- Log all detections

**Files**:
- New: `src/features/system-monitor/IssueDetector.ts`

---

## Phase 3: Enhanced Automation (Priority: MEDIUM)

### 3.1 Intelligent System Update Enhancement

**Objective**: Add optimization steps to existing SystemUpdate

**New Steps to Add**:

```typescript
// Add to SystemUpdate.ts after existing 13 steps:

Step 14: CPU Performance Check
- Verify power profile configuration
- Check CPU scaling governor
- Log current profile

Step 15: Memory Optimization Check
- Verify swappiness setting (180 for zram)
- Check for memory pressure
- Log memory stats

Step 16: Disk Optimization
- Run TRIM on SSD
- Check I/O scheduler
- Verify disk health (SMART)

Step 17: GPU Driver Verification
- Check Mesa version
- Verify Vulkan support
- Check for driver issues

Step 18: Audio System Health
- Verify PipeWire/WirePlumber status
- Check for zombie processes
- Test audio connection

Step 19: Kernel Module Audit
- Check for incompatible modules (v4l2loopback)
- Verify DKMS module status
- Rebuild if needed

Step 20: Post-Update Verification
- Run system health check
- Verify critical functionality
- Log any issues detected
```

**Features**:
- All steps optional (controlled by config)
- Detailed logging
- Auto-recovery on common issues
- Health report generation

**Files**:
- Modify: `src/features/system-update/SystemUpdate.ts`

### 3.2 Performance Profiling

**Objective**: Track performance trends over time

**Implementation**:

Create `src/features/performance-profiler/PerformanceProfiler.ts`:
```typescript
interface PerformanceSnapshot {
  timestamp: number;
  cpu: CPUStats;
  memory: MemoryStats;
  disk: DiskStats;
  gpu: GPUStats;
  powerProfile: string;
  uptime: number;
}

class PerformanceProfiler {
  // Take performance snapshot
  async snapshot(): Promise<PerformanceSnapshot>

  // Store historical data
  async recordSnapshot(snapshot: PerformanceSnapshot): Promise<void>

  // Analyze trends
  async analyzeTrends(hours: number): Promise<PerformanceTrends>

  // Detect regressions
  async detectRegressions(): Promise<Regression[]>

  // Generate performance report
  async generateReport(): Promise<PerformanceReport>
}
```

**Features**:
- Snapshot every 5 minutes
- Store last 7 days of data
- Trend analysis
- Performance regression detection
- Report generation

**Files**:
- New: `src/features/performance-profiler/PerformanceProfiler.ts`
- New: `src/features/performance-profiler/index.ts`

---

## Phase 4: Storage Management (Priority: LOW)

### 4.1 Unmounted Disk Strategy

**4 Unmounted HDDs**: sda (3.6TB), sdb (3.6TB), sdc (931.5GB), sdd (1.8TB)

**Recommended Usage**:
1. **sda (3.6TB)**: Media library and archives
2. **sdb (3.6TB)**: Time Machine-style backups (rsync snapshots)
3. **sdc (931.5GB)**: Google Drive local cache/mirror
4. **sdd (1.8TB)**: Development projects and build caches

**Implementation**:

Create `src/features/storage-manager/StorageManager.ts`:
```typescript
interface DiskInfo {
  device: string;
  size: number;
  filesystem?: string;
  mountPoint?: string;
  label?: string;
  usage?: DiskUsage;
}

class StorageManager {
  // Detect unmounted disks
  async detectUnmountedDisks(): Promise<DiskInfo[]>

  // Mount disk
  async mountDisk(device: string, mountPoint: string): Promise<void>

  // Setup fstab entry
  async addToFstab(device: string, mountPoint: string, options: string[]): Promise<void>

  // Monitor disk health
  async monitorDiskHealth(device: string): Promise<SmartStatus>
}
```

**Mount Strategy**:
- Create mount points: `/mnt/media`, `/mnt/backups`, `/mnt/gdrive-cache`, `/mnt/dev-projects`
- Use `noatime` option for performance
- Auto-mount on boot (fstab entries)
- Monitor disk health

**Files**:
- New: `src/features/storage-manager/StorageManager.ts`
- New: `src/features/storage-manager/index.ts`
- New: `scripts/setup-storage.sh` (interactive setup script)

### 4.2 Backup Automation

**Strategy**:
1. **System Snapshots** (sdb): Weekly rsync snapshots of `/home`, `/etc`
2. **Google Drive Cache** (sdc): Real-time cache for offline access
3. **Development Backups** (sdd): Daily incremental backups
4. **Media Archive** (sda): Long-term storage

**Implementation**:

Create `src/features/storage-manager/BackupManager.ts`:
```typescript
class BackupManager {
  // Create rsync snapshot
  async createSnapshot(source: string, destination: string): Promise<void>

  // Rotate old snapshots
  async rotateSnapshots(baseDir: string, keepCount: number): Promise<void>

  // Verify backup integrity
  async verifyBackup(backupPath: string): Promise<BackupStatus>

  // Schedule automatic backups
  async scheduleBackups(): Promise<void>
}
```

**Features**:
- Hardlink-based snapshots (space-efficient)
- Configurable retention (keep last 30 daily, 12 weekly)
- Backup verification
- Email/notification on failure

**Files**:
- New: `src/features/storage-manager/BackupManager.ts`

---

## Phase 5: Integration & Polish (Priority: LOW)

### 5.1 Unified CLI Interface

**New Commands**:
```bash
# Performance Management
daemira system:performance <performance|balanced|power-saver>
daemira system:performance-auto
daemira system:cpu-stats
daemira system:memory-stats
daemira system:gpu-stats

# Health Monitoring
daemira health:check
daemira health:auto-fix
daemira health:report

# Performance Profiling
daemira perf:snapshot
daemira perf:report [hours]
daemira perf:trends

# Storage Management
daemira storage:list
daemira storage:mount <device> <mountpoint>
daemira storage:health

# Backup Management
daemira backup:create
daemira backup:list
daemira backup:verify
```

**Files**:
- Modify: `src/main.ts` (add all new commands)
- Modify: `src/Daemira.ts` (add method routing)

### 5.2 Comprehensive Status Dashboard

**Enhanced `defaultFunction()`**:
```typescript
async defaultFunction() {
  // System Overview
  - Hostname, uptime, kernel version
  - Power profile (performance/balanced/power-saver)

  // Resource Usage
  - CPU: utilization %, temperature, frequency
  - Memory: used/total, swap usage, zram stats
  - Disk: usage %, I/O rate, SMART health
  - GPU: temperature, utilization, memory

  // Service Status
  - Google Drive: sync status, last sync, errors
  - System Update: last run, next scheduled, status
  - Audio: PipeWire/WirePlumber status

  // Health Status
  - Overall health: healthy/degraded/critical
  - Active issues: count, severity
  - Recent fixes: auto-recovery actions

  // Performance
  - Current power profile
  - Performance trends (last 24h)
  - Recommendations
}
```

**Files**:
- Modify: `src/Daemira.ts` (enhance defaultFunction)

---

## Configuration Updates

### New Environment Variables

Add to `.env`:
```bash
# Performance Management
POWER_PROFILE_AUTO_SWITCH=true
POWER_PROFILE_AUTO_INTERVAL=300000  # 5 minutes

# Memory Monitoring
MEMORY_SNAPSHOT_INTERVAL=300000  # 5 minutes
SWAPPINESS_TARGET=180  # Optimal for zram

# Disk Management
TRIM_INTERVAL=604800000  # Weekly (7 days)
SMART_CHECK_INTERVAL=86400000  # Daily

# Health Monitoring
HEALTH_CHECK_INTERVAL=21600000  # 6 hours
AUTO_FIX_ENABLED=true
ZOMBIE_PROCESS_CHECK=true

# Performance Profiling
PERF_SNAPSHOT_INTERVAL=300000  # 5 minutes
PERF_RETENTION_DAYS=7

# Storage Management
AUTO_MOUNT_DISKS=false  # Manual for safety
BACKUP_ENABLED=false  # Opt-in
```

### Config Schema Update

Modify `src/config/index.ts`:
```typescript
const configSchema = z.object({
  // ... existing config ...

  // New performance config
  powerProfileAutoSwitch: z.boolean().default(true),
  powerProfileAutoInterval: z.number().default(300000),

  // New monitoring config
  healthCheckInterval: z.number().default(21600000),
  autoFixEnabled: z.boolean().default(true),

  // New profiling config
  perfSnapshotInterval: z.number().default(300000),
  perfRetentionDays: z.number().default(7),
});
```

---

## Implementation Priority

### Immediate (Week 1)
1. ✅ ~~Fix Audio System~~ (COMPLETE)
2. ✅ ~~Fix Browser Performance~~ (COMPLETE)
3. **CPU Performance Manager** (1.1) - Integrate with power-profiles-daemon
4. **Memory Monitor** (1.2) - Check swappiness, track usage
5. **System Health Monitoring** (2.1) - Core health checks

### Short-term (Week 2-3)
6. **Disk I/O Optimization** (1.3) - TRIM, SMART, I/O scheduler
7. **GPU Monitor** (1.4) - Temperature, stats
8. **Enhanced System Update** (3.1) - Add new optimization steps
9. **Issue Detector** (2.2) - Proactive detection

### Medium-term (Week 4-6)
10. **Performance Profiling** (3.2) - Trend analysis
11. **Unified CLI Interface** (5.1) - All new commands
12. **Enhanced Status Dashboard** (5.2) - Comprehensive view

### Long-term (As Needed)
13. **Storage Management** (4.1) - Mount and manage HDDs
14. **Backup Automation** (4.2) - Automated backups

---

## Testing Strategy

### Unit Testing
- Test each utility class independently
- Mock Shell execution for testing
- Verify error handling
- Test configuration validation

### Integration Testing
- Test Daemira command flow
- Verify service auto-start
- Test error recovery
- Validate logging

### Performance Testing
- Benchmark CPU overhead of monitoring
- Test memory footprint
- Verify no performance regressions
- Load testing for long-running operations

### System Testing
- Test on actual system after updates
- Verify health checks detect real issues
- Test auto-recovery mechanisms
- Validate backup integrity

---

## Rollback & Safety

### Safety Measures
1. **Never modify system files directly** - Use well-tested utilities
2. **Backup before changes** - Especially for fstab, systemd configs
3. **Dry-run mode** - Test commands before execution
4. **Logging** - All actions logged for audit
5. **User confirmation** - Critical actions require confirmation

### Rollback Strategy
1. **Keep previous kernel** - Bootable fallback
2. **Configuration backups** - Before any modifications
3. **Service snapshots** - systemd service files backed up
4. **Package cache** - Keep package cache for downgrades

---

## Success Metrics

### Performance
- ✅ CPU frequency scaling optimized
- ✅ Memory usage < 60% under normal load
- ✅ Disk SMART health: PASSED
- ✅ GPU temperature < 80°C under load
- ✅ System uptime > 30 days without issues

### Reliability
- ✅ Zero audio failures per week
- ✅ Auto-recovery success rate > 95%
- ✅ System health checks pass > 99%
- ✅ Zero kernel panics

### Automation
- ✅ System updates automated and successful
- ✅ Health checks run every 6 hours
- ✅ Performance snapshots every 5 minutes
- ✅ Backups complete successfully (when enabled)

---

## Maintenance Schedule

### Automated (by Daemira)

**Every 5 minutes**:
- Performance snapshot
- Memory/CPU usage check

**Every 6 hours**:
- Full system health check
- System update (check and execute if needed)
- Issue detection

**Daily**:
- SMART disk health check
- Log rotation
- Zombie process check

**Weekly**:
- TRIM operation on SSD
- Performance report generation
- Backup verification (if enabled)

### Manual (User-initiated)

**After System Updates**:
- Verify health: `daemira health:check`
- Check performance: `daemira perf:report`

**Monthly**:
- Review performance trends
- Audit system configuration
- Check backup integrity

---

## Notes & Considerations

### Power Profile Integration
- Quickshell DMS already provides UI for power profiles
- Daemira integration provides CLI and automation
- Both can coexist (shared backend: power-profiles-daemon)
- Auto-switching should respect manual user changes

### Monitoring Overhead
- Keep monitoring lightweight (< 1% CPU usage)
- Use efficient polling intervals
- Minimize disk I/O for snapshots
- Consider memory footprint for long-running daemon

### OBS Virtual Camera
- **Status**: Deferred until kernel update
- **Reason**: v4l2loopback incompatible with kernel 6.18
- **Workaround**: Module blacklisted to prevent crashes
- **Future**: Monitor for v4l2loopback or kernel updates

### Future Enhancements
1. **Machine Learning**: Predict optimal power profile based on workload
2. **Network Monitoring**: Track bandwidth, latency, connections
3. **Process Profiling**: Identify resource-hungry processes
4. **Thermal Management**: Fan curve optimization
5. **Notification System**: Desktop notifications for issues
6. **Web Dashboard**: Real-time monitoring via web interface

---

## Appendix

### Useful Commands

**Power Management**:
```bash
powerprofilesctl                    # Show current profile
powerprofilesctl set performance    # Set performance mode
powerprofilesctl set balanced       # Set balanced mode
powerprofilesctl set power-saver    # Set power-saver mode
```

**CPU Monitoring**:
```bash
watch -n1 "grep MHz /proc/cpuinfo"  # Watch CPU frequency
lscpu                                # CPU information
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
```

**Memory**:
```bash
free -h                              # Memory usage
cat /proc/meminfo                    # Detailed memory info
cat /proc/sys/vm/swappiness          # Current swappiness
```

**Disk**:
```bash
fstrim -v /                          # Run TRIM
smartctl -H /dev/nvme0n1            # SMART health
smartctl -a /dev/nvme0n1            # Full SMART data
iostat -x 1                          # I/O stats
cat /sys/block/nvme0n1/queue/scheduler  # I/O scheduler
```

**GPU**:
```bash
lspci | grep -i vga                  # GPU info
glxinfo | grep -i "opengl"          # OpenGL info
vulkaninfo | head -20                # Vulkan info
cat /sys/class/drm/card0/device/gpu_busy_percent  # GPU usage (if available)
```

### Reference Links

- **CachyOS Wiki**: https://wiki.cachyos.org/
- **Arch Wiki - Performance**: https://wiki.archlinux.org/title/Performance
- **power-profiles-daemon**: https://gitlab.freedesktop.org/hadess/power-profiles-daemon
- **Quickshell DMS**: /usr/share/quickshell/dms/
- **Daemira Source**: /home/ln64/Source/daemira/

---

**End of Plan**

This plan provides a roadmap for transforming Daemira into a comprehensive system optimization and monitoring daemon while respecting the existing architecture and user preferences.
