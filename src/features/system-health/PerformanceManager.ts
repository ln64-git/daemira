import { Shell } from "../../utility/Shell.js";
import { Logger } from "../../utility/Logger.js";

/**
 * Power profile types
 */
export type PowerProfile = "performance" | "balanced" | "power-saver";

/**
 * Power profile information
 */
export interface PowerProfileInfo {
	name: PowerProfile;
	active: boolean;
	cpuDriver?: string;
	platformDriver?: string;
	degraded?: boolean;
}

/**
 * CPU statistics
 */
export interface CPUStats {
	cores: number;
	threads: number;
	currentFrequencyMHz: number[];
	averageFrequencyMHz: number;
	minFrequencyMHz: number;
	maxFrequencyMHz: number;
	governor?: string;
	powerProfile?: PowerProfile;
	utilization?: number;
}

/**
 * Performance manager
 * Integrates with power-profiles-daemon for CPU power management
 */
export class PerformanceManager {
	private static instance: PerformanceManager;
	private logger = Logger.getInstance();

	private constructor() { }

	static getInstance(): PerformanceManager {
		if (!PerformanceManager.instance) {
			PerformanceManager.instance = new PerformanceManager();
		}
		return PerformanceManager.instance;
	}

	/**
	 * Check if power-profiles-daemon is available
	 */
	async isPowerProfilesAvailable(): Promise<boolean> {
		try {
			const result = await Shell.execute("which powerprofilesctl");
			return result.exitCode === 0;
		} catch {
			return false;
		}
	}

	/**
	 * Get current power profile
	 */
	async getCurrentProfile(): Promise<PowerProfile | null> {
		try {
			if (!(await this.isPowerProfilesAvailable())) {
				this.logger.warn("power-profiles-daemon not available");
				return null;
			}

			const result = await Shell.execute("powerprofilesctl get");
			if (result.exitCode !== 0) {
				this.logger.error(`Failed to get power profile: ${result.stderr}`);
				return null;
			}

			const profile = result.stdout.trim() as PowerProfile;
			return profile;
		} catch (error) {
			this.logger.error(`Error getting power profile: ${error}`);
			return null;
		}
	}

	/**
	 * Get all available power profiles
	 */
	async getAllProfiles(): Promise<PowerProfileInfo[]> {
		try {
			if (!(await this.isPowerProfilesAvailable())) {
				this.logger.warn("power-profiles-daemon not available");
				return [];
			}

			const result = await Shell.execute("powerprofilesctl list");
			if (result.exitCode !== 0) {
				this.logger.error(`Failed to list power profiles: ${result.stderr}`);
				return [];
			}

			const profiles: PowerProfileInfo[] = [];
			const lines = result.stdout.split("\n");

			let currentProfile: Partial<PowerProfileInfo> | null = null;

			for (const line of lines) {
				const trimmed = line.trim();

				// Check for profile name (starts with * for active, or just profile name)
				if (trimmed.match(/^(\*\s+)?(\w+(-\w+)?):$/)) {
					// Save previous profile if exists
					if (currentProfile && currentProfile.name) {
						profiles.push(currentProfile as PowerProfileInfo);
					}

					// Start new profile
					const active = trimmed.startsWith("*");
					const name = trimmed
						.replace(/^\*\s+/, "")
						.replace(/:$/, "") as PowerProfile;

					currentProfile = {
						name,
						active,
					};
				} else if (currentProfile) {
					// Parse profile properties
					if (trimmed.startsWith("CpuDriver:")) {
						currentProfile.cpuDriver = trimmed
							.replace("CpuDriver:", "")
							.trim();
					} else if (trimmed.startsWith("PlatformDriver:")) {
						currentProfile.platformDriver = trimmed
							.replace("PlatformDriver:", "")
							.trim();
					} else if (trimmed.startsWith("Degraded:")) {
						currentProfile.degraded = trimmed.includes("yes");
					}
				}
			}

			// Add last profile
			if (currentProfile && currentProfile.name) {
				profiles.push(currentProfile as PowerProfileInfo);
			}

			return profiles;
		} catch (error) {
			this.logger.error(`Error getting power profiles: ${error}`);
			return [];
		}
	}

	/**
	 * Set power profile
	 */
	async setProfile(profile: PowerProfile): Promise<boolean> {
		try {
			if (!(await this.isPowerProfilesAvailable())) {
				this.logger.error("power-profiles-daemon not available");
				return false;
			}

			const result = await Shell.execute(
				`powerprofilesctl set ${profile}`,
				{ timeout: 10000 },
			);

			if (result.exitCode !== 0) {
				this.logger.error(`Failed to set power profile to ${profile}: ${result.stderr}`);
				return false;
			}

			this.logger.info(`Power profile set to: ${profile}`);
			return true;
		} catch (error) {
			this.logger.error(`Error setting power profile to ${profile}: ${error}`);
			return false;
		}
	}

