<script lang="ts">
	import Icon from '$lib/components/shared/Icon.svelte';
	import EntityCard from '$lib/components/entities/EntityCard.svelte';
	import { api } from '$lib/api/client';
	import type { Entity, EntityType } from '$lib/types';

	let entities = $state<Entity[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let searchQuery = $state('');
	let selectedType = $state<EntityType | ''>('');
	let entityCounts = $state<Record<string, number>>({});

	const entityTypes: { value: EntityType | ''; label: string }[] = [
		{ value: '', label: 'All Types' },
		{ value: 'loop', label: 'Loops' },
		{ value: 'task', label: 'Tasks' },
		{ value: 'code', label: 'Code' },
		{ value: 'spec', label: 'Specs' },
		{ value: 'proposal', label: 'Proposals' },
		{ value: 'activity', label: 'Activities' }
	];

	async function loadEntities(): Promise<void> {
		loading = true;
		error = null;
		try {
			const params: Record<string, unknown> = {};
			if (selectedType) params.type = selectedType;
			if (searchQuery) params.query = searchQuery;

			entities = await api.entities.list(params);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load entities';
			entities = [];
		} finally {
			loading = false;
		}
	}

	async function loadCounts(): Promise<void> {
		try {
			const result = await api.entities.count();
			entityCounts = result.byType;
		} catch {
			// Silently fail — counts are decorative
		}
	}

	// Debounced search — re-run whenever searchQuery changes
	let searchTimeout: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		const _q = searchQuery;
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(loadEntities, 300);

		return () => clearTimeout(searchTimeout);
	});

	// Immediate reload on type filter change
	$effect(() => {
		const _t = selectedType;
		loadEntities();
	});

	// Load counts once on mount
	$effect(() => {
		loadCounts();
	});
</script>

<svelte:head>
	<title>Entities - semsage</title>
</svelte:head>

<div class="entities-page">
	<header class="page-header">
		<div class="page-title-row">
			<h1 class="page-title">Entity Browser</h1>
			{#if !loading && entities.length > 0}
				<span class="entity-count">{entities.length} entities</span>
			{/if}
		</div>
		<p class="page-description">Browse and search the knowledge graph</p>
	</header>

	<div class="filters">
		<div class="search-box" class:focused={false}>
			<Icon name="search" size={16} />
			<input
				type="search"
				placeholder="Search entities..."
				bind:value={searchQuery}
				aria-label="Search entities"
				autocomplete="off"
				spellcheck={false}
			/>
			{#if searchQuery}
				<button
					class="clear-search"
					onclick={() => (searchQuery = '')}
					aria-label="Clear search"
				>
					<Icon name="x" size={14} />
				</button>
			{/if}
		</div>

		<div class="type-filters" role="group" aria-label="Filter by entity type">
			{#each entityTypes as type}
				<button
					class="type-chip"
					class:active={selectedType === type.value}
					onclick={() => (selectedType = type.value)}
					aria-pressed={selectedType === type.value}
				>
					{type.label}
					{#if type.value && entityCounts[type.value]}
						<span class="chip-count">{entityCounts[type.value]}</span>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	{#if loading}
		<div class="loading-state" role="status" aria-live="polite">
			<Icon name="loader" size={24} />
			<span>Loading entities...</span>
		</div>
	{:else if error}
		<div class="error-state" role="alert">
			<Icon name="alert-circle" size={24} />
			<span>{error}</span>
			<button class="retry-button" onclick={loadEntities}>Retry</button>
		</div>
	{:else if entities.length === 0}
		<div class="empty-state">
			<Icon name="database" size={48} />
			<h2>No entities found</h2>
			<p>
				{#if searchQuery || selectedType}
					Try adjusting your search or filters.
				{:else}
					The knowledge graph is empty. Start an agent loop to populate it.
				{/if}
			</p>
			{#if searchQuery || selectedType}
				<button
					class="retry-button"
					onclick={() => {
						searchQuery = '';
						selectedType = '';
					}}
				>
					Clear filters
				</button>
			{/if}
		</div>
	{:else}
		<div class="entity-grid" aria-label="Entity list">
			{#each entities as entity (entity.id)}
				<EntityCard {entity} />
			{/each}
		</div>
	{/if}
</div>

<style>
	.entities-page {
		padding: var(--space-6);
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: var(--space-5);
		height: 100%;
		overflow-y: auto;
	}

	.page-header {
		flex-shrink: 0;
	}

	.page-title-row {
		display: flex;
		align-items: baseline;
		gap: var(--space-3);
	}

	.page-title {
		font-size: var(--font-size-2xl);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		margin: 0 0 var(--space-1);
	}

	.entity-count {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.page-description {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin: 0;
	}

	.filters {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		flex-shrink: 0;
	}

	.search-box {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		max-width: 480px;
		transition: border-color var(--transition-fast);
	}

	.search-box:focus-within {
		border-color: var(--color-accent);
	}

	.search-box input {
		flex: 1;
		border: none;
		background: transparent;
		color: var(--color-text-primary);
		font-size: var(--font-size-sm);
		outline: none;
	}

	.search-box input::placeholder {
		color: var(--color-text-muted);
	}

	.clear-search {
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		padding: 2px;
		border-radius: var(--radius-sm);
		transition: color var(--transition-fast);
	}

	.clear-search:hover {
		color: var(--color-text-primary);
	}

	.type-filters {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.type-chip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		padding: var(--space-1) var(--space-3);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-full);
		font-size: var(--font-size-sm);
		color: var(--color-text-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
		white-space: nowrap;
	}

	.type-chip:hover {
		background: var(--color-bg-tertiary);
		color: var(--color-text-primary);
		border-color: var(--color-border-focus);
	}

	.type-chip.active {
		background: var(--color-accent-muted);
		border-color: var(--color-accent);
		color: var(--color-accent);
		font-weight: var(--font-weight-medium);
	}

	.type-chip:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.chip-count {
		font-size: var(--font-size-xs);
		background: var(--color-bg-tertiary);
		border-radius: var(--radius-full);
		padding: 0 5px;
		color: var(--color-text-muted);
	}

	.type-chip.active .chip-count {
		background: var(--color-accent-muted);
		color: var(--color-accent);
		border: 1px solid var(--color-accent);
	}

	.loading-state,
	.error-state,
	.empty-state {
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

	.empty-state h2 {
		margin: 0;
		font-size: var(--font-size-lg);
		color: var(--color-text-secondary);
	}

	.empty-state p {
		margin: 0;
		max-width: 400px;
		font-size: var(--font-size-sm);
	}

	.retry-button {
		margin-top: var(--space-2);
		padding: var(--space-2) var(--space-4);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		color: var(--color-text-primary);
		font-size: var(--font-size-sm);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.retry-button:hover {
		background: var(--color-bg-tertiary);
		border-color: var(--color-accent);
	}

	.error-state .retry-button {
		background: var(--color-error-muted);
		border-color: var(--color-error);
		color: var(--color-error);
	}

	/* Responsive grid: 1 col on mobile, 2 on tablet, 3 on desktop */
	.entity-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: var(--space-3);
	}

	@media (max-width: 600px) {
		.entities-page {
			padding: var(--space-4);
		}

		.entity-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
