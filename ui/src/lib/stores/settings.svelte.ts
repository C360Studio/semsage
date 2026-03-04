/**
 * Settings Store - Manages user preferences with localStorage persistence.
 *
 * Preferences are stored in localStorage and automatically synchronized.
 * Theme changes apply immediately via data-theme attribute on <html>.
 */

import { browser } from '$app/environment';

const STORAGE_KEY = 'semsage-settings';

export type Theme = 'dark' | 'light' | 'system';

export interface Settings {
	theme: Theme;
	activityLimit: number;
	reducedMotion: boolean;
}

const DEFAULT_SETTINGS: Settings = {
	theme: 'dark',
	activityLimit: 100,
	reducedMotion: false
};

class SettingsStore {
	private settings = $state<Settings>({ ...DEFAULT_SETTINGS });

	constructor() {
		if (browser) {
			this.loadFromStorage();
			this.applyTheme();
			this.watchSystemTheme();
		}
	}

	get theme(): Theme {
		return this.settings.theme;
	}

	get activityLimit(): number {
		return this.settings.activityLimit;
	}

	get reducedMotion(): boolean {
		return this.settings.reducedMotion;
	}

	get effectiveTheme(): 'dark' | 'light' {
		if (this.settings.theme === 'system') {
			return this.getSystemTheme();
		}
		return this.settings.theme;
	}

	setTheme(theme: Theme): void {
		this.settings.theme = theme;
		this.applyTheme();
		this.saveToStorage();
	}

	setActivityLimit(limit: number): void {
		this.settings.activityLimit = Math.max(10, Math.min(1000, limit));
		this.saveToStorage();
	}

	setReducedMotion(enabled: boolean): void {
		this.settings.reducedMotion = enabled;
		this.saveToStorage();
	}

	resetToDefaults(): void {
		this.settings = { ...DEFAULT_SETTINGS };
		this.applyTheme();
		this.saveToStorage();
	}

	getAll(): Settings {
		return { ...this.settings };
	}

	private loadFromStorage(): void {
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored) {
				const parsed = JSON.parse(stored) as Partial<Settings>;
				this.settings = { ...DEFAULT_SETTINGS, ...parsed };
			}
		} catch {
			// Ignore parse errors, use defaults
		}
	}

	private saveToStorage(): void {
		if (!browser) return;
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(this.settings));
		} catch {
			// Ignore storage errors (quota exceeded, etc.)
		}
	}

	private applyTheme(): void {
		if (!browser) return;
		const theme = this.effectiveTheme;
		document.documentElement.setAttribute('data-theme', theme);
	}

	private getSystemTheme(): 'dark' | 'light' {
		if (!browser) return 'dark';
		return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
	}

	private watchSystemTheme(): void {
		if (!browser) return;
		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		mediaQuery.addEventListener('change', () => {
			if (this.settings.theme === 'system') {
				this.applyTheme();
			}
		});
	}
}

export const settingsStore = new SettingsStore();
