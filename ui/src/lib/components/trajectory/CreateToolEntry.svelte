<script lang="ts">
	/**
	 * CreateToolEntry - Trajectory entry for create_tool calls.
	 *
	 * Displays the dynamically created tool name, description, list of
	 * processors composed into the tool, and the wiring summary.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';

	interface Props {
		toolName?: string;
		description?: string;
		processors?: string[];
		wiring?: Record<string, unknown>;
		status?: string;
		durationMs?: number;
	}

	let {
		toolName = 'unnamed',
		description,
		processors = [],
		wiring,
		status = 'unknown',
		durationMs
	}: Props = $props();

	let wiringExpanded = $state(false);

	const hasWiring = $derived(wiring && Object.keys(wiring).length > 0);

	function formatDuration(ms: number | undefined): string {
		if (ms === undefined) return '';
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(1)}s`;
	}
</script>

<div class="create-tool-entry" aria-label="Create tool call">
	<div class="entry-header">
		<span class="tool-badge">
			<Icon name="wrench" size={11} />
			create_tool
		</span>

		<div class="meta-group">
			{#if status === 'success'}
				<span class="status-chip success">
					<Icon name="check-circle" size={11} />
					registered
				</span>
			{:else if status === 'failed' || status === 'error'}
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

	<div class="tool-identity">
		<code class="tool-name">{toolName}</code>
		{#if description}
			<p class="tool-description">{description}</p>
		{/if}
	</div>

	{#if processors.length > 0}
		<div class="processors-section">
			<span class="section-label">
				<Icon name="layers" size={12} />
				Processors ({processors.length})
			</span>
			<ul class="processor-list">
				{#each processors as proc (proc)}
					<li class="processor-chip">{proc}</li>
				{/each}
			</ul>
		</div>
	{/if}

	{#if hasWiring}
		<div class="wiring-section">
			<button
				class="wiring-toggle"
				onclick={() => (wiringExpanded = !wiringExpanded)}
				aria-expanded={wiringExpanded}
			>
				<Icon name={wiringExpanded ? 'chevron-up' : 'chevron-down'} size={12} />
				<span>Wiring</span>
			</button>

			{#if wiringExpanded}
				<pre class="wiring-json">{JSON.stringify(wiring, null, 2)}</pre>
			{/if}
		</div>
	{/if}
</div>

<style>
	.create-tool-entry {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-3);
		background: var(--color-warning-muted);
		border: 1px solid var(--color-warning);
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
		color: var(--color-warning);
		padding: 2px var(--space-2);
		background: color-mix(in srgb, var(--color-warning) 15%, transparent);
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
		background: var(--color-success-muted);
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

	.tool-identity {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.tool-name {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		background: var(--color-bg-secondary);
		padding: 2px var(--space-2);
		border-radius: var(--radius-sm);
		display: inline-block;
	}

	.tool-description {
		margin: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		line-height: var(--line-height-relaxed);
	}

	.processors-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.section-label {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-secondary);
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
		background: color-mix(in srgb, var(--color-warning) 12%, var(--color-bg-secondary));
		border: 1px solid color-mix(in srgb, var(--color-warning) 30%, transparent);
		padding: 2px var(--space-2);
		border-radius: var(--radius-sm);
	}

	.wiring-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.wiring-toggle {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		background: transparent;
		border: none;
		cursor: pointer;
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		padding: 0;
		transition: color var(--transition-fast);
	}

	.wiring-toggle:hover {
		color: var(--color-text-primary);
	}

	.wiring-json {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		padding: var(--space-2) var(--space-3);
		margin: 0;
		white-space: pre-wrap;
		word-break: break-word;
		max-height: 200px;
		overflow-y: auto;
	}
</style>
