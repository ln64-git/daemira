/**
 * Daemira - Personal System Utility Daemon
 *
 * Main daemon class that orchestrates:
 * - Google Drive bidirectional sync
 * - Automated system updates
 * - Notion integration (future)
 */

import { GoogleDrive } from "./utility/GoogleDrive";
import { Logger } from "./utility/Logger";
import { config } from "./config";
import { DiskMonitor } from "./features/system-monitor/DiskMonitor.js";
import { PerformanceManager } from "./features/system-monitor/PerformanceManager.js";
import { MemoryMonitor } from "./features/system-monitor/MemoryMonitor.js";
import type { PowerProfile } from "./features/system-monitor/PerformanceManager.js";
import { SystemUpdate } from "./features/system-update/SystemUpdate.js";
import { DesktopIntegration } from "./features/desktop-monitor/DesktopIntegration.js";

export class Daemira {
	private logger = Logger.getInstance();
	private systemLog: string[] = [];
	private googleDrive: GoogleDrive | null = null;
	private googleDriveAutoStarted = false;
	private systemUpdate: SystemUpdate | null = null;
	private diskMonitor = DiskMonitor.getInstance();
	private performanceManager = PerformanceManager.getInstance();
	private memoryMonitor = MemoryMonitor.getInstance();
	private desktopIntegration = DesktopIntegration.getInstance();

	constructor() {
		this.logger.info("Daemira initializing...");
		this.autoStartServices();
	}

	/**
	 * Auto-start services in background (non-blocking)
	 */
	private autoStartServices(): void {
		setTimeout(async () => {
			// Auto-start Google Drive sync
			if (!this.googleDriveAutoStarted) {
				try {
					await this.startGoogleDriveSync();
					this.googleDriveAutoStarted = true;
				} catch (error) {
					this.logger.error(`Failed to auto-start Google Drive sync: ${error}`);
				}
			}

			// Auto-start system update scheduler
			try {
				this.systemUpdate = new SystemUpdate({ autoStart: true });
			} catch (error) {
				this.logger.error(`Failed to start system update scheduler: ${error}`);
			}
		}, 1000);
	}

	/**
	 * Default function - show Google Drive status
	 */
	async defaultFunction(): Promise<string> {
		return this.getGoogleDriveSyncStatus();
	}

	/**
	 * Add a system message to log
	 */
	setSystemMessage(message: string): void {
		this.systemLog.push(message);
		this.logger.info(message);
	}

	// ==================== Google Drive Methods ====================

	/**
	 * Start Google Drive sync service
	 */
	async startGoogleDriveSync(): Promise<string> {
		if (this.googleDrive && this.googleDrive.getStatus().running) {
			return "Google Drive sync is already running.";
		}

		try {
			this.googleDrive = new GoogleDrive(config.rcloneRemoteName);
			const result = await this.googleDrive.start();
			this.setSystemMessage(result);
			return result;
		} catch (error) {
			const errorMsg = error instanceof Error ? error.message : String(error);
			this.setSystemMessage(`Failed to start Google Drive sync: ${errorMsg}`);
			return `Error: ${errorMsg}`;
		}
	}

	/**
	 * Stop Google Drive sync service
	 */
	async stopGoogleDriveSync(): Promise<string> {
		if (!this.googleDrive) {
			return "Google Drive sync is not initialized.";
		}

		const result = await this.googleDrive.stop();
		this.setSystemMessage(result);
		return result;
	}

	/**
	 * Get Google Drive sync status
	 */
	getGoogleDriveSyncStatus(): string {
		if (!this.googleDrive) {
			return "Google Drive sync is not initialized.";
		}

		const status = this.googleDrive.getStatus();

		let output = "Google Drive Sync Status:\n";
		output += `  Running: ${status.running ? "Yes" : "No"}\n`;
		output += `  Mode: ${status.syncMode} (every ${status.syncInterval}s)\n`;
		output += `  Directories: ${status.directories}\n`;
		output += `  Queue Size: ${status.queueSize}\n\n`;

		if (Object.keys(status.syncStates.syncStatus).length > 0) {
			output += "  Directory States:\n";
			for (const [path, state] of Object.entries(
				status.syncStates.syncStatus,
			)) {
				const stateIcon =
					state === "idle" ? "✓" : state === "syncing" ? "↻" : "✗";
				const lastSync = status.syncStates.lastSyncTime[path];
				const lastSyncStr = lastSync
					? new Date(lastSync).toLocaleString()
					: "Never";

				output += `    ${stateIcon} ${path}\n`;
				output += `       Status: ${state}\n`;
				output += `       Last sync: ${lastSyncStr}\n`;

				if (status.syncStates.errorMessages[path]) {
					output += `       Error: ${status.syncStates.errorMessages[path]}\n`;
				}
			}
		}
		return output;
	}

	/**
	 * Force sync all directories immediately
	 */
	async syncAllGoogleDrive(): Promise<string> {
		if (!this.googleDrive) {
			return "Google Drive sync is not initialized. Start it first with startGoogleDriveSync().";
		}

		const result = await this.googleDrive.syncAll();
		this.setSystemMessage(result);
		return result;
	}

