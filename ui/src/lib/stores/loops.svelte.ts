import { api } from '$lib/api/client';
import type { Loop } from '$lib/types';

class LoopsStore {
	all = $state<Loop[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	get active(): Loop[] {
		return this.all.filter((l) =>
			['pending', 'exploring', 'executing', 'paused'].includes(l.state)
		);
	}

	get rootLoops(): Loop[] {
		// Root loops are those without a parent_loop_id
		return this.all.filter((l) => !l.parent_loop_id);
	}

	async fetch(): Promise<void> {
		this.loading = true;
		this.error = null;

		try {
			this.all = await api.loops.list();
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to fetch loops';
		} finally {
			this.loading = false;
		}
	}

	async sendSignal(loopId: string, type: 'pause' | 'resume' | 'cancel', reason?: string): Promise<void> {
		await api.loops.sendSignal(loopId, type, reason);
	}

	async fetchTree(loopId: string): Promise<Loop[]> {
		try {
			return await api.loops.getTree(loopId);
		} catch {
			return [];
		}
	}

}

export const loopsStore = new LoopsStore();
