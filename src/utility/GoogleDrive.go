package utility

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Configuration constants
const (
	DebounceDelayMS        = 2000  // 2 seconds
	PeriodicSyncDelayMS    = 30000 // 30 seconds
	QueueProcessIntervalMS = 1000  // 1 second
)

// SyncDirectory represents a directory to sync
type SyncDirectory struct {
	LocalPath        string
	RemotePath       string
	NeedsInitialSync bool
}

// SyncOperation represents a queued sync operation
type SyncOperation struct {
	Directory string
	Timestamp time.Time
	Retries   int
}

// SyncStatus represents the status of a directory sync
type SyncStatus string

const (
	StatusIdle    SyncStatus = "idle"
	StatusSyncing SyncStatus = "syncing"
	StatusError   SyncStatus = "error"
)

// SyncState holds the state of all directory syncs
type SyncState struct {
	LastSyncTime  map[string]time.Time
	SyncStatus    map[string]SyncStatus
	ErrorMessages map[string]string
	mu            sync.RWMutex
}

// GoogleDrive manages Google Drive synchronization using rclone
type GoogleDrive struct {
	logger             *Logger
	shell              *Shell
	directories        map[string]*SyncDirectory
	syncQueue          map[string]*SyncOperation
	debounceTimers     map[string]*time.Timer
	isRunning          bool
	remoteName         string
	debounceDelay      time.Duration
	periodicSyncDelay  time.Duration
	excludePatterns    []string
	state              *SyncState
	processInterval    *time.Ticker
	periodicSyncTicker *time.Ticker
	cancelFunc         context.CancelFunc
	mu                 sync.RWMutex
	wg                 sync.WaitGroup
}

// NewGoogleDrive creates a new GoogleDrive instance
func NewGoogleDrive(logger *Logger, remoteName string) *GoogleDrive {
	if remoteName == "" {
		remoteName = "gdrive"
	}

	gd := &GoogleDrive{
		logger:            logger,
		shell:             NewShell(logger),
		directories:       make(map[string]*SyncDirectory),
		syncQueue:         make(map[string]*SyncOperation),
		debounceTimers:    make(map[string]*time.Timer),
		remoteName:        remoteName,
		debounceDelay:     DebounceDelayMS * time.Millisecond,
		periodicSyncDelay: PeriodicSyncDelayMS * time.Millisecond,
		state: &SyncState{
			LastSyncTime:  make(map[string]time.Time),
			SyncStatus:    make(map[string]SyncStatus),
			ErrorMessages: make(map[string]string),
		},
	}

	gd.setupExcludePatterns()
	gd.logger.Info("GoogleDrive initialized with remote: %s", remoteName)

	return gd
}

// setupExcludePatterns initializes common exclude patterns
func (gd *GoogleDrive) setupExcludePatterns() {
	gd.excludePatterns = []string{
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

		// User-specific excludes (examples)
		"IK Multimedia/**",
		"Teamruns/**",
	}
}

// GetExcludeArgs returns rclone exclude arguments
func (gd *GoogleDrive) GetExcludeArgs() []string {
	args := make([]string, 0, len(gd.excludePatterns)*2)
	for _, pattern := range gd.excludePatterns {
		args = append(args, "--exclude", pattern)
	}
	return args
}

// AddDirectory adds a directory to sync
func (gd *GoogleDrive) AddDirectory(localPath, remotePath string) {
	gd.mu.Lock()
	defer gd.mu.Unlock()

	// Expand ~ to home directory
	if strings.HasPrefix(localPath, "~") {
		homeDir, _ := os.UserHomeDir()
		localPath = filepath.Join(homeDir, localPath[1:])
	}

	gd.directories[localPath] = &SyncDirectory{
		LocalPath:        localPath,
		RemotePath:       remotePath,
		NeedsInitialSync: true,
	}

	gd.state.mu.Lock()
	gd.state.SyncStatus[localPath] = StatusIdle
	gd.state.mu.Unlock()

	gd.logger.Debug("Added directory: %s -> %s", localPath, remotePath)
}

// SetupDefaultDirectories adds default home directories
func (gd *GoogleDrive) SetupDefaultDirectories() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	defaultDirs := []struct {
		local  string
		remote string
	}{
		{filepath.Join(homeDir, "Documents"), fmt.Sprintf("%s:Documents", gd.remoteName)},
		{filepath.Join(homeDir, "Downloads"), fmt.Sprintf("%s:Downloads", gd.remoteName)},
		{filepath.Join(homeDir, "Pictures"), fmt.Sprintf("%s:Pictures", gd.remoteName)},
		{filepath.Join(homeDir, "Desktop"), fmt.Sprintf("%s:Desktop", gd.remoteName)},
		{filepath.Join(homeDir, "Music"), fmt.Sprintf("%s:Music", gd.remoteName)},
		{filepath.Join(homeDir, "Source"), fmt.Sprintf("%s:Source", gd.remoteName)},
		{filepath.Join(homeDir, ".config"), fmt.Sprintf("%s:.config", gd.remoteName)},
	}

	for _, dir := range defaultDirs {
		gd.AddDirectory(dir.local, dir.remote)
	}

	return nil
}

