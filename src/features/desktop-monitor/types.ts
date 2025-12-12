export interface SessionInfo {
	sessionId: string;
	user: string;
	seat: string;
	type: string; // 'wayland' | 'x11'
	state: string;
	active: boolean;
	idle: boolean;
	locked: boolean;
	vt: number;
	display: string;
}

export interface CompositorInfo {
	name: string;
	version: string;
	available: boolean;
	branch?: string;
	commit?: string;
	buildDate?: string;
}

export interface WorkspaceInfo {
	id: number;
	name: string;
	monitor: string;
	monitorID: number;
	windows: number;
	hasfullscreen: boolean;
	lastwindow: string;
	lastwindowtitle: string;
}

export interface WindowInfo {
	address: string;
	title: string;
	class: string;
	workspace: {
		id: number;
		name: string;
	};
	monitor: number;
	pid: number;
	floating: boolean;
	fullscreen: boolean | number;
	mapped: boolean;
	hidden: boolean;
	pinned: boolean;
}

export interface MonitorInfo {
	id: number;
	name: string;
	description: string;
	make: string;
	model: string;
	serial: string;
	width: number;
	height: number;
	refreshRate: number;
	x: number;
	y: number;
	activeWorkspace: {
		id: number;
		name: string;
	};
	scale: number;
	transform: number;
	vrr: boolean;
	dpmsStatus: boolean;
}

export interface DesktopStatus {
	session: SessionInfo;
	compositor: CompositorInfo;
	workspaces: WorkspaceInfo[];
	windows: WindowInfo[];
	monitors: MonitorInfo[];
}

export type CompositorType = 'hyprland' | 'sway' | 'niri' | 'i3' | 'unknown';
