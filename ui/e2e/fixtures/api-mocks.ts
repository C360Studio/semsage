import type { Page, Route } from '@playwright/test';

// --- Canned data ---

export const mockLoops = [
	{
		loop_id: 'loop-001',
		task_id: 'task-001',
		state: 'executing',
		role: 'orchestrator',
		model: 'claude-sonnet-4-20250514',
		iterations: 3,
		max_iterations: 20,
		depth: 0,
		max_depth: 5,
		created_at: '2026-03-04T10:00:00Z'
	},
	{
		loop_id: 'loop-002',
		task_id: 'task-001',
		state: 'success',
		role: 'researcher',
		model: 'claude-sonnet-4-20250514',
		iterations: 5,
		max_iterations: 10,
		depth: 1,
		max_depth: 5,
		parent_loop_id: 'loop-001',
		created_at: '2026-03-04T10:01:00Z',
		completed_at: '2026-03-04T10:02:00Z'
	}
];

export const mockHealth = {
	healthy: true,
	components: [
		{ name: 'agent_loops_kv', status: 'running', uptime: 3600 }
	]
};

export const mockChildren = {
	loop_id: 'loop-001',
	children: ['loop-002']
};

export const mockTree = {
	root_loop_id: 'loop-001',
	entity_ids: ['loop-001', 'loop-002']
};

export const mockTrajectory = {
	loop_id: 'loop-001',
	steps: 3,
	model_calls: 2,
	tool_calls: 1,
	tokens_in: 1500,
	tokens_out: 800,
	duration_ms: 12000,
	status: 'executing',
	started_at: '2026-03-04T10:00:00Z',
	entries: [
		{
			type: 'model_call',
			timestamp: '2026-03-04T10:00:01Z',
			model: 'claude-sonnet-4-20250514',
			tokens_in: 500,
			tokens_out: 300,
			finish_reason: 'tool_use',
			duration_ms: 2000
		},
		{
			type: 'tool_call',
			timestamp: '2026-03-04T10:00:03Z',
			tool_name: 'spawn_agent',
			status: 'success',
			child_loop_id: 'loop-002',
			duration_ms: 100
		}
	]
};

export const mockTools = [
	{
		name: 'search_code',
		description: 'Search the codebase',
		root_loop_id: 'loop-001'
	}
];

export const mockSignalResponse = {
	loop_id: 'loop-001',
	signal: 'pause',
	accepted: true,
	message: 'signal sent',
	timestamp: '2026-03-04T10:05:00Z'
};

export const mockChatResponse = {
	message_id: 'msg-001',
	content: 'hello',
	timestamp: '2026-03-04T10:05:00Z'
};

export const mockGraphQLEntities = {
	data: {
		entitiesByPrefix: [
			{
				id: 'semsage.default.agentic.orchestrator.loop.loop-001',
				triples: [
					{
						subject: 'semsage.default.agentic.orchestrator.loop.loop-001',
						predicate: 'rdf.type',
						object: 'loop'
					},
					{
						subject: 'semsage.default.agentic.orchestrator.loop.loop-001',
						predicate: 'schema.name',
						object: 'Orchestrator Loop'
					}
				]
			}
		]
	}
};

export const mockGraphQLEntityDetail = {
	data: {
		entity: {
			id: 'semsage.default.agentic.orchestrator.loop.loop-001',
			triples: [
				{
					subject: 'semsage.default.agentic.orchestrator.loop.loop-001',
					predicate: 'rdf.type',
					object: 'loop'
				},
				{
					subject: 'semsage.default.agentic.orchestrator.loop.loop-001',
					predicate: 'schema.name',
					object: 'Orchestrator Loop'
				}
			]
		},
		relationships: []
	}
};

export const mockGraphQLCounts = {
	data: {
		entityIdHierarchy: {
			children: [
				{ name: 'loop', count: 5 },
				{ name: 'task', count: 3 }
			],
			totalEntities: 8
		}
	}
};

// --- Route mock setup ---

export interface MockOptions {
	/** Override specific endpoint responses */
	overrides?: Partial<Record<string, unknown>>;
	/** Endpoints that should return errors */
	errors?: string[];
}

/**
 * Set up API route mocking for all semsage endpoints.
 * Call this in beforeEach to intercept all API calls.
 */
