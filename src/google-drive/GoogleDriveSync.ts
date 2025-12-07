import { watch } from "fs";
import type { FSWatcher } from "fs";
import { homedir } from "os";
import { join } from "path";
import { runRcloneCommand, checkRcloneConfig, needsResync, ensureRemoteDirectory, clearBisyncLocks } from "./rclone";

interface SyncDirectory {
  localPath: string;
  remotePath: string;
  watcher?: FSWatcher;
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
 * Token bucket rate limiter for controlling sync operations
 */
class RateLimiter {
  private tokens: number;
  private lastRefill: number;
  private readonly maxTokens: number;
  private readonly refillRate: number; // tokens per second

  constructor(maxTokens: number = 10, refillRate: number = 10) {
    this.maxTokens = maxTokens;
    this.refillRate = refillRate;
    this.tokens = maxTokens;
    this.lastRefill = Date.now();
  }

  private refill(): void {
    const now = Date.now();
    const timePassed = (now - this.lastRefill) / 1000; // seconds
    const tokensToAdd = timePassed * this.refillRate;

    this.tokens = Math.min(this.maxTokens, this.tokens + tokensToAdd);
    this.lastRefill = now;
  }

  async acquire(tokens: number = 1): Promise<void> {
    while (true) {
      this.refill();

      if (this.tokens >= tokens) {
        this.tokens -= tokens;
        return;
      }

      // Wait until we have enough tokens
      const tokensNeeded = tokens - this.tokens;
      const waitTime = (tokensNeeded / this.refillRate) * 1000;
      await new Promise((resolve) => setTimeout(resolve, waitTime));
    }
  }

  getAvailableTokens(): number {
    this.refill();
    return Math.floor(this.tokens);
  }
}

/**
 * Google Drive sync service using rclone
 */
export class GoogleDriveSync {
  private directories: Map<string, SyncDirectory> = new Map();
  private syncQueue: Map<string, SyncOperation> = new Map();
  private debounceTimers: Map<string, NodeJS.Timeout> = new Map();
  private rateLimiter: RateLimiter;
  private isRunning: boolean = false;
  private processInterval?: NodeJS.Timeout;
  private periodicSyncInterval?: NodeJS.Timeout;
  private remoteName: string = "gdrive";
  private debounceDelay: number = 2000; // 2 seconds
  private periodicSyncDelay: number = 30000; // 30 seconds
  private excludePatterns: string[] = [];

  public state: SyncState = {
    lastSyncTime: {},
    syncStatus: {},
    errorMessages: {},
  };

  constructor(remoteName: string = "gdrive") {
    this.remoteName = remoteName;
    this.rateLimiter = new RateLimiter(10, 10); // 10 operations per second
    this.setupExcludePatterns();
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
      "IK Multimedia/**",  // Exclude IK Multimedia folder
      "Teamruns/**",       // Exclude Teamruns folder
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
    const configCheck = await checkRcloneConfig(this.remoteName);
    if (!configCheck.installed) {
      throw new Error("rclone is not installed. Install it with: sudo pacman -S rclone");
    }
    if (!configCheck.remoteConfigured) {
      throw new Error(
        `rclone remote '${this.remoteName}' is not configured. Run 'rclone config' to set it up.`
      );
    }
    if (!configCheck.connectionWorks) {
      throw new Error(
        configCheck.error || `Failed to connect to '${this.remoteName}'. Check your internet and authentication.`
      );
    }

    console.log("✅ Connection to Google Drive verified");

    this.isRunning = true;

    // Setup default directories if none configured
    if (this.directories.size === 0) {
      this.setupDefaultDirectories();
    }

    // Check which directories need initial sync
    for (const [path, dir] of this.directories.entries()) {
      const needsSync = await needsResync(dir.localPath, dir.remotePath);
      dir.needsInitialSync = needsSync;

      if (needsSync) {
        console.log(`🔄 Directory ${path} needs initial sync`);
      }
    }

    // Perform initial syncs
    await this.performInitialSyncs();

    // Start file watchers (now uses periodic sync)
    this.startWatchers();

    // Start queue processor
    this.processInterval = setInterval(() => this.processQueue(), 1000);

    // Start periodic sync timer
    this.periodicSyncInterval = setInterval(() => {
      console.log(`🔄 Periodic sync triggered for all directories`);
      for (const [path] of this.directories.entries()) {
        this.queueSync(path);
      }
    }, this.periodicSyncDelay);

    const dirCount = this.directories.size;
    return `✅ Google Drive sync started. Syncing ${dirCount} directories every ${this.periodicSyncDelay / 1000} seconds.`;
  }

