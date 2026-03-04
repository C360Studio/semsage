<script lang="ts">
	/**
	 * LoopCard - Displays a loop's status, role, model, iteration progress,
	 * duration, and pause/resume/cancel controls for active or paused loops.
	 *
	 * Used within AgentTreeNode on the activity page and in the loop detail view.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';
	import { loopsStore } from '$lib/stores/loops.svelte';
	import type { Loop, LoopState } from '$lib/types';

	interface Props {
		loop: Loop;
		onPause?: () => void;
		onResume?: () => void;
		onCancel?: () => void;
		/** Whether to show the navigation link on the loop ID */
		linkable?: boolean;
	}

	let { loop, onPause, onResume, onCancel, linkable = true }: Props = $props();

	const shortId = $derived(loop.loop_id.slice(0, 8));
	const loopState = $derived(loop.state as LoopState);

	const isActive = $derived(['pending', 'exploring', 'executing'].includes(loopState));
	const isPaused = $derived(loopState === 'paused');
	const isComplete = $derived(['complete', 'success', 'failed', 'cancelled'].includes(loopState));
	const isExecuting = $derived(loopState === 'executing');

	const progressPct = $derived(
		loop.max_iterations > 0
			? Math.min(100, (loop.iterations / loop.max_iterations) * 100)
			: 0
	);

	const truncatedModel = $derived(
		loop.model.length > 24 ? loop.model.slice(0, 22) + '…' : loop.model
	);

	function formatDuration(createdAt?: string): string {
		if (!createdAt) return '';
		const ms = Date.now() - new Date(createdAt).getTime();
		const s = Math.floor(ms / 1000);
		if (s < 60) return `${s}s`;
		const m = Math.floor(s / 60);
		if (m < 60) return `${m}m ${s % 60}s`;
		const h = Math.floor(m / 60);
		return `${h}h ${m % 60}m`;
	}

	const duration = $derived(formatDuration(loop.created_at));

	async function handlePause() {
		if (onPause) {
			onPause();
		} else {
			await loopsStore.sendSignal(loop.loop_id, 'pause');
		}
	}

	async function handleResume() {
		if (onResume) {
			onResume();
		} else {
			await loopsStore.sendSignal(loop.loop_id, 'resume');
		}
	}

	async function handleCancel() {
		if (onCancel) {
			onCancel();
		} else {
			await loopsStore.sendSignal(loop.loop_id, 'cancel');
		}
	}


</script>

<div
	class="loop-card"
	class:active={isActive}
	class:paused={isPaused}
	class:complete={isComplete}
	data-state={loopState}
>
	<!-- Header: ID + state badge -->
	<div class="loop-header">
		{#if linkable}
			<a
				href="/loops/{loop.loop_id}"
				class="loop-id linkable"
				title={loop.loop_id}
				aria-label="Navigate to loop {loop.loop_id}"
			>
				{shortId}
			</a>
		{:else}
			<span class="loop-id" title={loop.loop_id}>
				{shortId}
			</span>
		{/if}
		<span
			class="state-badge"
			class:executing={isExecuting || isActive}
			class:paused={isPaused}
			class:complete={isComplete}
			class:failed={loopState === 'failed'}
		>
			{loopState}
		</span>
	</div>

	<!-- Role -->
	<div class="loop-role" title={loop.role}>
		<Icon name="bot" size={12} />
		<span>{loop.role}</span>
	</div>

	<!-- Model -->
	<div class="loop-model" title={loop.model}>
		<Icon name="cpu" size={12} />
		<span>{truncatedModel}</span>
	</div>

	<!-- Iteration progress -->
	<div class="loop-progress">
		<div class="progress-bar" role="progressbar" aria-valuenow={loop.iterations} aria-valuemin={0} aria-valuemax={loop.max_iterations}>
			<div class="progress-fill" style:width="{progressPct}%"></div>
		</div>
		<span class="progress-text">{loop.iterations}/{loop.max_iterations}</span>
	</div>

	<!-- Duration (if started) -->
	{#if duration && !isComplete}
		<div class="loop-duration">
			<Icon name="clock" size={12} />
			<span>{duration}</span>
		</div>
	{/if}

	<!-- Controls -->
	{#if isActive || isPaused}
		<div class="loop-actions">
			{#if isActive}
				<button
					type="button"
					class="action-btn pause"
					onclick={handlePause}
					title="Pause loop"
					aria-label="Pause loop {shortId}"
				>
					<Icon name="pause" size={13} />
				</button>
			{/if}
			{#if isPaused}
				<button
					type="button"
					class="action-btn resume"
					onclick={handleResume}
					title="Resume loop"
					aria-label="Resume loop {shortId}"
				>
					<Icon name="play" size={13} />
				</button>
			{/if}
			<button
				type="button"
				class="action-btn cancel"
				onclick={handleCancel}
				title="Cancel loop (cascades to children)"
				aria-label="Cancel loop {shortId}"
			>
				<Icon name="x" size={13} />
			</button>
		</div>
	{/if}
</div>

<style>
	.loop-card {
		background: var(--color-bg-tertiary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		padding: var(--space-3);
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		transition: border-color var(--transition-fast);
	}

	.loop-card.active {
		border-color: var(--color-accent);
	}

	.loop-card.paused {
		border-color: var(--color-warning);
	}

	.loop-card.complete {
		opacity: 0.7;
	}

	.loop-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
	}

	.loop-id {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		background: none;
		border: none;
		padding: 0;
		cursor: default;
		text-decoration: none;
	}

	.loop-id.linkable {
		cursor: pointer;
		text-decoration: underline dotted;
		color: var(--color-accent);
	}

	.loop-id.linkable:hover {
		color: var(--color-accent-hover);
	}

	.state-badge {
		font-size: var(--font-size-xs);
		padding: 2px 6px;
		border-radius: var(--radius-full);
		background: var(--color-bg-elevated);
		color: var(--color-text-secondary);
		white-space: nowrap;
	}

	.state-badge.executing {
		background: var(--color-accent-muted);
		color: var(--color-accent);
	}

	.state-badge.paused {
		background: var(--color-warning-muted);
		color: var(--color-warning);
	}

	.state-badge.complete {
		background: var(--color-success-muted);
		color: var(--color-success);
	}

	.state-badge.failed {
		background: var(--color-error-muted);
		color: var(--color-error);
	}

	.loop-role,
	.loop-model,
	.loop-duration {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		min-width: 0;
	}

	.loop-role {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-primary);
	}

	.loop-role span,
	.loop-model span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.loop-progress {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.progress-bar {
		flex: 1;
		height: 4px;
		background: var(--color-bg-elevated);
		border-radius: var(--radius-full);
		overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: var(--color-accent);
		border-radius: var(--radius-full);
		transition: width var(--transition-base);
	}

	.progress-text {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	.loop-actions {
		display: flex;
		gap: var(--space-1);
		flex-wrap: wrap;
	}

	.action-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		border-radius: var(--radius-md);
		cursor: pointer;
		transition: filter var(--transition-fast);
		flex-shrink: 0;
	}

	.action-btn:hover {
		filter: brightness(1.2);
	}

	.action-btn:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.action-btn.pause {
		background: var(--color-warning-muted);
		color: var(--color-warning);
	}

	.action-btn.resume {
		background: var(--color-success-muted);
		color: var(--color-success);
	}

	.action-btn.cancel {
		background: var(--color-error-muted);
		color: var(--color-error);
	}
</style>
