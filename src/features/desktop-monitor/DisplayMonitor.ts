import { Shell } from '../../utility/Shell';
import { Logger } from '../../utility/Logger';
import type { MonitorInfo } from './types';

export class DisplayMonitor {
	private static instance: DisplayMonitor;
	private logger = Logger.getInstance();

	private constructor() {}

	static getInstance(): DisplayMonitor {
		if (!DisplayMonitor.instance) {
			DisplayMonitor.instance = new DisplayMonitor();
		}
		return DisplayMonitor.instance;
	}

	isAvailable(): boolean {
		return !!process.env.HYPRLAND_INSTANCE_SIGNATURE;
	}

	async getMonitors(): Promise<MonitorInfo[]> {
		if (!this.isAvailable()) {
			return [];
		}

		try {
			const result = await Shell.execute('hyprctl monitors -j', {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				this.logger.error(`hyprctl monitors failed: ${result.stderr}`);
				return [];
			}

			return JSON.parse(result.stdout);
		} catch (error) {
			this.logger.error(`Error getting monitors: ${error}`);
			return [];
		}
	}

	async getPrimaryMonitor(): Promise<MonitorInfo | null> {
		const monitors = await this.getMonitors();
		if (monitors.length === 0) {
			return null;
		}

		const activeMonitor = monitors.find((m) => m.activeWorkspace.id > 0);
		return activeMonitor || monitors[0];
	}

	async getMonitorCount(): Promise<number> {
		const monitors = await this.getMonitors();
		return monitors.length;
	}

	formatMonitorInfo(monitors: MonitorInfo[]): string {
		if (monitors.length === 0) {
			return 'Display Information:\n  No monitors detected';
		}

		const lines = ['Display Information:'];

		for (const monitor of monitors) {
			lines.push('');
			lines.push(`  ${monitor.name}:`);

			if (monitor.description && monitor.description !== monitor.name) {
				lines.push(`    Description: ${monitor.description}`);
			}

			if (monitor.make && monitor.model) {
				lines.push(`    Make/Model: ${monitor.make} ${monitor.model}`);
			}

			lines.push(`    Resolution: ${monitor.width}x${monitor.height}@${monitor.refreshRate.toFixed(2)}Hz`);
			lines.push(`    Position: ${monitor.x},${monitor.y}`);
			lines.push(`    Scale: ${monitor.scale.toFixed(2)}`);
			lines.push(`    VRR: ${monitor.vrr ? 'enabled' : 'disabled'}`);
			lines.push(`    DPMS: ${monitor.dpmsStatus ? 'on' : 'off'}`);
			lines.push(`    Active Workspace: ${monitor.activeWorkspace.name}`);

			if (monitor.transform !== 0) {
				lines.push(`    Transform: ${monitor.transform}`);
			}
		}

		return lines.join('\n');
	}

	formatMonitorSummary(monitors: MonitorInfo[]): string {
		if (monitors.length === 0) {
			return 'No monitors';
		}

		const summaries = monitors.map((m) => {
			return `${m.name} (${m.width}x${m.height}@${m.refreshRate.toFixed(0)}Hz${m.vrr ? ' VRR' : ''})`;
		});

		return summaries.join(', ');
	}
}