// Start begins watching and syncing directories
func (gd *GoogleDrive) Start(ctx context.Context) error {
	gd.mu.Lock()
	if gd.isRunning {
		gd.mu.Unlock()
		return fmt.Errorf("google Drive sync is already running")
	}

	// Check rclone configuration
	if err := gd.checkConfig(ctx); err != nil {
		gd.mu.Unlock()
		return err
	}

	gd.logger.Info("Connection to Google Drive verified")

	// Setup default directories if none configured (unlock first to avoid deadlock)
	needsSetup := len(gd.directories) == 0
	gd.mu.Unlock()

	if needsSetup {
		gd.logger.Info("Setting up default directories...")
		if err := gd.SetupDefaultDirectories(); err != nil {
			return err
		}
		gd.logger.Info("Default directories configured: %d directories", len(gd.directories))
	}

	gd.mu.Lock()
	gd.isRunning = true

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	gd.cancelFunc = cancel
	gd.mu.Unlock()

	gd.logger.Info("Marking directories for initial sync...")
	// Mark all directories as needing initial sync (check happens in background)
	// This prevents blocking on rclone checks during startup
	for path, dir := range gd.directories {
		dir.NeedsInitialSync = true
		gd.logger.Debug("Directory %s will be checked for initial sync", path)
	}
	gd.logger.Info("Marked %d directories for sync", len(gd.directories))

	// Start background workers first (non-blocking)
	gd.logger.Info("Starting background workers...")
	gd.startWorkers(ctx)
	gd.logger.Info("Background workers started")

	// Check which directories need initial sync in background (non-blocking)
	go func() {
		for path, dir := range gd.directories {
			needsSync, err := gd.needsResync(ctx, dir.LocalPath, dir.RemotePath)
			if err != nil {
				gd.logger.Warn("Failed to check resync for %s: %v", path, err)
				needsSync = true
			}
			dir.NeedsInitialSync = needsSync

			if needsSync {
				gd.logger.Info("Directory %s needs initial sync", path)
			} else {
				gd.logger.Info("Directory %s is already synced", path)
			}
		}
	}()

	dirCount := len(gd.directories)
	gd.logger.Info("Google Drive sync started. Syncing %d directories every %d seconds",
		dirCount, int(gd.periodicSyncDelay.Seconds()))
	fmt.Printf("✓ Google Drive sync started. Syncing %d directories every %d seconds\n",
		dirCount, int(gd.periodicSyncDelay.Seconds()))

	// Perform initial syncs in background (non-blocking)
	go func() {
		gd.logger.Info("Starting initial syncs in background...")
		if err := gd.performInitialSyncs(ctx); err != nil {
			gd.logger.Error("Initial syncs failed: %v", err)
		} else {
			gd.logger.Info("Initial syncs completed")
		}
	}()

	return nil
}

// startWorkers starts background goroutines for queue processing and periodic syncs
func (gd *GoogleDrive) startWorkers(ctx context.Context) {
	gd.logger.Info("startWorkers: Creating queue processor...")
	// Queue processor
	gd.processInterval = time.NewTicker(QueueProcessIntervalMS * time.Millisecond)
	gd.wg.Add(1)
	go func() {
		defer gd.wg.Done()
		gd.logger.Debug("Queue processor goroutine started")
		for {
			select {
			case <-ctx.Done():
				gd.logger.Debug("Queue processor stopping (context cancelled)")
				return
			case <-gd.processInterval.C:
				gd.processQueue(ctx)
			}
		}
	}()

	gd.logger.Info("startWorkers: Creating periodic sync timer...")
	// Periodic sync timer
	gd.periodicSyncTicker = time.NewTicker(gd.periodicSyncDelay)
	gd.wg.Add(1)
	go func() {
		defer gd.wg.Done()
		gd.logger.Debug("Periodic sync timer goroutine started")
		for {
			select {
			case <-ctx.Done():
				gd.logger.Debug("Periodic sync timer stopping (context cancelled)")
				return
			case <-gd.periodicSyncTicker.C:
				gd.logger.Debug("Periodic sync triggered for all directories")
				gd.mu.RLock()
				for path := range gd.directories {
					gd.QueueSync(path)
				}
				gd.mu.RUnlock()
			}
		}
	}()

	gd.logger.Info("startWorkers: Queueing all directories for immediate sync...")
	// Queue all directories for immediate sync after startup
	// Need to unlock before QueueSync (which needs write lock)
	gd.mu.RLock()
	paths := make([]string, 0, len(gd.directories))
	for path := range gd.directories {
		paths = append(paths, path)
	}
	gd.mu.RUnlock()

	queueCount := 0
	for _, path := range paths {
		gd.QueueSync(path)
		queueCount++
	}
	gd.logger.Info("startWorkers: Queued %d directories for sync", queueCount)
	gd.logger.Info("startWorkers: All workers started successfully")
}

