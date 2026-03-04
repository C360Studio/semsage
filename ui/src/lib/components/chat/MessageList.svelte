<script lang="ts">
	import Message from './Message.svelte';
	import type { Message as MessageType } from '$lib/types';

	interface Props {
		messages: MessageType[];
	}

	let { messages }: Props = $props();
	let container = $state<HTMLDivElement | undefined>(undefined);

	$effect(() => {
		// Track messages.length so the effect re-runs on every new message
		const _ = messages.length;
		if (container) {
			requestAnimationFrame(() => {
				if (container) {
					container.scrollTop = container.scrollHeight;
				}
			});
		}
	});
</script>

<div
	class="message-list"
	bind:this={container}
	role="log"
	aria-live="polite"
	aria-label="Chat messages"
>
	{#if messages.length === 0}
		<div class="empty-state">
			<p class="empty-title">Start a conversation with semsage</p>
			<p class="empty-hint">Ask about agents, loops, or the knowledge graph</p>
		</div>
	{:else}
		{#each messages as message (message.id)}
			<Message {message} />
		{/each}
	{/if}
</div>

<style>
	.message-list {
		flex: 1;
		overflow-y: auto;
		padding: var(--space-4) 0;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		scroll-behavior: smooth;
	}

	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
		color: var(--color-text-muted);
		padding: var(--space-8);
	}

	.empty-title {
		font-size: var(--font-size-lg);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-secondary);
		margin-bottom: var(--space-2);
	}

	.empty-hint {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	@media (prefers-reduced-motion: reduce) {
		.message-list {
			scroll-behavior: auto;
		}
	}
</style>
