import { api } from '$lib/api/client';
import type { Trajectory } from '$lib/types';

/**
 * Store for trajectory data — agent loop execution history.
 * Caches trajectory data per loop_id to avoid redundant API calls.
 */
class TrajectoryStore {
	cache = $state<Record<string, Trajectory>>({});
	loading = $state<Record<string, boolean>>({});
	errors = $state<Record<string, string | null>>({});

	/**
	 * Fetch trajectory for a loop, caching the result.
	 */
	async fetch(loopId: string): Promise<Trajectory | null> {
		if (this.cache[loopId]) return this.cache[loopId];

		this.loading[loopId] = true;
		this.errors[loopId] = null;

		try {
			const trajectory = await api.trajectory.getByLoop(loopId, 'json');
			this.cache[loopId] = trajectory;
			return trajectory;
		} catch (err) {
			this.errors[loopId] = err instanceof Error ? err.message : 'Failed to fetch trajectory';
			return null;
		} finally {
			this.loading[loopId] = false;
		}
	}

	get(loopId: string): Trajectory | undefined {
		return this.cache[loopId];
	}

	isLoading(loopId: string): boolean {
		return this.loading[loopId] ?? false;
	}

	getError(loopId: string): string | null {
		return this.errors[loopId] ?? null;
	}

	/**
	 * Uses object reassignment instead of `delete` to preserve $state reactivity.
	 */
	invalidate(loopId: string): void {
		const { [loopId]: _c, ...restCache } = this.cache;
		this.cache = restCache;
		const { [loopId]: _l, ...restLoading } = this.loading;
		this.loading = restLoading;
		const { [loopId]: _e, ...restErrors } = this.errors;
		this.errors = restErrors;
	}

	clear(): void {
		this.cache = {};
		this.loading = {};
		this.errors = {};
	}
}

export const trajectoryStore = new TrajectoryStore();
