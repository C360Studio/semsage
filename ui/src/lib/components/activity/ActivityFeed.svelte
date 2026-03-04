<script lang="ts">
	/**
	 * ActivityFeed - Displays the SSE activity event stream in reverse
	 * chronological order with color-coding by event type and auto-scroll.
	 *
	 * Connects to activityStore for live events. Supports a "Clear" button
	 * and shows an empty state when there are no events.
	 */

	import Icon from '$lib/components/shared/Icon.svelte';
	import { activityStore } from '$lib/stores/activity.svelte';
	import type { ActivityEvent } from '$lib/types';

	// No props — reads directly from activityStore

	let scrollEl = $state<HTMLElement | null>(null);
	// Track whether the user has scrolled up (to suppress auto-scroll)
	let userScrolled = $state(false);

	// Events in reverse chronological order (newest first)
	const events = $derived([...activityStore.recent].reverse());

	// Auto-scroll to top (newest) when new events arrive, unless user scrolled
	$effect(() => {
		// Touch events to track dependency
		const _count = activityStore.recent.length;
		if (!scrollEl || userScrolled) return;
		// Scroll to top since list is reversed (newest at top)
		scrollEl.scrollTop = 0;
	});

	function handleScroll() {
		if (!scrollEl) return;
		// If user scrolled away from top, mark as user-scrolled
		userScrolled = scrollEl.scrollTop > 40;
	}

	function handleClear() {
		activityStore.clear();
		userScrolled = false;
	}

	function getEventColor(type: string): string {
		switch (type) {
			case 'loop_created': return 'var(--color-success)';
			case 'loop_updated': return 'var(--color-accent)';
			case 'loop_deleted': return 'var(--color-error)';
			case 'tool_call': return 'var(--color-info)';
			case 'model_call': return 'var(--color-info)';
			default: return 'var(--color-text-muted)';
		}
	}

	function getEventIcon(type: string): string {
		switch (type) {
			case 'loop_created': return 'play';
			case 'loop_updated': return 'activity';
			case 'loop_deleted': return 'x';
			case 'tool_call': return 'wrench';
			case 'model_call': return 'brain';
			default: return 'circle';
		}
	}

	function formatEventType(type: string): string {
		return type.replace(/_/g, ' ');
	}

	function formatTime(ts: string): string {
		return new Date(ts).toLocaleTimeString(undefined, {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			hour12: false
		});
	}

	function getLoopId(event: ActivityEvent): string | null {
		const data = event.data as Record<string, unknown> | null;
		if (data && typeof data['loop_id'] === 'string') return data['loop_id'];
		return null;
	}
</script>

<div class="activity-feed">
	<!-- Feed header: connection status + clear button -->
	<div class="feed-header">
		<div class="connection-status" class:connected={activityStore.connected}>
			<span class="status-dot" aria-hidden="true"></span>
			<span class="status-label">{activityStore.connected ? 'Live' : 'Disconnected'}</span>
		</div>

		<span class="event-count" aria-live="polite" aria-atomic="true">
			{events.length} event{events.length === 1 ? '' : 's'}
		</span>

		{#if events.length > 0}
			<button
				type="button"
				class="clear-btn"
				onclick={handleClear}
				title="Clear all events"
				aria-label="Clear activity feed"
			>
				<Icon name="x" size={12} />
				Clear
			</button>
		{/if}
	</div>

	<!-- Event list -->
	{#if events.length === 0}
		<div class="empty-feed" aria-live="polite">
			<Icon name="activity" size={32} />
			<p>No activity yet</p>
			<p class="hint">Events will appear here as agents work</p>
		</div>
	{:else}
		<div
			class="events-list"
			role="log"
			aria-live="polite"
			aria-label="Activity events"
			bind:this={scrollEl}
			onscroll={handleScroll}
		>
			{#each events as event (event.id)}
				{@const loopId = getLoopId(event)}
				{@const color = getEventColor(event.type)}
				<div class="event-item">
					<!-- Color-coded icon -->
					<div class="event-icon" style:color={color} aria-hidden="true">
						<Icon name={getEventIcon(event.type)} size={13} />
					</div>

					<div class="event-body">
						<!-- Timestamp -->
						<span class="event-time">{formatTime(event.timestamp)}</span>

						<!-- Event type (color-coded label) -->
						<span
							class="event-type"
							class:created={event.type === 'loop_created'}
							class:updated={event.type === 'loop_updated'}
							class:deleted={event.type === 'loop_deleted'}
						>
							{formatEventType(event.type)}
						</span>

						<!-- Loop ID (if present) -->
						{#if loopId}
							<a
								href="/loops/{loopId}"
								class="loop-link"
								title="View loop {loopId}"
							>
								<Icon name="git-branch" size={10} />
								{loopId.slice(-8)}
							</a>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.activity-feed {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.feed-header {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
		min-height: 40px;
	}

	.connection-status {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: var(--radius-full);
		background: var(--color-error);
		flex-shrink: 0;
	}

	.connection-status.connected .status-dot {
		background: var(--color-success);
	}

	.status-label {
		font-size: var(--font-size-xs);
	}

	.event-count {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		margin-left: auto;
	}

	.clear-btn {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		padding: 2px var(--space-2);
		background: transparent;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		cursor: pointer;
		transition: all var(--transition-fast);
		white-space: nowrap;
	}

	.clear-btn:hover {
		background: var(--color-error-muted);
		border-color: var(--color-error);
		color: var(--color-error);
	}

	.clear-btn:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	.empty-feed {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		color: var(--color-text-muted);
		text-align: center;
		padding: var(--space-6);
	}

	.empty-feed p {
		margin: 0;
		font-size: var(--font-size-sm);
	}

	.hint {
		font-size: var(--font-size-xs) !important;
	}

	.events-list {
		flex: 1;
		overflow-y: auto;
		padding: var(--space-2);
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.event-item {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-2);
		background: var(--color-bg-secondary);
		border-radius: var(--radius-md);
		transition: background var(--transition-fast);
	}

	.event-item:hover {
		background: var(--color-bg-tertiary);
	}

	.event-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		background: var(--color-bg-tertiary);
		border-radius: var(--radius-full);
		flex-shrink: 0;
		margin-top: 1px;
	}

	.event-body {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
		flex: 1;
		min-width: 0;
	}

	.event-time {
		font-size: var(--font-size-xs);
		font-family: var(--font-family-mono);
		color: var(--color-text-muted);
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
	}

	.event-type {
		font-size: var(--font-size-xs);
		color: var(--color-text-primary);
		text-transform: capitalize;
		font-weight: var(--font-weight-medium);
	}

	.event-type.created {
		color: var(--color-success);
	}

	.event-type.updated {
		color: var(--color-accent);
	}

	.event-type.deleted {
		color: var(--color-error);
	}

	.loop-link {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		font-family: var(--font-family-mono);
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		text-decoration: none;
		padding: 1px 4px;
		background: var(--color-bg-elevated);
		border-radius: var(--radius-sm);
		transition: all var(--transition-fast);
	}

	.loop-link:hover {
		background: var(--color-accent-muted);
		color: var(--color-accent);
	}
</style>
