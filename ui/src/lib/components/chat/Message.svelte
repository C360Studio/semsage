<script lang="ts">
	import Icon from '$lib/components/shared/Icon.svelte';
	import type { Message } from '$lib/types';

	interface Props {
		message: Message;
	}

	let { message }: Props = $props();

	const typeConfig: Record<string, { icon: string; label: string }> = {
		user: { icon: 'user', label: 'You' },
		assistant: { icon: 'bot', label: 'semsage' },
		status: { icon: 'activity', label: 'Status' },
		error: { icon: 'alert-circle', label: 'Error' }
	};

	const config = $derived(typeConfig[message.type] ?? typeConfig.assistant);

	function formatTime(timestamp: string): string {
		const date = new Date(timestamp);
		return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
	}

	/**
	 * Render basic markdown: bold, italic, inline code, and fenced code blocks.
	 * Returns HTML-escaped string with markdown converted to HTML spans.
	 *
	 * SECURITY: HTML escaping MUST run before any regex substitutions to prevent XSS.
	 * Do not reorder.
	 */
	function renderMarkdown(text: string): string {
		// Escape HTML entities first to prevent XSS
		const escaped = text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;');

		// Fenced code blocks (```lang\n...\n```)
		const withCodeBlocks = escaped.replace(
			/```(?:\w+)?\n([\s\S]*?)```/g,
			(_, code) => `<pre class="code-block"><code>${code.trimEnd()}</code></pre>`
		);

		// Inline code (`code`)
		const withInlineCode = withCodeBlocks.replace(
			/`([^`]+)`/g,
			(_, code) => `<code class="inline-code">${code}</code>`
		);

		// Bold (**text** or __text__)
		const withBold = withInlineCode.replace(
			/\*\*(.+?)\*\*|__(.+?)__/g,
			(_, a, b) => `<strong>${a ?? b}</strong>`
		);

		// Italic (*text* or _text_) — avoid matching inside words
		const withItalic = withBold.replace(
			/(?<!\w)\*(.+?)\*(?!\w)|(?<!\w)_(.+?)_(?!\w)/g,
			(_, a, b) => `<em>${a ?? b}</em>`
		);

		return withItalic;
	}

	const isPending = $derived(message.type === 'assistant' && message.content === '');
	const renderedContent = $derived(
		message.type === 'user' || message.type === 'status' || message.type === 'error'
			? null
			: renderMarkdown(message.content)
	);
</script>

<div
	class="message"
	class:user={message.type === 'user'}
	class:assistant={message.type === 'assistant'}
	class:error={message.type === 'error'}
	class:status={message.type === 'status'}
>
	<div class="message-avatar" aria-hidden="true">
		<Icon name={config.icon} size={18} />
	</div>

	<div class="message-content">
		<div class="message-header">
			<span class="message-author">{config.label}</span>
			<span class="message-time">{formatTime(message.timestamp)}</span>
			{#if message.loopId}
				<span class="loop-ref" title={message.loopId}>
					loop:{message.loopId.slice(0, 8)}
				</span>
			{/if}
		</div>

		<div class="message-body">
			{#if isPending}
				<span class="loading-dots" aria-label="Thinking">
					<span></span>
					<span></span>
					<span></span>
				</span>
			{:else if renderedContent !== null}
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html renderedContent}
			{:else}
				{message.content}
			{/if}
		</div>
	</div>
</div>

<style>
	.message {
		display: flex;
		gap: var(--space-3);
		padding: var(--space-3);
		border-radius: var(--radius-lg);
		transition: background var(--transition-fast);
	}

	.message:hover {
		background: var(--color-bg-tertiary);
	}

	/* User messages: right-aligned with accent background */
	.message.user {
		flex-direction: row-reverse;
		background: var(--color-accent-muted);
	}

	.message.user:hover {
		background: var(--color-accent-muted);
		filter: brightness(1.1);
	}

	/* Assistant messages: left-aligned, slightly elevated */
	.message.assistant {
		background: var(--color-bg-secondary);
	}

	.message.error {
		background: var(--color-error-muted);
	}

	.message.status {
		opacity: 0.8;
	}

	.message-avatar {
		width: 32px;
		height: 32px;
		border-radius: var(--radius-full);
		background: var(--color-bg-tertiary);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		color: var(--color-text-secondary);
	}

	.message.user .message-avatar {
		background: var(--color-accent);
		color: white;
	}

	.message.assistant .message-avatar {
		background: var(--color-bg-elevated);
		color: var(--color-accent);
	}

	.message.error .message-avatar {
		background: var(--color-error-muted);
		color: var(--color-error);
	}

	.message-content {
		flex: 1;
		min-width: 0;
	}

	.message.user .message-content {
		align-items: flex-end;
		display: flex;
		flex-direction: column;
	}

	.message-header {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-1);
	}

	.message.user .message-header {
		flex-direction: row-reverse;
	}

	.message-author {
		font-size: var(--font-size-sm);
		font-weight: var(--font-weight-medium);
		color: var(--color-text-primary);
	}

	.message.user .message-author {
		color: var(--color-accent);
	}

	.message-time {
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
	}

	.loop-ref {
		font-size: var(--font-size-xs);
		font-family: var(--font-family-mono);
		color: var(--color-info);
		background: var(--color-info-muted);
		padding: 2px var(--space-2);
		border-radius: var(--radius-sm);
	}

	.message-body {
		font-size: var(--font-size-base);
		line-height: var(--line-height-relaxed);
		color: var(--color-text-primary);
		white-space: pre-wrap;
		word-break: break-word;
		max-width: 100%;
	}

	.message.error .message-body {
		color: var(--color-error);
	}

	.message.status .message-body {
		color: var(--color-text-secondary);
		font-size: var(--font-size-sm);
	}

	/* Markdown rendered elements */
	.message-body :global(strong) {
		font-weight: var(--font-weight-semibold);
	}

	.message-body :global(em) {
		font-style: italic;
	}

	.message-body :global(.inline-code) {
		font-family: var(--font-family-mono);
		font-size: 0.9em;
		background: var(--color-bg-tertiary);
		padding: 1px 4px;
		border-radius: var(--radius-sm);
		color: var(--color-accent);
	}

	.message-body :global(.code-block) {
		background: var(--color-bg-primary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-3) var(--space-4);
		overflow-x: auto;
		margin: var(--space-2) 0;
		white-space: pre;
	}

	.message-body :global(.code-block code) {
		font-family: var(--font-family-mono);
		font-size: var(--font-size-sm);
		color: var(--color-text-primary);
		background: none;
		padding: 0;
	}

	/* Animated loading dots for pending assistant messages */
	.loading-dots {
		display: inline-flex;
		gap: 4px;
		align-items: center;
		padding: 2px 0;
	}

	.loading-dots span {
		width: 6px;
		height: 6px;
		border-radius: var(--radius-full);
		background: var(--color-text-muted);
		animation: dot-pulse 1.4s ease-in-out infinite;
	}

	.loading-dots span:nth-child(2) {
		animation-delay: 0.2s;
	}

	.loading-dots span:nth-child(3) {
		animation-delay: 0.4s;
	}

	@keyframes dot-pulse {
		0%, 80%, 100% {
			opacity: 0.3;
			transform: scale(0.8);
		}
		40% {
			opacity: 1;
			transform: scale(1);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.loading-dots span {
			animation: none;
			opacity: 0.6;
		}
	}
</style>
