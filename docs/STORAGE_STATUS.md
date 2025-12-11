# Storage Configuration & Status

**Last Updated**: 2025-12-11

## Current Disk Configuration

### Mounted Disks

| Device | Size | Filesystem | Mount Point | Used | Free | Usage % | Label | Status |
|--------|------|------------|-------------|------|------|---------|-------|--------|
| **sda2** | 3.7TB | exfat | `/mnt/Media` | 3.6TB | **127GB** | **97%** ⚠️ | Media | **CRITICAL - Nearly Full!** |
| **sdb1** | 3.6TB | ext4 | `/mnt/Steam` | 2.6TB | 852GB | 72% | Games | Steam game library |
| **sdd2** | 1.9TB | exfat | `/mnt/Roms` | 963GB | 901GB | 52% | ROMs | ROM storage |

### Unmounted Disks

| Device | Size | Partitions | Status | Notes |
|--------|------|------------|--------|-------|
| **sdc** | 931.5GB | sdc1 (unused), sdc2 (ntfs), sdc3 (ntfs) | **DO NOT MOUNT** ⛔ | **Windows partition - Keep unmounted for safety** |

### Root/Boot Partitions

| Device | Size | Filesystem | Mount Point | Used | Free | Usage % |
|--------|------|------------|-------------|------|------|---------|
| nvme0n1p2 | 931.2GB | ext4 | `/` | 458GB | 411GB | 53% |
| nvme0n1p1 | 300MB | vfat | `/boot/efi` | - | - | - |

---

## ⚠️ Critical Issue: Media Drive Nearly Full

**Problem**: `/mnt/Media` (sda2) is at 97% capacity with only 127GB free

**Impact**:
- May run out of space soon
- Could cause application failures
- Potential data loss if writes fail
- Performance degradation when > 95% full

**Immediate Actions Needed**:
1. ✅ Add disk space monitoring to Daemira
2. ✅ Alert when free space < 100GB
3. 🔲 Review and clean up old/unused media files
4. 🔲 Consider moving some content to sdc or external storage

---

## Missing: fstab Auto-Mount Configuration

**Current State**: Disks are mounted manually (not in `/etc/fstab`)

**Problem**: Disks will NOT auto-mount after reboot

**Suggested fstab Entries**:

```fstab
# Media drive (exfat)
UUID=7636-7AF6    /mnt/Media    exfat    defaults,noatime,uid=1000,gid=1000    0 2

# Steam library (ext4)
UUID=4e74be0a-1fe9-4fe0-b781-2df2a15044e0    /mnt/Steam    ext4    defaults,noatime    0 2

# ROMs storage (exfat)
UUID=1C52-A83D    /mnt/Roms    exfat    defaults,noatime,uid=1000,gid=1000    0 2
```

**To Apply**:
```bash
# Backup current fstab
sudo cp /etc/fstab /etc/fstab.backup

# Add entries to fstab (manually edit or use script)
sudo nano /etc/fstab

# Test mounts without rebooting
sudo mount -a

# Verify all mounts successful
df -h | grep /mnt
```

---

## ⛔ Disk sdc: Windows Partition - DO NOT MOUNT

**Status**: **OFF-LIMITS** - Windows dual-boot partition

**Device**: sdc (931.5GB)
**Partitions**: sdc1, sdc2 (ntfs), sdc3 (ntfs)
**Purpose**: Windows operating system
**Action**: **Keep unmounted to avoid accidental writes or corruption**

**Safety Measures**:
- Daemira StorageManager must NEVER mount sdc
- Exclude sdc from automatic disk operations
- Add safety check to prevent accidental mounting
- Document clearly in all storage management code

**Code Implementation**:
```typescript
// Add to StorageManager safety checks
const PROTECTED_DISKS = ['sdc']; // Windows partition - never touch

async mountDisk(device: string) {
  if (PROTECTED_DISKS.some(d => device.includes(d))) {
    throw new Error(`Disk ${device} is protected (Windows partition)`);
  }
  // ... proceed with mount
}
```

---

## Daemira Integration Plan

### Phase 1: Disk Space Monitoring (HIGH PRIORITY)

**Implementation**:
```typescript
// src/features/storage-manager/DiskMonitor.ts

interface DiskStatus {
  device: string;
  mountPoint: string;
  total: number;
  used: number;
  free: number;
  percentUsed: number;
  filesystem: string;
  status: 'healthy' | 'warning' | 'critical';
}

class DiskMonitor {
  // Get status for all disks
  async getAllDiskStatus(): Promise<DiskStatus[]>

  // Check for low disk space
  async checkLowSpace(): Promise<DiskWarning[]>

  // Thresholds:
  // - Critical: < 100GB free OR > 95% used
  // - Warning: < 200GB free OR > 90% used
  // - Healthy: > 200GB free AND < 90% used
}
```

