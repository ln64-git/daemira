import { Shell } from "../../utility/Shell.js";
import { Logger } from "../../utility/Logger.js";

/**
 * Disk usage information
 */
export interface DiskUsage {
	device: string;
	mountPoint: string;
	filesystem: string;
	totalBytes: number;
	usedBytes: number;
	freeBytes: number;
	percentUsed: number;
	totalGB: number;
	usedGB: number;
	freeGB: number;
	status: "healthy" | "warning" | "critical";
}

/**
 * Disk health warning
 */
export interface DiskWarning {
	device: string;
	mountPoint: string;
	level: "warning" | "critical";
	message: string;
	freeGB: number;
	percentUsed: number;
}

/**
 * SMART health status
 */
export interface SmartStatus {
	device: string;
	passed: boolean;
	temperature?: number;
	powerOnHours?: number;
	powerCycles?: number;
	errors?: string[];
	rawOutput?: string;
}

/**
 * Protected disks that should never be mounted or modified
 */
const PROTECTED_DISKS = ["sdc"]; // Windows partition

/**
 * Disk monitoring utility
 * Monitors disk space, health (SMART), and provides alerts
 */
export class DiskMonitor {
	private static instance: DiskMonitor;
	private logger = Logger.getInstance();

	private constructor() { }

	static getInstance(): DiskMonitor {
		if (!DiskMonitor.instance) {
			DiskMonitor.instance = new DiskMonitor();
		}
		return DiskMonitor.instance;
	}

	/**
	 * Check if a disk is protected (e.g., Windows partition)
	 */
	isProtectedDisk(device: string): boolean {
		return PROTECTED_DISKS.some((protected_disk) =>
			device.includes(protected_disk),
		);
	}

	/**
	 * Get all mounted disk usage information
	 */
	async getAllDiskUsage(): Promise<DiskUsage[]> {
		try {
			// Get disk usage using df command
			const result = await Shell.execute(
				'df -B1 --output=source,target,fstype,size,used,avail,pcent | grep -E "^/dev/"',
			);

			if (result.exitCode !== 0) {
				this.logger.error(`Failed to get disk usage: ${result.stderr}`);
				return [];
			}

			const disks: DiskUsage[] = [];
			const lines = result.stdout.trim().split("\n");

			for (const line of lines) {
				const parts = line.trim().split(/\s+/);
				if (parts.length < 7) continue;

				const [device, mountPoint, filesystem, total, used, free, percent] =
					parts;

				// Validate all required fields exist
				if (!device || !mountPoint || !filesystem || !total || !used || !free || !percent) {
					continue;
				}

				const percentUsed = parseInt(percent.replace("%", ""), 10);
				const totalBytes = parseInt(total, 10);
				const usedBytes = parseInt(used, 10);
				const freeBytes = parseInt(free, 10);

				// Skip if parsing failed
				if (isNaN(percentUsed) || isNaN(totalBytes) || isNaN(usedBytes) || isNaN(freeBytes)) {
					continue;
				}

				// Determine status based on thresholds
				let status: "healthy" | "warning" | "critical" = "healthy";
				if (percentUsed >= 95 || freeBytes < 100 * 1024 * 1024 * 1024) {
					status = "critical";
				} else if (percentUsed >= 90 || freeBytes < 200 * 1024 * 1024 * 1024) {
					status = "warning";
				}

				disks.push({
					device,
					mountPoint,
					filesystem,
					totalBytes,
					usedBytes,
					freeBytes,
					percentUsed,
					totalGB: Math.round((totalBytes / 1024 / 1024 / 1024) * 10) / 10,
					usedGB: Math.round((usedBytes / 1024 / 1024 / 1024) * 10) / 10,
					freeGB: Math.round((freeBytes / 1024 / 1024 / 1024) * 10) / 10,
					status,
				});
			}

			return disks;
		} catch (error) {
			this.logger.error(`Error getting disk usage: ${error}`);
			return [];
		}
	}

	/**
	 * Check for low disk space warnings
	 */
	async checkLowSpace(): Promise<DiskWarning[]> {
		const disks = await this.getAllDiskUsage();
		const warnings: DiskWarning[] = [];

		for (const disk of disks) {
			if (disk.status === "critical") {
				warnings.push({
					device: disk.device,
					mountPoint: disk.mountPoint,
					level: "critical",
					message: `CRITICAL: ${disk.mountPoint} has only ${disk.freeGB}GB free (${disk.percentUsed}% used)`,
					freeGB: disk.freeGB,
					percentUsed: disk.percentUsed,
				});
			} else if (disk.status === "warning") {
				warnings.push({
					device: disk.device,
					mountPoint: disk.mountPoint,
					level: "warning",
					message: `WARNING: ${disk.mountPoint} has ${disk.freeGB}GB free (${disk.percentUsed}% used)`,
					freeGB: disk.freeGB,
					percentUsed: disk.percentUsed,
				});
			}
		}

		return warnings;
	}