	/**
	 * Get Google Drive exclude patterns
	 */
	getGoogleDriveExcludePatterns(): string {
		if (!this.googleDrive) {
			return "Google Drive sync is not initialized.";
		}

		const patterns = this.googleDrive.getExcludePatterns();
		let output = `Google Drive Exclude Patterns (${patterns.length} total):\n\n`;
		output += "These files/folders will NOT be synced:\n";
		patterns.forEach((pattern, index) => {
			output += `  ${index + 1}. ${pattern}\n`;
		});

		return output;
	}

	/**
	 * Add custom exclude pattern
	 */
	addGoogleDriveExcludePattern(pattern: string): string {
		if (!this.googleDrive) {
			return "Google Drive sync is not initialized.";
		}

		this.googleDrive.addExcludePattern(pattern);
		return `Added exclude pattern: ${pattern}`;
	}

	// ==================== System Update Methods ====================

	/**
	 * Get system update status
	 */
	getSystemUpdateStatus(): string {
		if (!this.systemUpdate) {
			return "System update scheduler is not initialized.";
		}

		const status = this.systemUpdate.getStatus();
		let output = "System Update Status:\n";
		output += `  Running: ${status.running ? "Yes" : "No"}\n`;

		if (status.lastUpdate) {
			output += `  Last Update: ${new Date(status.lastUpdate).toLocaleString()}\n`;
		}

		if (status.nextUpdate) {
			output += `  Next Update: ${new Date(status.nextUpdate).toLocaleString()}\n`;
		}

		if (status.history.length > 0) {
			output += "\n  Recent Updates:\n";
			status.history.slice(-5).forEach((entry: { timestamp: number; success: boolean; duration: number }) => {
				const success = entry.success ? "✓" : "✗";
				const duration = (entry.duration / 1000).toFixed(1);
				output += `    ${success} ${new Date(entry.timestamp).toLocaleString()} (${duration}s)\n`;
			});
		}

		return output;
	}

	/**
	 * Run system update immediately
	 */
	async runSystemUpdate(): Promise<string> {
		if (!this.systemUpdate) {
			this.systemUpdate = new SystemUpdate();
		}

		await this.systemUpdate.runUpdate();
		return "System update completed. Check logs for details.";
	}

	// ==================== Storage Monitoring Methods ====================

	/**
	 * Get disk usage summary
	 */
	async getDiskStatus(): Promise<string> {
		return await this.diskMonitor.getDiskSummary();
	}

	/**
	 * Check for low disk space warnings
	 */
	async checkDiskSpace(): Promise<string> {
		const warnings = await this.diskMonitor.checkLowSpace();

		if (warnings.length === 0) {
			return "All disks have sufficient space.";
		}

		let output = "⚠️  DISK SPACE WARNINGS:\n\n";
		for (const warning of warnings) {
			output += `${warning.level === "critical" ? "🔴" : "🟡"} ${warning.message}\n`;
		}

		return output;
	}

	/**
	 * Get SMART health status for all disks
	 */
	async getDiskHealth(): Promise<string> {
		const statuses = await this.diskMonitor.getAllSmartStatus();

		if (statuses.length === 0) {
			return "No SMART status available. Install smartmontools or run with sudo.";
		}

		let output = "=== Disk Health (SMART) ===\n\n";
		for (const status of statuses) {
			const healthIcon = status.passed ? "✓" : "✗";
			output += `${healthIcon} ${status.device}: ${status.passed ? "PASSED" : "FAILED"}\n`;

			if (status.temperature) {
				output += `  Temperature: ${status.temperature}°C\n`;
			}
			if (status.powerOnHours) {
				output += `  Power On Hours: ${status.powerOnHours}\n`;
			}
			if (status.errors && status.errors.length > 0) {
				output += `  Errors: ${status.errors.join(", ")}\n`;
			}
			output += "\n";
		}

		return output;
	}

	// ==================== Performance Management Methods ====================

	/**
	 * Get current power profile
	 */
	async getPowerProfile(): Promise<string> {
		const profile = await this.performanceManager.getCurrentProfile();

		if (!profile) {
			return "Power profiles not available (power-profiles-daemon not running)";
		}

		return `Current power profile: ${profile}`;
	}

	/**
	 * Set power profile
	 */
	async setPowerProfile(profile: PowerProfile): Promise<string> {
		const success = await this.performanceManager.setProfile(profile);

		if (!success) {
			return `Failed to set power profile to ${profile}`;
		}

		return `Power profile set to: ${profile}`;
	}

	/**
	 * List all available power profiles
	 */
	async listPowerProfiles(): Promise<string> {
		const profiles = await this.performanceManager.getAllProfiles();

		if (profiles.length === 0) {
			return "No power profiles available (power-profiles-daemon not running)";
		}

		let output = "=== Available Power Profiles ===\n\n";
		for (const profile of profiles) {
			const activeIcon = profile.active ? "●" : "○";
			output += `${activeIcon} ${profile.name}\n`;

			if (profile.cpuDriver) {
				output += `  CPU Driver: ${profile.cpuDriver}\n`;
			}
			if (profile.platformDriver) {
				output += `  Platform Driver: ${profile.platformDriver}\n`;
			}
			if (profile.degraded) {
				output += `  Status: Degraded\n`;
			}
			output += "\n";
		}

		return output;
	}

