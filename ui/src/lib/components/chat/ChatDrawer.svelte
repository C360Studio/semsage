<script lang="ts">
	import { fly } from 'svelte/transition';
	import { chatDrawerStore } from '$lib/stores/chatDrawer.svelte';
	import { settingsStore } from '$lib/stores/settings.svelte';
	import ChatPanel from './ChatPanel.svelte';
	import Icon from '$lib/components/shared/Icon.svelte';

	const FOCUSABLE_SELECTOR = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

	let drawerElement = $state<HTMLDivElement | null>(null);

	function handleKeydown(e: KeyboardEvent): void {
		if (!chatDrawerStore.isOpen) return;

		if (e.key === 'Escape') {
			chatDrawerStore.close();
		}

		// Focus trap: query live at keydown time so the list is never stale
		if (e.key === 'Tab' && drawerElement) {
			const elements = Array.from(
				drawerElement.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
			);
			if (elements.length === 0) return;
			const first = elements[0];
			const last = elements[elements.length - 1];
			if (e.shiftKey) {
				if (document.activeElement === first) {
					e.preventDefault();
					last.focus();
				}
			} else {
				if (document.activeElement === last) {
					e.preventDefault();
					first.focus();
				}
			}
		}
	}

	function handleBackdropClick(e: MouseEvent): void {
		if (e.target === e.currentTarget) {
			chatDrawerStore.close();
		}
	}

	$effect(() => {
		if (chatDrawerStore.isOpen && drawerElement) {
			requestAnimationFrame(() => {
				// Auto-focus the message textarea when drawer opens
				const firstInput = drawerElement?.querySelector<HTMLElement>('textarea, input');
				firstInput?.focus();
			});
		}
	});
</script>

<svelte:window onkeydown={handleKeydown} />

{#if chatDrawerStore.isOpen}
	<div
		class="chat-drawer-backdrop"
		onclick={handleBackdropClick}
		role="presentation"
		transition:fly={{
			x: 0,
			opacity: 0,
			duration: settingsStore.reducedMotion ? 0 : 200
		}}
	>
		<div
			bind:this={drawerElement}
			class="chat-drawer"
			role="dialog"
			aria-modal="true"
			aria-label={chatDrawerStore.contextTitle}
			transition:fly={{
				x: 400,
				duration: settingsStore.reducedMotion ? 0 : 200
			}}
		>
			<div class="drawer-header">
				<h2 class="drawer-title">{chatDrawerStore.contextTitle}</h2>
				<div class="header-actions">
					<kbd class="shortcut-hint" aria-label="Keyboard shortcut: Escape to close">Esc</kbd>
					<button
						class="close-button"
						onclick={() => chatDrawerStore.close()}
						aria-label="Close chat drawer"
					>
						<Icon name="x" size={20} />
					</button>
				</div>
			</div>

			<div class="drawer-content">
				<ChatPanel title={chatDrawerStore.contextTitle} />
			</div>
		</div>
	</div>
{/if}

<style>
	.chat-drawer-backdrop {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		backdrop-filter: blur(2px);
		z-index: 1000;
		display: flex;
		justify-content: flex-end;
	}

	.chat-drawer {
		width: var(--chat-drawer-width, 400px);
		height: 100%;
		background: var(--color-bg-primary);
		box-shadow: -4px 0 24px rgba(0, 0, 0, 0.3);
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.drawer-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: var(--space-4);
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
	}

	.drawer-title {
		font-size: var(--font-size-lg);
		font-weight: var(--font-weight-semibold);
		color: var(--color-text-primary);
		margin: 0;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.shortcut-hint {
		font-size: var(--font-size-xs);
		font-family: var(--font-family-mono);
		color: var(--color-text-muted);
		background: var(--color-bg-tertiary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		padding: 1px 6px;
	}

	.close-button {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		background: transparent;
		border: none;
		border-radius: var(--radius-md);
		color: var(--color-text-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.close-button:hover {
		background: var(--color-bg-tertiary);
		color: var(--color-text-primary);
	}

	.close-button:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: 2px;
	}

	/* Content area fills remaining height, padding handled inside ChatPanel */
	.drawer-content {
		flex: 1;
		overflow: hidden;
		padding: var(--space-4);
		padding-top: var(--space-3);
	}

	@media (max-width: 900px) {
		.chat-drawer {
			width: 100vw;
			height: 100vh;
		}
	}

	:global(.reduced-motion) .chat-drawer-backdrop,
	:global(.reduced-motion) .chat-drawer {
		transition: none !important;
	}
</style>
