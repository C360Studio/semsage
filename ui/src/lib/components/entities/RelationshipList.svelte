<script lang="ts">
	/**
	 * RelationshipList - Displays relationships grouped by predicate.
	 *
	 * Groups incoming and outgoing relationships separately.
	 * Color-codes by predicate type for quick visual scanning.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';
	import type { Relationship } from '$lib/types';

	interface Props {
		relationships: Relationship[];
	}

	let { relationships }: Props = $props();

	// Predicate color mapping for well-known semsage predicates
	function getPredicateColor(predicate: string): string {
		if (predicate.includes('loop.spawned') || predicate.includes('loop.parent')) {
			return 'var(--color-accent)';
		}
		if (predicate.includes('task.depends_on') || predicate.includes('task.blocks')) {
			return 'var(--color-warning)';
		}
		if (predicate.includes('prov.')) {
			return 'var(--color-info)';
		}
		if (predicate.includes('loop.') || predicate.includes('agentic.')) {
			return 'var(--color-success)';
		}
		return 'var(--color-text-muted)';
	}

	function getPredicateBg(predicate: string): string {
		if (predicate.includes('loop.spawned') || predicate.includes('loop.parent')) {
			return 'var(--color-accent-muted)';
		}
		if (predicate.includes('task.depends_on') || predicate.includes('task.blocks')) {
			return 'var(--color-warning-muted)';
		}
		if (predicate.includes('prov.')) {
			return 'var(--color-info-muted)';
		}
		if (predicate.includes('loop.') || predicate.includes('agentic.')) {
			return 'var(--color-success-muted)';
		}
		return 'var(--color-bg-tertiary)';
	}

	/** Group relationships by predicate, keeping direction. */
	interface PredicateGroup {
		predicate: string;
		predicateLabel: string;
		outgoing: Relationship[];
		incoming: Relationship[];
	}

	const grouped = $derived.by((): PredicateGroup[] => {
		const map = new Map<string, PredicateGroup>();

		for (const rel of relationships) {
			let group = map.get(rel.predicate);
			if (!group) {
				group = {
					predicate: rel.predicate,
					predicateLabel: rel.predicateLabel,
					outgoing: [],
					incoming: []
				};
				map.set(rel.predicate, group);
			}
			if (rel.direction === 'outgoing') {
				group.outgoing.push(rel);
			} else {
				group.incoming.push(rel);
			}
		}

		// Sort: outgoing predicates first, then by predicate name
		return Array.from(map.values()).sort((a, b) => {
			const aHasOut = a.outgoing.length > 0 ? 0 : 1;
			const bHasOut = b.outgoing.length > 0 ? 0 : 1;
			if (aHasOut !== bHasOut) return aHasOut - bHasOut;
			return a.predicate.localeCompare(b.predicate);
		});
	});

	const outgoingCount = $derived(relationships.filter((r) => r.direction === 'outgoing').length);
	const incomingCount = $derived(relationships.filter((r) => r.direction === 'incoming').length);

	/** Truncate a long entity ID for display. */
	function truncateId(id: string): string {
		if (id.length <= 40) return id;
		return id.slice(0, 20) + '…' + id.slice(-12);
	}
</script>

