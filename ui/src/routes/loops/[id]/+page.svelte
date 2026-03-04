<script lang="ts">
	/**
	 * Loop detail page (/loops/[id]).
	 *
	 * Renders: header metadata, LoopBreadcrumb, controls bar (active loops),
	 * ChildLoopList, Dynamic Tools, and TrajectoryPanel.
	 */

	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { trajectoryStore } from '$lib/stores/trajectory.svelte';
	import { dynamicToolsStore } from '$lib/stores/dynamicTools.svelte';
	import type { Loop, DynamicTool } from '$lib/types';

	import Icon from '$lib/components/shared/Icon.svelte';
	import CollapsiblePanel from '$lib/components/shared/CollapsiblePanel.svelte';
	import LoopBreadcrumb from '$lib/components/loops/LoopBreadcrumb.svelte';
	import ChildLoopList from '$lib/components/loops/ChildLoopList.svelte';
	import TrajectoryPanel from '$lib/components/trajectory/TrajectoryPanel.svelte';

	// -------------------------------------------------------------------------
	// State
	// -------------------------------------------------------------------------

	// The [id] route guarantees this param is always a non-empty string.
	const loopId = $derived($page.params.id as string);

	let loop = $state<Loop | null>(null);
	let loopLoading = $state(true);
	let loopError = $state<string | null>(null);

	let ancestors = $state<Loop[]>([]);
	let ancestorsLoading = $state(false);

	let children = $state<Loop[]>([]);
	let childrenLoading = $state(false);

	let tools = $state<DynamicTool[]>([]);
	let toolsLoading = $state(false);

	let signalError = $state<string | null>(null);
	let signaling = $state<'pause' | 'resume' | 'cancel' | null>(null);

	// -------------------------------------------------------------------------
	// Derived
	// -------------------------------------------------------------------------

	const isActive = $derived(
		loop !== null && ['pending', 'exploring', 'executing', 'paused'].includes(loop.state)
	);

	const canPause = $derived(loop?.state === 'executing' || loop?.state === 'exploring');
	const canResume = $derived(loop?.state === 'paused');
	const canCancel = $derived(isActive);

	const durationLabel = $derived.by(() => {
		if (!loop?.created_at) return null;
		const start = new Date(loop.created_at).getTime();
		const end = loop.completed_at ? new Date(loop.completed_at).getTime() : Date.now();
		const ms = end - start;
		if (ms < 1000) return `${ms}ms`;
		if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
		const mins = Math.floor(ms / 60_000);
		const secs = Math.floor((ms % 60_000) / 1000);
		return `${mins}m ${secs}s`;
	});

	const tokenTotal = $derived(
		loop !== null && loop.tokens_in !== undefined && loop.tokens_out !== undefined
			? loop.tokens_in + loop.tokens_out
			: null
	);

	const stateClass = $derived.by(() => {
		switch (loop?.state) {
			case 'executing':
			case 'exploring':
				return 'state-executing';
			case 'complete':
			case 'success':
				return 'state-success';
			case 'failed':
				return 'state-failed';
			case 'paused':
				return 'state-paused';
			case 'cancelled':
				return 'state-cancelled';
			default:
				return 'state-default';
		}
	});

	// Determine root loop ID by walking ancestors to the top
	const rootLoopId = $derived(ancestors.length > 0 ? ancestors[0].loop_id : loopId);

	// -------------------------------------------------------------------------
	// Data loading
	// -------------------------------------------------------------------------

	async function loadLoop(id: string) {
		loopLoading = true;
		loopError = null;
		try {
			loop = await api.loops.get(id);
		} catch (err) {
			loopError = err instanceof Error ? err.message : 'Failed to load loop';
		} finally {
			loopLoading = false;
		}
	}

	async function loadAncestors(startLoop: Loop) {
		ancestorsLoading = true;
		const chain: Loop[] = [];
		let current: Loop = startLoop;

		// Walk up the parent chain (max 20 hops to prevent runaway)
		for (let i = 0; i < 20 && current.parent_loop_id; i++) {
			try {
				const parent = await api.loops.get(current.parent_loop_id);
				chain.unshift(parent);
				current = parent;
			} catch {
				break;
			}
		}

		ancestors = chain;
		ancestorsLoading = false;
	}

	async function loadChildren(id: string) {
		childrenLoading = true;
		try {
			children = await api.loops.getChildren(id);
		} catch {
			children = [];
		} finally {
			childrenLoading = false;
		}
	}

	async function loadTools(id: string) {
		toolsLoading = true;
		try {
			tools = await dynamicToolsStore.fetch(id);
		} catch {
			tools = [];
		} finally {
			toolsLoading = false;
		}
	}

	// -------------------------------------------------------------------------
	// Signal controls
	// -------------------------------------------------------------------------

	async function sendSignal(type: 'pause' | 'resume' | 'cancel') {
		if (!loop) return;
		signaling = type;
		signalError = null;
		try {
			await api.loops.sendSignal(loop.loop_id, type);
			// Reload loop state
			await loadLoop(loop.loop_id);
		} catch (err) {
			signalError = err instanceof Error ? err.message : `Failed to ${type} loop`;
		} finally {
			signaling = null;
		}
	}

	// -------------------------------------------------------------------------
	// Lifecycle
	// -------------------------------------------------------------------------

	$effect(() => {
		const id = loopId;
		let cancelled = false;

		async function run() {
			loopLoading = true;
			loopError = null;
			try {
				loop = await api.loops.get(id);
			} catch (err) {
				if (cancelled) return;
				loopError = err instanceof Error ? err.message : 'Failed to load loop';
				loopLoading = false;
				return;
			}
			if (cancelled) return;
			loopLoading = false;

			if (loop) {
				await Promise.all([
					loadAncestors(loop),
					loadChildren(id),
					trajectoryStore.fetch(id)
				]);
				if (cancelled) return;
				loadTools(rootLoopId);
			}
		}

		run();

		return () => {
			cancelled = true;
		};
	});

	// -------------------------------------------------------------------------
	// Helpers
	// -------------------------------------------------------------------------

	function formatTokens(n: number): string {
		if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
		if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
		return String(n);
	}
