/**
 * Semsage UI type definitions.
 *
 * Defines types for loops, trajectories, entities, activity events,
 * agent tree structures, and dynamic tools.
 */

// ============================================================================
// Loop types
// ============================================================================

export type LoopState =
	| 'pending'
	| 'exploring'
	| 'executing'
	| 'paused'
	| 'complete'
	| 'success'
	| 'failed'
	| 'cancelled';

export interface Loop {
	loop_id: string;
	task_id: string;
	role: string;
	model: string;
	state: LoopState | string;
	iterations: number;
	max_iterations: number;
	depth: number;
	max_depth: number;
	parent_loop_id?: string;
	tokens_in?: number;
	tokens_out?: number;
	created_at?: string;
	completed_at?: string;
	result?: string;
}

export interface SignalRequest {
	type: 'pause' | 'resume' | 'cancel';
	reason?: string;
}

export interface SignalResponse {
	loop_id: string;
	signal: string;
	accepted: boolean;
}

// ============================================================================
// Activity event types
// ============================================================================

export interface ActivityEvent {
	/** Monotonic counter assigned by ActivityStore.addEvent() for stable {#each} keying. */
	id: number;
	type: string;
	timestamp: string;
	data: unknown;
}

// ============================================================================
// Trajectory types
// ============================================================================

export interface Trajectory {
	loop_id: string;
	trace_id?: string;
	steps: number;
	tool_calls: number;
	model_calls: number;
	tokens_in: number;
	tokens_out: number;
	duration_ms: number;
	status?: string;
	started_at?: string;
	ended_at?: string;
	entries?: TrajectoryEntry[];
}

export interface TrajectoryEntry {
	type: 'model_call' | 'tool_call';
	timestamp: string;
	duration_ms?: number;
	// model_call fields
	model?: string;
	provider?: string;
	capability?: string;
	tokens_in?: number;
	tokens_out?: number;
	finish_reason?: string;
	messages_count?: number;
	response_preview?: string;
	request_id?: string;
	// tool_call fields
	tool_name?: string;
	status?: string;
	result_preview?: string;
	// spawn_agent specific
	child_loop_id?: string;
	// shared
	error?: string;
	retries?: number;
}

export interface LLMCallRecord {
	request_id: string;
	loop_id: string;
	model: string;
	provider?: string;
	messages: unknown[];
	response: unknown;
	tokens_in: number;
	tokens_out: number;
	finish_reason: string;
	duration_ms: number;
	timestamp: string;
}

// ============================================================================
// Entity types
// ============================================================================

export type EntityType = 'code' | 'proposal' | 'spec' | 'task' | 'loop' | 'activity' | string;

export interface Entity {
	id: string;
	type: EntityType;
	name: string;
	predicates: Record<string, unknown>;
	createdAt?: string;
	updatedAt?: string;
}

export interface Relationship {
	predicate: string;
	predicateLabel: string;
	targetId: string;
	targetType: EntityType;
	targetName: string;
	direction: 'outgoing' | 'incoming';
}

export interface EntityWithRelationships extends Entity {
	relationships: Relationship[];
}

export interface EntityListParams extends Record<string, unknown> {
	type?: string;
	query?: string;
	limit?: number;
	offset?: number;
}

// ============================================================================
// System health types
// ============================================================================

export interface SystemHealth {
	healthy: boolean;
	components: ComponentHealth[];
}

export interface ComponentHealth {
	name: string;
	status: 'running' | 'stopped' | 'error';
	uptime: number;
}

// ============================================================================
// Agent tree types (semsage-specific)
// ============================================================================

export interface AgentTreeNode {
	loop: Loop;
	children: AgentTreeNode[];
	expanded: boolean;
}

// ============================================================================
// Dynamic tool types (semsage-specific)
// ============================================================================

export interface DynamicTool {
	name: string;
	description: string;
	root_loop_id: string;
	created_at?: string;
	processors?: string[];
}

// ============================================================================
// DAG types (semsage-specific — decompose_task)
// ============================================================================

export interface DagNode {
	id: string;
	prompt: string;
	role: string;
	depends_on: string[];
}

export interface Dag {
	nodes: DagNode[];
}

// ============================================================================
// Chat / message types
// ============================================================================

export interface Message {
	id: string;
	type: 'user' | 'assistant' | 'status' | 'error';
	content: string;
	timestamp: string;
	loopId?: string;
}

export interface MessageResponse {
	response_id: string;
	type: string;
	content: string;
	timestamp: string;
	in_reply_to?: string;
	error?: string;
}
