<script lang="ts">
	/**
	 * DynamicToolList - List of tools created via create_tool, scoped to a root loop.
	 *
	 * Phase 2 implementation placeholder.
	 */

	import type { DynamicTool } from '$lib/types';

	interface Props {
		tools: DynamicTool[];
		loading?: boolean;
	}

	let { tools, loading = false }: Props = $props();
</script>

<div class="dynamic-tool-list">
	{#if loading}
		<p class="loading">Loading tools...</p>
	{:else if tools.length === 0}
		<p class="empty">No dynamic tools created by this agent tree</p>
	{:else}
		<ul class="tools-list">
			{#each tools as tool (tool.name)}
				<li class="tool-item">
					<div class="tool-name">{tool.name}</div>
					{#if tool.description}
						<div class="tool-description">{tool.description}</div>
					{/if}
					{#if tool.processors && tool.processors.length > 0}
						<div class="tool-processors">
							{tool.processors.length} processor{tool.processors.length !== 1 ? 's' : ''}
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.dynamic-tool-list {
		display: flex;
		flex-direction: column;
	}

	.loading,
	.empty {
		color: var(--color-text-muted);
		font-size: var(--font-size-sm);
		padding: var(--space-3);
	}

	.tools-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.tool-item {
		padding: var(--space-3);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
	}

	.tool-name {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-semibold);
		font-family: var(--font-family-mono);
		color: var(--color-text-primary);
		margin-bottom: var(--space-1);
	}

	.tool-description {
		font-size: var(--font-size-xs);
		color: var(--color-text-secondary);
		margin-bottom: var(--space-1);
	}

	.tool-processors {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}
</style>