<div class="relationship-list">
	{#if relationships.length === 0}
		<p class="empty">No relationships found for this entity.</p>
	{:else}
		<div class="summary">
			<span class="summary-badge outgoing-badge">
				<Icon name="arrow-right" size={12} />
				{outgoingCount} outgoing
			</span>
			<span class="summary-badge incoming-badge">
				<Icon name="arrow-left" size={12} />
				{incomingCount} incoming
			</span>
		</div>

		{#each grouped as group (group.predicate)}
			<div class="predicate-group">
				<div class="predicate-header">
					<span
						class="predicate-pill"
						style="color: {getPredicateColor(group.predicate)}; background: {getPredicateBg(group.predicate)}"
						title={group.predicate}
					>
						{group.predicateLabel}
					</span>
					<span class="predicate-full" title={group.predicate}>{group.predicate}</span>
					<span class="predicate-count">
						{group.outgoing.length + group.incoming.length}
					</span>
				</div>

				{#if group.outgoing.length > 0}
					<div class="direction-section">
						<span class="direction-label outgoing">
							<Icon name="arrow-right" size={12} />
							outgoing
						</span>
						<ul class="relationship-items">
							{#each group.outgoing as rel (`out-${rel.targetId}`)}
								<li class="relationship-item">
									<a
										href="/entities/{encodeURIComponent(rel.targetId)}"
										class="rel-target"
										title={rel.targetId}
									>
										{truncateId(rel.targetName !== rel.targetId ? rel.targetName : rel.targetId)}
									</a>
									<span class="rel-type-badge">{rel.targetType}</span>
								</li>
							{/each}
						</ul>
					</div>
				{/if}

				{#if group.incoming.length > 0}
					<div class="direction-section">
						<span class="direction-label incoming">
							<Icon name="arrow-left" size={12} />
							incoming
						</span>
						<ul class="relationship-items">
							{#each group.incoming as rel (`in-${rel.targetId}`)}
								<li class="relationship-item">
									<a
										href="/entities/{encodeURIComponent(rel.targetId)}"
										class="rel-target"
										title={rel.targetId}
									>
										{truncateId(rel.targetName !== rel.targetId ? rel.targetName : rel.targetId)}
									</a>
									<span class="rel-type-badge">{rel.targetType}</span>
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			</div>
		{/each}
	{/if}
</div>

<style>
	.relationship-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.empty {
		color: var(--color-text-muted);
		font-size: var(--font-size-sm);
		text-align: center;
		padding: var(--space-6);
	}

	.summary {
		display: flex;
		gap: var(--space-3);
	}

	.summary-badge {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
	}

	.outgoing-badge {
		color: var(--color-success);
		background: var(--color-success-muted);
	}

	.incoming-badge {
		color: var(--color-info);
		background: var(--color-info-muted);
	}

	.predicate-group {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		overflow: hidden;
	}

	.predicate-header {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		background: var(--color-bg-tertiary);
		border-bottom: 1px solid var(--color-border);
	}

	.predicate-pill {
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-semibold);
		padding: 2px var(--space-2);
		border-radius: var(--radius-full);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		flex-shrink: 0;
	}

	.predicate-full {
		font-size: var(--font-size-xs);
		font-family: var(--font-family-mono);
		color: var(--color-text-muted);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		flex: 1;
		min-width: 0;
	}

	.predicate-count {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		background: var(--color-bg-elevated);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-full);
		padding: 0 6px;
		flex-shrink: 0;
	}

	.direction-section {
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--color-border);
	}

	.direction-section:last-child {
		border-bottom: none;
	}

	.direction-label {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-medium);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		margin-bottom: var(--space-2);
	}

	.direction-label.outgoing {
		color: var(--color-success);
	}

	.direction-label.incoming {
		color: var(--color-info);
	}

	.relationship-items {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		padding: 0;
		margin: 0;
	}

	.relationship-item {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		transition: background var(--transition-fast);
	}

	.relationship-item:hover {
		background: var(--color-bg-secondary);
	}

	.rel-target {
		flex: 1;
		font-size: var(--font-size-sm);
		font-family: var(--font-family-mono);
		color: var(--color-accent);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		text-decoration: none;
		min-width: 0;
	}

	.rel-target:hover {
		text-decoration: underline;
	}

	.rel-target:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
		border-radius: var(--radius-sm);
	}

	.rel-type-badge {
		font-size: var(--font-size-xs);
		padding: 1px 6px;
		background: var(--color-bg-tertiary);
		border-radius: var(--radius-sm);
		color: var(--color-text-muted);
		text-transform: uppercase;
		flex-shrink: 0;
	}
</style>
