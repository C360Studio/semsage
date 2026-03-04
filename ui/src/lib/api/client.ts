import { graphqlRequest } from './graphql';
import {
	transformEntity,
	transformRelationships,
	transformEntityCounts,
	type RawEntity,
	type RawRelationship,
	type EntityIdHierarchy
} from './transforms';
import type {
	Loop,
	SignalResponse,
	Entity,
	EntityWithRelationships,
	EntityListParams,
	Trajectory,
	LLMCallRecord,
	DynamicTool,
	MessageResponse,
	SystemHealth
} from '$lib/types';

const BASE_URL = import.meta.env.VITE_API_URL || '';

interface RequestOptions {
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	body?: unknown;
	headers?: Record<string, string>;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const { method = 'GET', body, headers = {} } = options;

	const response = await fetch(`${BASE_URL}${path}`, {
		method,
		headers: {
			'Content-Type': 'application/json',
			...headers
		},
		body: body ? JSON.stringify(body) : undefined
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ message: response.statusText }));
		throw new Error((error as { message?: string }).message || `Request failed: ${response.status}`);
	}

	return response.json();
}

function toQueryString(params?: Record<string, unknown>): string {
	if (!params) return '';
	const entries = Object.entries(params).filter(([, v]) => v !== undefined);
	if (entries.length === 0) return '';
	return '?' + new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString();
}

export const api = {
	system: {
		getHealth: () => request<SystemHealth>('/api/health')
	},

	loops: {
		list: (params?: { state?: string }) =>
			request<Loop[]>(`/api/loops${toQueryString(params)}`),

		get: (id: string) => request<Loop>(`/api/loops/${id}`),

		sendSignal: (loopId: string, type: 'pause' | 'resume' | 'cancel', reason?: string) =>
			request<SignalResponse>(`/api/loops/${loopId}/signal`, {
				method: 'POST',
				body: { type, reason }
			}),

		getChildren: (loopId: string) =>
			request<Loop[]>(`/api/loops/${loopId}/children`),

		getTree: (loopId: string) =>
			request<Loop[]>(`/api/loops/${loopId}/tree`)
	},

	trajectory: {
		getByLoop: (loopId: string, format?: 'summary' | 'json') =>
			request<Trajectory>(`/api/trajectory/loops/${loopId}?format=${format ?? 'json'}`),

		getCall: (loopId: string, requestId: string) =>
			request<LLMCallRecord>(`/api/trajectory/loops/${loopId}/calls/${requestId}`)
	},

	tools: {
		listByTree: (rootLoopId: string) =>
			request<DynamicTool[]>(`/api/tools?root_loop_id=${encodeURIComponent(rootLoopId)}`)
	},

	chat: {
		send: (content: string) =>
			request<MessageResponse>('/api/chat', {
				method: 'POST',
				body: { content }
			})
	},

	entities: {
		list: async (params?: EntityListParams): Promise<Entity[]> => {
			const prefix = params?.type ? `${params.type}.` : '';
			const limit = params?.limit || 100;

			const result = await graphqlRequest<{ entitiesByPrefix: RawEntity[] }>(
				`
				query($prefix: String!, $limit: Int) {
					entitiesByPrefix(prefix: $prefix, limit: $limit) {
						id
						triples { subject predicate object }
					}
				}
			`,
				{ prefix, limit }
			);

			let entities = result.entitiesByPrefix.map(transformEntity);

			if (params?.query) {
				const q = params.query.toLowerCase();
				entities = entities.filter(
					(e) =>
						e.name.toLowerCase().includes(q) ||
						e.id.toLowerCase().includes(q) ||
						JSON.stringify(e.predicates).toLowerCase().includes(q)
				);
			}

			return entities;
		},

		get: async (id: string): Promise<EntityWithRelationships> => {
			const result = await graphqlRequest<{
				entity: RawEntity;
				relationships: RawRelationship[];
			}>(
				`
				query($id: String!) {
					entity(id: $id) {
						id
						triples { subject predicate object }
					}
					relationships(entityId: $id) {
						from
						to
						predicate
						direction
					}
				}
			`,
				{ id }
			);

			if (!result.entity) {
				throw new Error('Entity not found');
			}

			return {
				...transformEntity(result.entity),
				relationships: transformRelationships(result.relationships || [])
			};
		},

		relationships: async (id: string) => {
			const result = await graphqlRequest<{ relationships: RawRelationship[] }>(
				`
				query($id: String!) {
					relationships(entityId: $id) {
						from
						to
						predicate
						direction
					}
				}
			`,
				{ id }
			);

			return transformRelationships(result.relationships || []);
		},

		count: async (): Promise<{ total: number; byType: Record<string, number> }> => {
			const result = await graphqlRequest<{ entityIdHierarchy: EntityIdHierarchy }>(
				`
				query {
					entityIdHierarchy(prefix: "") {
						children { name count }
						totalEntities
					}
				}
			`
			);

			return transformEntityCounts(result.entityIdHierarchy);
		}
	}
};
