/**
 * Google Drive Utility - Bidirectional sync using rclone bisync
 *
 * Features:
 * - Multi-directory sync with individual state tracking
 * - Queue-based processing (one sync at a time)
 * - Periodic sync intervals (default: 30 seconds)
 * - 60+ exclude patterns for build artifacts, caches, etc.
 * - Resilient bisync with conflict resolution
 * - Integration with Logger and Shell utilities
 */

import { homedir } from "node:os";
import { join } from "node:path";
import { existsSync, unlinkSync } from "node:fs";
import { Logger } from "./Logger";
import { Shell } from "./Shell";

// Configuration constants
const CONFIG = {
	DEBOUNCE_DELAY_MS: 2000, // Delay before syncing after file change (2 seconds)
	PERIODIC_SYNC_DELAY_MS: 30000, // Periodic sync interval (30 seconds)
	QUEUE_PROCESS_INTERVAL_MS: 1000, // How often to process sync queue (1 second)
} as const;

interface SyncDirectory {
	localPath: string;
	remotePath: string;
	needsInitialSync: boolean;
}

interface SyncOperation {
	directory: string;
	timestamp: number;
	retries: number;
}

interface SyncState {
	lastSyncTime: Record<string, number>;
	syncStatus: Record<string, "idle" | "syncing" | "error">;
	errorMessages: Record<string, string>;
}

/**
 * Google Drive sync service using rclone
 */
export class GoogleDrive {
	private logger = Logger.getInstance();
	private directories: Map<string, SyncDirectory> = new Map();
	private syncQueue: Map<string, SyncOperation> = new Map();
	private debounceTimers: Map<string, NodeJS.Timeout> = new Map();
	private isRunning = false;
	private processInterval?: NodeJS.Timeout;
	private periodicSyncInterval?: NodeJS.Timeout;
	private remoteName: string = "gdrive";
	private debounceDelay: number = CONFIG.DEBOUNCE_DELAY_MS;
	private periodicSyncDelay: number = CONFIG.PERIODIC_SYNC_DELAY_MS;
	private excludePatterns: string[] = [];

	public state: SyncState = {
		lastSyncTime: {},
		syncStatus: {},
		errorMessages: {},
	};

	constructor(remoteName: string = "gdrive") {
		this.remoteName = remoteName;
		this.setupExcludePatterns();
		this.logger.info(`GoogleDrive initialized with remote: ${remoteName}`);
	}

	/**
	 * Setup common exclude patterns for files/folders to ignore
	 */
	private setupExcludePatterns(): void {
		this.excludePatterns = [
			// Node.js / JavaScript / TypeScript
			"**/node_modules/**",
			"**/.npm/**",
			"**/.yarn/**",
			"**/.pnpm/**",
			"**/bower_components/**",
			"**/.turbo/**",
			"**/.vercel/**",
			"**/dist/**",
			"**/build/**",
			"**/.next/**",
			"**/.nuxt/**",
			"**/out/**",
			"**/.output/**",
			"**/.cache/**",
			"**/.parcel-cache/**",
			"**/coverage/**",
			"**/.nyc_output/**",

			// Python
			"**/.venv/**",
			"**/venv/**",
			"**/__pycache__/**",
			"**/*.pyc",
			"**/*.pyo",
			"**/*.pyd",
			"**/.Python/**",
			"**/pip-log.txt/**",
			"**/.pytest_cache/**",
			"**/.tox/**",
			"**/htmlcov/**",

			// Rust
			"**/target/**",
			"**/*.rs.bk",

			// Go
			"**/vendor/**",

			// Java
			"**/target/**",
			"**/.gradle/**",
			"**/build/**",

			// Ruby
			"**/vendor/bundle/**",
			"**/.bundle/**",

			// Version control
			"**/.git/**",
			"**/.svn/**",
			"**/.hg/**",
			"**/.gitignore",

			// IDE and editor files
			"**/.vscode/**",
			"**/.idea/**",
			"**/*.swp",
			"**/*.swo",
			"**/*~",
			"**/.*.swp",
			"**/.*.swo",

			// OS files
			"**/.DS_Store",
			"**/Thumbs.db",
			"**/.Trash-*/**",
			".local/share/Trash/**",

			// Temporary files
			"**/*.tmp",
			"**/*.temp",
			"**/*.log",
			"**/tmp/**",
			"**/temp/**",

			// Browser caches
			".mozilla/firefox/*/cache2/**",
			".cache/google-chrome/**",
			".cache/chromium/**",
			".cache/mozilla/**",

			// Environment and secrets
			"**/.env",
			"**/.env.local",
			"**/.env.*.local",

			// Database files
			"**/*.sqlite",
			"**/*.db",

			// Large media/game caches
			".local/share/Steam/**",
			".steam/**",

			// System cache
			".cache/**",

			// User-specific excludes
			"IK Multimedia/**", // Exclude IK Multimedia folder
			"Teamruns/**", // Exclude Teamruns folder
		];
	}