// performInitialSyncs performs initial syncs for directories that need it
func (gd *GoogleDrive) performInitialSyncs(ctx context.Context) error {
	for path, dir := range gd.directories {
		if !dir.NeedsInitialSync {
			continue
		}

		gd.logger.Info("Performing initial sync for %s...", path)
		gd.state.mu.Lock()
		gd.state.SyncStatus[path] = StatusSyncing
		gd.state.mu.Unlock()

		// Clear any stale lock files
		if err := gd.clearLocks(dir.LocalPath, dir.RemotePath); err != nil {
			gd.logger.Debug("Failed to clear locks: %v", err)
		}

		gd.logger.Debug("Starting initial bisync...")

		if err := gd.executeBisync(ctx, dir.LocalPath, dir.RemotePath, true); err != nil {
			gd.state.mu.Lock()
			gd.state.SyncStatus[path] = StatusError
			gd.state.ErrorMessages[path] = err.Error()
			gd.state.mu.Unlock()
			gd.logger.Error("Initial sync failed for %s: %v", path, err)
			continue
		}

		dir.NeedsInitialSync = false
		gd.state.mu.Lock()
		gd.state.LastSyncTime[path] = time.Now()
		gd.state.SyncStatus[path] = StatusIdle
		gd.state.mu.Unlock()
		gd.logger.Info("Initial sync completed for %s", path)
	}

	return nil
}

// QueueSync adds a directory to the sync queue
func (gd *GoogleDrive) QueueSync(directoryPath string) {
	gd.mu.Lock()
	defer gd.mu.Unlock()

	if !gd.isRunning {
		return
	}

	gd.syncQueue[directoryPath] = &SyncOperation{
		Directory: directoryPath,
		Timestamp: time.Now(),
		Retries:   0,
	}
}

// processQueue processes queued sync operations (one at a time)
func (gd *GoogleDrive) processQueue(ctx context.Context) {
	gd.mu.Lock()
	if len(gd.syncQueue) == 0 {
		gd.mu.Unlock()
		return
	}

	// Get oldest item from queue
	var oldestPath string
	var oldestTime time.Time
	for path, op := range gd.syncQueue {
		if oldestPath == "" || op.Timestamp.Before(oldestTime) {
			oldestPath = path
			oldestTime = op.Timestamp
		}
	}

	// Check if directory is already syncing
	gd.state.mu.RLock()
	status := gd.state.SyncStatus[oldestPath]
	gd.state.mu.RUnlock()

	if status == StatusSyncing {
		gd.mu.Unlock()
		return
	}

	// Remove from queue
	delete(gd.syncQueue, oldestPath)
	gd.mu.Unlock()

	// Sync directory
	gd.syncDirectory(ctx, oldestPath)
}

// syncDirectory syncs a specific directory
func (gd *GoogleDrive) syncDirectory(ctx context.Context, directoryPath string) {
	gd.mu.RLock()
	dir, exists := gd.directories[directoryPath]
	gd.mu.RUnlock()

	if !exists {
		gd.logger.Error("Directory not found: %s", directoryPath)
		return
	}

	gd.state.mu.Lock()
	gd.state.SyncStatus[directoryPath] = StatusSyncing
	gd.state.mu.Unlock()

	gd.logger.Info("Syncing %s...", directoryPath)

	// Clear any stale lock files before syncing
	if err := gd.clearLocks(dir.LocalPath, dir.RemotePath); err != nil {
		gd.logger.Debug("Failed to clear locks: %v", err)
	}

	if err := gd.executeBisync(ctx, dir.LocalPath, dir.RemotePath, false); err != nil {
		gd.state.mu.Lock()
		gd.state.SyncStatus[directoryPath] = StatusError
		gd.state.ErrorMessages[directoryPath] = err.Error()
		gd.state.mu.Unlock()
		gd.logger.Error("Sync failed for %s: %v", directoryPath, err)
		return
	}

	gd.state.mu.Lock()
	gd.state.LastSyncTime[directoryPath] = time.Now()
	gd.state.SyncStatus[directoryPath] = StatusIdle
	delete(gd.state.ErrorMessages, directoryPath)
	gd.state.mu.Unlock()

	gd.logger.Info("Synced %s", directoryPath)
}

