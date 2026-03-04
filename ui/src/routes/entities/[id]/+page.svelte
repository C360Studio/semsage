<script lang="ts">
	import { page } from '$app/stores';
	import Icon from '$lib/components/shared/Icon.svelte';
	import RelationshipList from '$lib/components/entities/RelationshipList.svelte';
	import { api } from '$lib/api/client';
	import type { EntityWithRelationships } from '$lib/types';

	let entity = $state<EntityWithRelationships | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Map entity types to visual colors
	const typeColors: Record<string, string> = {
		code: 'var(--color-info)',
		proposal: 'var(--color-warning)',
		spec: 'var(--color-success)',
		task: 'var(--color-accent)',
		loop: 'var(--color-info)',
		activity: 'var(--color-text-muted)'
	};

	const typeBgs: Record<string, string> = {
		code: 'var(--color-info-muted)',
		proposal: 'var(--color-warning-muted)',
		spec: 'var(--color-success-muted)',
		task: 'var(--color-accent-muted)',
		loop: 'var(--color-info-muted)',
		activity: 'var(--color-bg-tertiary)'
	};

	function getColor(type: string): string {
		return typeColors[type] ?? 'var(--color-text-muted)';
	}

	function getBg(type: string): string {
		return typeBgs[type] ?? 'var(--color-bg-tertiary)';
	}

	async function loadEntity(id: string): Promise<void> {
		loading = true;
		error = null;
		try {
			entity = await api.entities.get(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load entity';
			entity = null;
		} finally {
			loading = false;
		}
	}

	// Watch for id changes — handles both initial load and navigation.
	// Cancellation token prevents a stale in-flight response from overwriting
	// state when the user navigates away before the fetch completes.
	$effect(() => {
		const rawId = $page.params.id;
		if (!rawId) return;
		const id: string = rawId;
		let cancelled = false;

		async function run() {
			loading = true;
			error = null;
			try {
				const result = await api.entities.get(decodeURIComponent(id));
				if (cancelled) return;
				entity = result;
			} catch (e) {
				if (cancelled) return;
				error = e instanceof Error ? e.message : 'Failed to load entity';
				entity = null;
			} finally {
				if (!cancelled) loading = false;
			}
		}

		run();

		return () => {
			cancelled = true;
		};
	});

	/** Format a predicate value for display in the table. */
	function formatValue(value: unknown): string {
		if (typeof value === 'string') return value;
		if (typeof value === 'number' || typeof value === 'boolean') return String(value);
		return JSON.stringify(value);
	}

	/** Extract a human-readable label from a dotted predicate key. */
	function predicateLabel(key: string): string {
		const parts = key.split('.');
		return parts[parts.length - 1] ?? key;
	}

	/** Check if a predicate value looks like a timestamp. */
	function isTimestamp(value: unknown): boolean {
		if (typeof value !== 'string') return false;
		const d = new Date(value);
		return !isNaN(d.getTime()) && value.includes('T');
	}

	function formatTimestamp(value: string): string {
		return new Date(value).toLocaleString();
	}
</script>

<svelte:head>
	<title>{entity?.name ?? 'Entity'} - semsage</title>
</svelte:head>

<div class="entity-detail-page">
	<a href="/entities" class="back-button">
		<Icon name="arrow-left" size={16} />
		<span>Back to Entities</span>
	</a>

	{#if loading}
		<div class="loading-state" role="status" aria-live="polite">
			<Icon name="loader" size={24} />
			<span>Loading entity...</span>
		</div>
	{:else if error}
		<div class="error-state" role="alert">
			<Icon name="alert-circle" size={24} />
			<span>{error}</span>
			<button
				class="retry-button"
				onclick={() => {
					const id = $page.params.id;
					if (id) loadEntity(decodeURIComponent(id));
				}}
			>
				Retry
			</button>
		</div>
	{:else if entity}
		<header class="entity-header">
			<div class="entity-title-row">
				<h1 class="entity-name">{entity.name}</h1>
				<span
					class="type-badge"
					style="color: {getColor(entity.type)}; background: {getBg(entity.type)}"
				>
					{entity.type}
				</span>
			</div>

			<div class="entity-id-row">
				<code class="entity-id" title={entity.id}>{entity.id}</code>
				<button
					class="copy-button"
					onclick={() => navigator.clipboard.writeText(entity!.id)}
					aria-label="Copy entity ID"
					title="Copy ID to clipboard"
				>
					<Icon name="copy" size={14} />
				</button>
			</div>

			{#if entity.createdAt || entity.updatedAt}
				<div class="entity-timestamps">
					{#if entity.createdAt}
						<span class="timestamp-item">
							<Icon name="clock" size={12} />
							Created {formatTimestamp(entity.createdAt)}
						</span>
					{/if}
					{#if entity.updatedAt}
						<span class="timestamp-item">
							<Icon name="edit-3" size={12} />
							Updated {formatTimestamp(entity.updatedAt)}
						</span>
					{/if}
				</div>
			{/if}
		</header>

		{#if Object.keys(entity.predicates).length > 0}
			<section class="predicates-section" aria-labelledby="predicates-heading">
				<h2 id="predicates-heading" class="section-heading">
					<Icon name="tag" size={16} />
					Predicates
					<span class="section-count">{Object.keys(entity.predicates).length}</span>
				</h2>

				<div class="predicates-table">
					<div class="predicates-header">
						<span>Predicate</span>
						<span>Value</span>
					</div>
					{#each Object.entries(entity.predicates) as [key, value] (key)}
						<div class="predicate-row">
							<div class="predicate-key-cell">
								<span class="predicate-label-text" title={key}>
									{predicateLabel(key)}
								</span>
								<span class="predicate-key-full" title={key}>{key}</span>
							</div>
							<div class="predicate-value-cell">
								{#if isTimestamp(value)}
									<span class="predicate-value timestamp-value">
										{formatTimestamp(String(value))}
									</span>
								{:else}
									<span class="predicate-value">
										{formatValue(value)}
									</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</section>
		{:else}
			<div class="no-predicates">
				<Icon name="tag" size={20} />
				<p>No predicates</p>
			</div>
		{/if}

		{#if entity.relationships && entity.relationships.length > 0}
			<section class="relationships-section" aria-labelledby="relationships-heading">
				<h2 id="relationships-heading" class="section-heading">
					<Icon name="git-branch" size={16} />
					Relationships
					<span class="section-count">{entity.relationships.length}</span>
				</h2>

				<RelationshipList relationships={entity.relationships} />
			</section>
		{:else}
			<div class="no-relationships">
				<Icon name="git-branch" size={20} />
				<p>No relationships</p>
			</div>
		{/if}
	{/if}
</div>

<style>
	.entity-detail-page {
		padding: var(--space-6);
		max-width: 900px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
		height: 100%;
		overflow-y: auto;
	}

	.back-button {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		background: transparent;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		color: var(--color-text-secondary);
		text-decoration: none;
		font-size: var(--font-size-sm);
		cursor: pointer;
		transition: all var(--transition-fast);
		align-self: flex-start;
	}

	.back-button:hover {
		background: var(--color-bg-secondary);
		color: var(--color-text-primary);
		border-color: var(--color-border-focus);
	}

	.back-button:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.loading-state,
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-3);
		padding: var(--space-12);
		text-align: center;
		color: var(--color-text-muted);
		flex: 1;
	}

	.error-state {
		color: var(--color-error);
	}

	.retry-button {
		margin-top: var(--space-2);
		padding: var(--space-2) var(--space-4);
		background: var(--color-error-muted);
		border: 1px solid var(--color-error);
		border-radius: var(--radius-md);
		color: var(--color-error);
		font-size: var(--font-size-sm);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.retry-button:hover {
		background: var(--color-error);
		color: white;
	}

	.entity-header {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.entity-title-row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
	}

	.entity-name {
		font-size: var(--font-size-2xl);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		margin: 0;
		word-break: break-word;
	}

	.type-badge {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius-full);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		flex-shrink: 0;
	}

	.entity-id-row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.entity-id {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		word-break: break-all;
		padding: var(--space-1) var(--space-2);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
	}

	.copy-button {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--space-1);
		background: transparent;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: var(--radius-sm);
		transition: color var(--transition-fast);
		flex-shrink: 0;
	}

	.copy-button:hover {
		color: var(--color-text-primary);
	}

	.copy-button:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.entity-timestamps {
		display: flex;
		gap: var(--space-4);
		flex-wrap: wrap;
	}

	.timestamp-item {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}

	.section-heading {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		font-size: var(--font-size-lg);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		margin: 0 0 var(--space-3);
	}

	.section-count {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-normal);
		color: var(--color-text-muted);
		background: var(--color-bg-tertiary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-full);
		padding: 0 8px;
	}

	.predicates-table {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		overflow: hidden;
	}

	.predicates-header {
		display: grid;
		grid-template-columns: 1fr 2fr;
		padding: var(--space-2) var(--space-4);
		background: var(--color-bg-tertiary);
		border-bottom: 1px solid var(--color-border);
		font-size: var(--font-size-xs);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.predicate-row {
		display: grid;
		grid-template-columns: 1fr 2fr;
		border-bottom: 1px solid var(--color-border);
		transition: background var(--transition-fast);
	}

	.predicate-row:last-child {
		border-bottom: none;
	}

	.predicate-row:hover {
		background: var(--color-bg-secondary);
	}

	.predicate-key-cell,
	.predicate-value-cell {
		padding: var(--space-2) var(--space-4);
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 2px;
		min-width: 0;
	}

	.predicate-key-cell {
		border-right: 1px solid var(--color-border);
	}

	.predicate-label-text {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		color: var(--color-accent);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.predicate-key-full {
		font-size: var(--font-size-xs);
		font-family: var(--font-family-mono);
		color: var(--color-text-muted);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.predicate-value {
		font-size: var(--font-size-sm);
		font-family: var(--font-family-mono);
		color: var(--color-text-secondary);
		word-break: break-word;
		white-space: pre-wrap;
	}

	.timestamp-value {
		font-family: inherit;
		color: var(--color-text-secondary);
	}

	.no-predicates,
	.no-relationships {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--color-text-muted);
		font-size: var(--font-size-sm);
		padding: var(--space-4);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
	}

	.no-predicates p,
	.no-relationships p {
		margin: 0;
	}

	.predicates-section,
	.relationships-section {
		display: flex;
		flex-direction: column;
	}

	@media (max-width: 600px) {
		.entity-detail-page {
			padding: var(--space-4);
		}

		.predicates-header,
		.predicate-row {
			grid-template-columns: 1fr;
		}

		.predicate-key-cell {
			border-right: none;
			border-bottom: 1px solid var(--color-border);
		}

		.predicates-header span:last-child {
			display: none;
		}
	}
</style>