	/**
	 * Get SMART health status for a disk
	 * Requires smartmontools (smartctl)
	 */
	async getSmartStatus(device: string): Promise<SmartStatus | null> {
		try {
			// Check if smartctl is available
			const checkResult = await Shell.execute("which smartctl");
			if (checkResult.exitCode !== 0) {
				this.logger.warn("smartctl not found - install smartmontools package");
				return null;
			}

			// Get SMART health
			const result = await Shell.execute(`sudo smartctl -H ${device}`, {});

			const passed = result.stdout.includes("PASSED");
			const status: SmartStatus = {
				device,
				passed,
				rawOutput: result.stdout,
			};

			// Get detailed SMART data
			const detailResult = await Shell.execute(`sudo smartctl -a ${device}`, {});
			if (detailResult.exitCode === 0) {
				// Extract temperature
				const tempMatch = detailResult.stdout.match(
					/Temperature.*?(\d+)\s*Celsius/i,
				);
				if (tempMatch && tempMatch[1]) {
					status.temperature = parseInt(tempMatch[1], 10);
				}

				// Extract power on hours
				const hoursMatch = detailResult.stdout.match(
					/Power_On_Hours.*?(\d+)/i,
				);
				if (hoursMatch && hoursMatch[1]) {
					status.powerOnHours = parseInt(hoursMatch[1], 10);
				}

				// Extract power cycles
				const cyclesMatch = detailResult.stdout.match(
					/Power_Cycle_Count.*?(\d+)/i,
				);
				if (cyclesMatch && cyclesMatch[1]) {
					status.powerCycles = parseInt(cyclesMatch[1], 10);
				}

				// Check for errors
				const errors: string[] = [];
				if (detailResult.stdout.includes("FAILING_NOW")) {
					errors.push("Disk has attributes FAILING NOW");
				}
				if (detailResult.stdout.match(/Reallocated_Sector_Ct.*?\d+/)) {
					const match = detailResult.stdout.match(
						/Reallocated_Sector_Ct\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+(\d+)/,
					);
					if (match && match[1] && parseInt(match[1], 10) > 0) {
						errors.push(`Reallocated sectors: ${match[1]}`);
					}
				}
				if (errors.length > 0) {
					status.errors = errors;
				}
			}

			return status;
		} catch (error) {
			this.logger.error(`Error getting SMART status for ${device}: ${error}`);
			return null;
		}
	}

	/**
	 * Get SMART status for all physical disks
	 */
	async getAllSmartStatus(): Promise<SmartStatus[]> {
		try {
			// Get list of physical disks
			const result = await Shell.execute(
				'lsblk -d -n -o NAME,TYPE | grep disk | awk \'{print "/dev/"$1}\'',
				{},
			);

			if (result.exitCode !== 0) {
				this.logger.error(`Failed to list disks: ${result.stderr}`);
				return [];
			}

			const disks = result.stdout
				.trim()
				.split("\n")
				.filter((d) => d.trim());
			const statuses: SmartStatus[] = [];

			for (const disk of disks) {
				// Skip protected disks
				if (this.isProtectedDisk(disk)) {
					this.logger.info(`Skipping protected disk: ${disk}`);
					continue;
				}

				const status = await this.getSmartStatus(disk);
				if (status) {
					statuses.push(status);
				}
			}

			return statuses;
		} catch (error) {
			this.logger.error(`Error getting SMART status for all disks: ${error}`);
			return [];
		}
	}

	/**
	 * Format disk usage for display
	 */
	formatDiskUsage(disk: DiskUsage): string {
		const statusIcon =
			disk.status === "critical" ? "🔴" : disk.status === "warning" ? "🟡" : "🟢";
		return `${statusIcon} ${disk.mountPoint} (${disk.device}): ${disk.usedGB}GB / ${disk.totalGB}GB (${disk.percentUsed}%) - ${disk.freeGB}GB free`;
	}

	/**
	 * Get a summary of all disk usage
	 */
	async getDiskSummary(): Promise<string> {
		const disks = await this.getAllDiskUsage();
		const warnings = await this.checkLowSpace();

		let summary = "=== Disk Usage Summary ===\n\n";

		// Add warnings first
		if (warnings.length > 0) {
			summary += "⚠️  WARNINGS:\n";
			for (const warning of warnings) {
				summary += `  ${warning.level === "critical" ? "🔴" : "🟡"} ${warning.message}\n`;
			}
			summary += "\n";
		}

		// Add all disks
		summary += "All Disks:\n";
		for (const disk of disks) {
			summary += `  ${this.formatDiskUsage(disk)}\n`;
		}

		return summary;
	}
}
