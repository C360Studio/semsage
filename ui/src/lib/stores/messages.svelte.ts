import { api } from '$lib/api/client';
import type { Message, ActivityEvent } from '$lib/types';

class MessagesStore {
	messages = $state<Message[]>([]);
	sending = $state(false);

	private pendingLoops = new Set<string>();

	handleActivityEvent(event: ActivityEvent): void {
		if (event.type !== 'loop_updated') return;

		const data = event.data as {
			loop_id?: string;
			task_id?: string;
			outcome?: string;
			result?: string;
		};

		if (!data?.result) return;
		if (data.outcome !== 'success') return;

		const matchedId = [data.loop_id, data.task_id].find((id) => id && this.pendingLoops.has(id));
		if (!matchedId) return;

		this.pendingLoops.delete(matchedId);

		const responseMessage: Message = {
			id: crypto.randomUUID(),
			type: 'assistant',
			content: data.result,
			timestamp: new Date().toISOString(),
			loopId: data.loop_id
		};

		this.messages = [...this.messages, responseMessage];
	}

	async send(content: string): Promise<void> {
		if (!content.trim() || this.sending) return;

		const userMessage: Message = {
			id: crypto.randomUUID(),
			type: 'user',
			content,
			timestamp: new Date().toISOString()
		};

		this.messages = [...this.messages, userMessage];
		this.sending = true;

		try {
			const response = await api.chat.send(content);

			if (response.error) {
				const errorMessage: Message = {
					id: response.response_id,
					type: 'error',
					content: response.error,
					timestamp: response.timestamp
				};
				this.messages = [...this.messages, errorMessage];
				return;
			}

			const statusMessage: Message = {
				id: response.response_id,
				type: 'status',
				content: response.content,
				timestamp: response.timestamp,
				loopId: response.in_reply_to
			};

			this.messages = [...this.messages, statusMessage];

			if (response.in_reply_to) {
				this.pendingLoops.add(response.in_reply_to);
			}
		} catch (err) {
			const errorMessage: Message = {
				id: crypto.randomUUID(),
				type: 'error',
				content: err instanceof Error ? err.message : 'Failed to send message',
				timestamp: new Date().toISOString()
			};
			this.messages = [...this.messages, errorMessage];
		} finally {
			this.sending = false;
		}
	}

	clear(): void {
		this.messages = [];
	}

	addStatus(content: string): void {
		const statusMessage: Message = {
			id: crypto.randomUUID(),
			type: 'status',
			content,
			timestamp: new Date().toISOString()
		};
		this.messages = [...this.messages, statusMessage];
	}
}

export const messagesStore = new MessagesStore();