	/**
	 * Get exclude arguments for rclone
	 */
	private getExcludeArgs(): string[] {
		const args: string[] = [];
		for (const pattern of this.excludePatterns) {
			args.push("--exclude", pattern);
		}
		return args;
	}

	/**
	 * Add a directory to sync
	 */
	addDirectory(localPath: string, remotePath: string): void {
		const fullLocalPath = localPath.startsWith("~")
			? join(homedir(), localPath.slice(1))
			: localPath;

		this.directories.set(fullLocalPath, {
			localPath: fullLocalPath,
			remotePath,
			needsInitialSync: true,
		});

		this.state.syncStatus[fullLocalPath] = "idle";
		this.logger.debug(`Added directory: ${fullLocalPath} -> ${remotePath}`);
	}

	/**
	 * Setup default home directories
	 */
	setupDefaultDirectories(): void {
		const home = homedir();
		const defaultDirs = [
			{ local: join(home, "Documents"), remote: `${this.remoteName}:Documents` },
			{ local: join(home, "Downloads"), remote: `${this.remoteName}:Downloads` },
			{ local: join(home, "Pictures"), remote: `${this.remoteName}:Pictures` },
			{ local: join(home, "Desktop"), remote: `${this.remoteName}:Desktop` },
			{ local: join(home, "Music"), remote: `${this.remoteName}:Music` },
			{ local: join(home, "Source"), remote: `${this.remoteName}:Source` },
			{ local: join(home, ".config"), remote: `${this.remoteName}:.config` },
		];

		for (const dir of defaultDirs) {
			this.addDirectory(dir.local, dir.remote);
		}
	}

	/**
	 * Start watching directories and syncing
	 */
	async start(): Promise<string> {
		if (this.isRunning) {
			return "Google Drive sync is already running.";
		}

		// Check rclone configuration and connection
		const configCheck = await this._checkConfig(this.remoteName);
		if (!configCheck.installed) {
			throw new Error(
				"rclone is not installed. Install it with: sudo pacman -S rclone",
			);
		}
		if (!configCheck.remoteConfigured) {
			throw new Error(
				`rclone remote '${this.remoteName}' is not configured. Run 'rclone config' to set it up.`,
			);
		}
		if (!configCheck.connectionWorks) {
			throw new Error(
				configCheck.error ||
				`Failed to connect to '${this.remoteName}'. Check your internet and authentication.`,
			);
		}

		this.logger.info("Connection to Google Drive verified");

		this.isRunning = true;

		// Setup default directories if none configured
		if (this.directories.size === 0) {
			this.setupDefaultDirectories();
		}

		// Check which directories need initial sync
		for (const [path, dir] of this.directories.entries()) {
			const needsSync = await this._needsResync(dir.localPath, dir.remotePath);
			dir.needsInitialSync = needsSync;

			if (needsSync) {
				this.logger.info(`Directory ${path} needs initial sync`);
			}
		}

		// Perform initial syncs
		await this.performInitialSyncs();

		// Start file watchers (now uses periodic sync)
		this.startWatchers();

		// Start queue processor
		this.processInterval = setInterval(
			() => this.processQueue(),
			CONFIG.QUEUE_PROCESS_INTERVAL_MS,
		);

		// Start periodic sync timer
		this.periodicSyncInterval = setInterval(() => {
			this.logger.debug("Periodic sync triggered for all directories");
			for (const [path] of this.directories.entries()) {
				this.queueSync(path);
			}
		}, this.periodicSyncDelay);

		const dirCount = this.directories.size;
		return ` Google Drive sync started. Syncing ${dirCount} directories every ${this.periodicSyncDelay / 1000} seconds.`;
	}

