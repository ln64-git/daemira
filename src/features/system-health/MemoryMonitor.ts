import { Shell } from "../../utility/Shell.js";
import { Logger } from "../../utility/Logger.js";

/**
 * Memory statistics
 */
export interface MemoryStats {
	totalBytes: number;
	usedBytes: number;
	freeBytes: number;
	availableBytes: number;
	buffersBytes: number;
	cachedBytes: number;
	totalGB: number;
	usedGB: number;
	freeGB: number;
	availableGB: number;
	percentUsed: number;
	swap: SwapStats;
	zram?: ZramStats;
}

/**
 * Swap statistics
 */
export interface SwapStats {
	totalBytes: number;
	usedBytes: number;
	freeBytes: number;
	totalGB: number;
	usedGB: number;
	freeGB: number;
	percentUsed: number;
}

/**
 * zram statistics
 */
export interface ZramStats {
	device: string;
	totalBytes: number;
	usedBytes: number;
	compressedBytes: number;
	compressionRatio: number;
	percentUsed: number;
}

/**
 * Memory monitor
 * Tracks memory usage, swap, and zram statistics
 */
export class MemoryMonitor {
	private static instance: MemoryMonitor;
	private logger = Logger.getInstance();

	// Optimal swappiness for zram
	private static readonly OPTIMAL_SWAPPINESS_ZRAM = 180;

	private constructor() { }

	static getInstance(): MemoryMonitor {
		if (!MemoryMonitor.instance) {
			MemoryMonitor.instance = new MemoryMonitor();
		}
		return MemoryMonitor.instance;
	}

	/**
	 * Get current swappiness value
	 */
	async getSwappiness(): Promise<number | null> {
		try {
			const result = await Shell.execute("cat /proc/sys/vm/swappiness");

			if (result.exitCode !== 0) {
				this.logger.error(`Failed to get swappiness: ${result.stderr}`);
				return null;
			}

			return parseInt(result.stdout.trim(), 10);
		} catch (error) {
			this.logger.error(`Error getting swappiness: ${error}`);
			return null;
		}
	}

	/**
	 * Get recommended swappiness value
	 */
	getRecommendedSwappiness(): number {
		// For zram, recommended is 180
		// For regular swap, recommended is 60
		return MemoryMonitor.OPTIMAL_SWAPPINESS_ZRAM;
	}

	/**
	 * Get memory statistics from /proc/meminfo
	 */
	async getMemoryStats(): Promise<MemoryStats> {
		try {
			const result = await Shell.execute("cat /proc/meminfo");

			if (result.exitCode !== 0) {
				throw new Error(`Failed to read /proc/meminfo: ${result.stderr}`);
			}

			const lines = result.stdout.split("\n");
			const memInfo: Record<string, number> = {};

			for (const line of lines) {
				const match = line.match(/^(\w+):\s+(\d+)/);
				if (match) {
					memInfo[match[1]] = parseInt(match[2], 10) * 1024; // Convert kB to bytes
				}
			}

			const totalBytes = memInfo.MemTotal || 0;
			const freeBytes = memInfo.MemFree || 0;
			const availableBytes = memInfo.MemAvailable || 0;
			const buffersBytes = memInfo.Buffers || 0;
			const cachedBytes = memInfo.Cached || 0;
			const usedBytes = totalBytes - freeBytes - buffersBytes - cachedBytes;

			const swapTotalBytes = memInfo.SwapTotal || 0;
			const swapFreeBytes = memInfo.SwapFree || 0;
			const swapUsedBytes = swapTotalBytes - swapFreeBytes;

			// Get zram stats
			const zram = await this.getZramStats();

			return {
				totalBytes,
				usedBytes,
				freeBytes,
				availableBytes,
				buffersBytes,
				cachedBytes,
				totalGB: Math.round((totalBytes / 1024 / 1024 / 1024) * 10) / 10,
				usedGB: Math.round((usedBytes / 1024 / 1024 / 1024) * 10) / 10,
				freeGB: Math.round((freeBytes / 1024 / 1024 / 1024) * 10) / 10,
				availableGB:
					Math.round((availableBytes / 1024 / 1024 / 1024) * 10) / 10,
				percentUsed: Math.round((usedBytes / totalBytes) * 100 * 10) / 10,
				swap: {
					totalBytes: swapTotalBytes,
					usedBytes: swapUsedBytes,
					freeBytes: swapFreeBytes,
					totalGB: Math.round((swapTotalBytes / 1024 / 1024 / 1024) * 10) / 10,
					usedGB: Math.round((swapUsedBytes / 1024 / 1024 / 1024) * 10) / 10,
					freeGB: Math.round((swapFreeBytes / 1024 / 1024 / 1024) * 10) / 10,
					percentUsed:
						swapTotalBytes > 0
							? Math.round((swapUsedBytes / swapTotalBytes) * 100 * 10) / 10
							: 0,
				},
				zram: zram || undefined,
			};
		} catch (error) {
			this.logger.error(`Error getting memory stats: ${error}`);
			throw error;
		}
	}

