<script lang="ts">
	import { activityStore } from '$lib/stores/activity.svelte';
</script>

<header class="header">
	<div class="header-content">
		<div class="app-info">
			<span class="app-name">Semsage</span>
			<span class="app-tagline">Agent Hierarchy Monitor</span>
		</div>
		<div class="connection-status" class:connected={activityStore.connected}>
			<span class="status-dot"></span>
			<span class="status-text">{activityStore.connected ? 'Connected' : 'Disconnected'}</span>
		</div>
	</div>
</header>

<style>
	.header {
		height: var(--header-height);
		background: var(--color-bg-secondary);
		border-bottom: 1px solid var(--color-border);
		display: flex;
		align-items: center;
		padding: 0 var(--space-4);
	}

	.header-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
	}

	.app-info {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-width: 0;
		overflow: hidden;
	}

	.app-name {
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		white-space: nowrap;
	}

	.app-tagline {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	@media (max-width: 768px) {
		.app-tagline {
			display: none;
		}
	}

	.connection-status {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: var(--radius-full);
		background: var(--color-error);
	}

	.connection-status.connected .status-dot {
		background: var(--color-success);
	}

	.status-text {
		color: var(--color-text-muted);
	}
</style>
