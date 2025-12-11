# System Optimization Plan - Quick Summary

**System**: CachyOS Linux (archDuke) | **User**: ln64 | **Date**: 2025-12-11

---

## 🎯 Goals

Transform Daemira into a comprehensive system optimization and monitoring daemon while respecting existing architecture and user setup.

---

## ✅ Current Status

### Resolved
- ✅ **Audio System** - Working (zombie WirePlumber fixed)
- ✅ **Browser Performance** - Zen Browser restored
- ✅ **Power Management** - QuickShell DMS handles UI via power-profiles-daemon

### Deferred
- ⏸️ **OBS Virtual Camera** - Waiting for kernel update (v4l2loopback crashes on 6.18)

### Critical Alert
- ⚠️ **`/mnt/Media` is 97% full** - Only 127GB free of 3.7TB! Needs cleanup

---

## 📦 Storage Configuration (Actual)

| Mount | Device | Size | Used | Free | Usage | Purpose |
|-------|--------|------|------|------|-------|---------|
| `/mnt/Media` | sda2 (exfat) | 3.7TB | 3.6TB | **127GB** | **97%** ⚠️ | Media library |
| `/mnt/Steam` | sdb1 (ext4) | 3.6TB | 2.6TB | 852GB | 72% | Steam games |
| `/mnt/Roms` | sdd2 (exfat) | 1.9TB | 963GB | 901GB | 52% | ROM storage |
| `/` | nvme0n1p2 (ext4) | 931GB | 458GB | 411GB | 53% | System root |
| ⛔ *unmounted* | **sdc** (ntfs) | 931GB | - | - | - | **Windows partition - DO NOT MOUNT** |

**Issues**:
1. ⚠️ No fstab entries - Disks not auto-mounted on boot
2. ⚠️ Media drive critically low on space
3. ⛔ sdc must stay unmounted (Windows dual-boot)

---

## 🚀 Implementation Phases

### Phase 1: Performance Monitoring (HIGH PRIORITY)

**1.1 CPU Performance Manager**
- Integrate with existing `power-profiles-daemon`
- Commands: `daemira system:performance <performance|balanced|power-saver>`
- Auto-detect optimal profile based on workload
- **Files**: `src/utility/PerformanceManager.ts`, updates to `Daemira.ts` and `main.ts`

**1.2 Memory Monitor**
- Track 32GB RAM usage and 31GB zram swap
- Check swappiness setting (optimal: 180 for zram)
- Memory pressure detection
- **Files**: `src/utility/MemoryMonitor.ts`

**1.3 Disk I/O Optimization**
- Enable weekly TRIM on NVMe SSD
- Verify I/O scheduler (should be `none` or `mq-deadline` for NVMe)
- SMART health monitoring for all disks
- **⚠️ CRITICAL: Disk space monitoring for Media drive**
- **Files**: Enhance `SystemUpdate.ts`

**1.4 GPU Monitor**
- Track AMD RX 5700 XT temperature and usage
- Verify Vulkan/OpenGL functionality
- **Files**: `src/utility/GPUMonitor.ts`

### Phase 2: System Health Monitoring (HIGH PRIORITY)

**2.1 Comprehensive Health Checks**
- Monitor: CPU, memory, disk, GPU, audio, services
- Auto-detect zombie processes (like WirePlumber issue)
- Auto-recovery for common issues
- **Files**: `src/features/system-monitor/SystemMonitor.ts`, `HealthCheck.ts`

**2.2 Proactive Issue Detection**
- Performance anomaly detection
- Post-update health verification
- Kernel module incompatibility detection
- **Files**: `src/features/system-monitor/IssueDetector.ts`

### Phase 3: Enhanced Automation (MEDIUM PRIORITY)

**3.1 Enhanced System Update**
- Add 7 new optimization steps to existing 13-step workflow:
  - Step 14: CPU performance check
  - Step 15: Memory swappiness tuning
  - Step 16: TRIM + I/O scheduler verification
  - Step 17: GPU driver check
  - Step 18: Audio system health
  - Step 19: Kernel module audit
  - Step 20: Post-update verification
- **Files**: Enhance `SystemUpdate.ts`

**3.2 Performance Profiling**
- Snapshot every 5 minutes
- Track trends over 7 days
- Detect performance regressions
- Generate reports
- **Files**: `src/features/performance-profiler/PerformanceProfiler.ts`

### Phase 4: Storage Management (MEDIUM PRIORITY)

**4.1 Disk Space Monitoring** (⚠️ URGENT)
- **Critical**: Alert when `/mnt/Media` < 100GB free
- Monitor all disks for low space
- Track usage trends
- **Files**: `src/features/storage-manager/DiskMonitor.ts`

**4.2 fstab Auto-Mount**
- Generate fstab entries for current mounts (sda2, sdb1, sdd2)
- Detect missing fstab entries
- ⛔ **Safety**: Never touch sdc (Windows partition)
- **Files**: `src/features/storage-manager/FstabManager.ts`

---

## 🛠️ New Daemira Commands