  /**
   * Perform initial syncs for directories that need it
   */
  private async performInitialSyncs(): Promise<void> {
    for (const [path, dir] of this.directories.entries()) {
      if (dir.needsInitialSync) {
        console.log(`🔄 Performing initial sync for ${path}...`);
        this.state.syncStatus[path] = "syncing";

        // Clear any stale lock files from previous interrupted syncs
        await clearBisyncLocks(dir.localPath, dir.remotePath);

        // Skip mkdir - let bisync --resync create directories automatically
        // This avoids timeout issues and is more reliable
        console.log(`  📁 Starting initial bisync (will create remote directory if needed)...`);

        const result = await this.executeBisync(dir.localPath, dir.remotePath, true);

        if (result.success) {
          dir.needsInitialSync = false;
          this.state.lastSyncTime[path] = Date.now();
          this.state.syncStatus[path] = "idle";
          console.log(`✅ Initial sync completed for ${path}`);
        } else {
          this.state.syncStatus[path] = "error";
          this.state.errorMessages[path] = result.error || "Unknown error";
          console.error(`❌ Initial sync failed for ${path}: ${result.error}`);

          // If lock file error, try clearing it again for next run
          if (result.error && result.error.includes("prior lock")) {
            console.log(`  💡 Lock file detected - will be cleared on next attempt`);
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
    console.log(`📅 Using periodic sync mode (every 30 seconds) instead of file watching`);
    console.log(`   This avoids system limits and respects rate limiting better.`);

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
          .replace(/\?/g, "[^/]")
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
   * Process queued sync operations with rate limiting
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

    // Acquire rate limit token
    await this.rateLimiter.acquire(1);

    // Remove from queue and sync
    this.syncQueue.delete(path);
    await this.syncDirectory(path);
  }

  /**
   * Sync a specific directory
   */
  async syncDirectory(directoryPath: string): Promise<void> {
    const dir = this.directories.get(directoryPath);
    if (!dir) {
      console.error(`❌ Directory not found: ${directoryPath}`);
      return;
    }

    this.state.syncStatus[directoryPath] = "syncing";
    console.log(`🔄 Syncing ${directoryPath}...`);

    const result = await this.executeBisync(dir.localPath, dir.remotePath, false);

    if (result.success) {
      this.state.lastSyncTime[directoryPath] = Date.now();
      this.state.syncStatus[directoryPath] = "idle";
      delete this.state.errorMessages[directoryPath];
      console.log(`✅ Synced ${directoryPath}`);
    } else {
      this.state.syncStatus[directoryPath] = "error";
      this.state.errorMessages[directoryPath] = result.error || "Unknown error";
      console.error(`❌ Sync failed for ${directoryPath}: ${result.error}`);
    }
  }

  /**
   * Execute rclone bisync command
   */
  private async executeBisync(
    localPath: string,
    remotePath: string,
    isInitial: boolean
  ): Promise<{ success: boolean; error?: string }> {
    const args = [
      "bisync",
      localPath,
      remotePath,
      ...this.getExcludeArgs(),
      "--resilient",
      "--recover",
      "--conflict-resolve", "newer",
      "--conflict-loser", "num",
      "--create-empty-src-dirs",
      "--skip-links",  // Skip symlinks instead of failing on them
      "--progress",    // Show progress during transfers
      "--stats", "30s", // Show stats every 30 seconds
      "--max-size", "10G",  // Skip files larger than 10 GB
      "--drive-chunk-size", "64M",  // Use 64MB chunks for large file uploads
      "--transfers", "4",  // Allow 4 parallel transfers
      "--checkers", "8",   // Use 8 parallel checkers
    ];

    if (isInitial) {
      args.push("--resync");
    }

    // No timeout - let it run as long as needed for large syncs
    const timeout = 0;  // 0 = no timeout

    let lastProgressTime = Date.now();
    const result = await runRcloneCommand(args, (line) => {
      // Log important rclone output (skip NOTICE about symlinks to reduce noise)
      if (!line.includes("Can't follow symlink")) {
        // Show progress updates
        const now = Date.now();
        if (line.includes("Transferred:") || line.includes("INFO") || now - lastProgressTime > 5000) {
          console.log(`    ${line}`);
          lastProgressTime = now;
        }
      }
    }, timeout);

    if (result.timedOut) {
      return {
        success: false,
        error: `Bisync timed out unexpectedly.`,
      };
    }

    if (result.exitCode === 0) {
      return { success: true };
    } else {
      // Combine both error and output for better debugging
      const errorMsg = result.error || result.output || "Sync failed";
      // Extract the most relevant error lines
      const errorLines = errorMsg.split('\n')
        .filter(line => line.includes('ERROR') || line.includes('NOTICE') || line.includes('Failed'))
        .slice(-5)  // Last 5 error lines
        .join('\n');

      return {
        success: false,
        error: errorLines || errorMsg,
      };
    }
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

    // Close all watchers (none in periodic mode, but keep for compatibility)
    for (const dir of this.directories.values()) {
      if (dir.watcher) {
        dir.watcher.close();
      }
    }

    return "✅ Google Drive sync stopped.";
  }

  /**
   * Get current sync status
   */
  getStatus(): {
    running: boolean;
    directories: number;
    queueSize: number;
    rateLimitTokens: number;
    syncStates: SyncState;
    syncMode: string;
    syncInterval: number;
  } {
    return {
      running: this.isRunning,
      directories: this.directories.size,
      queueSize: this.syncQueue.size,
      rateLimitTokens: this.rateLimiter.getAvailableTokens(),
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

    return `✅ Queued ${paths.length} directories for sync.`;
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
    }
  }

  /**
   * Remove an exclude pattern
   */
  removeExcludePattern(pattern: string): void {
    const index = this.excludePatterns.indexOf(pattern);
    if (index > -1) {
      this.excludePatterns.splice(index, 1);
    }
  }
}