	/**
	 * Get zram statistics if available
	 */
	async getZramStats(): Promise<ZramStats | null> {
		try {
			// Check if zram0 exists
			const checkResult = await Shell.execute(
				"test -e /sys/block/zram0 && echo exists",
			);
			if (checkResult.exitCode !== 0 || !checkResult.stdout.includes("exists")) {
				return null;
			}

			// Get zram stats from sysfs
			const diskSizeResult = await Shell.execute(
				"cat /sys/block/zram0/disksize 2>/dev/null",
			);
			const memUsedResult = await Shell.execute(
				"cat /sys/block/zram0/mem_used_total 2>/dev/null",
			);
			const origDataSizeResult = await Shell.execute(
				"cat /sys/block/zram0/orig_data_size 2>/dev/null",
			);

			if (
				diskSizeResult.exitCode !== 0 ||
				memUsedResult.exitCode !== 0 ||
				origDataSizeResult.exitCode !== 0
			) {
				return null;
			}

			const totalBytes = parseInt(diskSizeResult.stdout.trim(), 10);
			const compressedBytes = parseInt(memUsedResult.stdout.trim(), 10);
			const usedBytes = parseInt(origDataSizeResult.stdout.trim(), 10);

			const compressionRatio =
				usedBytes > 0 ? usedBytes / compressedBytes : 1;
			const percentUsed =
				totalBytes > 0 ? (usedBytes / totalBytes) * 100 : 0;

			return {
				device: "/dev/zram0",
				totalBytes,
				usedBytes,
				compressedBytes,
				compressionRatio: Math.round(compressionRatio * 100) / 100,
				percentUsed: Math.round(percentUsed * 10) / 10,
			};
		} catch (error) {
			this.logger.error(`Error getting zram stats: ${error}`);
			return null;
		}
	}

	/**
	 * Check if swappiness is optimal
	 */
	async checkSwappiness(): Promise<{
		current: number;
		recommended: number;
		optimal: boolean;
		message: string;
	}> {
		const current = await this.getSwappiness();
		const recommended = this.getRecommendedSwappiness();

		if (current === null) {
			return {
				current: -1,
				recommended,
				optimal: false,
				message: "Unable to read swappiness value",
			};
		}

		const optimal = current === recommended;

		let message = "";
		if (optimal) {
			message = `Swappiness is optimal for zram (${current})`;
		} else {
			message = `Swappiness is ${current}, recommended ${recommended} for zram. Run: sudo sysctl vm.swappiness=${recommended}`;
		}

		return {
			current,
			recommended,
			optimal,
			message,
		};
	}

	/**
	 * Format memory stats for display
	 */
	formatMemoryStats(stats: MemoryStats): string {
		let output = "=== Memory Statistics ===\n\n";

		output += `Total: ${stats.totalGB} GB\n`;
		output += `Used: ${stats.usedGB} GB (${stats.percentUsed}%)\n`;
		output += `Free: ${stats.freeGB} GB\n`;
		output += `Available: ${stats.availableGB} GB\n`;
		output += `Buffers/Cache: ${Math.round((stats.buffersBytes + stats.cachedBytes) / 1024 / 1024 / 1024 * 10) / 10} GB\n\n`;

		output += "Swap:\n";
		output += `  Total: ${stats.swap.totalGB} GB\n`;
		output += `  Used: ${stats.swap.usedGB} GB (${stats.swap.percentUsed}%)\n`;
		output += `  Free: ${stats.swap.freeGB} GB\n`;

		if (stats.zram) {
			output += `\nzram (${stats.zram.device}):\n`;
			output += `  Size: ${Math.round((stats.zram.totalBytes / 1024 / 1024 / 1024) * 10) / 10} GB\n`;
			output += `  Used (uncompressed): ${Math.round((stats.zram.usedBytes / 1024 / 1024 / 1024) * 10) / 10} GB\n`;
			output += `  Used (compressed): ${Math.round((stats.zram.compressedBytes / 1024 / 1024 / 1024) * 10) / 10} GB\n`;
			output += `  Compression Ratio: ${stats.zram.compressionRatio}x\n`;
			output += `  Usage: ${stats.zram.percentUsed}%\n`;
		}

		return output;
	}
}
