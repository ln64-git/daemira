/**
 * SystemUpdate Feature - Automated system maintenance for Arch Linux
 *
 * Features:
 * - Periodic system updates (default: 6 hours)
 * - Comprehensive update steps (pacman, AUR, firmware, cleanup)
 * - Update history tracking
 * - .pacnew file detection
 * - Reboot requirement detection
 * - Integration with Shell utility and Logger
 */

import { Shell } from "../../utility/Shell";
import { Logger } from "../../utility/Logger";

export interface SystemUpdateOptions {
	interval?: number; // milliseconds (default: 6 hours)
	autoStart?: boolean; // Start scheduler immediately
}

export interface UpdateStep {
	name: string;
	cmd: string;
	optional?: boolean;
}

/**
 * System update scheduler and executor for Arch Linux
 */
export class SystemUpdate {
	private logger = Logger.getInstance();
	private isRunning = false;
	private intervalHandle?: NodeJS.Timeout;
	private updateInterval: number;

	// Track update history
	private lastUpdateTime?: number;
	private updateHistory: Array<{
		timestamp: number;
		success: boolean;
		duration: number;
	}> = [];

	constructor(options: SystemUpdateOptions = {}) {
		this.updateInterval = options.interval || 6 * 60 * 60 * 1000; // 6 hours default

		if (options.autoStart) {
			this.start();
		}
	}

	/**
	 * Start the periodic update scheduler
	 */
	start(): void {
		if (this.isRunning) {
			this.logger.warn("SystemUpdate scheduler already running");
			return;
		}

		this.isRunning = true;
		this.logger.info(
			`Starting system update scheduler (interval: ${this.updateInterval / 1000 / 60} minutes)`,
		);

		// Run immediately
		this.runUpdate();

		// Schedule periodic updates
		this.intervalHandle = setInterval(() => {
			this.runUpdate();
		}, this.updateInterval);
	}

	/**
	 * Stop the scheduler
	 */
	stop(): void {
		if (!this.isRunning) {
			this.logger.warn("SystemUpdate scheduler not running");
			return;
		}

		this.isRunning = false;
		if (this.intervalHandle) {
			clearInterval(this.intervalHandle);
			this.intervalHandle = undefined;
		}

		this.logger.info("System update scheduler stopped");
	}

	/**
	 * Run system update immediately
	 */
	async runUpdate(): Promise<void> {
		this.logger.info("Starting system update...");
		const startTime = Date.now();

		try {
			await this._executeUpdateSteps();
			await this._checkPacnewFiles();
			await this._checkRebootRequired();

			const duration = Date.now() - startTime;
			this.lastUpdateTime = Date.now();
			this.updateHistory.push({
				timestamp: this.lastUpdateTime,
				success: true,
				duration,
			});

			this.logger.info(
				`System update completed successfully in ${duration / 1000}s`,
			);
		} catch (error) {
			const duration = Date.now() - startTime;
			this.updateHistory.push({
				timestamp: Date.now(),
				success: false,
				duration,
			});
			this.logger.error(`System update failed: ${error}`);
		}
	}

	/**
	 * Get update status
	 */
	getStatus(): {
		running: boolean;
		lastUpdate?: number;
		nextUpdate?: number;
		history: typeof this.updateHistory;
	} {
		return {
			running: this.isRunning,
			lastUpdate: this.lastUpdateTime,
			nextUpdate: this.lastUpdateTime
				? this.lastUpdateTime + this.updateInterval
				: undefined,
			history: this.updateHistory.slice(-10), // Last 10 updates
		};
	}

	/**
	 * Execute all update steps
	 */
	private async _executeUpdateSteps(): Promise<void> {
		const steps: UpdateStep[] = [
			{
				name: "Refreshing mirrorlist",
				cmd: "sudo pacman-mirrors --fasttrack",
				optional: true,
			},
			{
				name: "Updating keyrings",
				cmd: "sudo pacman -Sy --needed --noconfirm archlinux-keyring cachyos-keyring",
			},
			{
				name: "Updating package databases",
				cmd: "sudo pacman -Syy --noconfirm",
			},
			{
				name: "Upgrading packages",
				cmd: "sudo pacman -Syu --noconfirm",
			},
			{
				name: "Updating AUR packages",
				cmd: "yay -Sua --noconfirm --answerclean All --answerdiff None --answeredit None --removemake --cleanafter",
			},
			{
				name: "Updating firmware",
				cmd: "sudo fwupdmgr refresh --force && sudo fwupdmgr update -y",
				optional: true,
			},
			{
				name: "Removing orphaned packages",
				cmd: 'orphans=$(pacman -Qdtq 2>/dev/null); [ -z "$orphans" ] || sudo pacman -Rns --noconfirm $orphans',
			},
			{
				name: "Cleaning package cache",
				cmd: "sudo paccache -rk2",
			},
			{
				name: "Cleaning uninstalled cache",
				cmd: "sudo paccache -ruk0",
			},
			{
				name: "Cleaning yay cache",
				cmd: "yay -Sc --noconfirm --answerclean All",
			},
			{
				name: "Optimizing pacman database",
				cmd: "sudo pacman-optimize",
				optional: true,
			},
			{
				name: "Updating GRUB",
				cmd: "sudo grub-mkconfig -o /boot/grub/grub.cfg",
			},
			{
				name: "Reloading systemd daemon",
				cmd: "sudo systemctl daemon-reload",
			},
		];

		for (const step of steps) {
			this.logger.info(`Step: ${step.name}`);

			try {
				const result = await Shell.execute(step.cmd, {
					timeout: 600000, // 10 minutes max per step
					onStdout: (line) => this.logger.debug(`  ${line}`),
				});

				if (result.exitCode === 0) {
					this.logger.info(`Completed: ${step.name}`);
				} else {
					if (step.optional) {
						this.logger.warn(
							`Skipped (optional): ${step.name} (exit code ${result.exitCode})`,
						);
					} else {
						this.logger.warn(
							`Warning: ${step.name} exited with code ${result.exitCode}`,
						);
					}
				}
			} catch (error) {
				const errorMsg = error instanceof Error ? error.message : String(error);
				if (step.optional) {
					this.logger.warn(`Skipped (optional): ${step.name} - ${errorMsg}`);
				} else {
					this.logger.error(`Failed: ${step.name} - ${errorMsg}`);
					throw error;
				}
			}
		}
	}

	/**
	 * Check for .pacnew files
	 */
	private async _checkPacnewFiles(): Promise<void> {
		try {
			const result = await Shell.execute(
				"find /etc -name '*.pacnew' 2>/dev/null",
				{
					timeout: 10000,
				},
			);

			const files = result.stdout
				.split("\n")
				.filter((line) => line.trim());

			if (files.length > 0) {
				this.logger.warn(
					`Found ${files.length} .pacnew file(s) that may need manual merging:`,
				);
				files.forEach((file) => this.logger.warn(`  ${file}`));
				this.logger.info(
					"Consider using 'pacdiff' to merge configuration changes.",
				);
			}
		} catch (error) {
			this.logger.debug("Could not check for .pacnew files");
		}
	}

	/**
	 * Check if reboot is required
	 */
	private async _checkRebootRequired(): Promise<void> {
		try {
			const result = await Shell.execute(
				"[ -f /usr/lib/modules/$(uname -r)/modules.dep ]",
				{ timeout: 5000 },
			);

			const needsReboot = result.exitCode !== 0;
			if (needsReboot) {
				this.logger.warn(
					"Kernel update detected - reboot recommended for changes to take effect",
				);
			}
		} catch (error) {
			this.logger.debug("Could not check reboot status");
		}
	}
}
