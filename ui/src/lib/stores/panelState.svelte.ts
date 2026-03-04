/**
 * Panel State Store - Manages collapsible panel visibility with localStorage persistence.
 */

const STORAGE_KEY = 'semsage-panel-state';

interface PanelConfig {
	id: string;
	defaultOpen?: boolean;
}

class PanelStateStore {
	private panelStates = $state<Record<string, boolean>>({});
	private defaults: Record<string, boolean> = {};

	constructor() {
		if (typeof window !== 'undefined') {
			this.loadFromStorage();
		}
	}

	private loadFromStorage(): void {
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored) {
				this.panelStates = JSON.parse(stored);
			}
		} catch {
			// Ignore parse errors, use defaults
		}
	}

	private saveToStorage(): void {
		if (typeof window === 'undefined') return;
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(this.panelStates));
		} catch {
			// Ignore storage errors
		}
	}

	register(config: PanelConfig): boolean {
		const { id, defaultOpen = true } = config;
		this.defaults[id] = defaultOpen;

		if (!(id in this.panelStates)) {
			this.panelStates[id] = defaultOpen;
		}

		return this.panelStates[id];
	}

	isOpen(id: string): boolean {
		return this.panelStates[id] ?? this.defaults[id] ?? true;
	}

	toggle(id: string): void {
		this.panelStates[id] = !this.isOpen(id);
		this.saveToStorage();
	}

	setOpen(id: string, open: boolean): void {
		this.panelStates[id] = open;
		this.saveToStorage();
	}

	get openCount(): number {
		return Object.values(this.panelStates).filter(Boolean).length;
	}

	get totalCount(): number {
		return Object.keys(this.defaults).length;
	}
}

export const panelState = new PanelStateStore();
