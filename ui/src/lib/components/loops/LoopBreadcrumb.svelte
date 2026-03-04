<script lang="ts">
	/**
	 * LoopBreadcrumb - Parent chain breadcrumb showing root → ... → parent → current.
	 *
	 * Receives pre-resolved ancestors from the parent page (which fetches them
	 * by walking parent_loop_id chain). Each ancestor is a clickable link.
	 * The current loop is shown as non-clickable, bold.
	 */

	import type { Loop } from '$lib/types';
	import Icon from '$lib/components/shared/Icon.svelte';

	interface Props {
		loop: Loop;
		ancestors?: Loop[];
	}

	let { loop, ancestors = [] }: Props = $props();

	function roleLabel(l: Loop): string {
		return l.role || l.loop_id.slice(0, 8);
	}
</script>

<nav class="loop-breadcrumb" aria-label="Loop ancestry">
	<ol class="breadcrumb-list">
		{#each ancestors as ancestor, i (ancestor.loop_id)}
			<li class="breadcrumb-item">
				{#if i > 0}
					<Icon name="chevron-right" size={12} class="separator-icon" />
				{/if}
				<a href="/loops/{ancestor.loop_id}" class="breadcrumb-link" title="Loop {ancestor.loop_id}">
					<span class="ancestor-role">{roleLabel(ancestor)}</span>
				</a>
			</li>
		{/each}

		{#if ancestors.length > 0}
			<li class="breadcrumb-sep" aria-hidden="true">
				<Icon name="chevron-right" size={12} class="separator-icon" />
			</li>
		{/if}

		<li class="breadcrumb-item current" aria-current="page">
			<span class="current-role">{roleLabel(loop)}</span>
		</li>
	</ol>
</nav>

<style>
	.loop-breadcrumb {
		padding: var(--space-1) 0;
	}

	.breadcrumb-list {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0;
		list-style: none;
		padding: 0;
		margin: 0;
		font-size: var(--font-size-sm);
	}

	.breadcrumb-item {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	.breadcrumb-sep {
		display: flex;
		align-items: center;
	}

	.breadcrumb-link {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--color-accent);
		text-decoration: none;
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		transition: background var(--transition-fast);
	}

	.breadcrumb-link:hover {
		background: var(--color-accent-muted);
		text-decoration: none;
	}

	.ancestor-role {
		font-weight: var(--font-weight-medium);
	}

	:global(.separator-icon) {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.breadcrumb-item.current {
		padding: var(--space-1) var(--space-2);
	}

	.current-role {
		color: var(--color-text-primary);
		font-weight: var(--font-weight-semibold);
	}
</style>
