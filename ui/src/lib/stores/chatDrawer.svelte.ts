/**
 * ChatDrawer Store - Manages global chat drawer state.
 */

export interface ChatDrawerContext {
	type: 'global';
}

class ChatDrawerStore {
	isOpen = $state(false);
	context = $state<ChatDrawerContext>({ type: 'global' });

	open(context?: ChatDrawerContext): void {
		if (context) this.context = context;
		this.isOpen = true;
	}

	close(): void {
		this.isOpen = false;
	}

	toggle(context?: ChatDrawerContext): void {
		if (this.isOpen) {
			this.close();
		} else {
			this.open(context);
		}
	}

	get contextTitle(): string {
		return 'Chat';
	}
}

export const chatDrawerStore = new ChatDrawerStore();
