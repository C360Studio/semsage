<script lang="ts">
	import MessageList from './MessageList.svelte';
	import MessageInput from './MessageInput.svelte';
	import { messagesStore } from '$lib/stores/messages.svelte';

	interface Props {
		title?: string;
	}

	let { title = 'Chat' }: Props = $props();

	async function handleSend(content: string): Promise<void> {
		await messagesStore.send(content);
	}
</script>

<div class="chat-panel">
	<div class="chat-messages">
		<MessageList messages={messagesStore.messages} />
	</div>

	{#if messagesStore.sending}
		<div class="loading-bar" aria-hidden="true"></div>
	{/if}

	<div class="chat-input">
		<MessageInput
			onSend={handleSend}
			disabled={messagesStore.sending}
			placeholder="Ask semsage anything..."
		/>
	</div>
</div>

<style>
	.chat-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
		position: relative;
	}

	.chat-messages {
		flex: 1;
		overflow-y: auto;
		min-height: 0;
	}

	.loading-bar {
		height: 2px;
		background: linear-gradient(
			90deg,
			transparent 0%,
			var(--color-accent) 40%,
			var(--color-accent) 60%,
			transparent 100%
		);
		background-size: 200% 100%;
		animation: loading-slide 1.5s ease-in-out infinite;
		flex-shrink: 0;
	}

	@keyframes loading-slide {
		0% { background-position: 200% 0; }
		100% { background-position: -200% 0; }
	}

	.chat-input {
		flex-shrink: 0;
		padding-top: var(--space-2);
		border-top: 1px solid var(--color-border);
	}

	@media (prefers-reduced-motion: reduce) {
		.loading-bar {
			animation: none;
			background: var(--color-accent);
			opacity: 0.5;
		}
	}
</style>
