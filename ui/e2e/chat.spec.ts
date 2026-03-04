import { test, expect } from '@playwright/test';
import { setupApiMocks, waitForHydration } from './fixtures/api-mocks';

test.describe('Chat Drawer', () => {
	test.beforeEach(async ({ page }) => {
		await setupApiMocks(page);
		await page.goto('/activity');
		await waitForHydration(page);
	});

	test('chat drawer opens with Ctrl+K and closes with Escape', async ({ page }) => {
		// Open
		await page.keyboard.press('Meta+k');

		// Chat drawer uses role="dialog"
		const drawer = page.getByRole('dialog');
		await expect(drawer).toBeVisible();

		// Close
		await page.keyboard.press('Escape');
		await expect(drawer).not.toBeVisible();
	});

	test('chat drawer has a message input', async ({ page }) => {
		await page.keyboard.press('Meta+k');

		const input = page.getByLabel('Message input');
		await expect(input).toBeVisible();
	});

	test('sending a message calls POST /api/chat', async ({ page }) => {
		let chatBody: { content?: string } = {};
		await page.route('**/api/chat', async (route) => {
			if (route.request().method() === 'POST') {
				chatBody = route.request().postDataJSON() as { content?: string };
				return route.fulfill({
					json: {
						message_id: 'msg-001',
						content: 'hello test',
						timestamp: new Date().toISOString()
					}
				});
			}
			return route.fallback();
		});

		// Open chat
		await page.keyboard.press('Meta+k');

		// Type message
		const input = page.getByLabel('Message input');
		await input.fill('hello test');

		// Click send
		const sendBtn = page.getByLabel('Send message');
		await sendBtn.click();

		// Verify API was called
		await page.waitForTimeout(500);
		expect(chatBody.content).toBe('hello test');
	});

	test('sent message appears in the message list', async ({ page }) => {
		await page.route('**/api/chat', async (route) => {
			if (route.request().method() === 'POST') {
				return route.fulfill({
					json: {
						message_id: 'msg-001',
						content: 'test message',
						timestamp: new Date().toISOString()
					}
				});
			}
			return route.fallback();
		});

		// Open chat
		await page.keyboard.press('Meta+k');

		const input = page.getByLabel('Message input');
		await input.fill('test message');

		const sendBtn = page.getByLabel('Send message');
		await sendBtn.click();

		// User message should appear in the dialog
		await expect(page.getByRole('dialog').getByText('test message').first()).toBeVisible();
	});

	test('empty message cannot be sent', async ({ page }) => {
		// Open chat
		await page.keyboard.press('Meta+k');

		// Send button should be disabled when input is empty
		const sendBtn = page.getByLabel('Send message');
		await expect(sendBtn).toBeDisabled();
	});
});
