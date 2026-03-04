import { test, expect } from '@playwright/test';
import { setupApiMocks, waitForHydration } from './fixtures/api-mocks';

test.describe('Entity Browser', () => {
	test.beforeEach(async ({ page }) => {
		await setupApiMocks(page);
	});

	test('entities page renders with search', async ({ page }) => {
		await page.goto('/entities');
		await waitForHydration(page);

		// Search input should exist
		const searchInput = page.getByPlaceholder(/search/i);
		await expect(searchInput).toBeVisible();
	});

	test('entity list calls GraphQL', async ({ page }) => {
		let graphqlCalled = false;
		await page.route('**/graphql/**', async (route) => {
			if (route.request().method() === 'POST') {
				graphqlCalled = true;
				const body = route.request().postDataJSON();
				const query = (body?.query as string) ?? '';
				if (query.includes('entityIdHierarchy')) {
					return route.fulfill({
						json: {
							data: {
								entityIdHierarchy: {
									children: [{ name: 'loop', count: 1 }],
									totalEntities: 1
								}
							}
						}
					});
				}
				return route.fulfill({
					json: {
						data: {
							entitiesByPrefix: [
								{
									id: 'test.entity.001',
									triples: [
										{ subject: 'test.entity.001', predicate: 'rdf.type', object: 'loop' },
										{ subject: 'test.entity.001', predicate: 'schema.name', object: 'Test Entity' }
									]
								}
							]
						}
					}
				});
			}
			return route.fallback();
		});

		await page.goto('/entities');
		await waitForHydration(page);
		await page.waitForTimeout(500);

		expect(graphqlCalled).toBe(true);
	});

	test('search input debounces API calls', async ({ page }) => {
		let requestCount = 0;
		await page.route('**/graphql/**', async (route) => {
			if (route.request().method() === 'POST') {
				requestCount++;
				return route.fulfill({
					json: {
						data: {
							entitiesByPrefix: [],
							entityIdHierarchy: { children: [], totalEntities: 0 }
						}
					}
				});
			}
			return route.fallback();
		});

		await page.goto('/entities');
		await waitForHydration(page);

		const initialCount = requestCount;
		const searchInput = page.getByPlaceholder(/search/i);
		await searchInput.fill('test query');

		// Wait for debounce (300ms) + buffer
		await page.waitForTimeout(500);

		expect(requestCount).toBeGreaterThan(initialCount);
	});

	test('shows empty state when no entities', async ({ page }) => {
		await page.route('**/graphql/**', async (route) => {
			if (route.request().method() === 'POST') {
				return route.fulfill({
					json: {
						data: {
							entitiesByPrefix: [],
							entityIdHierarchy: { children: [], totalEntities: 0 }
						}
					}
				});
			}
			return route.fallback();
		});

		await page.goto('/entities');
		await waitForHydration(page);

		await expect(page.getByText(/no entities/i)).toBeVisible();
	});

	test('entity detail page loads by ID', async ({ page }) => {
		await page.goto('/entities/test-entity-001');
		await waitForHydration(page);

		// Should display entity heading
		const heading = page.getByRole('heading').first();
		await expect(heading).toBeVisible();
	});

	test('entity detail has back link to /entities', async ({ page }) => {
		await page.goto('/entities/test-entity-001');
		await waitForHydration(page);

		// Use specific back button link text
		const back = page.getByRole('link', { name: 'Back to Entities' });
		await expect(back).toBeVisible();
	});

	test('entity detail handles errors gracefully', async ({ page }) => {
		await setupApiMocks(page, { errors: ['graphql'] });

		await page.goto('/entities/nonexistent');
		await waitForHydration(page);

		await expect(page.getByText(/error|failed/i)).toBeVisible();
	});
});
