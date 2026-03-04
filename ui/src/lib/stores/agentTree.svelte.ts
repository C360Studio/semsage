import { api } from '$lib/api/client';
import type { Loop } from '$lib/types';

/**
 * Agent Tree Store - Manages expanded/collapsed tree state and caches children per loop.
 *
 * Used on the activity page to render expandable agent hierarchies.
 * Each root loop can be expanded to show its spawned child loops.
 */
class AgentTreeStore {
	// Set of loop IDs that are currently expanded
	private expanded = $state<Set<string>>(new Set());

	// Cache of children per loop ID
	private childrenCache = $state<Record<string, Loop[]>>({});

	// Loading state per loop ID
	private loadingMap = $state<Record<string, boolean>>({});

	// Error state per loop ID
	private errorMap = $state<Record<string, string | null>>({});

	/**
	 * Check if a loop node is expanded.
	 */
	isExpanded(loopId: string): boolean {
		return this.expanded.has(loopId);
	}

	/**
	 * Toggle expand/collapse for a loop node.
	 * Fetches children if expanding and not yet cached.
	 */
	async toggle(loopId: string): Promise<void> {
		if (this.expanded.has(loopId)) {
			// Collapse — create new Set without this ID
			const next = new Set(this.expanded);
			next.delete(loopId);
			this.expanded = next;
		} else {
			// Expand — fetch children if not cached
			if (!this.childrenCache[loopId]) {
				await this.fetchChildren(loopId);
			}
			const next = new Set(this.expanded);
			next.add(loopId);
			this.expanded = next;
		}
	}

	/**
	 * Fetch and cache children for a loop.
	 */
	async fetchChildren(loopId: string): Promise<Loop[]> {
		if (this.childrenCache[loopId]) return this.childrenCache[loopId];

		this.loadingMap[loopId] = true;
		this.errorMap[loopId] = null;

		try {
			const children = await api.loops.getChildren(loopId);
			this.childrenCache[loopId] = children;
			return children;
		} catch (err) {
			this.errorMap[loopId] = err instanceof Error ? err.message : 'Failed to fetch children';
			this.childrenCache[loopId] = [];
			return [];
		} finally {
			this.loadingMap[loopId] = false;
		}
	}

	/**
	 * Get cached children for a loop (empty array if not fetched).
	 */
	getChildren(loopId: string): Loop[] {
		return this.childrenCache[loopId] ?? [];
	}

	/**
	 * Check if children are currently loading for a loop.
	 */
	isLoading(loopId: string): boolean {
		return this.loadingMap[loopId] ?? false;
	}

	/**
	 * Get fetch error for a loop's children.
	 */
	getError(loopId: string): string | null {
		return this.errorMap[loopId] ?? null;
	}

	/**
	 * Invalidate cached children for a loop (forces re-fetch on next expand).
	 * Uses object reassignment instead of `delete` to preserve $state reactivity.
	 */
	invalidate(loopId: string): void {
		const { [loopId]: _c, ...restCache } = this.childrenCache;
		this.childrenCache = restCache;
		const { [loopId]: _l, ...restLoading } = this.loadingMap;
		this.loadingMap = restLoading;
		const { [loopId]: _e, ...restError } = this.errorMap;
		this.errorMap = restError;
	}

	/**
	 * Collapse all expanded nodes.
	 */
	collapseAll(): void {
		this.expanded = new Set();
	}

	/**
	 * Clear all caches and collapse all nodes.
	 */
	reset(): void {
		this.expanded = new Set();
		this.childrenCache = {};
		this.loadingMap = {};
		this.errorMap = {};
	}
}

export const agentTreeStore = new AgentTreeStore();
