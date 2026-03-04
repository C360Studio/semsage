import type { Entity, EntityType, Relationship } from '$lib/types';

/**
 * Raw entity format from graph-query GraphQL responses.
 */
export interface RawTriple {
	subject: string;
	predicate: string;
	object: unknown;
}

export interface RawEntity {
	id: string;
	triples: RawTriple[];
}

export interface RawRelationship {
	from: string;
	to: string;
	predicate: string;
	direction: 'outgoing' | 'incoming';
}

export interface HierarchyChild {
	name: string;
	count: number;
}

export interface EntityIdHierarchy {
	children: HierarchyChild[];
	totalEntities: number;
}

/**
 * Extract entity type from ID prefix.
 * Semsage agent entities use the "semsage" prefix.
 * Example: "semsage.default.agentic.orchestrator.loop.abc" → "loop"
 */
function extractTypeFromId(id: string): EntityType {
	// Semsage agent loop entities
	if (id.startsWith('semsage.') && id.includes('.loop.')) return 'loop';
	if (id.startsWith('semsage.') && id.includes('.task.')) return 'task';

	const firstDot = id.indexOf('.');
	if (firstDot === -1) return 'code';

	const prefix = id.substring(0, firstDot);
	const validTypes = ['code', 'proposal', 'spec', 'task', 'loop', 'activity'];
	return validTypes.includes(prefix) ? (prefix as EntityType) : 'code';
}

/**
 * Transform a raw entity from graph-query format to UI Entity format.
 */
export function transformEntity(raw: RawEntity): Entity {
	const predicates: Record<string, unknown> = {};
	let name = raw.id;
	let createdAt: string | undefined;
	let updatedAt: string | undefined;

	for (const triple of raw.triples) {
		predicates[triple.predicate] = triple.object;

		if (triple.predicate === 'dc.terms.title') {
			name = triple.object as string;
		} else if (triple.predicate === 'code.artifact.path') {
			const path = triple.object as string;
			const filename = path.split('/').pop();
			if (filename && name === raw.id) {
				name = filename;
			}
		} else if (triple.predicate === 'prov.generatedAtTime') {
			createdAt = triple.object as string;
		} else if (triple.predicate === 'prov.invalidatedAtTime') {
			updatedAt = triple.object as string;
		}
	}

	const type = extractTypeFromId(raw.id);

	return {
		id: raw.id,
		type,
		name,
		predicates,
		...(createdAt && { createdAt }),
		...(updatedAt && { updatedAt })
	};
}

/**
 * Transform raw relationships from graph-query to UI Relationship format.
 */
export function transformRelationships(raw: RawRelationship[]): Relationship[] {
	return raw.map((r) => {
		const targetId = r.direction === 'outgoing' ? r.to : r.from;

		const predicateParts = r.predicate.split('.');
		const predicateLabel = predicateParts[predicateParts.length - 1] || r.predicate;

		return {
			predicate: r.predicate,
			predicateLabel,
			targetId,
			targetType: extractTypeFromId(targetId),
			targetName: targetId,
			direction: r.direction
		};
	});
}

/**
 * Transform entity hierarchy counts to the format expected by the UI.
 */
export function transformEntityCounts(hierarchy: EntityIdHierarchy): {
	total: number;
	byType: Record<string, number>;
} {
	const byType: Record<string, number> = {};
	for (const child of hierarchy.children) {
		byType[child.name] = child.count;
	}
	return {
		total: hierarchy.totalEntities,
		byType
	};
}
