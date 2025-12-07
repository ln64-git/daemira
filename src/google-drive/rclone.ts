import { spawn } from "child_process";
import { existsSync } from "fs";
import { homedir } from "os";
import { join } from "path";

/**
 * Execute an rclone command and return the output
 */
export async function runRcloneCommand(
  args: string[],
  onData?: (line: string) => void,
  timeoutMs: number = 30000
): Promise<{ exitCode: number; output: string; error: string; timedOut?: boolean }> {
  return new Promise((resolve) => {
    const proc = spawn("rclone", args, {
      stdio: ["ignore", "pipe", "pipe"],
    });

    let output = "";
    let error = "";
    let completed = false;

    // Set timeout only if timeoutMs > 0 (0 means no timeout)
    let timeout: NodeJS.Timeout | undefined;
    if (timeoutMs > 0) {
      timeout = setTimeout(() => {
        if (!completed) {
          completed = true;
          proc.kill("SIGTERM");
          resolve({
            exitCode: -1,
            output,
            error: error || "Command timed out after " + (timeoutMs / 1000) + " seconds",
            timedOut: true
          });
        }
      }, timeoutMs);
    }

    proc.stdout.on("data", (data: Buffer) => {
      const text = data.toString();
      output += text;
      if (onData) {
        text.split("\n").forEach((line) => {
          if (line.trim()) onData(line);
        });
      }
    });

    proc.stderr.on("data", (data: Buffer) => {
      const text = data.toString();
      error += text;
      if (onData) {
        text.split("\n").forEach((line) => {
          if (line.trim()) onData(line);
        });
      }
    });

    proc.on("close", (code) => {
      if (!completed) {
        completed = true;
        if (timeout) clearTimeout(timeout);
        resolve({ exitCode: code || 0, output, error });
      }
    });

    proc.on("error", (err) => {
      if (!completed) {
        completed = true;
        if (timeout) clearTimeout(timeout);
        resolve({ exitCode: -1, output, error: err.message });
      }
    });
  });
}

/**
 * Check if rclone is installed and the remote is configured
 */
export async function checkRcloneConfig(remoteName: string = "gdrive"): Promise<{
  installed: boolean;
  remoteConfigured: boolean;
  connectionWorks: boolean;
  error?: string;
}> {
  // Check if rclone is installed
  const versionCheck = await runRcloneCommand(["version"], undefined, 5000);
  if (versionCheck.exitCode !== 0) {
    return {
      installed: false,
      remoteConfigured: false,
      connectionWorks: false,
      error: "rclone is not installed or not in PATH",
    };
  }

  // Check if remote is configured
  const listRemotes = await runRcloneCommand(["listremotes"], undefined, 5000);
  const remotes = listRemotes.output.split("\n").filter((line) => line.trim());
  const remoteExists = remotes.some((remote) => remote.startsWith(`${remoteName}:`));

  if (!remoteExists) {
    return {
      installed: true,
      remoteConfigured: false,
      connectionWorks: false,
      error: `Remote '${remoteName}' is not configured. Run 'rclone config' to set it up.`,
    };
  }

  // Test actual connection to remote
  console.log(`🔍 Testing connection to ${remoteName}...`);
  const aboutResult = await runRcloneCommand(["about", `${remoteName}:`], undefined, 15000);

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
      error: `Failed to connect to ${remoteName}: ${aboutResult.error || aboutResult.output}`,
    };
  }

  return {
    installed: true,
    remoteConfigured: true,
    connectionWorks: true,
  };
}

/**
 * Get the rclone config file path
 */
export function getRcloneConfigPath(): string {
  const configHome = process.env.XDG_CONFIG_HOME || join(homedir(), ".config");
  return join(configHome, "rclone", "rclone.conf");
}

/**
 * Check if a directory needs initial resync (first time sync)
 */
export async function needsResync(localPath: string, remotePath: string): Promise<boolean> {
  // Check if bisync workdir exists for this path pair
  const configDir = process.env.XDG_CONFIG_HOME || join(homedir(), ".config");
  const bisyncDir = join(configDir, "rclone", "bisync");

  // Bisync creates a workdir based on the paths
  // If it doesn't exist or is empty, we need to resync
  if (!existsSync(bisyncDir)) {
    return true;
  }

  // Try a test bisync to see if it complains about needing resync
  const testResult = await runRcloneCommand([
    "bisync",
    localPath,
    remotePath,
    "--dry-run",
  ]);

  // If it mentions resync or first run, we need to do initial sync
  return testResult.error.includes("--resync") || testResult.error.includes("first run");
}

/**
 * Create a directory on the remote if it doesn't exist
 */
export async function ensureRemoteDirectory(remotePath: string): Promise<{
  success: boolean;
  error?: string;
}> {
  // Just try to create the directory - mkdir will succeed if it already exists
  // or if it needs to be created. Add verbose flag for debugging and use shorter timeout.
  const mkdirResult = await runRcloneCommand(["mkdir", remotePath, "-v"], undefined, 15000);

  if (mkdirResult.timedOut) {
    return {
      success: false,
      error: "Command timed out - check your Google Drive connection and authentication",
    };
  }

  // Exit code 0 means success (created or already exists)
  if (mkdirResult.exitCode === 0) {
    return { success: true };
  } else {
    return {
      success: false,
      error: mkdirResult.error || mkdirResult.output || "Failed to create directory",
    };
  }
}

/**
 * Clean up bisync lock files for a path pair
 */
export async function clearBisyncLocks(localPath: string, remotePath: string): Promise<void> {
  const cacheDir = process.env.XDG_CACHE_HOME || join(homedir(), ".cache");
  const bisyncCacheDir = join(cacheDir, "rclone", "bisync");

  // Generate the lock file name pattern matching rclone's format:
  // local__{sanitized_local_path}..{sanitized_remote_path}.lck
  const sanitizedLocal = localPath.replace(/\//g, "_");
  const sanitizedRemote = remotePath.replace(/:/g, "_").replace(/\//g, "_");
  const lockFile = join(bisyncCacheDir, `local_${sanitizedLocal}..${sanitizedRemote}.lck`);

  // Try to delete the lock file if it exists
  try {
    if (existsSync(lockFile)) {
      const { unlinkSync } = await import("fs");
      unlinkSync(lockFile);
      console.log(`  🧹 Cleaned up stale lock file`);
    }
  } catch (error) {
    // Ignore errors - lock file might not exist or already be deleted
  }
}

