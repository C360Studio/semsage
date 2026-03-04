import { test, expect } from '@playwright/test';
import { setupApiMocks, waitForHydration, mockSignalResponse } from './fixtures/api-mocks';

test.describe('Loop Detail Page', () => {
	test.beforeEach(async ({ page }) => {
		await setupApiMocks(page);
	});

	test('renders loop metadata', async ({ page }) => {
		await page.goto('/loops/loop-001');
		await waitForHydration(page);

		// Should show loop ID somewhere on page
		await expect(page.getByText('loop-001')).toBeVisible();

		// Should show state badge
		await expect(page.getByText('executing')).toBeVisible();
	});

	test('shows pause button for executing loop', async ({ page }) => {
		await page.goto('/loops/loop-001');
		await waitForHydration(page);

		const pauseBtn = page.getByRole('button', { name: /pause/i });
		await expect(pauseBtn).toBeVisible();
	});

	test('pause button sends correct signal', async ({ page }) => {
		let signalReceived: { type?: string } = {};
		await page.route('**/api/loops/*/signal', async (route) => {
			if (route.request().method() === 'POST') {
				signalReceived = route.request().postDataJSON() as { type?: string };
				return route.fulfill({ json: mockSignalResponse });
			}
			return route.fallback();
		});

		await page.goto('/loops/loop-001');
		await waitForHydration(page);

		const pauseBtn = page.getByRole('button', { name: /pause/i });
		await pauseBtn.click();

		expect(signalReceived.type).toBe('pause');
	});

	test('cancel button visible for active loop', async ({ page }) => {
		await page.goto('/loops/loop-001');
		await waitForHydration(page);

		const cancelBtn = page.getByRole('button', { name: /cancel/i });
		await expect(cancelBtn).toBeVisible();
	});

	test('no control buttons for completed loop', async ({ page }) => {
		await page.goto('/loops/loop-002');
		await waitForHydration(page);

		// loop-002 is in 'success' state — no signal buttons
		await expect(page.getByRole('button', { name: /pause/i })).not.toBeVisible();
		await expect(page.getByRole('button', { name: /resume/i })).not.toBeVisible();
	});

	test('fetches children for loop', async ({ page }) => {
		let childrenFetched = false;
		await page.route('**/api/loops/*/children', async (route) => {
			childrenFetched = true;
			return route.fulfill({
				json: { loop_id: 'loop-001', children: ['loop-002'] }
			});
		});

		await page.goto('/loops/loop-001');
		await waitForHydration(page);
		await page.waitForTimeout(500);

		expect(childrenFetched).toBe(true);
	});

	test('fetches trajectory data', async ({ page }) => {
		let trajectoryFetched = false;
		await page.route('**/api/trajectory/loops/*', async (route) => {
			if (route.request().url().includes('/calls/')) return route.fallback();
			trajectoryFetched = true;
			return route.fulfill({
				json: {
					loop_id: 'loop-001',
					steps: 3,
					model_calls: 2,
					tool_calls: 1,
					tokens_in: 1500,
					tokens_out: 800,
					duration_ms: 12000,
					entries: []
				}
			});
		});

		await page.goto('/loops/loop-001');
		await waitForHydration(page);
		await page.waitForTimeout(500);

		expect(trajectoryFetched).toBe(true);
	});

	test('shows error state when loop not found', async ({ page }) => {
		await setupApiMocks(page, { errors: ['loop-detail'] });

		await page.goto('/loops/nonexistent');
		await waitForHydration(page);

		// Should show error message
		await expect(page.getByText(/error|not found|failed|couldn't load/i)).toBeVisible();
	});

	test('has back navigation link', async ({ page }) => {
		await page.goto('/loops/loop-001');
		await waitForHydration(page);

		// Should have a link to /activity
		const backLink = page.locator('a[href="/activity"]');
		await expect(backLink.first()).toBeVisible();
	});
});
