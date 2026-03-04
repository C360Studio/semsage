<script lang="ts">
	import { settingsStore, type Theme } from '$lib/stores/settings.svelte';

	const themes: { value: Theme; label: string }[] = [
		{ value: 'dark', label: 'Dark' },
		{ value: 'light', label: 'Light' },
		{ value: 'system', label: 'System' }
	];
</script>

<div class="page">
	<div class="page-header">
		<h1 class="page-title">Settings</h1>
		<p class="page-description">Application preferences</p>
	</div>

	<div class="settings-sections">
		<section class="settings-section">
			<h2 class="section-title">Appearance</h2>

			<div class="setting-row">
				<div class="setting-info">
					<label class="setting-label" for="theme-select">Theme</label>
					<p class="setting-hint">Choose your preferred color scheme</p>
				</div>
				<select
					id="theme-select"
					class="input setting-control"
					value={settingsStore.theme}
					onchange={(e) => settingsStore.setTheme((e.currentTarget as HTMLSelectElement).value as Theme)}
				>
					{#each themes as theme}
						<option value={theme.value}>{theme.label}</option>
					{/each}
				</select>
			</div>

			<div class="setting-row">
				<div class="setting-info">
					<label class="setting-label" for="reduced-motion-toggle">Reduced motion</label>
					<p class="setting-hint">Minimize animations and transitions</p>
				</div>
				<input
					id="reduced-motion-toggle"
					type="checkbox"
					checked={settingsStore.reducedMotion}
					onchange={(e) => settingsStore.setReducedMotion((e.currentTarget as HTMLInputElement).checked)}
				/>
			</div>
		</section>

		<section class="settings-section">
			<h2 class="section-title">Activity Feed</h2>

			<div class="setting-row">
				<div class="setting-info">
					<label class="setting-label" for="activity-limit">Event limit</label>
					<p class="setting-hint">Maximum number of activity events to keep in memory</p>
				</div>
				<input
					id="activity-limit"
					type="number"
					class="input setting-control"
					value={settingsStore.activityLimit}
					min="10"
					max="1000"
					onchange={(e) => settingsStore.setActivityLimit(parseInt((e.currentTarget as HTMLInputElement).value, 10))}
				/>
			</div>
		</section>

		<section class="settings-section">
			<div class="setting-row">
				<div class="setting-info">
					<p class="setting-label">Reset settings</p>
					<p class="setting-hint">Restore all settings to their defaults</p>
				</div>
				<button class="btn btn-secondary" onclick={() => settingsStore.resetToDefaults()}>
					Reset to defaults
				</button>
			</div>
		</section>
	</div>
</div>

<style>
	.page {
		padding: var(--space-6);
		max-width: 640px;
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
	}

	.page-header {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.page-title {
		font-size: var(--font-size-2xl);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
	}

	.page-description {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.settings-sections {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
	}

	.settings-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}

	.section-title {
		font-size: var(--font-size-base);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		padding-bottom: var(--space-3);
		border-bottom: 1px solid var(--color-border);
	}

	.setting-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
	}

	.setting-info {
		flex: 1;
		min-width: 0;
	}

	.setting-label {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-primary);
		display: block;
		margin-bottom: var(--space-1);
	}

	.setting-hint {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}

	.setting-control {
		width: 140px;
		flex-shrink: 0;
	}
</style>
