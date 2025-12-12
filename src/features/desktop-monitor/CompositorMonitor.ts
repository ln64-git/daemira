import { Shell } from '../../utility/Shell';
import { Logger } from '../../utility/Logger';
import type { CompositorInfo, WorkspaceInfo, WindowInfo } from './types';

export class CompositorMonitor {
	private static instance: CompositorMonitor;
	private logger = Logger.getInstance();

	private constructor() {}

	static getInstance(): CompositorMonitor {
		if (!CompositorMonitor.instance) {
			CompositorMonitor.instance = new CompositorMonitor();
		}
		return CompositorMonitor.instance;
	}

	isAvailable(): boolean {
		return !!process.env.HYPRLAND_INSTANCE_SIGNATURE;
	}

	async getCompositorInfo(): Promise<CompositorInfo> {
		if (!this.isAvailable()) {
			return {
				name: 'unknown',
				version: 'unknown',
				available: false,
			};
		}

		try {
			const result = await Shell.execute('hyprctl version -j', {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				this.logger.error(`hyprctl version failed: ${result.stderr}`);
				return {
					name: 'Hyprland',
					version: 'unknown',
					available: false,
				};
			}

			const versionData = JSON.parse(result.stdout);

			return {
				name: 'Hyprland',
				version: versionData.tag || versionData.commit?.substring(0, 7) || 'unknown',
				available: true,
				branch: versionData.branch,
				commit: versionData.commit,
				buildDate: versionData.date,
			};
		} catch (error) {
			this.logger.error(`Error getting compositor info: ${error}`);
			return {
				name: 'Hyprland',
				version: 'unknown',
				available: false,
			};
		}
	}

	async getWorkspaces(): Promise<WorkspaceInfo[]> {
		if (!this.isAvailable()) {
			return [];
		}

		try {
			const result = await Shell.execute('hyprctl workspaces -j', {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				this.logger.error(`hyprctl workspaces failed: ${result.stderr}`);
				return [];
			}

			return JSON.parse(result.stdout);
		} catch (error) {
			this.logger.error(`Error getting workspaces: ${error}`);
			return [];
		}
	}

	async getActiveWindow(): Promise<WindowInfo | null> {
		if (!this.isAvailable()) {
			return null;
		}

		try {
			const result = await Shell.execute('hyprctl activewindow -j', {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				return null;
			}

			const window = JSON.parse(result.stdout);

			if (!window.address || window.address === '0x') {
				return null;
			}

			return window;
		} catch (error) {
			this.logger.error(`Error getting active window: ${error}`);
			return null;
		}
	}

	async getWindows(): Promise<WindowInfo[]> {
		if (!this.isAvailable()) {
			return [];
		}

		try {
			const result = await Shell.execute('hyprctl clients -j', {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				this.logger.error(`hyprctl clients failed: ${result.stderr}`);
				return [];
			}

			return JSON.parse(result.stdout);
		} catch (error) {
			this.logger.error(`Error getting windows: ${error}`);
			return [];
		}
	}

	async getWindowCount(): Promise<number> {
		const windows = await this.getWindows();
		return windows.length;
	}

	formatCompositorInfo(info: CompositorInfo, workspaces: WorkspaceInfo[], activeWindow: WindowInfo | null, windowCount: number): string {
		const lines = [
			'Compositor Information:',
			`  Name: ${info.name}`,
			`  Version: ${info.version}`,
			`  Available: ${info.available ? 'yes' : 'no'}`,
		];

		if (info.branch) {
			lines.push(`  Branch: ${info.branch}`);
		}

		if (info.commit) {
			lines.push(`  Commit: ${info.commit.substring(0, 7)}`);
		}

		if (info.available) {
			lines.push('');
			lines.push('Workspaces:');

			if (workspaces.length === 0) {
				lines.push('  No workspaces found');
			} else {
				const sortedWorkspaces = workspaces.sort((a, b) => a.id - b.id);
				for (const ws of sortedWorkspaces) {
					lines.push(`  ${ws.id} (${ws.name}): ${ws.windows} windows on ${ws.monitor}`);
				}
			}

			lines.push('');
			lines.push(`Total Windows: ${windowCount}`);

			if (activeWindow) {
				lines.push('');
				lines.push('Active Window:');
				lines.push(`  Title: ${activeWindow.title}`);
				lines.push(`  Class: ${activeWindow.class}`);
				lines.push(`  Workspace: ${activeWindow.workspace.name}`);
			} else {
				lines.push('');
				lines.push('Active Window: None');
			}
		}

		return lines.join('\n');
	}
}