// executeBisync executes rclone bisync command
func (gd *GoogleDrive) executeBisync(ctx context.Context, localPath, remotePath string, isInitial bool) error {
	args := []string{
		"bisync",
		localPath,
		remotePath,
	}
	args = append(args, gd.GetExcludeArgs()...)
	args = append(args,
		"--resilient",
		"--recover",
		"--conflict-resolve", "newer",
		"--conflict-loser", "num",
		"--create-empty-src-dirs",
		"--skip-links",
		"--progress",
		"--stats", "30s",
		"--max-size", "10G",
		"--drive-chunk-size", "64M",
		"--transfers", "4",
		"--checkers", "8",
	)

	if isInitial {
		args = append(args, "--resync")
	}

	// Build command with proper quoting for arguments that contain spaces
	// This prevents bash from splitting arguments like "IK Multimedia/**" into two separate arguments
	quotedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.Contains(arg, " ") {
			// Use single quotes for shell safety, but escape single quotes inside
			quoted := strings.ReplaceAll(arg, "'", "'\"'\"'")
			quotedArgs = append(quotedArgs, "'"+quoted+"'")
		} else {
			quotedArgs = append(quotedArgs, arg)
		}
	}
	command := "rclone " + strings.Join(quotedArgs, " ")

	lastProgressTime := time.Now()
	result, err := gd.shell.Execute(ctx, command, &ExecOptions{
		Timeout: 0, // No timeout for large syncs
		StdoutCallback: func(line string) {
			if !strings.Contains(line, "Can't follow symlink") {
				now := time.Now()
				// Log important information about deletions and transfers
				if strings.Contains(line, "Deleted:") ||
					strings.Contains(line, "Transferred:") ||
					strings.Contains(line, "INFO") ||
					strings.Contains(line, "Deleting") ||
					strings.Contains(line, "Copied") ||
					now.Sub(lastProgressTime) > 5*time.Second {
					gd.logger.Info("  %s", line)
					lastProgressTime = now
				} else {
					gd.logger.Debug("  %s", line)
				}
			}
		},
		StderrCallback: func(line string) {
			if !strings.Contains(line, "Can't follow symlink") {
				// Log errors and important notices
				if strings.Contains(line, "ERROR") ||
					strings.Contains(line, "NOTICE") ||
					strings.Contains(line, "Deleted") ||
					strings.Contains(line, "Deleting") {
					gd.logger.Info("  %s", line)
				} else {
					gd.logger.Debug("  %s", line)
				}
			}
		},
	})

	if err != nil {
		return fmt.Errorf("bisync failed: %w", err)
	}

	if result.TimedOut {
		return fmt.Errorf("bisync timed out unexpectedly")
	}

	if result.ExitCode != 0 {
		// Check if error is due to lock file
		errorMsg := result.Stderr
		if errorMsg == "" {
			errorMsg = result.Stdout
		}

		// Check if remote directory doesn't exist
		remoteDirMissing := strings.Contains(errorMsg, "directory not found") &&
			strings.Contains(errorMsg, "error reading source root directory")

		// Check for missing cache files (path1.lst, path2.lst) - requires resync
		needsResync := strings.Contains(errorMsg, "Failed loading prior Path") ||
			strings.Contains(errorMsg, "no such file or directory") ||
			strings.Contains(errorMsg, "path1.lst") ||
			strings.Contains(errorMsg, "path2.lst") ||
			strings.Contains(errorMsg, "Bisync aborted. Please try again")

		// If remote directory doesn't exist, create it first
		if remoteDirMissing {
			gd.logger.Warn("Remote directory %s doesn't exist on Google Drive, creating it...", remotePath)
			// Create the remote directory using rclone mkdir
			mkdirCmd := fmt.Sprintf("rclone mkdir %s", remotePath)
			mkdirResult, mkdirErr := gd.shell.Execute(ctx, mkdirCmd, &ExecOptions{Timeout: 30 * time.Second})
			if mkdirErr == nil && mkdirResult.ExitCode == 0 {
				gd.logger.Info("Remote directory created successfully, retrying sync with --resync...")
				// Now retry with --resync since this is a new directory
				resyncArgs := []string{
					"bisync",
					localPath,
					remotePath,
				}
				resyncArgs = append(resyncArgs, gd.GetExcludeArgs()...)
				resyncArgs = append(resyncArgs,
					"--resync",
					"--resilient",
					"--recover",
					"--conflict-resolve", "newer",
					"--conflict-loser", "num",
					"--create-empty-src-dirs",
					"--skip-links",
					"--progress",
					"--stats", "30s",
					"--max-size", "10G",
					"--drive-chunk-size", "64M",
					"--transfers", "4",
					"--checkers", "8",
				)

				quotedResyncArgs := make([]string, 0, len(resyncArgs))
				for _, arg := range resyncArgs {
					if strings.Contains(arg, " ") {
						quoted := strings.ReplaceAll(arg, "'", "'\"'\"'")
						quotedResyncArgs = append(quotedResyncArgs, "'"+quoted+"'")
					} else {
						quotedResyncArgs = append(quotedResyncArgs, arg)
					}
				}
				resyncCommand := "rclone " + strings.Join(quotedResyncArgs, " ")

				resyncResult, resyncErr := gd.shell.Execute(ctx, resyncCommand, &ExecOptions{
					Timeout: 0,
					StdoutCallback: func(line string) {
						if !strings.Contains(line, "Can't follow symlink") {
							now := time.Now()
							if strings.Contains(line, "Transferred:") ||
								strings.Contains(line, "INFO") ||
								strings.Contains(line, "Deleted:") ||
								strings.Contains(line, "Deleting") ||
								strings.Contains(line, "Copied") ||
								now.Sub(lastProgressTime) > 5*time.Second {
								gd.logger.Info("  %s", line)
								lastProgressTime = now
							} else {
								gd.logger.Debug("  %s", line)
							}
						}
					},
					StderrCallback: func(line string) {
						if !strings.Contains(line, "Can't follow symlink") {
							if strings.Contains(line, "ERROR") ||
								strings.Contains(line, "NOTICE") ||
								strings.Contains(line, "Deleted") ||
								strings.Contains(line, "Deleting") {
								gd.logger.Info("  %s", line)
							} else {
								gd.logger.Debug("  %s", line)
							}
						}
					},
				})

				if resyncErr == nil && !resyncResult.TimedOut && resyncResult.ExitCode == 0 {
					gd.logger.Info("Sync completed successfully after creating remote directory")
					return nil
				}
				// If resync failed, fall through to error handling
				if resyncResult != nil {
					result = resyncResult
					errorMsg = resyncResult.Stderr
					if errorMsg == "" {
						errorMsg = resyncResult.Stdout
					}
				}
			} else {
				gd.logger.Warn("Failed to create remote directory: %v", mkdirErr)
				if mkdirResult != nil {
					gd.logger.Warn("mkdir output: %s", mkdirResult.Stderr)
				}
			}
		}

		// Check for lock file error and automatically retry after clearing
		if strings.Contains(errorMsg, "prior lock file found") || strings.Contains(errorMsg, "lock file found") {
			gd.logger.Warn("Lock file detected, clearing and retrying...")
			if err := gd.clearLocks(localPath, remotePath); err != nil {
				gd.logger.Warn("Failed to clear lock file: %v", err)
			} else {
				gd.logger.Info("Lock file cleared, retrying sync...")
				// Retry the sync once after clearing lock
				retryResult, retryErr := gd.shell.Execute(ctx, command, &ExecOptions{
					Timeout: 0, // No timeout for large syncs
					StdoutCallback: func(line string) {
						if !strings.Contains(line, "Can't follow symlink") {
							now := time.Now()
							if strings.Contains(line, "Transferred:") ||
								strings.Contains(line, "INFO") ||
								strings.Contains(line, "Deleted:") ||
								strings.Contains(line, "Deleting") ||
								now.Sub(lastProgressTime) > 5*time.Second {
								gd.logger.Info("  %s", line)
								lastProgressTime = now
							} else {
								gd.logger.Debug("  %s", line)
							}
						}
					},
					StderrCallback: func(line string) {
						if !strings.Contains(line, "Can't follow symlink") {
							if strings.Contains(line, "ERROR") ||
								strings.Contains(line, "NOTICE") ||
								strings.Contains(line, "Deleted") ||
								strings.Contains(line, "Deleting") {
								gd.logger.Info("  %s", line)
							} else {
								gd.logger.Debug("  %s", line)
							}
						}
					},
				})

				if retryErr == nil && !retryResult.TimedOut && retryResult.ExitCode == 0 {
					gd.logger.Info("Sync succeeded after clearing lock file")
					return nil
				}
				// If retry also failed, fall through to error handling
				if retryResult != nil {
					result = retryResult
					errorMsg = retryResult.Stderr
					if errorMsg == "" {
						errorMsg = retryResult.Stdout
					}
					// Re-check if resync is needed after retry
					needsResync = strings.Contains(errorMsg, "Failed loading prior Path") ||
						strings.Contains(errorMsg, "no such file or directory") ||
						strings.Contains(errorMsg, "path1.lst") ||
						strings.Contains(errorMsg, "path2.lst") ||
						strings.Contains(errorMsg, "Bisync aborted. Please try again")
				}
			}
		}

		// If cache files are missing, retry with --resync to rebuild cache
		if needsResync && !isInitial {
			gd.logger.Warn("Bisync cache files missing or corrupted, performing resync to rebuild cache...")
			// Build resync command
			resyncArgs := []string{
				"bisync",
				localPath,
				remotePath,
			}
			resyncArgs = append(resyncArgs, gd.GetExcludeArgs()...)
			resyncArgs = append(resyncArgs,
				"--resync",
				"--resilient",
				"--recover",
				"--conflict-resolve", "newer",
				"--conflict-loser", "num",
				"--create-empty-src-dirs",
				"--skip-links",
				"--progress",
				"--stats", "30s",
				"--max-size", "10G",
				"--drive-chunk-size", "64M",
				"--transfers", "4",
				"--checkers", "8",
			)

			quotedResyncArgs := make([]string, 0, len(resyncArgs))
			for _, arg := range resyncArgs {
				if strings.Contains(arg, " ") {
					quoted := strings.ReplaceAll(arg, "'", "'\"'\"'")
					quotedResyncArgs = append(quotedResyncArgs, "'"+quoted+"'")
				} else {
					quotedResyncArgs = append(quotedResyncArgs, arg)
				}
			}
			resyncCommand := "rclone " + strings.Join(quotedResyncArgs, " ")

			gd.logger.Info("Running resync to rebuild cache and sync deletions...")
			resyncResult, resyncErr := gd.shell.Execute(ctx, resyncCommand, &ExecOptions{
				Timeout: 0, // No timeout for large syncs
				StdoutCallback: func(line string) {
					if !strings.Contains(line, "Can't follow symlink") {
						now := time.Now()
						if strings.Contains(line, "Transferred:") ||
							strings.Contains(line, "INFO") ||
							strings.Contains(line, "Deleted:") ||
							strings.Contains(line, "Deleting") ||
							strings.Contains(line, "Copied") ||
							now.Sub(lastProgressTime) > 5*time.Second {
							gd.logger.Info("  %s", line)
							lastProgressTime = now
						} else {
							gd.logger.Debug("  %s", line)
						}
					}
				},
				StderrCallback: func(line string) {
					if !strings.Contains(line, "Can't follow symlink") {
						if strings.Contains(line, "ERROR") ||
							strings.Contains(line, "NOTICE") ||
							strings.Contains(line, "Deleted") ||
							strings.Contains(line, "Deleting") {
							gd.logger.Info("  %s", line)
						} else {
							gd.logger.Debug("  %s", line)
						}
					}
				},
			})

			if resyncErr == nil && !resyncResult.TimedOut && resyncResult.ExitCode == 0 {
				gd.logger.Info("Resync completed successfully, cache rebuilt and deletions synced")
				return nil
			}
			// If resync also failed, fall through to error handling
			if resyncResult != nil {
				result = resyncResult
				errorMsg = resyncResult.Stderr
				if errorMsg == "" {
					errorMsg = resyncResult.Stdout
				}
			}
		}

		// Extract relevant error lines
		lines := strings.Split(errorMsg, "\n")
		errorLines := []string{}
		for _, line := range lines {
			if strings.Contains(line, "ERROR") ||
				strings.Contains(line, "NOTICE") ||
				strings.Contains(line, "Failed") {
				errorLines = append(errorLines, line)
			}
		}
		if len(errorLines) > 5 {
			errorLines = errorLines[len(errorLines)-5:]
		}

		// Log full error for debugging
		gd.logger.Error("Rclone bisync error (exit code %d) for %s -> %s:\nStderr: %s\nStdout: %s",
			result.ExitCode, localPath, remotePath, result.Stderr, result.Stdout)

		if len(errorLines) > 0 {
			return fmt.Errorf("sync failed: %s", strings.Join(errorLines, "\n"))
		}
		return fmt.Errorf("sync failed with exit code %d, check logs for details", result.ExitCode)
	}

	return nil
}

