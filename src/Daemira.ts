/**
 * Daemira - Personal System Utility Daemon
 *
 * Main daemon class that orchestrates:
 * - Google Drive bidirectional sync
 * - Automated system updates
 * - Notion integration (future)
 */

import { GoogleDrive } from "./utility/GoogleDrive";
import { SystemUpdate } from "./features/system-update";
import { Logger } from "./utility/Logger";
import { config } from "./config";

export class Daemira {
	private logger = Logger.getInstance();
	private systemLog: string[] = [];
	private googleDrive: GoogleDrive | null = null;
	private googleDriveAutoStarted = false;
	private systemUpdate: SystemUpdate | null = null;

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
}
