
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
import { DiskMonitor } from "../system-health/DiskMonitor.js";
import { PerformanceManager } from "../system-health/PerformanceManager.js";
import { MemoryMonitor } from "../system-health/MemoryMonitor.js";

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
	private diskMonitor = DiskMonitor.getInstance();
	private performanceManager = PerformanceManager.getInstance();
	private memoryMonitor = MemoryMonitor.getInstance();

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
	 * Check if passwordless sudo is available
	 */
	private async checkPasswordlessSudo(): Promise<boolean> {
		try {
			// Use -n flag to test if passwordless sudo works
			const result = await Shell.execute("sudo -n true", {
				timeout: 5000,
				logCommand: false,
			});
			return result.exitCode === 0;
		} catch {
			return false;
		}
	}

	/**
	 * Check if running as root
	 */
	private isRoot(): boolean {
		return process.getuid && process.getuid() === 0;
	}

	/**
	 * Check if a command exists in PATH
	 */
	private async commandExists(command: string): Promise<boolean> {
		try {
			// Extract the base command (first word before space)
			const baseCmd = command.split(/\s+/)[0].replace(/^sudo\s+-n\s+/, "").replace(/^sudo\s+/, "");
			const result = await Shell.execute(`command -v ${baseCmd}`, {
				timeout: 2000,
				logCommand: false,
			});
			return result.exitCode === 0;
		} catch {
			return false;
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
		console.log("=== Starting System Update ===");
		const startTime = Date.now();

		// Check if passwordless sudo is available
		if (!this.isRoot()) {
			const hasPasswordlessSudo = await this.checkPasswordlessSudo();
			if (!hasPasswordlessSudo) {
				const username = process.env.USER || "ln64";
				const errorMsg = "Passwordless sudo is not configured. System updates require sudo access without password prompts.";
				console.error(`\n✗ ERROR: ${errorMsg}`);
				console.error("\nEASIEST: Run the setup script:");
				console.error("  sudo ./scripts/setup-passwordless-sudo.sh");
				console.error("\nMANUAL: Or configure manually:");
				console.error("  sudo visudo");
				console.error(`\nThen add this line (replace '${username}' with your username if different):`);
				console.error(`  ${username} ALL=(ALL) NOPASSWD: /usr/bin/pacman, /usr/bin/paccache, /usr/bin/pacman-optimize, /usr/bin/grub-mkconfig, /usr/bin/systemctl, /usr/bin/fwupdmgr, /usr/bin/fstrim, /usr/bin/dkms`);
				console.error("\nOr for all commands (less secure but simpler):");
				console.error(`  ${username} ALL=(ALL) NOPASSWD: ALL`);
				this.logger.error(errorMsg);
				throw new Error(errorMsg);
			}
		}

		try {
			await this._executeUpdateSteps();
			await this._executeOptimizationSteps();
			await this._checkPacnewFiles();
			await this._checkRebootRequired();
			await this._postUpdateVerification();

			const duration = Date.now() - startTime;
			this.lastUpdateTime = Date.now();
			this.updateHistory.push({
				timestamp: this.lastUpdateTime,
				success: true,
				duration,
			});

			const successMsg = `System update completed successfully in ${(duration / 1000).toFixed(1)}s`;
			this.logger.info(successMsg);
			console.log(`\n✓ ${successMsg}`);
		} catch (error) {
			const duration = Date.now() - startTime;
			this.updateHistory.push({
				timestamp: Date.now(),
				success: false,
				duration,
			});
			const errorMsg = `System update failed: ${error}`;
			this.logger.error(errorMsg);
			console.error(`\n✗ ${errorMsg}`);
			throw error; // Re-throw so caller knows it failed
		}
	}

	/**
	 * Get update status
	 */
	getStatus(): {
		running: boolean;
		lastUpdate?: number;
		nextUpdate?: number;
		history: Array<{
			timestamp: number;
			success: boolean;
			duration: number;
		}>;
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
		console.log("\n=== Executing Update Steps ===");
		const steps: UpdateStep[] = [
			{
				name: "Refreshing mirrorlist",
				cmd: "sudo -n pacman-mirrors --fasttrack",
				optional: true,
			},
			{
				name: "Updating keyrings",
				cmd: "sudo -n pacman -Sy --needed --noconfirm archlinux-keyring cachyos-keyring",
			},
			{
				name: "Updating package databases",
				cmd: "sudo -n pacman -Syy --noconfirm",
			},
			{
				name: "Upgrading packages",
				cmd: "sudo -n pacman -Syu --noconfirm",
			},
			{
				name: "Updating AUR packages",
				cmd: "yay -Sua --noconfirm --answerclean All --answerdiff None --answeredit None --removemake --cleanafter",
			},
			{
				name: "Updating firmware",
				cmd: "sudo -n fwupdmgr refresh --force && sudo -n fwupdmgr update -y",
				optional: true,
			},
			{
				name: "Removing orphaned packages",
				cmd: 'orphans=$(pacman -Qdtq 2>/dev/null); [ -z "$orphans" ] || sudo -n pacman -Rns --noconfirm $orphans',
			},
			{
				name: "Cleaning package cache",
				cmd: "sudo -n paccache -rk2",
			},
			{
				name: "Cleaning uninstalled cache",
				cmd: "sudo -n paccache -ruk0",
			},
			{
				name: "Cleaning yay cache",
				cmd: "yay -Sc --noconfirm --answerclean All",
			},
			{
				name: "Optimizing pacman database",
				cmd: "sudo -n pacman-optimize",
				optional: true,
			},
			{
				name: "Updating GRUB",
				cmd: "sudo -n grub-mkconfig -o /boot/grub/grub.cfg",
			},
			{
				name: "Reloading systemd daemon",
				cmd: "sudo -n systemctl daemon-reload",
			},
		];

		for (let i = 0; i < steps.length; i++) {
			const step = steps[i];
			if (!step) {
				continue; // Skip if step is undefined (shouldn't happen, but TypeScript safety)
			}
			const stepNum = i + 1;
			const currentStep = step; // Create a const reference for TypeScript narrowing
			this.logger.info(`Step ${stepNum}/${steps.length}: ${currentStep.name}`);
			console.log(`\n[${stepNum}/${steps.length}] ${currentStep.name}...`);

			// For optional steps, check if command exists first
			if (currentStep.optional) {
				const cmdExists = await this.commandExists(currentStep.cmd);
				if (!cmdExists) {
					const skipMsg = `Skipped (optional): ${currentStep.name} - command not available on this system`;
					this.logger.info(skipMsg);
					console.log(`  ⚠ ${skipMsg}`);
					continue;
				}
			}

			try {
				// Use shorter timeout for first few commands to fail fast
				const timeout = i < 3 ? 30000 : 600000; // 30s for first 3, 10min for others
				let passwordDetected = false;
				const result = await Shell.execute(currentStep.cmd, {
					timeout,
					onStdout: (line) => {
						this.logger.debug(`  ${line}`);
						// Only log non-empty lines to reduce noise
						if (line.trim()) {
							console.log(`  ${line}`);
						}
					},
					onStderr: (line) => {
						this.logger.warn(`  [stderr] ${line}`);
						// Check for password prompt
						if (line.toLowerCase().includes("password") ||
							line.toLowerCase().includes("sudo: a password is required")) {
							passwordDetected = true;
						}
						// Only show stderr if it's not a password prompt and not common pacman warnings
						if (line.trim() && !passwordDetected) {
							const lowerLine = line.toLowerCase();
							// Filter out common pacman/yay warnings that are normal
							const isNormalWarning =
								lowerLine.includes("warning:") && (
									lowerLine.includes("is newer than") ||
									lowerLine.includes("is up to date") ||
									lowerLine.includes("-- skipping")
								);
							if (!isNormalWarning) {
								console.error(`  [stderr] ${line}`);
							}
						}
					},
				});

				// Check for password prompt after command completes
				if (passwordDetected ||
					(result.stderr && (
						result.stderr.toLowerCase().includes("password") ||
						result.stderr.toLowerCase().includes("sudo: a password is required")
					))) {
					const errorMsg = `Sudo password required for: ${currentStep.name}`;
					console.error(`\n✗ ERROR: ${errorMsg}`);
					console.error(`  Command: ${currentStep.cmd}`);
					console.error(`\n  Solutions:`);
					console.error(`  1. Configure passwordless sudo for this command`);
					console.error(`  2. Run manually: ${currentStep.cmd}`);
					console.error(`  3. Run entire update with sudo: sudo bun start system:update`);
					throw new Error(errorMsg);
				}

				if (result.timedOut) {
					const errorMsg = `Command timed out: ${currentStep.name}`;
					this.logger.error(errorMsg);
					console.error(`  ✗ ${errorMsg}`);
					if (currentStep.optional) {
						this.logger.warn(`Skipping optional step due to timeout`);
						console.log(`  ⚠ Skipping optional step`);
						continue;
					} else {
						throw new Error(`Step timed out: ${currentStep.name}`);
					}
				}

				if (result.exitCode === 0) {
					this.logger.info(`Completed: ${currentStep.name}`);
					console.log(`  ✓ ${currentStep.name}`);
				} else {
					// Check for "command not found" errors
					const isCommandNotFound = result.stderr && (
						result.stderr.toLowerCase().includes("command not found") ||
						result.stderr.toLowerCase().includes("no such file or directory")
					);

					if (currentStep.optional) {
						if (isCommandNotFound) {
							const skipMsg = `Skipped (optional): ${currentStep.name} - command not available on this system`;
							this.logger.info(skipMsg);
							console.log(`  ⚠ ${skipMsg}`);
						} else {
							const warnMsg = `Skipped (optional): ${currentStep.name} (exit code ${result.exitCode})`;
							this.logger.warn(warnMsg);
							console.log(`  ⚠ ${warnMsg}`);
						}
					} else {
						const warnMsg = `Warning: ${currentStep.name} exited with code ${result.exitCode}`;
						this.logger.warn(warnMsg);
						console.log(`  ⚠ ${warnMsg}`);
					}
					if (result.stderr && !isCommandNotFound) {
						// Check if it's a password prompt
						if (result.stderr.toLowerCase().includes("password") ||
							result.stderr.toLowerCase().includes("sudo: a password is required")) {
							throw new Error(`Sudo password required for: ${currentStep.name}. Configure passwordless sudo.`);
						}
						console.error(`  Error output: ${result.stderr.substring(0, 200)}`);
					}
				}
			} catch (error) {
				const errorMsg = error instanceof Error ? error.message : String(error);
				if (currentStep.optional) {
					this.logger.warn(`Skipped (optional): ${currentStep.name} - ${errorMsg}`);
				} else {
					this.logger.error(`Failed: ${currentStep.name} - ${errorMsg}`);
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

	/**
	 * Execute optimization steps after system update
	 */
	private async _executeOptimizationSteps(): Promise<void> {
		this.logger.info("Running post-update optimization...");
		console.log("\n=== Running Post-Update Optimization ===");

		// Step 14: Run TRIM on SSD
		await this._runTrimOperation(14);

		// Step 15: Check I/O scheduler
		await this._checkIOScheduler(15);

		// Step 16: Check SMART health
		await this._checkSmartHealth(16);

		// Step 17: Verify power profile
		await this._checkPowerProfile(17);

		// Step 18: Check memory swappiness
		await this._checkSwappiness(18);

		// Step 19: Check disk space
		await this._checkDiskSpace(19);

		// Step 20: Rebuild DKMS modules if needed
		await this._rebuildDKMSModules(20);
	}

	/**
	 * Run TRIM operation on SSD
	 */
	private async _runTrimOperation(stepNum: number): Promise<void> {
		this.logger.info(`Step ${stepNum}/20: Running TRIM on SSD`);
		console.log(`  [${stepNum}/20] Running TRIM on SSD...`);

		try {
			let passwordDetected = false;
			const result = await Shell.execute("sudo -n fstrim -v /", {
				timeout: 30000,
				onStderr: (line) => {
					// Check for password prompt
					if (line.toLowerCase().includes("password") ||
						line.toLowerCase().includes("sudo: a password is required")) {
						passwordDetected = true;
					}
					this.logger.debug(`  [stderr] ${line}`);
				},
			});

			// Check for password requirement
			if (passwordDetected ||
				(result.stderr && (
					result.stderr.toLowerCase().includes("password") ||
					result.stderr.toLowerCase().includes("sudo: a password is required")
				))) {
				const warnMsg = "TRIM skipped: sudo password required (run manually: sudo fstrim -v /)";
				this.logger.warn(warnMsg);
				console.log(`    ⚠ ${warnMsg}`);
				return;
			}

			if (result.exitCode === 0) {
				const msg = `TRIM completed: ${result.stdout.trim()}`;
				this.logger.info(msg);
				console.log(`    ✓ ${msg}`);
			} else if (result.timedOut) {
				const warnMsg = "TRIM operation timed out";
				this.logger.warn(warnMsg);
				console.log(`    ⚠ ${warnMsg}`);
			} else {
				const warnMsg = `TRIM operation returned exit code ${result.exitCode}`;
				this.logger.warn(warnMsg);
				console.log(`    ⚠ ${warnMsg}`);
			}
		} catch (error) {
			const warnMsg = `TRIM operation failed: ${error}`;
			this.logger.warn(warnMsg);
			console.log(`    ⚠ ${warnMsg}`);
		}
	}

	/**
	 * Check I/O scheduler for NVMe
	 */
	private async _checkIOScheduler(stepNum: number): Promise<void> {
		this.logger.info(`Step ${stepNum}/20: Checking I/O scheduler`);
		console.log(`  [${stepNum}/20] Checking I/O scheduler...`);

		try {
			const result = await Shell.execute(
				"cat /sys/block/nvme0n1/queue/scheduler 2>/dev/null",
				{ timeout: 5000 },
			);

			if (result.exitCode === 0) {
				const scheduler = result.stdout.trim();
				this.logger.info(`I/O Scheduler: ${scheduler}`);

				// For NVMe, 'none' or 'mq-deadline' is optimal
				if (scheduler.includes("[none]") || scheduler.includes("[mq-deadline]")) {
					const msg = `I/O scheduler is optimal: ${scheduler}`;
					this.logger.info(msg);
					console.log(`    ✓ ${msg}`);
				} else {
					const msg = `I/O scheduler: ${scheduler} (consider 'none' or 'mq-deadline' for NVMe)`;
					this.logger.warn(msg);
					console.log(`    ⚠ ${msg}`);
				}
			} else {
				console.log(`    ⚠ Could not check I/O scheduler (NVMe device may not exist)`);
			}
		} catch (error) {
			this.logger.debug("Could not check I/O scheduler");
		}
	}

	/**
	 * Check SMART health for all disks
	 */
	private async _checkSmartHealth(stepNum: number): Promise<void> {
		this.logger.info(`Step ${stepNum}/20: Checking SMART disk health`);
		console.log(`  [${stepNum}/20] Checking SMART disk health...`);

		try {
			const statuses = await this.diskMonitor.getAllSmartStatus();

			if (statuses.length === 0) {
				this.logger.debug("No SMART status available");
				return;
			}

			let healthyCount = 0;
			let failedCount = 0;
			const failedDisks: string[] = [];
			for (const status of statuses) {
				if (status.passed) {
					healthyCount++;
					this.logger.info(`${status.device}: SMART health PASSED`);
					if (status.temperature) {
						this.logger.debug(`  Temperature: ${status.temperature}°C`);
					}
				} else {
					failedCount++;
					failedDisks.push(status.device);
					this.logger.error(`${status.device}: SMART health FAILED`);
					if (status.errors && status.errors.length > 0) {
						status.errors.forEach((err) => this.logger.error(`  Error: ${err}`));
					}
				}
			}
			if (healthyCount > 0 && failedCount === 0) {
				console.log(`    ✓ All ${healthyCount} disk(s) passed SMART health check`);
			} else if (failedCount > 0) {
				console.log(`    ⚠ ${failedCount} disk(s) failed SMART check: ${failedDisks.join(", ")}`);
				console.log(`    ✓ ${healthyCount} disk(s) passed`);
			}
		} catch (error) {
			this.logger.debug(`Could not check SMART health: ${error}`);
		}
	}

	/**
	 * Verify power profile configuration
	 */
	private async _checkPowerProfile(stepNum: number): Promise<void> {
		this.logger.info(`Step ${stepNum}/20: Checking power profile`);
		console.log(`  [${stepNum}/20] Checking power profile...`);

		try {
			const profile = await this.performanceManager.getCurrentProfile();

			if (profile) {
				const msg = `Current power profile: ${profile}`;
				this.logger.info(msg);
				console.log(`    ✓ ${msg}`);
			} else {
				const msg = "power-profiles-daemon not available";
				this.logger.debug(msg);
				console.log(`    ⚠ ${msg}`);
			}
		} catch (error) {
			this.logger.debug(`Could not check power profile: ${error}`);
		}
	}

	/**
	 * Check memory swappiness configuration
	 */
	private async _checkSwappiness(stepNum: number): Promise<void> {
		this.logger.info(`Step ${stepNum}/20: Checking memory swappiness`);
		console.log(`  [${stepNum}/20] Checking memory swappiness...`);

		try {
			const check = await this.memoryMonitor.checkSwappiness();

			if (check.optimal) {
				const msg = `Swappiness is optimal: ${check.current}`;
				this.logger.info(msg);
				console.log(`    ✓ ${msg}`);
			} else {
				this.logger.warn(check.message);
				console.log(`    ⚠ ${check.message}`);
			}
		} catch (error) {
			this.logger.debug(`Could not check swappiness: ${error}`);
		}
	}

	/**
	 * Check disk space for low space warnings
	 */
	private async _checkDiskSpace(stepNum: number): Promise<void> {
		this.logger.info(`Step ${stepNum}/20: Checking disk space`);
		console.log(`  [${stepNum}/20] Checking disk space...`);

		try {
			const warnings = await this.diskMonitor.checkLowSpace();

			if (warnings.length === 0) {
				const msg = "All disks have sufficient space";
				this.logger.info(msg);
				console.log(`    ✓ ${msg}`);
			} else {
				const msg = `Found ${warnings.length} disk space warning(s)`;
				this.logger.warn(msg);
				console.log(`    ⚠ ${msg}`);
				for (const warning of warnings) {
					// Message already includes CRITICAL/WARNING prefix
					this.logger.warn(`  ${warning.message}`);
					const icon = warning.level === "critical" ? "🔴" : "🟡";
					console.log(`      ${icon} ${warning.message}`);
				}
			}
		} catch (error) {
			this.logger.debug(`Could not check disk space: ${error}`);
		}
	}

	/**
	 * Rebuild DKMS modules after kernel update
	 */
	private async _rebuildDKMSModules(stepNum: number): Promise<void> {
		this.logger.info(`Step ${stepNum}/20: Checking DKMS modules`);
		console.log(`  [${stepNum}/20] Checking DKMS modules...`);

		try {
			// Check if any DKMS modules need rebuilding
			const statusResult = await Shell.execute("dkms status", {
				timeout: 10000,
			});

			if (statusResult.exitCode === 0 && statusResult.stdout.trim()) {
				this.logger.info("DKMS modules present, verifying installation");

				// Run dkms autoinstall to rebuild if needed
				let passwordDetected = false;
				const result = await Shell.execute("sudo -n dkms autoinstall", {
					timeout: 120000, // 2 minutes
					onStdout: (line) => this.logger.debug(`  ${line}`),
					onStderr: (line) => {
						if (line.toLowerCase().includes("password") ||
							line.toLowerCase().includes("sudo: a password is required")) {
							passwordDetected = true;
						}
					},
				});

				if (passwordDetected ||
					(result.stderr && (
						result.stderr.toLowerCase().includes("password") ||
						result.stderr.toLowerCase().includes("sudo: a password is required")
					))) {
					const msg = "DKMS check skipped: sudo password required";
					this.logger.warn(msg);
					console.log(`    ⚠ ${msg}`);
				} else if (result.exitCode === 0) {
					const msg = "DKMS modules verified/rebuilt successfully";
					this.logger.info(msg);
					console.log(`    ✓ ${msg}`);
				} else {
					const msg = `DKMS autoinstall exited with code ${result.exitCode}`;
					this.logger.warn(msg);
					console.log(`    ⚠ ${msg}`);
				}
			} else {
				const msg = "No DKMS modules installed";
				this.logger.debug(msg);
				console.log(`    ✓ ${msg}`);
			}
		} catch (error) {
			this.logger.debug(`Could not check DKMS modules: ${error}`);
		}
	}

	/**
	 * Post-update system verification
	 */
	private async _postUpdateVerification(): Promise<void> {
		this.logger.info("Running post-update verification...");

		// Check for any systemd service failures
		try {
			const result = await Shell.execute(
				"systemctl --failed --no-legend --no-pager",
				{ timeout: 10000 },
			);

			if (result.stdout.trim()) {
				const failedServices = result.stdout
					.trim()
					.split("\n")
					.map((line) => line.split(/\s+/)[0]);
				this.logger.warn(
					`Found ${failedServices.length} failed service(s): ${failedServices.join(", ")}`,
				);
			} else {
				this.logger.info("No failed system services detected");
			}
		} catch (error) {
			this.logger.debug("Could not check systemd services");
		}

		// Verify critical functionality
		this.logger.info("System update verification complete");
	}
}
