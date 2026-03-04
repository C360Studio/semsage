<script lang="ts">
	/**
	 * EntityCard - Entity preview card for the entity browser.
	 *
	 * Shows entity ID (mono, truncated), type badge derived from ID structure,
	 * and up to 4 key predicates as label:value pairs.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';
	import type { Entity } from '$lib/types';

	interface Props {
		entity: Entity;
	}

	let { entity }: Props = $props();

	// Map entity types to icons
	const typeIcons: Record<string, string> = {
		code: 'file-code',
		proposal: 'lightbulb',
		spec: 'file-text',
		task: 'check-square',
		loop: 'refresh-cw',
		activity: 'activity'
	};

	// Map entity types to CSS color tokens
	const typeColors: Record<string, string> = {
		code: 'var(--color-info)',
		proposal: 'var(--color-warning)',
		spec: 'var(--color-success)',
		task: 'var(--color-accent)',
		loop: 'var(--color-info)',
		activity: 'var(--color-text-muted)'
	};

	// Map entity types to background muted tokens
	const typeBgs: Record<string, string> = {
		code: 'var(--color-info-muted)',
		proposal: 'var(--color-warning-muted)',
		spec: 'var(--color-success-muted)',
		task: 'var(--color-accent-muted)',
		loop: 'var(--color-info-muted)',
		activity: 'var(--color-bg-tertiary)'
	};

	function getIcon(type: string): string {
		return typeIcons[type] ?? 'circle';
	}

	function getColor(type: string): string {
		return typeColors[type] ?? 'var(--color-text-muted)';
	}

	function getBg(type: string): string {
		return typeBgs[type] ?? 'var(--color-bg-tertiary)';
	}

	/**
	 * Select up to 4 key predicates for display.
	 * Prefers predicates with short, human-readable values.
	 */
	const keyPredicates = $derived.by(() => {
		const entries = Object.entries(entity.predicates);
		const MAX = 4;

		// Priority predicates to show first
		const priority = [
			'dc.terms.title',
			'agentic.loop.role',
			'agentic.loop.state',
			'agentic.task.status',
			'code.artifact.path',
			'code.artifact.language',
			'prov.generatedAtTime'
		];

		const sorted = entries.sort(([a], [b]) => {
			const ai = priority.indexOf(a);
			const bi = priority.indexOf(b);
			if (ai !== -1 && bi !== -1) return ai - bi;
			if (ai !== -1) return -1;
			if (bi !== -1) return 1;
			return 0;
		});

		return sorted.slice(0, MAX).map(([key, value]) => {
			// Use the last segment of the predicate key as the label
			const parts = key.split('.');
			const label = parts[parts.length - 1] ?? key;
			const raw = typeof value === 'string' ? value : JSON.stringify(value);
			// Truncate long values
			const display = raw.length > 40 ? raw.slice(0, 38) + '…' : raw;
			return { key, label, display };
		});
	});

	// Truncate long IDs for display
	const displayId = $derived(
		entity.id.length > 50 ? entity.id.slice(0, 24) + '…' + entity.id.slice(-16) : entity.id
	);
</script>

<a
	href="/entities/{encodeURIComponent(entity.id)}"
	class="entity-card"
	aria-label="View entity {entity.name}"
	title={entity.id}
>
	<div class="entity-header">
		<div
			class="entity-icon"
			style="color: {getColor(entity.type)}; background: {getBg(entity.type)}"
			aria-hidden="true"
		>
			<Icon name={getIcon(entity.type)} size={16} />
		</div>

		<div class="entity-title">
			<span class="entity-name">{entity.name}</span>
			<span
				class="entity-type-badge"
				style="color: {getColor(entity.type)}; background: {getBg(entity.type)}"
			>
				{entity.type}
			</span>
		</div>

		<div class="entity-arrow" aria-hidden="true">
			<Icon name="chevron-right" size={14} />
		</div>
	</div>

	<div class="entity-id" title={entity.id}>{displayId}</div>

	{#if keyPredicates.length > 0}
		<div class="entity-predicates">
			{#each keyPredicates as { label, display } (label)}
				<div class="predicate-row">
					<span class="predicate-label">{label}</span>
					<span class="predicate-value">{display}</span>
				</div>
			{/each}
		</div>
	{/if}

	{#if entity.createdAt}
		<div class="entity-date">
			{new Date(entity.createdAt).toLocaleDateString(undefined, {
				month: 'short',
				day: 'numeric',
				year: 'numeric'
			})}
		</div>
	{/if}
</a>

<style>
	.entity-card {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-4);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		text-decoration: none;
		transition: all var(--transition-fast);
		cursor: pointer;
	}

	.entity-card:hover {
		border-color: var(--color-accent);
		background: var(--color-bg-tertiary);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
	}

	.entity-card:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.entity-header {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.entity-icon {
		width: 28px;
		height: 28px;
		border-radius: var(--radius-md);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.entity-title {
		flex: 1;
		min-width: 0;
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.entity-name {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.entity-type-badge {
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		padding: 1px 6px;
		border-radius: var(--radius-full);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		flex-shrink: 0;
	}

	.entity-arrow {
		color: var(--color-text-muted);
		flex-shrink: 0;
		opacity: 0;
		transition: opacity var(--transition-fast);
	}

	.entity-card:hover .entity-arrow {
		opacity: 1;
	}

	.entity-id {
		font-size: var(--font-size-xs);
		font-family: var(--font-family-mono);
		color: var(--color-text-muted);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.entity-predicates {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding-top: var(--space-1);
		border-top: 1px solid var(--color-border);
	}

	.predicate-row {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		font-size: var(--font-size-xs);
	}

	.predicate-label {
		font-family: var(--font-family-mono);
		color: var(--color-text-muted);
		white-space: nowrap;
		flex-shrink: 0;
		min-width: 60px;
	}

	.predicate-value {
		color: var(--color-text-secondary);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.entity-date {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}
</style>