**Daemira Commands**:
```bash
daemira storage:status          # Show all disk usage
daemira storage:check           # Check for low space warnings
daemira storage:health          # SMART health for all disks
```

**Health Check Integration**:
- Add disk space check to `SystemMonitor.checkSystemHealth()`
- Alert on critical disk space (< 100GB or > 95%)
- Log disk usage trends

### Phase 2: fstab Management (MEDIUM PRIORITY)

**Implementation**:
```typescript
// src/features/storage-manager/FstabManager.ts

class FstabManager {
  // Detect mounts not in fstab
  async detectMissingFstabEntries(): Promise<MountEntry[]>

  // Generate fstab entry for a mount
  async generateFstabEntry(device: string, mountPoint: string): Promise<string>

  // Suggest adding entries (dry-run)
  async suggestFstabEntries(): Promise<string[]>
}
```

**Daemira Commands**:
```bash
daemira storage:check-fstab     # Check for missing fstab entries
daemira storage:suggest-fstab   # Show suggested fstab entries
```

### Phase 3: sdc Setup & Google Drive Cache (LOW PRIORITY)

**Enhancement to GoogleDrive Utility**:
- Option to use local cache directory (sdc)
- Faster file access from cache
- Reduce Google API calls

---

## Disk Health Monitoring

### SMART Status Commands

```bash
# Check all disks
sudo smartctl -H /dev/sda    # Should be: PASSED
sudo smartctl -H /dev/sdb
sudo smartctl -H /dev/sdc
sudo smartctl -H /dev/sdd
sudo smartctl -H /dev/nvme0n1

# Detailed info
sudo smartctl -a /dev/sda
```

### Add to Daemira SystemUpdate

**New Steps**:
- Step 16: Check SMART health for all disks
- Step 17: Alert on SMART warnings or errors
- Step 18: Log disk temperatures and stats

---

## Filesystem Optimization

### exfat Considerations (sda2, sdd2)

**Pros**:
- Cross-platform (Linux, Windows, macOS)
- No file size limits
- Good for media files

**Cons**:
- No journaling (risk of corruption on power loss)
- No native Linux permissions (uses mount options)
- Slower than ext4 for small files

**Recommendations**:
- Keep for compatibility
- Use `noatime` mount option (already doing this)
- Regular backups important (no journaling)

### ext4 Considerations (sdb1)

**Pros**:
- Journaling (safer)
- Better performance
- Native Linux filesystem

**Cons**:
- Not readable by Windows (without drivers)

**Current**: Optimal for Steam library (sdb1)

---

## Action Items

### Immediate (Do Now)
1. ✅ Add disk space monitoring to Daemira health checks
2. ✅ Create `DiskMonitor` utility class
3. ⚠️ Clean up `/mnt/Media` to free space (manual)

### Short-term (This Week)
4. 🔲 Add fstab entries for auto-mount
5. 🔲 Implement fstab checker in Daemira
6. 🔲 Add SMART health checks to SystemUpdate

### Long-term (As Needed)
7. 🔲 Setup sdc for Google Drive cache or backups
8. 🔲 Implement disk usage trend analysis
9. 🔲 Add automated cleanup suggestions

---

## Monitoring Thresholds

### Disk Space Alerts

| Level | Condition | Action |
|-------|-----------|--------|
| **Critical** | Free < 100GB OR Used > 95% | Alert immediately, block operations |
| **Warning** | Free < 200GB OR Used > 90% | Alert, suggest cleanup |
| **Low** | Free < 500GB OR Used > 85% | Log, monitor closely |
| **Healthy** | Free > 500GB AND Used < 85% | Normal operation |

### Current Status

| Disk | Free Space | Usage | Alert Level |
|------|------------|-------|-------------|
| `/mnt/Media` (sda2) | 127GB | 97% | 🔴 **CRITICAL** |
| `/mnt/Steam` (sdb1) | 852GB | 72% | 🟢 Healthy |
| `/mnt/Roms` (sdd2) | 901GB | 52% | 🟢 Healthy |
| `/` (nvme0n1p2) | 411GB | 53% | 🟢 Healthy |

---

## Notes

- All disk UUIDs verified via `lsblk -f`
- fstab entries use UUID (more reliable than /dev/sdX)
- `noatime` option reduces SSD/HDD wear
- exfat requires `uid=1000,gid=1000` for proper permissions
- SMART monitoring requires `smartmontools` package (verify installed)