export async function setupApiMocks(page: Page, opts: MockOptions = {}): Promise<void> {
	const { overrides = {}, errors = [] } = opts;

	// Health
	await page.route('**/api/health', async (route: Route) => {
		if (errors.includes('health')) {
			return route.fulfill({ status: 500, json: { error: 'unavailable' } });
		}
		return route.fulfill({
			json: (overrides['health'] as object) ?? mockHealth
		});
	});

	// List loops
	await page.route('**/api/loops', async (route: Route) => {
		if (route.request().url().includes('/api/loops/')) return route.fallback();
		if (errors.includes('loops')) {
			return route.fulfill({ status: 500, json: { error: 'unavailable' } });
		}
		return route.fulfill({
			json: (overrides['loops'] as object) ?? mockLoops
		});
	});

	// Get single loop
	await page.route('**/api/loops/*', async (route: Route) => {
		const url = route.request().url();
		// Skip sub-routes (children, tree, signal)
		if (
			url.includes('/children') ||
			url.includes('/tree') ||
			url.includes('/signal')
		) {
			return route.fallback();
		}
		if (errors.includes('loop-detail')) {
			return route.fulfill({ status: 404, json: { error: 'not found' } });
		}
		// Return the matching loop or first mock
		const id = url.split('/api/loops/')[1]?.split('?')[0];
		const loop = mockLoops.find((l) => l.loop_id === id) ?? mockLoops[0];
		return route.fulfill({
			json: (overrides['loop-detail'] as object) ?? loop
		});
	});

	// Loop signal
	await page.route('**/api/loops/*/signal', async (route: Route) => {
		if (route.request().method() !== 'POST') return route.fallback();
		if (errors.includes('signal')) {
			return route.fulfill({ status: 500, json: { error: 'failed' } });
		}
		return route.fulfill({
			json: (overrides['signal'] as object) ?? mockSignalResponse
		});
	});

	// Loop children
	await page.route('**/api/loops/*/children', async (route: Route) => {
		if (errors.includes('children')) {
			return route.fulfill({ status: 500, json: { error: 'unavailable' } });
		}
		return route.fulfill({
			json: (overrides['children'] as object) ?? mockChildren
		});
	});

	// Loop tree
	await page.route('**/api/loops/*/tree', async (route: Route) => {
		return route.fulfill({
			json: (overrides['tree'] as object) ?? mockTree
		});
	});

	// Trajectory
	await page.route('**/api/trajectory/loops/*', async (route: Route) => {
		if (route.request().url().includes('/calls/')) return route.fallback();
		return route.fulfill({
			json: (overrides['trajectory'] as object) ?? mockTrajectory
		});
	});

	// Tools
	await page.route('**/api/tools*', async (route: Route) => {
		return route.fulfill({
			json: (overrides['tools'] as object) ?? mockTools
		});
	});

	// Chat
	await page.route('**/api/chat', async (route: Route) => {
		if (route.request().method() !== 'POST') return route.fallback();
		if (errors.includes('chat')) {
			return route.fulfill({ status: 500, json: { error: 'failed' } });
		}
		return route.fulfill({
			json: (overrides['chat'] as object) ?? mockChatResponse
		});
	});

	// GraphQL
	await page.route('**/graphql/**', async (route: Route) => {
		if (route.request().method() !== 'POST') return route.fallback();
		if (errors.includes('graphql')) {
			return route.fulfill({ status: 500, json: { errors: [{ message: 'unavailable' }] } });
		}

		const body = route.request().postDataJSON();
		const query = (body?.query as string) ?? '';

		// Route to appropriate mock based on query content
		if (query.includes('entityIdHierarchy')) {
			return route.fulfill({
				json: (overrides['graphql-counts'] as object) ?? mockGraphQLCounts
			});
		}
		if (query.includes('entity(id:') || query.includes('relationships(entityId:')) {
			return route.fulfill({
				json: (overrides['graphql-entity'] as object) ?? mockGraphQLEntityDetail
			});
		}
		// Default: entity list
		return route.fulfill({
			json: (overrides['graphql-entities'] as object) ?? mockGraphQLEntities
		});
	});

	// SSE activity stream — return empty stream that stays open briefly
	await page.route('**/api/activity*', async (route: Route) => {
		return route.fulfill({
			status: 200,
			headers: {
				'Content-Type': 'text/event-stream',
				'Cache-Control': 'no-cache',
				Connection: 'keep-alive'
			},
			body: 'event: connected\ndata: {"message":"connected"}\n\n'
		});
	});
}

/** Wait for SvelteKit hydration to complete. */
export async function waitForHydration(page: Page): Promise<void> {
	await page.waitForSelector('body.hydrated', { timeout: 10_000 });
}
