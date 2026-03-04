import { api } from '$lib/api/client';
import type { DynamicTool } from '$lib/types';

/**
 * Dynamic Tools Store - Fetches tools scoped to a root loop ID.
 *
 * Tools are created by `create_tool` calls within an agent tree and are
 * scoped to the root loop that created them. When the root loop terminates,
 * its tools are cleaned up by the backend.
 */
class DynamicToolsStore {
	// Cache of tools per root loop ID
	private toolsCache = $state<Record<string, DynamicTool[]>>({});

	// Loading state per root loop ID
	private loadingMap = $state<Record<string, boolean>>({});

	// Error state per root loop ID
	private errorMap = $state<Record<string, string | null>>({});

	/**
	 * Fetch tools scoped to a root loop ID, with caching.
	 */
	async fetch(rootLoopId: string): Promise<DynamicTool[]> {
		if (this.toolsCache[rootLoopId]) return this.toolsCache[rootLoopId];

		this.loadingMap[rootLoopId] = true;
		this.errorMap[rootLoopId] = null;

		try {
			const tools = await api.tools.listByTree(rootLoopId);
			this.toolsCache[rootLoopId] = tools;
			return tools;
		} catch (err) {
			this.errorMap[rootLoopId] = err instanceof Error ? err.message : 'Failed to fetch tools';
			return [];
		} finally {
			this.loadingMap[rootLoopId] = false;
		}
	}

	/**
	 * Get cached tools for a root loop (empty array if not fetched).
	 */
	get(rootLoopId: string): DynamicTool[] {
		return this.toolsCache[rootLoopId] ?? [];
	}

	/**
	 * Check if tools are currently loading for a root loop.
	 */
	isLoading(rootLoopId: string): boolean {
		return this.loadingMap[rootLoopId] ?? false;
	}

	/**
	 * Get fetch error for a root loop's tools.
	 */
	getError(rootLoopId: string): string | null {
		return this.errorMap[rootLoopId] ?? null;
	}

	/**
	 * Invalidate cached tools for a root loop (forces re-fetch on next access).
	 * Uses object reassignment instead of `delete` to preserve $state reactivity.
	 */
	invalidate(rootLoopId: string): void {
		const { [rootLoopId]: _t, ...restTools } = this.toolsCache;
		this.toolsCache = restTools;
		const { [rootLoopId]: _l, ...restLoading } = this.loadingMap;
		this.loadingMap = restLoading;
		const { [rootLoopId]: _e, ...restError } = this.errorMap;
		this.errorMap = restError;
	}

	/**
	 * Clear all cached tool data.
	 */
	clear(): void {
		this.toolsCache = {};
		this.loadingMap = {};
		this.errorMap = {};
	}
}

export const dynamicToolsStore = new DynamicToolsStore();