	/**
	 * Perform initial syncs for directories that need it
	 */
	private async performInitialSyncs(): Promise<void> {
		for (const [path, dir] of this.directories.entries()) {
			if (dir.needsInitialSync) {
				this.logger.info(`Performing initial sync for ${path}...`);
				this.state.syncStatus[path] = "syncing";

				// Clear any stale lock files from previous interrupted syncs
				await this._clearLocks(dir.localPath, dir.remotePath);

				this.logger.debug(
					"Starting initial bisync (will create remote directory if needed)...",
				);

				const result = await this.executeBisync(
					dir.localPath,
					dir.remotePath,
					true,
				);

				if (result.success) {
					dir.needsInitialSync = false;
					this.state.lastSyncTime[path] = Date.now();
					this.state.syncStatus[path] = "idle";
					this.logger.info(`Initial sync completed for ${path}`);
				} else {
					this.state.syncStatus[path] = "error";
					this.state.errorMessages[path] = result.error || "Unknown error";
					this.logger.error(`Initial sync failed for ${path}: ${result.error}`);

					// If lock file error, try clearing it again for next run
					if (result.error && result.error.includes("prior lock")) {
						this.logger.info("Lock file detected - will be cleared on next attempt");
					}
				}
			}
		}
	}

	/**
	 * Start file system watchers for all directories
	 * NOTE: We use a periodic sync approach instead of recursive file watching
	 * to avoid hitting system inotify limits (especially with node_modules, etc.)
	 */
	private startWatchers(): void {
		// Use periodic syncing instead of file watching to avoid inotify limits
		this.logger.info(
			"Using periodic sync mode (every 30 seconds) instead of file watching",
		);
		this.logger.info("This avoids system limits and respects rate limiting better.");

		// Queue all directories for immediate sync after startup
		for (const [path] of this.directories.entries()) {
			this.queueSync(path);
		}
	}

	/**
	 * Handle file change events
	 */
	private onFileChange(directoryPath: string, filename: string): void {
		// Check if file matches exclude patterns
		if (this.shouldExclude(filename)) {
			return;
		}

		// Debounce: clear existing timer and set new one
		const existingTimer = this.debounceTimers.get(directoryPath);
		if (existingTimer) {
			clearTimeout(existingTimer);
		}

		const timer = setTimeout(() => {
			this.queueSync(directoryPath);
			this.debounceTimers.delete(directoryPath);
		}, this.debounceDelay);

		this.debounceTimers.set(directoryPath, timer);
	}

	/**
	 * Check if a file should be excluded based on patterns
	 */
	private shouldExclude(filepath: string): boolean {
		return this.excludePatterns.some((pattern) => {
			const regex = new RegExp(
				pattern
					.replace(/\*\*/g, ".*")
					.replace(/\*/g, "[^/]*")
					.replace(/\?/g, "[^/]"),
			);
			return regex.test(filepath);
		});
	}

	/**
	 * Queue a sync operation for a directory
	 */
	queueSync(directoryPath: string): void {
		if (!this.isRunning) return;

		this.syncQueue.set(directoryPath, {
			directory: directoryPath,
			timestamp: Date.now(),
			retries: 0,
		});
	}

	/**
	 * Process queued sync operations (one at a time to avoid overwhelming the system)
	 */
	private async processQueue(): Promise<void> {
		if (this.syncQueue.size === 0) return;

		// Get oldest item from queue
		const entries = Array.from(this.syncQueue.entries());
		const firstEntry = entries[0];
		if (!firstEntry) return;

		const [path, operation] = firstEntry;

		// Check if directory is already syncing
		if (this.state.syncStatus[path] === "syncing") {
			return;
		}

		// Remove from queue and sync (one at a time)
		this.syncQueue.delete(path);
		await this.syncDirectory(path);
	}

	/**
	 * Sync a specific directory
	 */
	async syncDirectory(directoryPath: string): Promise<void> {
		const dir = this.directories.get(directoryPath);
		if (!dir) {
			this.logger.error(`Directory not found: ${directoryPath}`);
			return;
		}

		this.state.syncStatus[directoryPath] = "syncing";
		this.logger.info(`Syncing ${directoryPath}...`);

		const result = await this.executeBisync(dir.localPath, dir.remotePath, false);

		if (result.success) {
			this.state.lastSyncTime[directoryPath] = Date.now();
			this.state.syncStatus[directoryPath] = "idle";
			delete this.state.errorMessages[directoryPath];
			this.logger.info(`Synced ${directoryPath}`);
		} else {
			this.state.syncStatus[directoryPath] = "error";
			this.state.errorMessages[directoryPath] = result.error || "Unknown error";
			this.logger.error(`Sync failed for ${directoryPath}: ${result.error}`);
		}
	}