```bash
# Performance Management
daemira system:performance <performance|balanced|power-saver>
daemira system:performance-auto        # Auto-detect and set optimal profile
daemira system:cpu-stats               # CPU frequency, utilization, temperature
daemira system:memory-stats            # RAM/zram detailed statistics
daemira system:gpu-stats               # GPU temperature, usage

# Health Monitoring
daemira health:check                   # Full system health scan
daemira health:auto-fix                # Auto-fix detected issues
daemira health:report                  # Detailed health report

# Performance Profiling
daemira perf:snapshot                  # Take performance snapshot
daemira perf:report [hours]            # Generate performance report
daemira perf:trends                    # Show performance trends

# Storage Management
daemira storage:status                 # Show all disk usage
daemira storage:check                  # Check for low space warnings
daemira storage:health                 # SMART status for all disks
daemira storage:check-fstab            # Detect missing fstab entries
daemira storage:suggest-fstab          # Show suggested fstab entries
```

---

## 📊 Key Metrics & Thresholds

### Disk Space Alerts
| Level | Condition | Action |
|-------|-----------|--------|
| 🔴 Critical | Free < 100GB OR Used > 95% | Alert immediately |
| 🟡 Warning | Free < 200GB OR Used > 90% | Alert, suggest cleanup |
| 🟢 Healthy | Free > 200GB AND Used < 90% | Normal |

**Current Alerts**:
- 🔴 `/mnt/Media`: 127GB free, 97% used - **CRITICAL**

### Memory Thresholds
- Swappiness: Check if set to 180 (optimal for zram)
- Memory pressure: Alert if > 90% used

### CPU Monitoring
- Track frequency scaling
- Detect thermal throttling
- Monitor power profile changes

---

## 🔧 Suggested fstab Entries

Add to `/etc/fstab` for auto-mount on boot:

```fstab
# Media drive (exfat) - CRITICAL: 97% full, needs cleanup!
UUID=7636-7AF6    /mnt/Media    exfat    defaults,noatime,uid=1000,gid=1000    0 2

# Steam library (ext4)
UUID=4e74be0a-1fe9-4fe0-b781-2df2a15044e0    /mnt/Steam    ext4    defaults,noatime    0 2

# ROMs storage (exfat)
UUID=1C52-A83D    /mnt/Roms    exfat    defaults,noatime,uid=1000,gid=1000    0 2

# ⛔ DO NOT ADD sdc - Windows partition must stay unmounted!
```

**To apply**:
```bash
sudo cp /etc/fstab /etc/fstab.backup    # Backup first!
sudo nano /etc/fstab                     # Add entries above
sudo mount -a                            # Test without reboot
df -h | grep /mnt                        # Verify mounts
```

---

## 🎬 Recommended Implementation Order

### Week 1: Critical Fixes
1. ⚠️ **Disk Space Monitoring** (Phase 4.1) - Media drive critical!
2. **CPU Performance Manager** (Phase 1.1) - Quick win
3. **Memory Monitor** (Phase 1.2) - Check swappiness

### Week 2: Core Monitoring
4. **System Health Monitoring** (Phase 2.1) - Foundation
5. **Disk I/O Optimization** (Phase 1.3) - TRIM, SMART
6. **GPU Monitor** (Phase 1.4) - Complete the monitoring suite

### Week 3: Automation
7. **Enhanced System Update** (Phase 3.1) - Add new steps
8. **Issue Detector** (Phase 2.2) - Proactive detection
9. **fstab Manager** (Phase 4.2) - Auto-mount setup

### Week 4+: Advanced Features
10. **Performance Profiling** (Phase 3.2) - Long-term tracking
11. **Unified CLI** - All new commands
12. **Status Dashboard** - Comprehensive view

---

## 🔒 Safety Measures

1. **Never touch sdc** - Windows partition protection
2. **Backup before changes** - Especially fstab, systemd configs
3. **Dry-run mode** - Test before execution
4. **Comprehensive logging** - All actions logged
5. **User confirmation** - Critical actions require approval

### Protected Disks
```typescript
const PROTECTED_DISKS = ['sdc']; // Windows partition - NEVER MOUNT OR WRITE
```

---

## 📈 Expected Outcomes

### Immediate (Week 1-2)
- ✅ Disk space alerts prevent data loss
- ✅ CPU running at optimal performance profile
- ✅ Memory usage optimized
- ✅ fstab auto-mount configured

### Medium-term (Week 3-4)
- ✅ Comprehensive health monitoring
- ✅ Auto-recovery for common issues
- ✅ Performance profiling active
- ✅ SMART health checks automated

### Long-term (Month 2+)
- ✅ Self-healing system
- ✅ Predictive maintenance
- ✅ Performance trend analysis
- ✅ Optimized resource utilization

---

## 📝 Next Steps

**Ready to proceed?** Choose a phase to start:

1. **Start with critical** - Disk space monitoring (⚠️ Media drive at 97%)
2. **Performance first** - CPU/Memory/GPU monitoring
3. **Full implementation** - All phases in order

Let me know which approach you prefer, and I'll begin implementation!

---

## 📚 Documentation

- **Full Plan**: [SYSTEM_OPTIMIZATION_PLAN.md](SYSTEM_OPTIMIZATION_PLAN.md)
- **Storage Details**: [STORAGE_STATUS.md](STORAGE_STATUS.md)
- **System Config**: [system-config/README.md](system-config/README.md)

---

**Last Updated**: 2025-12-11
