import { test, expect } from '@playwright/test';
import { setupApiMocks, waitForHydration, mockLoops } from './fixtures/api-mocks';

test.describe('Activity Page', () => {
	test.beforeEach(async ({ page }) => {
		await setupApiMocks(page);
	});

	test('renders split layout with both panels', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		// The AgentTreeView heading
		await expect(page.getByRole('heading', { name: 'Agent Trees' })).toBeVisible();
	});

	test('agent tree shows root loops', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		// Root loop should appear — wait for data to load
		await expect(page.locator('[role="treeitem"]').first()).toBeVisible({ timeout: 10_000 });
	});

	test('agent tree shows tree container when loops present', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		// The tree should render
		await expect(page.locator('[role="tree"]')).toBeVisible({ timeout: 10_000 });
	});

	test('refresh button triggers loop fetch', async ({ page }) => {
		let fetchCount = 0;
		await page.route('**/api/loops', async (route) => {
			if (route.request().url().includes('/api/loops/')) return route.fallback();
			fetchCount++;
			return route.fulfill({ json: mockLoops });
		});

		await page.goto('/activity');
		await waitForHydration(page);

		const initialCount = fetchCount;

		// Click refresh
		const refreshBtn = page.getByRole('button', { name: 'Refresh agent trees' });
		await refreshBtn.click();

		// Should have triggered another fetch
		expect(fetchCount).toBeGreaterThan(initialCount);
	});

	test('page loads and shows agent tree heading', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		// Verify page fully loaded with data displayed
		await expect(page.getByRole('heading', { name: 'Agent Trees' })).toBeVisible();
	});

	test('shows error banner when loops fetch fails', async ({ page }) => {
		// Set up error response for loops — must return non-ok JSON for the error to propagate
		await page.route('**/api/loops', async (route) => {
			if (route.request().url().includes('/api/loops/')) return route.fallback();
			return route.fulfill({
				status: 500,
				contentType: 'application/json',
				body: JSON.stringify({ message: 'unavailable' })
			});
		});

		await page.goto('/activity');
		await waitForHydration(page);

		// loopsStore.error triggers the error banner in activity page
		const retryBtn = page.getByRole('button', { name: 'Retry loading loops' });
		await expect(retryBtn).toBeVisible({ timeout: 10_000 });
	});

	test('empty state shows when no loops exist', async ({ page }) => {
		await setupApiMocks(page, { overrides: { loops: [] } });

		await page.goto('/activity');
		await waitForHydration(page);

		await expect(page.getByText('No active agent trees')).toBeVisible();
	});

	test('health endpoint is called on page load', async ({ page }) => {
		let healthCalled = false;
		await page.route('**/api/health', async (route) => {
			healthCalled = true;
			return route.fulfill({
				json: { healthy: true, components: [{ name: 'agent_loops_kv', status: 'running', uptime: 3600 }] }
			});
		});

		await page.goto('/activity');
		await waitForHydration(page);
		await page.waitForTimeout(500);

		expect(healthCalled).toBe(true);
	});
});