// Stop stops all watchers and sync operations
func (gd *GoogleDrive) Stop() error {
	gd.mu.Lock()
	if !gd.isRunning {
		gd.mu.Unlock()
		return fmt.Errorf("google Drive sync is not running")
	}

	gd.isRunning = false

	// Cancel context
	if gd.cancelFunc != nil {
		gd.cancelFunc()
	}

	// Stop tickers
	if gd.processInterval != nil {
		gd.processInterval.Stop()
	}
	if gd.periodicSyncTicker != nil {
		gd.periodicSyncTicker.Stop()
	}

	// Clear timers
	for _, timer := range gd.debounceTimers {
		timer.Stop()
	}
	gd.debounceTimers = make(map[string]*time.Timer)

	gd.mu.Unlock()

	// Wait for workers to finish
	gd.wg.Wait()

	gd.logger.Info("Google Drive sync stopped")
	return nil
}

// GetStatus returns current sync status
func (gd *GoogleDrive) GetStatus() map[string]interface{} {
	gd.mu.RLock()
	defer gd.mu.RUnlock()

	gd.state.mu.RLock()
	defer gd.state.mu.RUnlock()

	return map[string]interface{}{
		"running":      gd.isRunning,
		"directories":  len(gd.directories),
		"queueSize":    len(gd.syncQueue),
		"syncMode":     "periodic",
		"syncInterval": int(gd.periodicSyncDelay.Seconds()),
		"syncStates":   gd.state,
	}
}

