import { browser } from '$app/environment';
import type { ActivityEvent } from '$lib/types';
import { settingsStore } from '$lib/stores/settings.svelte';

type ActivityCallback = (event: ActivityEvent) => void;

class ActivityStore {
	recent = $state<ActivityEvent[]>([]);
	connected = $state(false);

	private eventSource: EventSource | null = null;
	private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
	private callbacks: Set<ActivityCallback> = new Set();
	private nextId = 0;

	private get maxEvents(): number {
		return settingsStore.activityLimit;
	}

	connect(filter?: string): void {
		if (!browser) return;
		// Idempotency guard: do not open a second connection if one already exists
		if (this.eventSource) return;

		const url = filter
			? `/api/activity?filter=${encodeURIComponent(filter)}`
			: '/api/activity';

		this.eventSource = new EventSource(url);

		this.eventSource.addEventListener('connected', () => {
			this.connected = true;
		});

		this.eventSource.addEventListener('activity', (event) => {
			const activity = JSON.parse(event.data) as ActivityEvent;
			this.addEvent(activity);
		});

		this.eventSource.onmessage = (event) => {
			const activity = JSON.parse(event.data) as ActivityEvent;
			this.addEvent(activity);
		};

		this.eventSource.onerror = () => {
			this.connected = false;
			this.eventSource?.close();
			this.eventSource = null;
			this.reconnectTimeout = setTimeout(() => this.connect(filter), 3000);
		};
	}

	private addEvent(event: ActivityEvent): void {
		event.id = this.nextId++;
		this.recent = [...this.recent.slice(-(this.maxEvents - 1)), event];
		for (const callback of this.callbacks) {
			callback(event);
		}
	}

	onEvent(callback: ActivityCallback): () => void {
		this.callbacks.add(callback);
		return () => this.callbacks.delete(callback);
	}

	disconnect(): void {
		if (this.reconnectTimeout) {
			clearTimeout(this.reconnectTimeout);
			this.reconnectTimeout = null;
		}
		if (this.eventSource) {
			this.eventSource.close();
			this.eventSource = null;
		}
		this.connected = false;
	}

	clear(): void {
		this.recent = [];
	}
}

export const activityStore = new ActivityStore();
