<script lang="ts">
	/**
	 * AgentTreeView - Renders the full agent hierarchy as an expandable tree.
	 *
	 * Root loops (no parent_loop_id) are tree roots. Each can be expanded
	 * to show children fetched via agentTreeStore. Used on the activity page.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';
	import AgentTreeNode from './AgentTreeNode.svelte';
	import { agentTreeStore } from '$lib/stores/agentTree.svelte';
	import type { Loop } from '$lib/types';

	interface Props {
		rootLoops: Loop[];
		loading?: boolean;
	}

	let { rootLoops, loading = false }: Props = $props();

	const loopCount = $derived(rootLoops.length);
</script>

<div class="agent-tree-view">
	<!-- Header -->
	<div class="tree-header">
		<div class="header-left">
			<Icon name="tree" size={14} />
			<span class="header-label">Agent Trees</span>
			{#if loopCount > 0}
				<span class="count-badge">{loopCount}</span>
			{/if}
		</div>
		{#if loopCount > 0}
			<button
				type="button"
				class="collapse-all-btn"
				onclick={() => agentTreeStore.collapseAll()}
				title="Collapse all"
				aria-label="Collapse all agent trees"
			>
				<Icon name="chevron-up" size={14} />
			</button>
		{/if}
	</div>

	<!-- Tree content -->
	{#if loading}
		<div class="loading-state">
			<Icon name="loader" size={20} />
			<span>Loading agent trees…</span>
		</div>
	{:else if rootLoops.length === 0}
		<div class="empty-state" aria-live="polite">
			<Icon name="network" size={32} />
			<p>No active agent trees</p>
			<p class="hint">Active agent hierarchies will appear here</p>
		</div>
	{:else}
		<div class="tree-scroll">
			<ul
				class="tree-list"
				role="tree"
				aria-label="Agent hierarchy"
			>
				{#each rootLoops as loop (loop.loop_id)}
					<AgentTreeNode {loop} depth={0} />
				{/each}
			</ul>
		</div>
	{/if}
</div>

<style>
	.agent-tree-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.tree-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
		min-height: 40px;
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--color-text-muted);
		font-size: var(--font-size-xs);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.header-label {
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-secondary);
	}

	.count-badge {
		background: var(--color-accent-muted);
		color: var(--color-accent);
		padding: 1px 6px;
		border-radius: var(--radius-full);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
	}

	.collapse-all-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border: none;
		background: transparent;
		border-radius: var(--radius-sm);
		cursor: pointer;
		color: var(--color-text-muted);
		transition: color var(--transition-fast), background var(--transition-fast);
	}

	.collapse-all-btn:hover {
		background: var(--color-bg-elevated);
		color: var(--color-text-primary);
	}

	.collapse-all-btn:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.loading-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-3);
		color: var(--color-text-muted);
		font-size: var(--font-size-sm);
	}

	.loading-state :global(svg) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		color: var(--color-text-muted);
		text-align: center;
		padding: var(--space-6);
	}

	.empty-state p {
		margin: 0;
		font-size: var(--font-size-sm);
	}

	.hint {
		font-size: var(--font-size-xs) !important;
		color: var(--color-text-muted);
	}

	.tree-scroll {
		flex: 1;
		overflow-y: auto;
		overflow-x: hidden;
	}

	.tree-list {
		list-style: none;
		padding: var(--space-2);
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}
</style>
