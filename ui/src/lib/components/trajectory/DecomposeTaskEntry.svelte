<script lang="ts">
	/**
	 * DecomposeTaskEntry - Trajectory entry for decompose_task calls.
	 *
	 * Renders the returned DAG inline using DagView. Collapsible to save space
	 * when the user doesn't need the full breakdown.
	 */

	import type { Dag } from '$lib/types';
	import Icon from '$lib/components/shared/Icon.svelte';
	import DagView from './DagView.svelte';

	interface Props {
		dag?: Dag | null;
		goal?: string;
		status?: string;
		durationMs?: number;
	}

	let { dag, goal, status = 'unknown', durationMs }: Props = $props();

	let dagExpanded = $state(true);

	const nodeCount = $derived(dag?.nodes.length ?? 0);
	const isSuccess = $derived(status === 'success');
	const isFailed = $derived(status === 'failed' || status === 'error');

	function formatDuration(ms: number | undefined): string {
		if (ms === undefined) return '';
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(1)}s`;
	}
</script>

<div class="decompose-task-entry" aria-label="Decompose task call">
	<div class="entry-header">
		<span class="tool-badge">
			<Icon name="network" size={11} />
			decompose_task
		</span>

		<div class="meta-group">
			{#if isSuccess}
				<span class="status-chip success">
					<Icon name="check-circle" size={11} />
					{nodeCount} node{nodeCount !== 1 ? 's' : ''}
				</span>
			{:else if isFailed}
				<span class="status-chip failed">
					<Icon name="alert-circle" size={11} />
					failed
				</span>
			{/if}

			{#if durationMs !== undefined}
				<span class="duration">
					<Icon name="clock" size={11} />
					{formatDuration(durationMs)}
				</span>
			{/if}
		</div>
	</div>

	{#if goal}
		<p class="goal-text">
			<span class="goal-label">Goal:</span> {goal}
		</p>
	{/if}

	{#if dag && nodeCount > 0}
		<div class="dag-section">
			<button
				class="dag-toggle"
				onclick={() => (dagExpanded = !dagExpanded)}
				aria-expanded={dagExpanded}
			>
				<Icon name={dagExpanded ? 'chevron-up' : 'chevron-down'} size={12} />
				<span>Task DAG ({nodeCount} node{nodeCount !== 1 ? 's' : ''})</span>
			</button>

			{#if dagExpanded}
				<div class="dag-container">
					<DagView {dag} />
				</div>
			{/if}
		</div>
	{:else if !dag}
		<p class="no-dag">No DAG data available</p>
	{/if}
</div>

<style>
	.decompose-task-entry {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-3);
		background: var(--color-success-muted);
		border: 1px solid var(--color-success);
		border-radius: var(--radius-md);
		font-size: var(--font-size-sm);
	}

	.entry-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
	}

	.tool-badge {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-semibold);
		color: var(--color-success);
		padding: 2px var(--space-2);
		background: color-mix(in srgb, var(--color-success) 15%, transparent);
		border-radius: var(--radius-full);
	}

	.meta-group {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.status-chip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
	}

	.status-chip.success {
		background: color-mix(in srgb, var(--color-success) 15%, transparent);
		color: var(--color-success);
	}

	.status-chip.failed {
		background: var(--color-error-muted);
		color: var(--color-error);
	}

	.duration {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		font-size: var(--font-size-xs);
		font-family: var(--font-family-mono);
		color: var(--color-text-muted);
	}

	.goal-text {
		margin: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		line-height: var(--line-height-relaxed);
	}

	.goal-label {
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
	}

	.dag-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.dag-toggle {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		background: transparent;
		border: none;
		cursor: pointer;
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		color: var(--color-success);
		padding: 0;
		transition: opacity var(--transition-fast);
	}

	.dag-toggle:hover {
		opacity: 0.75;
	}

	.dag-container {
		padding: var(--space-3);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		overflow-x: auto;
		min-height: 120px;
	}

	.no-dag {
		margin: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		font-style: italic;
	}
</style>