// SyncAll queues all directories for immediate sync
func (gd *GoogleDrive) SyncAll() string {
	gd.mu.RLock()
	defer gd.mu.RUnlock()

	if !gd.isRunning {
		return "Google Drive sync is not running. Start it first."
	}

	count := 0
	for path := range gd.directories {
		gd.QueueSync(path)
		count++
	}

	return fmt.Sprintf("Queued %d directories for sync", count)
}

// SyncDirectory queues a specific directory for immediate sync
func (gd *GoogleDrive) SyncDirectory(directoryPath string) string {
	gd.mu.RLock()
	defer gd.mu.RUnlock()

	if !gd.isRunning {
		return "Google Drive sync is not running. Start it first."
	}

	// Check if directory exists
	if _, exists := gd.directories[directoryPath]; !exists {
		return fmt.Sprintf("Directory not found: %s", directoryPath)
	}

	gd.QueueSync(directoryPath)
	return fmt.Sprintf("Queued %s for immediate sync", directoryPath)
}

// ResyncDirectory forces a resync of a specific directory (rebuilds cache and syncs deletions)
func (gd *GoogleDrive) ResyncDirectory(ctx context.Context, directoryPath string) error {
	gd.mu.RLock()
	dir, exists := gd.directories[directoryPath]
	gd.mu.RUnlock()

	if !exists {
		return fmt.Errorf("directory not found: %s", directoryPath)
	}

	gd.logger.Info("Forcing resync of %s (will rebuild cache and sync deletions)...", directoryPath)

	// Clear locks first
	if err := gd.clearLocks(dir.LocalPath, dir.RemotePath); err != nil {
		gd.logger.Debug("Failed to clear locks: %v", err)
	}

	// Clear bisync cache files to force a true resync
	if err := gd.clearBisyncCache(dir.LocalPath, dir.RemotePath); err != nil {
		gd.logger.Warn("Failed to clear bisync cache: %v (will continue anyway)", err)
	} else {
		gd.logger.Info("Cleared bisync cache files to force fresh resync")
	}

	// First, use rclone sync with --delete-after to ensure deletions are synced
	// This ensures files that exist on remote but not locally are deleted
	gd.logger.Info("Syncing deletions from local to remote...")
	syncArgs := []string{
		"sync",
		dir.LocalPath,
		dir.RemotePath,
		"--delete-after",
		"--progress",
		"--stats", "30s",
		"--max-size", "10G",
		"--drive-chunk-size", "64M",
		"--transfers", "4",
		"--checkers", "8",
	}
	syncArgs = append(syncArgs, gd.GetExcludeArgs()...)

	quotedSyncArgs := make([]string, 0, len(syncArgs))
	for _, arg := range syncArgs {
		if strings.Contains(arg, " ") {
			quoted := strings.ReplaceAll(arg, "'", "'\"'\"'")
			quotedSyncArgs = append(quotedSyncArgs, "'"+quoted+"'")
		} else {
			quotedSyncArgs = append(quotedSyncArgs, arg)
		}
	}
	syncCommand := "rclone " + strings.Join(quotedSyncArgs, " ")

	syncResult, syncErr := gd.shell.Execute(ctx, syncCommand, &ExecOptions{
		Timeout: 0,
		StdoutCallback: func(line string) {
			if strings.Contains(line, "Deleted:") ||
				strings.Contains(line, "Transferred:") ||
				strings.Contains(line, "Deleting") ||
				strings.Contains(line, "INFO") {
				gd.logger.Info("  %s", line)
			} else {
				gd.logger.Debug("  %s", line)
			}
		},
		StderrCallback: func(line string) {
			if strings.Contains(line, "ERROR") ||
				strings.Contains(line, "NOTICE") ||
				strings.Contains(line, "Deleted") ||
				strings.Contains(line, "Deleting") {
				gd.logger.Info("  %s", line)
			} else {
				gd.logger.Debug("  %s", line)
			}
		},
	})

	if syncErr != nil {
		gd.logger.Warn("Sync with --delete-after failed: %v", syncErr)
	} else if syncResult != nil && syncResult.ExitCode == 0 {
		gd.logger.Info("Deletions synced successfully")
	} else if syncResult != nil {
		gd.logger.Warn("Sync with --delete-after exited with code %d: %s", syncResult.ExitCode, syncResult.Stderr)
	}

	// Now execute bisync resync to rebuild cache and sync both ways
	gd.logger.Info("Rebuilding bisync cache with full resync...")
	return gd.executeBisync(ctx, dir.LocalPath, dir.RemotePath, true)
}

