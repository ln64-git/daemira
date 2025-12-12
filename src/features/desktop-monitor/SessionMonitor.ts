import { Shell } from '../../utility/Shell';
import { Logger } from '../../utility/Logger';
import type { SessionInfo } from './types';

export class SessionMonitor {
	private static instance: SessionMonitor;
	private logger = Logger.getInstance();

	private constructor() {}

	static getInstance(): SessionMonitor {
		if (!SessionMonitor.instance) {
			SessionMonitor.instance = new SessionMonitor();
		}
		return SessionMonitor.instance;
	}

	async getSessionInfo(): Promise<SessionInfo> {
		try {
			const sessionId = process.env.XDG_SESSION_ID || '';
			if (!sessionId) {
				this.logger.warn('XDG_SESSION_ID not set, session monitoring unavailable');
				return this.getDefaultSessionInfo();
			}

			const result = await Shell.execute(`loginctl show-session ${sessionId}`, {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				this.logger.error(`loginctl failed: ${result.stderr}`);
				return this.getDefaultSessionInfo();
			}

			return this.parseLoginctlOutput(result.stdout);
		} catch (error) {
			this.logger.error(`Error getting session info: ${error}`);
			return this.getDefaultSessionInfo();
		}
	}

	private parseLoginctlOutput(output: string): SessionInfo {
		const lines = output.split('\n');
		const props: Record<string, string> = {};

		for (const line of lines) {
			const match = line.match(/^([^=]+)=(.*)$/);
			if (match) {
				props[match[1].trim()] = match[2].trim();
			}
		}

		return {
			sessionId: props.Id || process.env.XDG_SESSION_ID || 'unknown',
			user: props.Name || process.env.USER || 'unknown',
			seat: props.Seat || 'seat0',
			type: (props.Type || process.env.XDG_SESSION_TYPE || 'unknown').toLowerCase(),
			state: props.State || 'unknown',
			active: props.Active === 'yes',
			idle: props.IdleHint === 'yes',
			locked: props.LockedHint === 'yes',
			vt: parseInt(props.VTNr || '0', 10),
			display: props.Display || process.env.DISPLAY || '',
		};
	}

	private getDefaultSessionInfo(): SessionInfo {
		return {
			sessionId: process.env.XDG_SESSION_ID || 'unknown',
			user: process.env.USER || 'unknown',
			seat: 'seat0',
			type: (process.env.XDG_SESSION_TYPE || 'unknown').toLowerCase(),
			state: 'unknown',
			active: false,
			idle: false,
			locked: false,
			vt: 0,
			display: process.env.DISPLAY || '',
		};
	}

	async isSessionLocked(): Promise<boolean> {
		try {
			const info = await this.getSessionInfo();
			return info.locked;
		} catch (error) {
			this.logger.error(`Error checking lock status: ${error}`);
			return false;
		}
	}

	async getIdleStatus(): Promise<boolean> {
		try {
			const info = await this.getSessionInfo();
			return info.idle;
		} catch (error) {
			this.logger.error(`Error checking idle status: ${error}`);
			return false;
		}
	}

	async lockSession(): Promise<void> {
		try {
			const sessionId = process.env.XDG_SESSION_ID;
			if (!sessionId) {
				throw new Error('XDG_SESSION_ID not set');
			}

			const result = await Shell.execute(`loginctl lock-session ${sessionId}`, {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				throw new Error(`loginctl lock-session failed: ${result.stderr}`);
			}

			this.logger.info('Session locked successfully');
		} catch (error) {
			this.logger.error(`Error locking session: ${error}`);
			throw error;
		}
	}

	async unlockSession(): Promise<void> {
		try {
			const sessionId = process.env.XDG_SESSION_ID;
			if (!sessionId) {
				throw new Error('XDG_SESSION_ID not set');
			}

			const result = await Shell.execute(`loginctl unlock-session ${sessionId}`, {
				timeout: 5000,
			});

			if (result.exitCode !== 0) {
				throw new Error(`loginctl unlock-session failed: ${result.stderr}`);
			}

			this.logger.info('Session unlocked successfully');
		} catch (error) {
			this.logger.error(`Error unlocking session: ${error}`);
			throw error;
		}
	}

	formatSessionInfo(info: SessionInfo): string {
		const lines = [
			'Session Information:',
			`  Session ID: ${info.sessionId}`,
			`  User: ${info.user}`,
			`  Seat: ${info.seat}`,
			`  Type: ${info.type}`,
			`  State: ${info.state}`,
			`  Active: ${info.active ? 'yes' : 'no'}`,
			`  Idle: ${info.idle ? 'yes' : 'no'}`,
			`  Locked: ${info.locked ? 'yes' : 'no'}`,
		];

		if (info.vt > 0) {
			lines.push(`  VT: ${info.vt}`);
		}

		if (info.display) {
			lines.push(`  Display: ${info.display}`);
		}

		return lines.join('\n');
	}
}
