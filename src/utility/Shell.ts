/**
 * Shell Utility - Execute system commands with proper error handling
 *
 * Features:
 * - Async-only API for non-blocking operations
 * - Timeout support with graceful termination
 * - Structured result with exit code, stdout, stderr
 * - Optional streaming callbacks for real-time output
 * - Integration with Logger for command logging
 */

import { spawn } from "node:child_process";
import { Logger } from "./Logger";

export interface ShellResult {
	exitCode: number;
	stdout: string;
	stderr: string;
	timedOut: boolean;
	command: string;
}

export interface ShellOptions {
	timeout?: number; // Timeout in milliseconds (default: 30000)
	onStdout?: (line: string) => void; // Callback for each stdout line
	onStderr?: (line: string) => void; // Callback for each stderr line
	cwd?: string; // Working directory
	env?: Record<string, string>; // Environment variables
	logCommand?: boolean; // Log command execution (default: true)
}

export class Shell {
	private static logger = Logger.getInstance();

	private constructor() {
		// Prevent instantiation - this is a static utility class
	}

	/**
	 * Execute a shell command asynchronously
	 *
	 * @param command - The command to execute (passed to bash -c)
	 * @param options - Execution options
	 * @returns Promise resolving to ShellResult
	 */
	static async execute(
		command: string,
		options: ShellOptions = {},
	): Promise<ShellResult> {
		const {
			timeout = 30000,
			onStdout,
			onStderr,
			cwd,
			env,
			logCommand = true,
		} = options;

		if (logCommand) {
			this.logger.debug(`Executing: ${command}`);
		}

		return new Promise((resolve) => {
			const proc = spawn("bash", ["-c", command], {
				stdio: ["ignore", "pipe", "pipe"],
				cwd,
				env: { ...process.env, ...env },
			});

			let stdout = "";
			let stderr = "";
			let completed = false;
			let timeoutHandle: NodeJS.Timeout | undefined;

			// Setup timeout handler
			if (timeout > 0) {
				timeoutHandle = setTimeout(() => {
					if (!completed) {
						completed = true;
						proc.kill("SIGTERM");
						this.logger.warn(
							`Command timed out after ${timeout}ms: ${command}`,
						);
						resolve({
							exitCode: -1,
							stdout,
							stderr: stderr || `Timed out after ${timeout}ms`,
							timedOut: true,
							command,
						});
					}
				}, timeout);
			}

			// Capture stdout
			proc.stdout.on("data", (data: Buffer) => {
				const text = data.toString();
				stdout += text;
				if (onStdout) {
					text.split("\n").forEach((line) => {
						if (line.trim()) onStdout(line);
					});
				}
			});

			// Capture stderr
			proc.stderr.on("data", (data: Buffer) => {
				const text = data.toString();
				stderr += text;
				if (onStderr) {
					text.split("\n").forEach((line) => {
						if (line.trim()) onStderr(line);
					});
				}
			});

			// Handle process completion
			proc.on("close", (code) => {
				if (!completed) {
					completed = true;
					if (timeoutHandle) clearTimeout(timeoutHandle);
					resolve({
						exitCode: code || 0,
						stdout,
						stderr,
						timedOut: false,
						command,
					});
				}
			});

			// Handle process errors
			proc.on("error", (err) => {
				if (!completed) {
					completed = true;
					if (timeoutHandle) clearTimeout(timeoutHandle);
					this.logger.error(`Command error: ${err.message}`);
					resolve({
						exitCode: -1,
						stdout,
						stderr: err.message,
						timedOut: false,
						command,
					});
				}
			});
		});
	}
}
