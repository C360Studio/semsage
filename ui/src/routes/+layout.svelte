<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import Sidebar from '$lib/components/shared/Sidebar.svelte';
	import Header from '$lib/components/shared/Header.svelte';
	import ChatDrawer from '$lib/components/chat/ChatDrawer.svelte';
	import Icon from '$lib/components/shared/Icon.svelte';
	import { activityStore } from '$lib/stores/activity.svelte';
	import { loopsStore } from '$lib/stores/loops.svelte';
	import { systemStore } from '$lib/stores/system.svelte';
	import { messagesStore } from '$lib/stores/messages.svelte';
	import { settingsStore } from '$lib/stores/settings.svelte';
	import { chatDrawerStore } from '$lib/stores/chatDrawer.svelte';
	import { sidebarStore } from '$lib/stores/sidebar.svelte';
	import '../app.css';

	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	function handleKeydown(e: KeyboardEvent): void {
		// Cmd+K (Mac) or Ctrl+K (Windows/Linux) - Toggle chat drawer
		if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
			e.preventDefault();
			chatDrawerStore.toggle();
		}
	}

	// Mark hydration complete for e2e tests
	onMount(() => {
		document.body.classList.add('hydrated');
	});

	// Apply reduced motion setting
	$effect(() => {
		if (settingsStore.reducedMotion) {
			document.documentElement.classList.add('reduced-motion');
		} else {
			document.documentElement.classList.remove('reduced-motion');
		}
	});

	// Initialize connections on mount
	$effect(() => {
		activityStore.connect();
		loopsStore.fetch();
		systemStore.fetch();

		// Subscribe to activity events for chat responses
		const unsubscribe = activityStore.onEvent((event) => {
			messagesStore.handleActivityEvent(event);
		});

		// Periodic refresh for non-SSE data
		const interval = setInterval(() => {
			loopsStore.fetch();
			systemStore.fetch();
		}, 30000);

		return () => {
			activityStore.disconnect();
			unsubscribe();
			clearInterval(interval);
		};
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="app-layout">
	<Sidebar currentPath={$page.url.pathname} />

	<!-- Mobile sidebar backdrop -->
	{#if sidebarStore.isOpen}
		<button
			class="sidebar-backdrop"
			onclick={() => sidebarStore.close()}
			aria-label="Close navigation"
		></button>
	{/if}

	<div class="main-area">
		<!-- Mobile hamburger button -->
		<button
			class="hamburger-btn"
			onclick={() => sidebarStore.open()}
			aria-label="Open navigation"
			aria-expanded={sidebarStore.isOpen}
		>
			<Icon name="menu" size={24} />
		</button>

		<Header />

		<main class="content">
			{@render children()}
		</main>
	</div>
</div>

<!-- Global ChatDrawer (Cmd+K) -->
<ChatDrawer />

<style>
	.app-layout {
		display: flex;
		height: 100vh;
		overflow: hidden;
	}

	.main-area {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.content {
		flex: 1;
		overflow: auto;
	}

	/* Mobile hamburger button - hidden on desktop */
	.hamburger-btn {
		display: none;
		position: fixed;
		top: var(--space-3);
		left: var(--space-3);
		z-index: 50;
		width: 40px;
		height: 40px;
		padding: 0;
		border: none;
		background: var(--color-bg-secondary);
		color: var(--color-text-primary);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
		cursor: pointer;
		align-items: center;
		justify-content: center;
	}

	.hamburger-btn:hover {
		background: var(--color-bg-tertiary);
	}

	/* Mobile sidebar backdrop */
	.sidebar-backdrop {
		display: none;
		position: fixed;
		inset: 0;
		z-index: 99;
		background: rgba(0, 0, 0, 0.5);
		border: none;
		cursor: pointer;
	}

	@media (max-width: 768px) {
		.hamburger-btn {
			display: flex;
		}

		.sidebar-backdrop {
			display: block;
		}

		.main-area {
			padding-top: calc(40px + var(--space-3) * 2);
		}
	}
</style>
