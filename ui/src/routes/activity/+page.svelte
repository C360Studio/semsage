<script lang="ts">
	/**
	 * Activity page — the home page of semsage.
	 *
	 * Left panel: ActivityFeed (SSE event stream from activityStore)
	 * Right panel: AgentTreeView (expandable loop hierarchy from loopsStore)
	 *
	 * Uses ResizableSplit for a draggable 40/60 split layout.
	 * Mobile: panels stack vertically.
	 */

	import { onMount } from 'svelte';
	import ActivityFeed from '$lib/components/activity/ActivityFeed.svelte';
	import AgentTreeView from '$lib/components/tree/AgentTreeView.svelte';
	import ResizableSplit from '$lib/components/shared/ResizableSplit.svelte';
	import Icon from '$lib/components/shared/Icon.svelte';
	import { loopsStore } from '$lib/stores/loops.svelte';
	import { agentTreeStore } from '$lib/stores/agentTree.svelte';

	const rootLoops = $derived(loopsStore.rootLoops);
	const isLoadingLoops = $derived(loopsStore.loading);
	const loopsError = $derived(loopsStore.error);

	// Poll loops every 5 seconds so the tree stays fresh.
	// SSE connection is managed by the layout — no connect/disconnect here.
	const POLL_INTERVAL = 5000;

	onMount(() => {
		// Initial fetch
		loopsStore.fetch();

		// Poll for loop updates
		const interval = setInterval(() => loopsStore.fetch(), POLL_INTERVAL);

		return () => {
			clearInterval(interval);
		};
	});

	async function handleRefresh() {
		await loopsStore.fetch();
		agentTreeStore.reset();
	}
</script>

<svelte:head>
	<title>Activity - Semsage</title>
</svelte:head>

<div class="activity-page">
	<ResizableSplit
		id="activity-split"
		defaultRatio={0.4}
		minLeftWidth={280}
		minRightWidth={300}
		leftTitle="Activity Feed"
		rightTitle="Agent Trees"
	>
		{#snippet left()}
			<ActivityFeed />
		{/snippet}

		{#snippet right()}
			<div class="tree-panel">
				{#if loopsError}
					<div class="error-banner" role="alert">
						<Icon name="alert-circle" size={14} />
						<span>{loopsError}</span>
						<button
							type="button"
							class="retry-btn"
							onclick={handleRefresh}
							aria-label="Retry loading loops"
						>
							<Icon name="refresh-cw" size={12} />
							Retry
						</button>
					</div>
				{/if}
				<AgentTreeView rootLoops={rootLoops} loading={isLoadingLoops} />
			</div>
		{/snippet}

		{#snippet rightActions()}
			<button
				type="button"
				class="refresh-btn"
				onclick={handleRefresh}
				title="Refresh agent trees"
				aria-label="Refresh agent trees"
				disabled={isLoadingLoops}
			>
				<Icon name="refresh-cw" size={13} />
			</button>
		{/snippet}
	</ResizableSplit>
</div>

<style>
	.activity-page {
		display: flex;
		flex: 1;
		min-height: 0;
		padding: var(--space-3);
		background: var(--color-bg-primary);
	}

	.tree-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.error-banner {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		background: var(--color-error-muted);
		color: var(--color-error);
		font-size: var(--font-size-xs);
		border-bottom: 1px solid var(--color-error);
		flex-shrink: 0;
	}

	.error-banner span {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.retry-btn {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		padding: 2px var(--space-2);
		background: transparent;
		border: 1px solid var(--color-error);
		border-radius: var(--radius-sm);
		color: var(--color-error);
		font-size: var(--font-size-xs);
		cursor: pointer;
		white-space: nowrap;
		transition: all var(--transition-fast);
		flex-shrink: 0;
	}

	.retry-btn:hover {
		background: var(--color-error);
		color: white;
	}

	.refresh-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		background: transparent;
		border: none;
		border-radius: var(--radius-sm);
		cursor: pointer;
		color: var(--color-text-muted);
		transition: color var(--transition-fast), background var(--transition-fast);
	}

	.refresh-btn:hover {
		background: var(--color-bg-elevated);
		color: var(--color-text-primary);
	}

	.refresh-btn:disabled {
		cursor: not-allowed;
		opacity: 0.4;
	}

	.refresh-btn:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	/* On mobile the ResizableSplit goes vertical — no extra override needed
	   since ResizableSplit.svelte already handles flex-direction: column at 900px */
	@media (max-width: 900px) {
		.activity-page {
			padding: var(--space-2);
		}
	}
</style>
