import { test, expect } from '@playwright/test';
import { setupApiMocks, waitForHydration } from './fixtures/api-mocks';

test.describe('Settings Page', () => {
	test.beforeEach(async ({ page }) => {
		await setupApiMocks(page);
		// Clear localStorage to start fresh
		await page.goto('/settings');
		await page.evaluate(() => localStorage.clear());
		await page.reload();
		await waitForHydration(page);
	});

	test('renders settings page with all sections', async ({ page }) => {
		await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible();
		await expect(page.getByText('Appearance')).toBeVisible();
		await expect(page.getByText('Activity Feed')).toBeVisible();
	});

	test('theme select has dark/light/system options', async ({ page }) => {
		const select = page.locator('#theme-select');
		await expect(select).toBeVisible();

		const options = select.locator('option');
		await expect(options).toHaveCount(3);

		const values = await options.allTextContents();
		expect(values).toContain('Dark');
		expect(values).toContain('Light');
		expect(values).toContain('System');
	});

	test('changing theme applies to document', async ({ page }) => {
		const select = page.locator('#theme-select');

		// Switch to light theme
		await select.selectOption('light');

		// The theme attribute should be applied to document
		const theme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
		expect(theme).toBe('light');
	});

	test('reduced motion checkbox works', async ({ page }) => {
		const checkbox = page.locator('#reduced-motion-toggle');
		await expect(checkbox).toBeVisible();

		// Enable reduced motion
		await checkbox.check();
		await expect(checkbox).toBeChecked();

		// Should add class to document
		const hasClass = await page.evaluate(() =>
			document.documentElement.classList.contains('reduced-motion')
		);
		expect(hasClass).toBe(true);

		// Disable
		await checkbox.uncheck();
		await expect(checkbox).not.toBeChecked();
	});

	test('activity limit input accepts numeric values', async ({ page }) => {
		const input = page.locator('#activity-limit');
		await expect(input).toBeVisible();

		// Clear and type new value
		await input.fill('50');
		await input.dispatchEvent('change');

		// Value should be reflected
		await expect(input).toHaveValue('50');
	});

	test('reset to defaults button restores settings', async ({ page }) => {
		// Change a setting first
		const select = page.locator('#theme-select');
		await select.selectOption('light');

		// Click reset
		const resetBtn = page.getByRole('button', { name: /reset to defaults/i });
		await resetBtn.click();

		// Theme should be back to default (dark)
		const theme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
		expect(theme).toBe('dark');
	});

	test('settings persist across navigation', async ({ page }) => {
		// Change theme
		const select = page.locator('#theme-select');
		await select.selectOption('light');

		// Navigate away
		const nav = page.locator('nav[aria-label="Main navigation"]');
		await nav.getByText('Activity').click();
		await expect(page).toHaveURL(/\/activity/);

		// Navigate back
		await nav.getByText('Settings').click();
		await expect(page).toHaveURL(/\/settings/);

		// Theme should still be light
		const theme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
		expect(theme).toBe('light');
	});
});