	/**
	 * Execute rclone bisync command
	 */
	private async executeBisync(
		localPath: string,
		remotePath: string,
		isInitial: boolean,
	): Promise<{ success: boolean; error?: string }> {
		// Helper to escape shell arguments (quote if needed)
		const escapeArg = (arg: string): string => {
			// If it's a flag (starts with --), don't quote
			if (arg.startsWith("--")) {
				return arg;
			}
			// Quote arguments that contain special characters, spaces, or wildcards
			if (arg.includes(" ") || arg.includes("*") || arg.includes("?") || arg.includes("$") || arg.includes("`") || arg.includes("\\")) {
				return `"${arg.replace(/"/g, '\\"')}"`;
			}
			return arg;
		};

		const args = [
			"bisync",
			localPath,
			remotePath,
			...this.getExcludeArgs(),
			"--resilient",
			"--recover",
			"--conflict-resolve",
			"newer",
			"--conflict-loser",
			"num",
			"--create-empty-src-dirs",
			"--skip-links", // Skip symlinks instead of failing on them
			"--progress", // Show progress during transfers
			"--stats",
			"30s", // Show stats every 30 seconds
			"--max-size",
			"10G", // Skip files larger than 10 GB
			"--drive-chunk-size",
			"64M", // Use 64MB chunks for large file uploads
			"--transfers",
			"4", // Allow 4 parallel transfers
			"--checkers",
			"8", // Use 8 parallel checkers
		];

		if (isInitial) {
			args.push("--resync");
		}

		const command = `rclone ${args.map(escapeArg).join(" ")}`;

		// No timeout - let it run as long as needed for large syncs
		let lastProgressTime = Date.now();
		const result = await Shell.execute(command, {
			timeout: 0, // 0 = no timeout
			onStdout: (line) => {
				// Log important rclone output (skip NOTICE about symlinks to reduce noise)
				if (!line.includes("Can't follow symlink")) {
					// Show progress updates
					const now = Date.now();
					if (
						line.includes("Transferred:") ||
						line.includes("INFO") ||
						now - lastProgressTime > 5000
					) {
						this.logger.debug(`  ${line}`);
						lastProgressTime = now;
					}
				}
			},
			onStderr: (line) => {
				// Log errors
				if (!line.includes("Can't follow symlink")) {
					this.logger.debug(`  ${line}`);
				}
			},
			logCommand: false, // We'll log ourselves
		});

		if (result.timedOut) {
			return {
				success: false,
				error: "Bisync timed out unexpectedly.",
			};
		}

		if (result.exitCode === 0) {
			return { success: true };
		}

		// Combine both error and output for better debugging
		const errorMsg = result.stderr || result.stdout || "Sync failed";
		// Extract the most relevant error lines
		const errorLines = errorMsg
			.split("\n")
			.filter(
				(line) =>
					line.includes("ERROR") ||
					line.includes("NOTICE") ||
					line.includes("Failed"),
			)
			.slice(-5) // Last 5 error lines
			.join("\n");

		return {
			success: false,
			error: errorLines || errorMsg,
		};
	}

	/**
	 * Stop all watchers and sync operations
	 */
	async stop(): Promise<string> {
		if (!this.isRunning) {
			return "Google Drive sync is not running.";
		}

		this.isRunning = false;

		// Clear all debounce timers
		for (const timer of this.debounceTimers.values()) {
			clearTimeout(timer);
		}
		this.debounceTimers.clear();

		// Stop queue processor
		if (this.processInterval) {
			clearInterval(this.processInterval);
			this.processInterval = undefined;
		}

		// Stop periodic sync timer
		if (this.periodicSyncInterval) {
			clearInterval(this.periodicSyncInterval);
			this.periodicSyncInterval = undefined;
		}

		this.logger.info("Google Drive sync stopped");
		return " Google Drive sync stopped.";
	}

	/**
	 * Get current sync status
	 */
	getStatus(): {
		running: boolean;
		directories: number;
		queueSize: number;
		syncStates: SyncState;
		syncMode: string;
		syncInterval: number;
	} {
		return {
			running: this.isRunning,
			directories: this.directories.size,
			queueSize: this.syncQueue.size,
			syncStates: this.state,
			syncMode: "periodic",
			syncInterval: this.periodicSyncDelay / 1000, // in seconds
		};
	}

	/**
	 * Force sync all directories immediately
	 */
	async syncAll(): Promise<string> {
		if (!this.isRunning) {
			return "Google Drive sync is not running. Start it first with start().";
		}

		const paths = Array.from(this.directories.keys());
		for (const path of paths) {
			this.queueSync(path);
		}

		return ` Queued ${paths.length} directories for sync.`;
	}

	/**
	 * Get the list of exclude patterns
	 */
	getExcludePatterns(): string[] {
		return [...this.excludePatterns];
	}

	/**
	 * Add custom exclude patterns
	 */
	addExcludePattern(pattern: string): void {
		if (!this.excludePatterns.includes(pattern)) {
			this.excludePatterns.push(pattern);
			this.logger.info(`Added exclude pattern: ${pattern}`);
		}
	}