	/**
	 * Get CPU statistics
	 */
	async getCPUStats(): Promise<string> {
		const stats = await this.performanceManager.getCPUStats();
		return this.performanceManager.formatCPUStats(stats);
	}

	/**
	 * Auto-suggest optimal power profile
	 */
	async suggestPowerProfile(): Promise<string> {
		const suggested = await this.performanceManager.suggestProfile();
		const current = await this.performanceManager.getCurrentProfile();

		let output = `Suggested power profile: ${suggested}\n`;
		if (current) {
			output += `Current power profile: ${current}\n`;

			if (current !== suggested) {
				output += `\nRecommendation: Switch to ${suggested} for better performance/efficiency`;
			} else {
				output += `\n✓ Current profile is optimal`;
			}
		}

		return output;
	}

	// ==================== Memory Monitoring Methods ====================

	/**
	 * Get memory statistics
	 */
	async getMemoryStats(): Promise<string> {
		const stats = await this.memoryMonitor.getMemoryStats();
		return this.memoryMonitor.formatMemoryStats(stats);
	}

	/**
	 * Check swappiness configuration
	 */
	async checkSwappiness(): Promise<string> {
		const check = await this.memoryMonitor.checkSwappiness();
		return check.message;
	}

	// ==================== System Health Overview ====================

	/**
	 * Get comprehensive system status
	 */
	async getSystemStatus(): Promise<string> {
		let output = "=== Daemira System Status ===\n\n";

		// CPU & Performance
		try {
			const cpuStats = await this.performanceManager.getCPUStats();
			output += `CPU: ${cpuStats.cores}C/${cpuStats.threads}T @ ${cpuStats.averageFrequencyMHz}MHz`;
			if (cpuStats.powerProfile) {
				output += ` (${cpuStats.powerProfile})`;
			}
			if (cpuStats.utilization !== undefined) {
				output += ` - ${cpuStats.utilization}% utilized`;
			}
			output += "\n";
		} catch {
			output += "CPU: Unable to read stats\n";
		}

		// Memory
		try {
			const memStats = await this.memoryMonitor.getMemoryStats();
			output += `Memory: ${memStats.usedGB}GB / ${memStats.totalGB}GB (${memStats.percentUsed}%)`;
			if (memStats.swap.usedGB > 0) {
				output += ` + ${memStats.swap.usedGB}GB swap`;
			}
			output += "\n";
		} catch {
			output += "Memory: Unable to read stats\n";
		}

		// Disk space warnings
		try {
			const warnings = await this.diskMonitor.checkLowSpace();
			if (warnings.length > 0) {
				output += `\n⚠️  Disk Warnings: ${warnings.length}\n`;
				for (const warning of warnings) {
					output += `  ${warning.level === "critical" ? "🔴" : "🟡"} ${warning.mountPoint}: ${warning.freeGB}GB free\n`;
				}
			} else {
				output += "Disk Space: All healthy\n";
			}
		} catch {
			output += "Disk Space: Unable to check\n";
		}

		// Google Drive status
		output += "\n";
		if (this.googleDrive) {
			const gdStatus = this.googleDrive.getStatus();
			output += `Google Drive: ${gdStatus.running ? "Running" : "Stopped"} (${gdStatus.queueSize} queued)\n`;
		} else {
			output += "Google Drive: Not initialized\n";
		}

		// System Update status
		if (this.systemUpdate) {
			const suStatus = this.systemUpdate.getStatus();
			if (suStatus.lastUpdate) {
				const lastUpdate = new Date(suStatus.lastUpdate);
				const hoursSince = (Date.now() - suStatus.lastUpdate) / 1000 / 60 / 60;
				output += `System Update: Last ${hoursSince.toFixed(1)}h ago\n`;
			} else {
				output += "System Update: Never run\n";
			}
		} else {
			output += "System Update: Not initialized\n";
		}

		// Desktop Environment
		try {
			const desktopSummary = await this.desktopIntegration.getDesktopSummary();
			output += `\nDesktop Environment:\n  ${desktopSummary}\n`;
		} catch {
			output += "\nDesktop Environment: Unable to query\n";
		}

		return output;
	}

	async getDesktopStatus(): Promise<string> {
		return await this.desktopIntegration.getFormattedStatus();
	}

	async getSessionInfo(): Promise<string> {
		return await this.desktopIntegration.getSessionStatus();
	}

	async getCompositorInfo(): Promise<string> {
		return await this.desktopIntegration.getCompositorStatus();
	}

	async getDisplayInfo(): Promise<string> {
		return await this.desktopIntegration.getDisplayStatus();
	}

	async lockSession(): Promise<string> {
		return await this.desktopIntegration.lockSession();
	}

	async unlockSession(): Promise<string> {
		return await this.desktopIntegration.unlockSession();
	}
}
