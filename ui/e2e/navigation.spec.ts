import { test, expect } from '@playwright/test';
import { setupApiMocks, waitForHydration } from './fixtures/api-mocks';

test.describe('Navigation', () => {
	test.beforeEach(async ({ page }) => {
		await setupApiMocks(page);
	});

	test('root path redirects to /activity', async ({ page }) => {
		await page.goto('/');
		await waitForHydration(page);
		// The redirect uses goto() in onMount — need to wait a tick
		await page.waitForURL(/\/activity/, { timeout: 10_000 });
	});

	test('sidebar shows navigation links', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		const nav = page.locator('nav[aria-label="Main navigation"]');
		await expect(nav).toBeVisible();
		await expect(nav.getByText('Activity')).toBeVisible();
		await expect(nav.getByText('Entities')).toBeVisible();
		await expect(nav.getByText('Settings')).toBeVisible();
	});

	test('clicking sidebar nav links navigates between pages', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		const nav = page.locator('nav[aria-label="Main navigation"]');

		// Navigate to Settings
		await nav.getByText('Settings').click();
		await expect(page).toHaveURL(/\/settings/);

		// Navigate to Entities
		await nav.getByText('Entities').click();
		await expect(page).toHaveURL(/\/entities/);

		// Navigate back to Activity
		await nav.getByText('Activity').click();
		await expect(page).toHaveURL(/\/activity/);
	});

	test('activity nav link shows active state', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		// The active link gets aria-current="page"
		const activityLink = page.locator('a[href="/activity"][aria-current="page"]');
		await expect(activityLink).toBeVisible();
	});

	test('Cmd+K toggles chat drawer', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		// Chat drawer should be closed initially
		await expect(page.getByRole('dialog')).not.toBeVisible();

		// Open with Ctrl+K
		await page.keyboard.press('Meta+k');
		await expect(page.getByRole('dialog')).toBeVisible();

		// Close with Escape
		await page.keyboard.press('Escape');
		await expect(page.getByRole('dialog')).not.toBeVisible();
	});

	test('sidebar shows system health status', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);

		await expect(page.getByText('System healthy')).toBeVisible();
	});

	test('page title includes route name', async ({ page }) => {
		await page.goto('/activity');
		await waitForHydration(page);
		await expect(page).toHaveTitle(/Activity.*Semsage/);
	});
});
