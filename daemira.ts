#!/usr/bin/env bun
/**
 * DAEMIRA - Clean Entry Point
 *
 * A minimal daemon for Google Drive sync with zero framework bloat.
 * Just instantiate and run - that's it.
 */

import { GoogleDriveSync } from "./src/google-drive/GoogleDriveSync";

// Parse command line arguments
const args = process.argv.slice(2);
const command = args[0];

// Create Google Drive sync instance
const sync = new GoogleDriveSync("gdrive");

// Execute command
try {
  switch (command) {
    case "start":
      console.log(await sync.start());
      // Keep process alive
      process.on("SIGINT", async () => {
        console.log("\n🛑 Shutting down...");
        await sync.stop();
        process.exit(0);
      });
      break;

    case "stop":
      console.log(await sync.stop());
      break;

    case "status":
      const status = sync.getStatus();
      console.log(`📊 Google Drive Sync Status:`);
      console.log(`  Running: ${status.running ? "✅" : "❌"}`);
      console.log(`  Mode: ${status.syncMode} (every ${status.syncInterval}s)`);
      console.log(`  Directories: ${status.directories}`);
      console.log(`  Queue Size: ${status.queueSize}`);
      break;

    case "sync":
      console.log(await sync.syncAll());
      break;

    case "patterns":
      const patterns = sync.getExcludePatterns();
      console.log(`🚫 Exclude Patterns (${patterns.length} total):`);
      patterns.forEach((p, i) => console.log(`  ${i + 1}. ${p}`));
      break;

    default:
      // Default: start sync and show status
      console.log(await sync.start());
      console.log("\n" + sync.getStatus());
      console.log("\n💡 Press Ctrl+C to stop");

      process.on("SIGINT", async () => {
        console.log("\n🛑 Shutting down...");
        await sync.stop();
        process.exit(0);
      });
  }
} catch (error) {
  console.error("❌ Error:", error instanceof Error ? error.message : error);
  process.exit(1);
}