	/**
	 * Remove an exclude pattern
	 */
	removeExcludePattern(pattern: string): void {
		const index = this.excludePatterns.indexOf(pattern);
		if (index > -1) {
			this.excludePatterns.splice(index, 1);
			this.logger.info(`Removed exclude pattern: ${pattern}`);
		}
	}

	// Private helper methods (merged from rclone.ts)

	/**
	 * Check if rclone is installed and the remote is configured
	 */
	private async _checkConfig(remoteName: string = "gdrive"): Promise<{
		installed: boolean;
		remoteConfigured: boolean;
		connectionWorks: boolean;
		error?: string;
	}> {
		// Check if rclone is installed
		const versionCheck = await Shell.execute("rclone version", {
			timeout: 5000,
			logCommand: false,
		});
		if (versionCheck.exitCode !== 0) {
			return {
				installed: false,
				remoteConfigured: false,
				connectionWorks: false,
				error: "rclone is not installed or not in PATH",
			};
		}

		// Check if remote is configured
		const listRemotes = await Shell.execute("rclone listremotes", {
			timeout: 5000,
			logCommand: false,
		});
		const remotes = listRemotes.stdout
			.split("\n")
			.filter((line) => line.trim());
		const remoteExists = remotes.some((remote) =>
			remote.startsWith(`${remoteName}:`),
		);

		if (!remoteExists) {
			return {
				installed: true,
				remoteConfigured: false,
				connectionWorks: false,
				error: `Remote '${remoteName}' is not configured. Run 'rclone config' to set it up.`,
			};
		}

		// Test actual connection to remote
		this.logger.info(`Testing connection to ${remoteName}...`);
		const aboutResult = await Shell.execute(`rclone about ${remoteName}:`, {
			timeout: 15000,
			logCommand: false,
		});

		if (aboutResult.timedOut) {
			return {
				installed: true,
				remoteConfigured: true,
				connectionWorks: false,
				error: `Connection to ${remoteName} timed out. Check your internet connection and authentication.`,
			};
		}

		if (aboutResult.exitCode !== 0) {
			return {
				installed: true,
				remoteConfigured: true,
				connectionWorks: false,
				error: `Failed to connect to ${remoteName}: ${aboutResult.stderr || aboutResult.stdout}`,
			};
		}

		return {
			installed: true,
			remoteConfigured: true,
			connectionWorks: true,
		};
	}

	/**
	 * Check if a directory needs initial resync (first time sync)
	 */
	private async _needsResync(
		localPath: string,
		remotePath: string,
	): Promise<boolean> {
		// Check if bisync workdir exists for this path pair
		const configDir = process.env.XDG_CONFIG_HOME || join(homedir(), ".config");
		const bisyncDir = join(configDir, "rclone", "bisync");

		// Bisync creates a workdir based on the paths
		// If it doesn't exist or is empty, we need to resync
		if (!existsSync(bisyncDir)) {
			return true;
		}

		// Try a test bisync to see if it complains about needing resync
		const testResult = await Shell.execute(
			`rclone bisync ${localPath} ${remotePath} --dry-run`,
			{
				timeout: 10000,
				logCommand: false,
			},
		);

		// If it mentions resync or first run, we need to do initial sync
		return (
			testResult.stderr.includes("--resync") ||
			testResult.stderr.includes("first run")
		);
	}

	/**
	 * Clean up bisync lock files for a path pair
	 */
	private async _clearLocks(
		localPath: string,
		remotePath: string,
	): Promise<void> {
		const cacheDir = process.env.XDG_CACHE_HOME || join(homedir(), ".cache");
		const bisyncCacheDir = join(cacheDir, "rclone", "bisync");

		// Generate the lock file name pattern matching rclone's format:
		// local__{sanitized_local_path}..{sanitized_remote_path}.lck
		const sanitizedLocal = localPath.replace(/\//g, "_");
		const sanitizedRemote = remotePath.replace(/:/g, "_").replace(/\//g, "_");
		const lockFile = join(
			bisyncCacheDir,
			`local_${sanitizedLocal}..${sanitizedRemote}.lck`,
		);

		// Try to delete the lock file if it exists
		try {
			if (existsSync(lockFile)) {
				unlinkSync(lockFile);
				this.logger.info("Cleaned up stale lock file");
			}
		} catch (error) {
			// Ignore errors - lock file might not exist or already be deleted
			this.logger.debug(`Could not clear lock file: ${error}`);
		}
	}
}