	/**
	 * Get CPU frequency for all cores
	 */
	async getCPUFrequencies(): Promise<number[]> {
		try {
			const result = await Shell.execute(
				"grep MHz /proc/cpuinfo | awk '{print $4}'",
			);

			if (result.exitCode !== 0) {
				this.logger.error(`Failed to get CPU frequencies: ${result.stderr}`);
				return [];
			}

			return result.stdout
				.trim()
				.split("\n")
				.map((freq) => parseFloat(freq))
				.filter((freq) => !isNaN(freq));
		} catch (error) {
			this.logger.error(`Error getting CPU frequencies: ${error}`);
			return [];
		}
	}

	/**
	 * Get CPU governor
	 */
	async getCPUGovernor(): Promise<string | null> {
		try {
			const result = await Shell.execute(
				"cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null",
			);

			if (result.exitCode !== 0) {
				return null;
			}

			return result.stdout.trim();
		} catch {
			return null;
		}
	}

	/**
	 * Get comprehensive CPU statistics
	 */
	async getCPUStats(): Promise<CPUStats> {
		try {
			// Get CPU info
			const cpuInfoResult = await Shell.execute("lscpu");
			const cpuInfo = cpuInfoResult.stdout;

			// Extract cores and threads
			const coresMatch = cpuInfo.match(/Core\(s\) per socket:\s*(\d+)/);
			const threadsMatch = cpuInfo.match(/Thread\(s\) per core:\s*(\d+)/);
			const socketsMatch = cpuInfo.match(/Socket\(s\):\s*(\d+)/);

			const sockets = socketsMatch && socketsMatch[1] ? parseInt(socketsMatch[1], 10) : 1;
			const coresPerSocket = coresMatch && coresMatch[1] ? parseInt(coresMatch[1], 10) : 1;
			const threadsPerCore = threadsMatch && threadsMatch[1] ? parseInt(threadsMatch[1], 10) : 1;

			const cores = sockets * coresPerSocket;
			const threads = cores * threadsPerCore;

			// Get frequencies
			const frequencies = await this.getCPUFrequencies();
			const averageFrequency =
				frequencies.length > 0
					? frequencies.reduce((a, b) => a + b, 0) / frequencies.length
					: 0;
			const minFrequency =
				frequencies.length > 0 ? Math.min(...frequencies) : 0;
			const maxFrequency =
				frequencies.length > 0 ? Math.max(...frequencies) : 0;

			// Get governor
			const governor = await this.getCPUGovernor();

			// Get power profile
			const powerProfile = await this.getCurrentProfile();

			// Get CPU utilization (simple average from uptime)
			let utilization: number | undefined;
			try {
				const uptimeResult = await Shell.execute("cat /proc/loadavg");
				if (uptimeResult.exitCode === 0) {
					const loadAvgStr = uptimeResult.stdout.split(" ")[0];
					if (loadAvgStr) {
						const loadAvg = parseFloat(loadAvgStr);
						utilization = Math.min((loadAvg / threads) * 100, 100);
					}
				}
			} catch {
				// Ignore if can't get utilization
			}

			return {
				cores,
				threads,
				currentFrequencyMHz: frequencies,
				averageFrequencyMHz: Math.round(averageFrequency),
				minFrequencyMHz: Math.round(minFrequency),
				maxFrequencyMHz: Math.round(maxFrequency),
				governor: governor || undefined,
				powerProfile: powerProfile || undefined,
				utilization: utilization ? Math.round(utilization * 10) / 10 : undefined,
			};
		} catch (error) {
			this.logger.error(`Error getting CPU stats: ${error}`);
			throw error;
		}
	}

	/**
	 * Suggest optimal power profile based on CPU utilization
	 */
	async suggestProfile(): Promise<PowerProfile> {
		try {
			const stats = await this.getCPUStats();

			if (!stats.utilization) {
				// Default to balanced if can't determine
				return "balanced";
			}

			// Suggest based on utilization
			if (stats.utilization > 70) {
				return "performance";
			} else if (stats.utilization < 30) {
				return "power-saver";
			} else {
				return "balanced";
			}
		} catch (error) {
			this.logger.error(`Error suggesting power profile: ${error}`);
			return "balanced";
		}
	}

	/**
	 * Format CPU stats for display
	 */
	formatCPUStats(stats: CPUStats): string {
		let output = "=== CPU Statistics ===\n\n";
		output += `Cores: ${stats.cores} (${stats.threads} threads)\n`;
		output += `Average Frequency: ${stats.averageFrequencyMHz} MHz\n`;
		output += `Frequency Range: ${stats.minFrequencyMHz} - ${stats.maxFrequencyMHz} MHz\n`;

		if (stats.governor) {
			output += `Governor: ${stats.governor}\n`;
		}

		if (stats.powerProfile) {
			output += `Power Profile: ${stats.powerProfile}\n`;
		}

		if (stats.utilization !== undefined) {
			output += `CPU Utilization: ${stats.utilization}%\n`;
		}

		return output;
	}
}
