# Filesystem Configuration

## Root Filesystem

- **Device**: `/dev/nvme0n1p2`
- **Size**: 931.2 GB
- **Used**: 458 GB (53%)
- **Available**: 411 GB
- **Filesystem Type**: ext4
- **Mount Point**: `/`

## Boot Partition

- **Device**: `/dev/nvme0n1p1`
- **Size**: 300 MB
- **Filesystem Type**: vfat (FAT32)
- **Mount Point**: `/boot/efi`
- **Purpose**: UEFI boot partition

## Swap

- **Type**: zram (compressed RAM)
- **Size**: 31.2 GB
- **Device**: `/dev/zram0`
- **Usage**: 0 GB (currently unused)
- **Mount Point**: `[SWAP]`

## Additional Storage Devices

### Unmounted Disks

- **sda**: 3.6 TB (unmounted, filesystem unknown)
- **sdb**: 3.6 TB (unmounted, filesystem unknown)
- **sdc**: 931.5 GB (unmounted, filesystem unknown)
- **sdd**: 1.8 TB (unmounted, filesystem unknown)

### Disk Layout

```
nvme0n1 (931.5 GB)
├── nvme0n1p1 (300 MB) - /boot/efi (vfat)
└── nvme0n1p2 (931.2 GB) - / (ext4)

sda (3.6 TB) - unmounted
sdb (3.6 TB) - unmounted
sdc (931.5 GB) - unmounted
sdd (1.8 TB) - unmounted

zram0 (31.2 GB) - swap
```

## Filesystem Features

### ext4 Features (Root Partition)
- Standard Linux filesystem
- Journaling enabled
- Supports large files and directories
- Compatible with all Linux tools

### zram Swap
- Compressed RAM-based swap
- Faster than disk-based swap
- Reduces memory pressure
- Automatically managed by systemd

## Disk Usage

### Current Usage
- **Root**: 53% used (458 GB / 931 GB)
- **Boot**: Size unknown (300 MB partition)
- **Swap**: 0% used (0 GB / 31 GB)

### Monitoring Disk Usage

```bash
# Check disk usage
df -h

# Check specific directory
du -sh /path/to/directory

# Find large files
find / -type f -size +1G 2>/dev/null

# Check inode usage
df -i
```

## Mount Points

### Current Mounts
- `/` - Root filesystem (ext4)
- `/boot/efi` - EFI boot partition (vfat)
- `[SWAP]` - zram swap

### Additional Mounts (if any)
Check with: `mount | grep -v tmpfs | grep -v devtmpfs`

## Filesystem Maintenance

### Regular Maintenance

```bash
# Check filesystem (should be unmounted or read-only)
sudo fsck -n /dev/nvme0n1p2

# Check disk health (if SMART available)
sudo smartctl -a /dev/nvme0n1

# Monitor disk I/O
iostat -x 1

# Check for bad blocks (long operation)
sudo badblocks -v /dev/nvme0n1p2
```

### Backup Considerations

- Root partition: 53% used - monitor for growth
- Unmounted disks: Available for backups/storage
- Consider setting up backup strategy for:
  - System configuration (`/etc`)
  - User data (`/home`)
  - Application data (`~/.config`)

## Notes

- Root filesystem is ext4 (standard, reliable)
- zram swap provides fast swap without disk wear
- Large unmounted disks available for expansion/backups
- Boot partition uses vfat for UEFI compatibility

