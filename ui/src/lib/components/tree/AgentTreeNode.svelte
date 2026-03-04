<script lang="ts">
	/**
	 * AgentTreeNode - A single node in the agent tree hierarchy.
	 *
	 * Renders a loop's role badge, state indicator, model, iteration count,
	 * and depth indicator. Supports lazy-loading of children on expand.
	 * Recursive — renders child AgentTreeNode components when expanded.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';
	import LoopCard from '$lib/components/loops/LoopCard.svelte';
	import { agentTreeStore } from '$lib/stores/agentTree.svelte';
	import { goto } from '$app/navigation';
	import type { Loop, LoopState } from '$lib/types';

	// Import self for recursion — Svelte handles this fine when the file references itself
	import AgentTreeNode from './AgentTreeNode.svelte';

	interface Props {
		loop: Loop;
		depth?: number;
	}

	let { loop, depth = 0 }: Props = $props();

	const isExpanded = $derived(agentTreeStore.isExpanded(loop.loop_id));
	const isLoading = $derived(agentTreeStore.isLoading(loop.loop_id));
	const children = $derived(agentTreeStore.getChildren(loop.loop_id));
	const fetchError = $derived(agentTreeStore.getError(loop.loop_id));

	const loopState = $derived(loop.state as LoopState);
	const isActive = $derived(['pending', 'exploring', 'executing'].includes(loopState));
	const isPaused = $derived(loopState === 'paused');

	// Depth indicator: we use indentation via padding-left on the node row
	const indentPx = $derived(depth * 20);

	async function handleToggle(e: MouseEvent) {
		e.stopPropagation();
		await agentTreeStore.toggle(loop.loop_id);
	}

	function handleNavigate() {
		goto(`/loops/${loop.loop_id}`);
	}
</script>

<li
	class="agent-tree-node"
	role="treeitem"
	aria-expanded={isExpanded && children.length > 0 ? isExpanded : undefined}
	aria-selected="false"
>
	<div class="node-row" style:padding-left="{indentPx}px">
		<!-- Expand/collapse toggle.
		     aria-hidden="true" because the treeitem li already carries aria-expanded;
		     exposing it on the button too would create a redundant/conflicting announcement. -->
		<button
			type="button"
			class="expand-btn"
			class:expanded={isExpanded}
			class:loading={isLoading}
			onclick={handleToggle}
			aria-label={isExpanded ? 'Collapse children' : 'Expand children'}
			title={isExpanded ? 'Collapse' : 'Expand children'}
			aria-hidden="true"
		>
			{#if isLoading}
				<Icon name="loader" size={14} />
			{:else}
				<Icon name={isExpanded ? 'chevron-down' : 'chevron-right'} size={14} />
			{/if}
		</button>

		<!-- State dot -->
		<span
			class="state-dot"
			class:executing={loopState === 'executing'}
			class:active={isActive && loopState !== 'executing'}
			class:paused={isPaused}
			class:complete={loopState === 'complete' || loopState === 'success'}
			class:failed={loopState === 'failed'}
			class:cancelled={loopState === 'cancelled'}
			title="State: {loopState}"
			aria-hidden="true"
		></span>

		<!-- Role badge -->
		<span class="role-badge" title={loop.role}>
			{loop.role}
		</span>

		<!-- Depth indicator (only shown when depth > 0) -->
		{#if depth > 0}
			<span class="depth-badge" title="Depth {loop.depth}/{loop.max_depth}">
				d{loop.depth}
			</span>
		{/if}

		<!-- Spacer -->
		<span class="spacer"></span>

		<!-- Model name -->
		<span class="model-name" title={loop.model}>
			{loop.model.length > 16 ? loop.model.slice(0, 14) + '…' : loop.model}
		</span>

		<!-- Iteration count -->
		<span class="iter-count" title="Iteration {loop.iterations} of {loop.max_iterations}">
			{loop.iterations}/{loop.max_iterations}
		</span>

		<!-- Navigate to detail page -->
		<button
			type="button"
			class="detail-btn"
			onclick={handleNavigate}
			title="View loop detail"
			aria-label="View detail for loop {loop.loop_id.slice(0, 8)}"
		>
			<Icon name="arrow-up-right" size={13} />
		</button>
	</div>

	<!-- Loop card with controls (only when active or paused) -->
	{#if isActive || isPaused}
		<div class="node-card" style:margin-left="{indentPx + 32}px">
			<LoopCard {loop} linkable={true} />
		</div>
	{/if}

	<!-- Children (recursive) -->
	{#if isExpanded}
		{#if fetchError}
			<div class="fetch-error" style:padding-left="{indentPx + 32}px">
				<Icon name="alert-circle" size={12} />
				<span>{fetchError}</span>
			</div>
		{:else if children.length === 0 && !isLoading}
			<div class="no-children" style:padding-left="{indentPx + 32}px">
				<span>No child agents</span>
			</div>
		{:else}
			<ul class="children-list" role="group">
				{#each children as child (child.loop_id)}
					<AgentTreeNode loop={child} depth={depth + 1} />
				{/each}
			</ul>
		{/if}
	{/if}
</li>

<style>
	.agent-tree-node {
		display: flex;
		flex-direction: column;
		list-style: none;
	}

	.node-row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding-top: var(--space-2);
		padding-bottom: var(--space-2);
		padding-right: var(--space-3);
		border-radius: var(--radius-sm);
		cursor: default;
		min-width: 0;
		transition: background var(--transition-fast);
	}

	.node-row:hover {
		background: var(--color-bg-tertiary);
	}

	.expand-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		border: none;
		background: transparent;
		border-radius: var(--radius-sm);
		cursor: pointer;
		color: var(--color-text-muted);
		flex-shrink: 0;
		transition: color var(--transition-fast), background var(--transition-fast);
	}

	.expand-btn:hover {
		background: var(--color-bg-elevated);
		color: var(--color-text-primary);
	}

	.expand-btn:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.expand-btn.loading {
		animation: spin 1s linear infinite;
		cursor: wait;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	.state-dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
		background: var(--color-text-muted);
		flex-shrink: 0;
	}

	.state-dot.executing {
		background: var(--color-accent);
		box-shadow: 0 0 4px var(--color-accent);
		animation: pulse 2s infinite;
	}

	.state-dot.active {
		background: var(--color-info);
	}

	.state-dot.paused {
		background: var(--color-warning);
	}

	.state-dot.complete {
		background: var(--color-success);
	}

	.state-dot.failed {
		background: var(--color-error);
	}

	.state-dot.cancelled {
		background: var(--color-text-muted);
		opacity: 0.5;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}

	.role-badge {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-primary);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		max-width: 160px;
	}

	.depth-badge {
		font-family: var(--font-family-mono);
		font-size: 10px;
		padding: 1px 4px;
		background: var(--color-bg-elevated);
		color: var(--color-text-muted);
		border-radius: var(--radius-sm);
		flex-shrink: 0;
	}

	.spacer {
		flex: 1;
	}

	.model-name {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		white-space: nowrap;
		flex-shrink: 0;
	}

	.iter-count {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		white-space: nowrap;
		flex-shrink: 0;
	}

	.detail-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border: none;
		background: transparent;
		border-radius: var(--radius-sm);
		cursor: pointer;
		color: var(--color-text-muted);
		flex-shrink: 0;
		transition: color var(--transition-fast), background var(--transition-fast);
	}

	.detail-btn:hover {
		background: var(--color-accent-muted);
		color: var(--color-accent);
	}

	.detail-btn:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.node-card {
		margin-top: var(--space-1);
		margin-bottom: var(--space-1);
		margin-right: var(--space-3);
	}

	.children-list {
		list-style: none;
		padding: 0;
		margin: 0;
		border-left: 1px solid var(--color-border);
		margin-left: 12px;
	}

	.fetch-error,
	.no-children {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		padding: var(--space-1) var(--space-3);
		padding-right: var(--space-3);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		list-style: none;
	}

	.fetch-error {
		color: var(--color-error);
	}
</style>