// clearBisyncCache removes all bisync cache files for a directory pair
func (gd *GoogleDrive) clearBisyncCache(localPath, remotePath string) error {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		cacheDir = filepath.Join(homeDir, ".cache")
	}

	bisyncCacheDir := filepath.Join(cacheDir, "rclone", "bisync")

	// Generate cache file name patterns
	sanitizedLocal := strings.ReplaceAll(localPath, "/", "_")
	sanitizedRemote := strings.ReplaceAll(strings.ReplaceAll(remotePath, ":", "_"), "/", "_")
	prefix := fmt.Sprintf("local_%s..%s", sanitizedLocal, sanitizedRemote)

	// Find and remove all cache files for this directory pair
	entries, err := os.ReadDir(bisyncCacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Cache directory doesn't exist, nothing to clear
		}
		return err
	}

	cleared := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			cacheFile := filepath.Join(bisyncCacheDir, entry.Name())
			if err := os.Remove(cacheFile); err != nil {
				gd.logger.Debug("Could not remove cache file %s: %v", cacheFile, err)
			} else {
				cleared++
				gd.logger.Debug("Removed cache file: %s", entry.Name())
			}
		}
	}

	if cleared > 0 {
		gd.logger.Info("Cleared %d bisync cache file(s)", cleared)
	}

	return nil
}