</script>

<div class="page">
	<!-- Back navigation -->
	<a href="/activity" class="back-link">
		<Icon name="chevron-left" size={16} />
		Activity
	</a>

	{#if loopLoading && !loop}
		<div class="loading-state">
			<Icon name="loader" size={24} class="spin" />
			<p>Loading loop…</p>
		</div>
	{:else if loopError}
		<div class="error-state">
			<Icon name="alert-triangle" size={24} />
			<p>{loopError}</p>
			<button class="btn btn-secondary" onclick={() => loadLoop(loopId)}>Retry</button>
		</div>
	{:else if loop}
		<!-- ================================================================
		     Header
		     ================================================================ -->
		<header class="loop-header">
			<div class="header-top">
				<code class="loop-id">{loop.loop_id}</code>
				<div class="badges">
					<span class="role-badge">{loop.role}</span>
					<span class="state-badge {stateClass}">{loop.state}</span>
				</div>
			</div>

			<div class="header-meta">
				{#if loop.model}
					<span class="meta-item">
						<Icon name="brain" size={13} />
						{loop.model}
					</span>
				{/if}

				<span class="meta-item">
					<Icon name="refresh-cw" size={13} />
					Iteration {loop.iterations} of {loop.max_iterations}
				</span>

				{#if loop.depth !== undefined && loop.max_depth !== undefined}
					<span class="meta-item">
						<Icon name="layers" size={13} />
						Depth {loop.depth}/{loop.max_depth}
					</span>
				{/if}

				{#if durationLabel}
					<span class="meta-item">
						<Icon name="clock" size={13} />
						{durationLabel}
					</span>
				{/if}

				{#if tokenTotal !== null}
					<span class="meta-item">
						<Icon name="cpu" size={13} />
						{formatTokens(tokenTotal)} tokens
						{#if loop.tokens_in !== undefined && loop.tokens_out !== undefined}
							<span class="token-detail">({formatTokens(loop.tokens_in)} in / {formatTokens(loop.tokens_out)} out)</span>
						{/if}
					</span>
				{/if}
			</div>

			<!-- Breadcrumb -->
			{#if !ancestorsLoading}
				<LoopBreadcrumb {loop} {ancestors} />
			{/if}
		</header>

		<!-- ================================================================
		     Controls (active loops only)
		     ================================================================ -->
		{#if isActive}
			<div class="controls-bar" role="group" aria-label="Loop controls">
				{#if signalError}
					<span class="signal-error">
						<Icon name="alert-circle" size={14} />
						{signalError}
					</span>
				{/if}

				<button
					class="btn btn-warning"
					onclick={() => sendSignal('pause')}
					disabled={!canPause || signaling !== null}
					title="Pause this loop"
				>
					{#if signaling === 'pause'}
						<Icon name="loader" size={14} class="spin" />
					{:else}
						<Icon name="pause" size={14} />
					{/if}
					Pause
				</button>

				<button
					class="btn btn-success"
					onclick={() => sendSignal('resume')}
					disabled={!canResume || signaling !== null}
					title="Resume this loop"
				>
					{#if signaling === 'resume'}
						<Icon name="loader" size={14} class="spin" />
					{:else}
						<Icon name="play" size={14} />
					{/if}
					Resume
				</button>

				<button
					class="btn btn-danger"
					onclick={() => sendSignal('cancel')}
					disabled={!canCancel || signaling !== null}
					title="Cancel this loop and all children"
				>
					{#if signaling === 'cancel'}
						<Icon name="loader" size={14} class="spin" />
					{:else}
						<Icon name="square" size={14} />
					{/if}
					Cancel
				</button>
			</div>
		{/if}

		<!-- ================================================================
		     Children
		     ================================================================ -->
		<CollapsiblePanel
			id="loop-children-{loopId}"
			title="Children ({children.length})"
			defaultOpen={true}
		>
			<div class="panel-inner">
				<ChildLoopList {children} loading={childrenLoading} />
			</div>
		</CollapsiblePanel>

		<!-- ================================================================
		     Dynamic Tools
		     ================================================================ -->
		{#if toolsLoading || tools.length > 0}
			<CollapsiblePanel
				id="loop-tools-{loopId}"
				title="Dynamic Tools ({tools.length})"
				defaultOpen={false}
			>
				<div class="panel-inner">
					{#if toolsLoading}
						<div class="tools-loading">
							<Icon name="loader" size={16} class="spin" />
							<span>Loading tools…</span>
						</div>
					{:else if tools.length === 0}
						<p class="tools-empty">No dynamic tools created by this agent tree.</p>
					{:else}
						<ul class="tools-list">
							{#each tools as tool (tool.name)}
								<li class="tool-item">
									<div class="tool-header">
										<code class="tool-name">{tool.name}</code>
										{#if tool.processors && tool.processors.length > 0}
											<span class="processor-count">
												{tool.processors.length} processor{tool.processors.length !== 1 ? 's' : ''}
											</span>
										{/if}
									</div>
									{#if tool.description}
										<p class="tool-description">{tool.description}</p>
									{/if}
									{#if tool.processors && tool.processors.length > 0}
										<ul class="processor-list">
											{#each tool.processors as proc (proc)}
												<li class="processor-chip">{proc}</li>
											{/each}
										</ul>
									{/if}
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</CollapsiblePanel>
		{/if}

		<!-- ================================================================
		     Trajectory
		     ================================================================ -->
		<div class="trajectory-section">
			<TrajectoryPanel {loopId} />
		</div>
	{/if}
</div>

<style>
	.page {
		padding: var(--space-6);
		max-width: 900px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		min-height: 100%;
	}

	/* ------------------------------------------------------------------ */
	/* Back link                                                            */
	/* ------------------------------------------------------------------ */

	.back-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--color-text-muted);
		text-decoration: none;
		font-size: var(--font-size-sm);
		align-self: flex-start;
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		transition: all var(--transition-fast);
	}

	.back-link:hover {
		color: var(--color-text-primary);
		background: var(--color-bg-tertiary);
	}

	/* ------------------------------------------------------------------ */
	/* Loading / error states                                              */
	/* ------------------------------------------------------------------ */

	.loading-state,
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-12);
		text-align: center;
		color: var(--color-text-secondary);
	}

	.error-state {
		color: var(--color-error);
	}

	.error-state p {
		color: var(--color-text-secondary);
		margin: 0;
	}

	/* ------------------------------------------------------------------ */
	/* Loop header                                                          */
	/* ------------------------------------------------------------------ */

	.loop-header {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-4) var(--space-5);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
	}

	.header-top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		flex-wrap: wrap;
	}

	.loop-id {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-sm);
		color: var(--color-info);
		background: var(--color-info-muted);
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		word-break: break-all;
	}

	.badges {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
	}

	.role-badge {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		background: var(--color-bg-tertiary);
		border: 1px solid var(--color-border);
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius-full);
	}

	.state-badge {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius-full);
	}

	.state-executing {
		background: var(--color-accent-muted);
		color: var(--color-accent);
	}

	.state-success {
		background: var(--color-success-muted);
		color: var(--color-success);
	}

	.state-failed {
		background: var(--color-error-muted);
		color: var(--color-error);
	}

	.state-paused {
		background: var(--color-warning-muted);
		color: var(--color-warning);
	}

	.state-cancelled,
	.state-default {
		background: var(--color-bg-tertiary);
		color: var(--color-text-muted);
	}

	.header-meta {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: var(--space-4);
	}

	.meta-item {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-sm);
		color: var(--color-text-secondary);
	}

	.token-detail {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		font-family: var(--font-family-mono);
	}

	/* ------------------------------------------------------------------ */
	/* Controls bar                                                        */
	/* ------------------------------------------------------------------ */

	.controls-bar {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
		padding: var(--space-3) var(--space-4);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
	}

	.signal-error {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		color: var(--color-error);
		margin-right: var(--space-2);
	}

	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-4);
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		border-radius: var(--radius-md);
		border: 1px solid transparent;
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.btn-secondary {
		background: var(--color-bg-tertiary);
		color: var(--color-text-primary);
		border-color: var(--color-border);
	}

	.btn-secondary:hover:not(:disabled) {
		background: var(--color-bg-elevated);
	}

	.btn-warning {
		background: var(--color-warning-muted);
		color: var(--color-warning);
		border-color: var(--color-warning);
	}

	.btn-warning:hover:not(:disabled) {
		background: var(--color-warning);
		color: white;
	}

	.btn-success {
		background: var(--color-success-muted);
		color: var(--color-success);
		border-color: var(--color-success);
	}

	.btn-success:hover:not(:disabled) {
		background: var(--color-success);
		color: white;
	}

	.btn-danger {
		background: var(--color-error-muted);
		color: var(--color-error);
		border-color: var(--color-error);
	}

	.btn-danger:hover:not(:disabled) {
		background: var(--color-error);
		color: white;
	}

	/* ------------------------------------------------------------------ */
	/* CollapsiblePanel inner content                                      */
	/* ------------------------------------------------------------------ */

	.panel-inner {
		padding: var(--space-3);
	}

	/* ------------------------------------------------------------------ */
	/* Dynamic Tools                                                       */
	/* ------------------------------------------------------------------ */

	.tools-loading {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-4);
		color: var(--color-text-muted);
		font-size: var(--font-size-sm);
	}

	.tools-empty {
		padding: var(--space-4);
		color: var(--color-text-muted);
		font-size: var(--font-size-sm);
		text-align: center;
		margin: 0;
	}

	.tools-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.tool-item {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		padding: var(--space-3);
		background: var(--color-bg-primary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
	}

	.tool-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
	}

	.tool-name {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
	}

	.processor-count {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}

	.tool-description {
		margin: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		line-height: var(--line-height-relaxed);
	}

	.processor-list {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.processor-chip {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-warning);
		background: var(--color-warning-muted);
		border: 1px solid color-mix(in srgb, var(--color-warning) 30%, transparent);
		padding: 2px var(--space-2);
		border-radius: var(--radius-sm);
	}

	/* ------------------------------------------------------------------ */
	/* Trajectory section                                                  */
	/* ------------------------------------------------------------------ */

	.trajectory-section {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-height: 400px;
	}

	/* ------------------------------------------------------------------ */
	/* Spinner                                                             */
	/* ------------------------------------------------------------------ */

	:global(.spin) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
