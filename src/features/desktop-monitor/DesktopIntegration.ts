import { Logger } from '../../utility/Logger';
import { SessionMonitor } from './SessionMonitor';
import { CompositorMonitor } from './CompositorMonitor';
import { DisplayMonitor } from './DisplayMonitor';
import type { DesktopStatus, CompositorType } from './types';

export class DesktopIntegration {
	private static instance: DesktopIntegration;
	private logger = Logger.getInstance();
	private sessionMonitor = SessionMonitor.getInstance();
	private compositorMonitor = CompositorMonitor.getInstance();
	private displayMonitor = DisplayMonitor.getInstance();

	private constructor() {}

	static getInstance(): DesktopIntegration {
		if (!DesktopIntegration.instance) {
			DesktopIntegration.instance = new DesktopIntegration();
		}
		return DesktopIntegration.instance;
	}

	detectCompositor(): CompositorType {
		if (process.env.HYPRLAND_INSTANCE_SIGNATURE) return 'hyprland';
		if (process.env.NIRI_SOCKET) return 'niri';
		if (process.env.SWAYSOCK) return 'sway';
		if (process.env.I3SOCK) return 'i3';
		return 'unknown';
	}

	isDesktopMonitoringAvailable(): boolean {
		return this.compositorMonitor.isAvailable() || process.env.XDG_SESSION_ID !== undefined;
	}

	async getDesktopStatus(): Promise<DesktopStatus> {
		const [session, compositor, workspaces, windows, monitors] = await Promise.all([
			this.sessionMonitor.getSessionInfo(),
			this.compositorMonitor.getCompositorInfo(),
			this.compositorMonitor.getWorkspaces(),
			this.compositorMonitor.getWindows(),
			this.displayMonitor.getMonitors(),
		]);

		return {
			session,
			compositor,
			workspaces,
			windows,
			monitors,
		};
	}

	async getFormattedStatus(): Promise<string> {
		const status = await this.getDesktopStatus();

		const lines = [
			'Desktop Environment Status',
			'='.repeat(50),
			'',
		];

		lines.push(this.sessionMonitor.formatSessionInfo(status.session));
		lines.push('');

		const activeWindow = await this.compositorMonitor.getActiveWindow();
		const windowCount = status.windows.length;
		lines.push(this.compositorMonitor.formatCompositorInfo(status.compositor, status.workspaces, activeWindow, windowCount));
		lines.push('');

		lines.push(this.displayMonitor.formatMonitorInfo(status.monitors));

		return lines.join('\n');
	}

	async getSessionStatus(): Promise<string> {
		const session = await this.sessionMonitor.getSessionInfo();
		return this.sessionMonitor.formatSessionInfo(session);
	}

	async getCompositorStatus(): Promise<string> {
		const [compositor, workspaces, activeWindow, windowCount] = await Promise.all([
			this.compositorMonitor.getCompositorInfo(),
			this.compositorMonitor.getWorkspaces(),
			this.compositorMonitor.getActiveWindow(),
			this.compositorMonitor.getWindowCount(),
		]);

		return this.compositorMonitor.formatCompositorInfo(compositor, workspaces, activeWindow, windowCount);
	}

	async getDisplayStatus(): Promise<string> {
		const monitors = await this.displayMonitor.getMonitors();
		return this.displayMonitor.formatMonitorInfo(monitors);
	}

	async lockSession(): Promise<string> {
		try {
			await this.sessionMonitor.lockSession();
			return 'Session locked successfully';
		} catch (error) {
			return `Failed to lock session: ${error}`;
		}
	}

	async unlockSession(): Promise<string> {
		try {
			await this.sessionMonitor.unlockSession();
			return 'Session unlocked successfully';
		} catch (error) {
			return `Failed to unlock session: ${error}`;
		}
	}

	async getDesktopSummary(): Promise<string> {
		const status = await this.getDesktopStatus();
		const monitors = status.monitors;

		const lines: string[] = [];

		lines.push(`Compositor: ${status.compositor.name} ${status.compositor.version}`);
		lines.push(`Session: ${status.session.type} (${status.session.seat})`);

		if (status.workspaces.length > 0) {
			lines.push(`Workspaces: ${status.workspaces.length} active`);
		}

		if (status.windows.length > 0) {
			lines.push(`Windows: ${status.windows.length} open`);
		}

		if (monitors.length > 0) {
			lines.push(`Displays: ${this.displayMonitor.formatMonitorSummary(monitors)}`);
		}

		lines.push(`Lock State: ${status.session.locked ? 'locked' : 'unlocked'}`);

		return lines.join('\n  ');
	}
}