// GetExcludePatterns returns a copy of exclude patterns
func (gd *GoogleDrive) GetExcludePatterns() []string {
	return append([]string{}, gd.excludePatterns...)
}

// AddExcludePattern adds a custom exclude pattern
func (gd *GoogleDrive) AddExcludePattern(pattern string) {
	gd.mu.Lock()
	defer gd.mu.Unlock()

	for _, p := range gd.excludePatterns {
		if p == pattern {
			return
		}
	}

	gd.excludePatterns = append(gd.excludePatterns, pattern)
	gd.logger.Info("Added exclude pattern: %s", pattern)
}

// RemoveExcludePattern removes an exclude pattern
func (gd *GoogleDrive) RemoveExcludePattern(pattern string) {
	gd.mu.Lock()
	defer gd.mu.Unlock()

	for i, p := range gd.excludePatterns {
		if p == pattern {
			gd.excludePatterns = append(gd.excludePatterns[:i], gd.excludePatterns[i+1:]...)
			gd.logger.Info("Removed exclude pattern: %s", pattern)
			return
		}
	}
}

// checkConfig verifies rclone is installed and configured
func (gd *GoogleDrive) checkConfig(ctx context.Context) error {
	// Check if rclone is installed
	result, err := gd.shell.Execute(ctx, "rclone version", &ExecOptions{Timeout: 5 * time.Second})
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("rclone is not installed or not in PATH. Install it with: sudo pacman -S rclone")
	}

	// Check if remote is configured
	result, err = gd.shell.Execute(ctx, "rclone listremotes", &ExecOptions{Timeout: 5 * time.Second})
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to list rclone remotes")
	}

	remotes := strings.Split(result.Stdout, "\n")
	remoteExists := false
	for _, remote := range remotes {
		if strings.HasPrefix(strings.TrimSpace(remote), gd.remoteName+":") {
			remoteExists = true
			break
		}
	}

	if !remoteExists {
		return fmt.Errorf("rclone remote '%s' is not configured. Run 'rclone config' to set it up", gd.remoteName)
	}

	// Test actual connection
	gd.logger.Info("Testing connection to %s...", gd.remoteName)
	result, err = gd.shell.Execute(ctx, fmt.Sprintf("rclone about %s:", gd.remoteName), &ExecOptions{Timeout: 15 * time.Second})

	if err != nil && result != nil && result.TimedOut {
		return fmt.Errorf("connection to %s timed out. Check your internet connection and authentication", gd.remoteName)
	}

	if result.ExitCode != 0 {
		errorMsg := result.Stderr
		if errorMsg == "" {
			errorMsg = result.Stdout
		}
		return fmt.Errorf("failed to connect to %s: %s", gd.remoteName, errorMsg)
	}

	return nil
}

// needsResync checks if a directory needs initial resync
func (gd *GoogleDrive) needsResync(ctx context.Context, localPath, remotePath string) (bool, error) {
	// Try a dry-run bisync to see if it complains about needing resync
	command := fmt.Sprintf("rclone bisync %s %s --dry-run", localPath, remotePath)
	result, err := gd.shell.Execute(ctx, command, &ExecOptions{Timeout: 10 * time.Second})

	if err != nil {
		return true, nil // Assume needs resync on error
	}

	// If it mentions resync or first run, we need to do initial sync
	return strings.Contains(result.Stderr, "--resync") ||
		strings.Contains(result.Stderr, "first run"), nil
}

// clearLocks cleans up bisync lock files
func (gd *GoogleDrive) clearLocks(localPath, remotePath string) error {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		cacheDir = filepath.Join(homeDir, ".cache")
	}

	bisyncCacheDir := filepath.Join(cacheDir, "rclone", "bisync")

	// Generate lock file name pattern
	sanitizedLocal := strings.ReplaceAll(localPath, "/", "_")
	sanitizedRemote := strings.ReplaceAll(strings.ReplaceAll(remotePath, ":", "_"), "/", "_")
	lockFile := filepath.Join(bisyncCacheDir, fmt.Sprintf("local_%s..%s.lck", sanitizedLocal, sanitizedRemote))

	// Try to delete the lock file if it exists
	if _, err := os.Stat(lockFile); err == nil {
		if err := os.Remove(lockFile); err != nil {
			gd.logger.Debug("Could not clear lock file: %v", err)
			return err
		}
		gd.logger.Info("Cleaned up stale lock file")
	}

	return nil
}
