<script lang="ts">
	/**
	 * ChildLoopList - List of child loops on the loop detail page.
	 *
	 * Shows role, state badge, and result summary (if completed) for each child.
	 * Each child navigates to its own detail page on click.
	 */

	import type { Loop, LoopState } from '$lib/types';
	import Icon from '$lib/components/shared/Icon.svelte';

	interface Props {
		children: Loop[];
		loading?: boolean;
	}

	let { children, loading = false }: Props = $props();

	function stateClass(state: LoopState | string): string {
		switch (state) {
			case 'executing':
				return 'state-executing';
			case 'complete':
			case 'success':
				return 'state-success';
			case 'failed':
				return 'state-failed';
			case 'paused':
				return 'state-paused';
			case 'pending':
			case 'exploring':
				return 'state-pending';
			case 'cancelled':
				return 'state-cancelled';
			default:
				return 'state-default';
		}
	}

	function stateIcon(state: LoopState | string): string {
		switch (state) {
			case 'executing':
			case 'exploring':
				return 'loader';
			case 'complete':
			case 'success':
				return 'check-circle';
			case 'failed':
				return 'alert-circle';
			case 'paused':
				return 'pause';
			case 'cancelled':
				return 'x';
			default:
				return 'circle';
		}
	}

	function isSpinning(state: LoopState | string): boolean {
		return state === 'executing' || state === 'exploring';
	}

	function truncate(text: string, max = 80): string {
		return text.length > max ? text.slice(0, max) + '…' : text;
	}
</script>

<div class="child-loop-list">
	{#if loading}
		<div class="state-message">
			<Icon name="loader" size={16} class="spin" />
			<span>Loading children…</span>
		</div>
	{:else if children.length === 0}
		<div class="state-message empty">
			<Icon name="git-branch" size={16} />
			<span>No child loops spawned</span>
		</div>
	{:else}
		<ul class="children-list" aria-label="Child loops">
			{#each children as child (child.loop_id)}
				<li class="child-item">
					<a href="/loops/{child.loop_id}" class="child-link">
						<div class="child-main">
							<span class="child-role">{child.role || 'unknown'}</span>
							<code class="child-id">{child.loop_id.slice(0, 12)}…</code>
						</div>

						<div class="child-meta">
							{#if child.iterations > 0}
								<span class="iteration-count">
									<Icon name="refresh-cw" size={11} />
									{child.iterations}/{child.max_iterations}
								</span>
							{/if}

							<span class="state-badge {stateClass(child.state)}">
								<Icon
									name={stateIcon(child.state)}
									size={11}
									class={isSpinning(child.state) ? 'spin' : ''}
								/>
								{child.state}
							</span>

							<Icon name="chevron-right" size={14} class="nav-arrow" />
						</div>
					</a>

					{#if child.result && (child.state === 'complete' || child.state === 'success')}
						<div class="child-result">
							<span class="result-label">Result:</span>
							<span class="result-text">{truncate(child.result)}</span>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.child-loop-list {
		display: flex;
		flex-direction: column;
	}

	.state-message {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-4);
		color: var(--color-text-muted);
		font-size: var(--font-size-sm);
	}

	.state-message.empty {
		justify-content: center;
		padding: var(--space-6);
	}

	.children-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.child-item {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		overflow: hidden;
		background: var(--color-bg-secondary);
		transition: border-color var(--transition-fast);
	}

	.child-item:hover {
		border-color: var(--color-accent);
	}

	.child-link {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		text-decoration: none;
		transition: background var(--transition-fast);
	}

	.child-link:hover {
		background: var(--color-bg-tertiary);
		text-decoration: none;
	}

	.child-main {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.child-role {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		white-space: nowrap;
	}

	.child-id {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	.child-meta {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.iteration-count {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}

	.state-badge {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
		white-space: nowrap;
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

	.state-pending {
		background: var(--color-bg-tertiary);
		color: var(--color-text-muted);
	}

	.state-cancelled {
		background: var(--color-bg-tertiary);
		color: var(--color-text-muted);
	}

	.state-default {
		background: var(--color-bg-tertiary);
		color: var(--color-text-muted);
	}

	:global(.nav-arrow) {
		color: var(--color-text-muted);
	}

	.child-result {
		padding: var(--space-2) var(--space-4);
		border-top: 1px solid var(--color-border);
		background: var(--color-bg-tertiary);
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		display: flex;
		gap: var(--space-1);
	}

	.result-label {
		font-weight: var(--font-weight-medium);
		color: var(--color-text-primary);
		flex-shrink: 0;
	}

	.result-text {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	:global(.spin) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
