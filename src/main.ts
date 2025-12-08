#!/usr/bin/env bun
/**
 * Daemira - Personal System Utility Daemon
 *
 * Single entry point for all daemon operations:
 * - Google Drive bidirectional sync
 * - Automated system updates
 * - System monitoring and maintenance
 */

import { Daemira } from "./Daemira";
import { Logger, interceptConsole } from "./utility/Logger";

// Initialize logger and intercept console
interceptConsole();
const logger = Logger.getInstance();

// Parse command line arguments
const args = process.argv.slice(2);
const command = args[0];
const commandArgs = args.slice(1);

// Create Daemira instance
const daemon = new Daemira();

// Setup signal handlers for graceful shutdown
process.on("SIGINT", async () => {
	logger.info("Shutting down Daemira...");
	await daemon.stopGoogleDriveSync();
	process.exit(0);
});

process.on("SIGTERM", async () => {
	logger.info("Shutting down Daemira...");
	await daemon.stopGoogleDriveSync();
	process.exit(0);
});

// Execute command
try {
	let result: string;

	switch (command) {
		// Google Drive commands
		case "gdrive:start":
			result = await daemon.startGoogleDriveSync();
			console.log(result);
			console.log("\nPress Ctrl+C to stop");
			// Keep process alive
			await new Promise(() => { });
			break;

		case "gdrive:stop":
			result = await daemon.stopGoogleDriveSync();
			console.log(result);
			break;

		case "gdrive:status":
			result = daemon.getGoogleDriveSyncStatus();
			console.log(result);
			break;

		case "gdrive:sync":
			result = await daemon.syncAllGoogleDrive();
			console.log(result);
			break;

		case "gdrive:patterns":
			result = daemon.getGoogleDriveExcludePatterns();
			console.log(result);
			break;

		case "gdrive:exclude":
			if (!commandArgs[0]) {
				console.error("Error: Pattern required. Usage: daemira gdrive:exclude <pattern>");
				process.exit(1);
			}
			result = daemon.addGoogleDriveExcludePattern(commandArgs[0]);
			console.log(result);
			break;

		// System update commands
		case "system:update":
			result = await daemon.runSystemUpdate();
			console.log(result);
			break;

		case "system:status":
			result = daemon.getSystemUpdateStatus();
			console.log(result);
			break;

		// Default: start daemon with all services
		case "start":
		default:
			logger.info("Starting Daemira daemon...");
			result = await daemon.defaultFunction();
			console.log(result);
			// Keep process alive
			await new Promise(() => { });
	}
} catch (error) {
	logger.error(`Error: ${error instanceof Error ? error.message : error}`);
	console.error("Error:", error instanceof Error ? error.message : error);
	process.exit(1);
}
