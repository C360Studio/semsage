<script lang="ts">
	/**
	 * SpawnAgentEntry - Trajectory entry for spawn_agent tool calls.
	 *
	 * Shows child loop ID as a link, role, inline status badge, prompt snippet,
	 * and duration. Styled distinctly from generic tool calls to emphasize
	 * agent hierarchy relationships.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';

	interface Props {
		toolCallId: string;
		childLoopId?: string;
		childRole?: string;
		childState?: string;
		promptSnippet?: string;
		status?: string;
		durationMs?: number;
	}

	let {
		toolCallId,
		childLoopId,
		childRole,
		childState,
		promptSnippet,
		status = 'unknown',
		durationMs
	}: Props = $props();

	const isSuccess = $derived(status === 'success' || childState === 'success' || childState === 'complete');
	const isFailed = $derived(status === 'failed' || childState === 'failed');
	const isPending = $derived(!isSuccess && !isFailed);

	function formatDuration(ms: number | undefined): string {
		if (ms === undefined) return '';
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(1)}s`;
	}

	function truncate(text: string, max = 100): string {
		return text.length > max ? text.slice(0, max) + '…' : text;
	}
</script>

<div class="spawn-agent-entry" aria-label="Spawn agent call">
	<div class="entry-header">
		<span class="tool-badge">
			<Icon name="git-branch" size={11} />
			spawn_agent
		</span>

		<div class="status-group">
			{#if isPending}
				<span class="status-indicator pending" title="Waiting">
					<Icon name="loader" size={12} class="spin" />
					waiting
				</span>
			{:else if isSuccess}
				<span class="status-indicator success" title="Completed">
					<Icon name="check-circle" size={12} />
					completed
				</span>
			{:else if isFailed}
				<span class="status-indicator failed" title="Failed">
					<Icon name="alert-circle" size={12} />
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

	<div class="child-info">
		{#if childRole}
			<span class="child-role">{childRole}</span>
		{/if}

		{#if childLoopId}
			<a href="/loops/{childLoopId}" class="child-link" title="View child loop detail">
				<Icon name="arrow-right" size={12} />
				<code class="child-id">{childLoopId.slice(0, 12)}…</code>
				<Icon name="external-link" size={11} class="link-icon" />
			</a>
		{:else}
			<span class="no-child">Spawning…</span>
		{/if}
	</div>

	{#if promptSnippet}
		<p class="prompt-snippet">{truncate(promptSnippet)}</p>
	{/if}
</div>

<style>
	.spawn-agent-entry {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-3);
		background: var(--color-accent-muted);
		border: 1px solid var(--color-accent);
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
		color: var(--color-accent);
		padding: 2px var(--space-2);
		background: color-mix(in srgb, var(--color-accent) 15%, transparent);
		border-radius: var(--radius-full);
	}

	.status-group {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.status-indicator {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
	}

	.status-indicator.pending {
		color: var(--color-text-muted);
	}

	.status-indicator.success {
		color: var(--color-success);
	}

	.status-indicator.failed {
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

	.child-info {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.child-role {
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		background: var(--color-bg-tertiary);
		padding: 2px var(--space-2);
		border-radius: var(--radius-sm);
		border: 1px solid var(--color-border);
	}

	.child-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--color-accent);
		text-decoration: none;
		font-size: var(--font-size-xs);
		transition: opacity var(--transition-fast);
	}

	.child-link:hover {
		opacity: 0.8;
		text-decoration: underline;
	}

	.child-id {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
	}

	:global(.link-icon) {
		opacity: 0.6;
	}

	.no-child {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		font-style: italic;
	}

	.prompt-snippet {
		margin: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		line-height: var(--line-height-relaxed);
		padding: var(--space-2) var(--space-3);
		background: var(--color-bg-secondary);
		border-radius: var(--radius-sm);
		border-left: 2px solid var(--color-accent);
	}

	:global(.spin) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}
</style>
