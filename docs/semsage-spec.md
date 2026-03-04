# Semsage Architectural Specification

Semsage exposes SemStreams' existing agentic capabilities through a clean tool interface. It is a separate Go module (`github.com/c360studio/semsage`) that imports SemStreams as a dependency. SemStreams never imports Semsage.

## Origin

Semsage is inspired by the [OpenSage whitepaper](https://arxiv.org/abs/2602.16891) ("Self-Programming Agent Generation Engine" by Li et al.) and [Ian Blenke's SageAgent](https://github.com/ianblenke/sageagent) — an open-source Python implementation of those concepts. SageAgent implements self-generated topology (LLM-driven DAG decomposition), dynamic tool creation, hierarchical graph-based memory, and agent coordination via a pub/sub message bus.

The realization: SemStreams already provides all of these capabilities natively — with better persistence (NATS JetStream), better type safety (Go + typed payloads), and a governance model (filter chain at processor boundary). SageAgent built custom memory graph, message bus, tool registry, and topology manager in Python to approximate what SemStreams already has.

## Core Principle

**Agents are just another consumer of SemStreams flows.** No new framework, no DSL.

- Tools are flows — processors wired as reactive definitions
- `spawn_agent` composes flows by instantiating child loops
- `decompose_task` produces a DAG of flows to instantiate
- The knowledge graph is shared state everything reads/writes
- Reactive definitions ARE the execution model
- The governance filter chain sits at the processor boundary, covering everything automatically

Every Semsage capability is a `ToolExecutor` or a `reactive.Definition`. No new extension mechanisms — it plugs into existing SemStreams contracts.

## SemStreams Contracts

### ToolExecutor Interface

```go
// semstreams/processor/agentic-tools/executor.go
type ToolExecutor interface {
    Execute(ctx context.Context, call agentic.ToolCall) (agentic.ToolResult, error)
    ListTools() []agentic.ToolDefinition
}
```

Registration via `ExecutorRegistry`:

```go
// semstreams/processor/agentic-tools/global.go
func RegisterTool(name string, executor ToolExecutor) error
func GetGlobalRegistry() *ExecutorRegistry
```

### Tool Types

```go
// semstreams/agentic/tools.go
type ToolDefinition struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Parameters  map[string]any `json:"parameters"`
}

type ToolCall struct {
    ID        string         `json:"id"`
    Name      string         `json:"name"`
    Arguments map[string]any `json:"arguments,omitempty"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    LoopID    string         `json:"loop_id,omitempty"`
    TraceID   string         `json:"trace_id,omitempty"`
}

type ToolResult struct {
    CallID   string         `json:"call_id"`
    Content  string         `json:"content,omitempty"`
    Error    string         `json:"error,omitempty"`
    Metadata map[string]any `json:"metadata,omitempty"`
    LoopID   string         `json:"loop_id,omitempty"`
    TraceID  string         `json:"trace_id,omitempty"`
}
```

### TaskMessage

```go
// semstreams/agentic/user_types.go (key fields)
type TaskMessage struct {
    LoopID       string                   `json:"loop_id,omitempty"`
    TaskID       string                   `json:"task_id"`       // required
    Role         string                   `json:"role"`          // required
    Model        string                   `json:"model"`         // required
    Prompt       string                   `json:"prompt"`        // required
    ParentLoopID string                   `json:"parent_loop_id,omitempty"`
    Depth        int                      `json:"depth,omitempty"`
    MaxDepth     int                      `json:"max_depth,omitempty"`
    Context      *types.ConstructedContext `json:"context,omitempty"`
    Tools        []ToolDefinition         `json:"tools,omitempty"`
    Metadata     map[string]any           `json:"metadata,omitempty"`
}
```

### Loop Events

```go
// semstreams/agentic/events.go
type LoopCompletedEvent struct {
    LoopID       string    `json:"loop_id"`
    TaskID       string    `json:"task_id"`
    Outcome      string    `json:"outcome"` // "success"
    Result       string    `json:"result"`
    ParentLoopID string    `json:"parent_loop,omitempty"`
    CompletedAt  time.Time `json:"completed_at"`
    // ... plus Role, Model, Iterations, token counts, routing fields
}

type LoopFailedEvent struct {
    LoopID   string    `json:"loop_id"`
    TaskID   string    `json:"task_id"`
    Outcome  string    `json:"outcome"` // "failed"
    Reason   string    `json:"reason"`
    Error    string    `json:"error"`
    FailedAt time.Time `json:"failed_at"`
    // ... plus Role, Model, Iterations, token counts, routing fields
}
```

### reactive.Definition

```go
// semstreams/processor/reactive/types.go
type Definition struct {
    ID            string
    Description   string
    StateBucket   string
    StateFactory  func() any
    MaxIterations int
    Timeout       time.Duration
    Rules         []RuleDef
    Events        EventConfig
}
```

Used for Phase 2 DAG execution workflow.

## Tool Specifications

### spawn_agent

Publishes a `TaskMessage` to create a child agentic-loop. Blocks until the child completes, times out, or fails. Returns the child's result as a normal `ToolResult`.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `prompt` | string | yes | Task prompt for the child agent |
| `role` | string | yes | System role for the child agent |
| `model` | string | no | LLM model (defaults to parent's model) |
| `tools` | array | no | Tool subset for the child (defaults to parent's tools) |
| `timeout` | string | no | Duration string (defaults to "5m") |
| `metadata` | object | no | Additional context passed to child |

**Implementation contract:**

1. Subscribe to `agent.complete.{childLoopID}` BEFORE publishing (no race)
2. Build `TaskMessage` with `ParentLoopID` from `call.LoopID`, enforce `Depth < MaxDepth`
3. Create graph entity for child loop + `agentic.loop.spawned` relationship triple
4. Publish to `agent.task.{childLoopID}`
5. Block on: completion channel, `ctx.Done()`, timeout
6. Return child's `Result` as `ToolResult.Content`, or error `ToolResult` on failure
7. Unsubscribe on all exit paths (defer)

**Concurrency:** The existing agentic-loop already handles parallel tool calls. If the LLM emits three `spawn_agent` calls in one response, three children run concurrently for free.

### create_tool

Lets the LLM compose existing processors into named reactive definitions at runtime. Instead of a code sandbox, `create_tool` wires existing pieces — not writing new code.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Tool name (must be unique within agent tree) |
| `description` | string | yes | Human-readable description |
| `processors` | array | yes | List of processor references to compose |
| `wiring` | object | yes | Input/output mappings between processors |
| `parameters` | object | no | Tool parameter schema |

**Implementation contract:**

1. Validate all referenced processors exist in the SemStreams component registry
2. Build a `reactive.Definition` from the spec
3. Register the definition with the workflow engine
4. Register the new flow as a callable tool via `agentictools.RegisterTool()`
5. Tools are scoped to the agent tree that created them (keyed by root loop ID)
6. Governance filter chain covers automatically since it sits at the processor boundary

### decompose_task

Returns a DAG of subtasks as structured JSON. The parent agent decides whether to spawn nodes individually or delegate to the DAG execution workflow.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `goal` | string | yes | High-level goal to decompose |
| `context` | string | no | Additional context for decomposition |
| `max_depth` | int | no | Maximum decomposition depth |

**Returns:**

```json
{
  "dag": {
    "nodes": [
      {
        "id": "node-1",
        "prompt": "Research current market data",
        "role": "researcher",
        "depends_on": []
      },
      {
        "id": "node-2",
        "prompt": "Analyze findings from research",
        "role": "analyst",
        "depends_on": ["node-1"]
      }
    ]
  }
}
```

### query_agent_tree

Queries the agent hierarchy via SemStreams' graph query infrastructure. No separate KV bucket — relationships are stored as graph triples alongside loop entities.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `operation` | string | yes | One of: `get_tree`, `get_children`, `get_status` |
| `loop_id` | string | no | Target loop ID (required for `get_children`, `get_status`) |

**Implementation:** Thin wrapper around graph query client:
- `get_children` → `GetOutgoingRelationships(ctx, loopEntityID, "agentic.loop.spawned")`
- `get_tree` → `ExecutePathQuery` with `PredicateFilter: ["agentic.loop.spawned"]`
- `get_status` → `GetEntity(ctx, loopEntityID)` + read from `AGENT_LOOPS` KV

## Agent Hierarchy via Graph Layer (Option C — Hybrid)

Agent hierarchy uses a hybrid storage model:

- **`AGENT_LOOPS` KV bucket** (existing) — mutable loop state machine, fast KV Watch for SSE delivery of real-time status updates
- **Graph entity references** — lightweight entities in `ENTITY_STATES` holding relationship triples for parent-child edges, DAG dependencies, and tree structure

This separation respects the KV-or-stream heuristic: `AGENT_LOOPS` needs real-time watch semantics for the hot mutable state path, while the graph layer provides relationship queries that would otherwise require a separate `AGENT_TREE` bucket with manual children-list maintenance.

### Entity ID Mapping

Agent loops map to the 6-part entity ID format:

```
semsage.default.agentic.orchestrator.loop.<loop-id>
semsage.default.agentic.orchestrator.task.<task-id>
```

The `Type` field (`loop`, `task`) enables type-based queries. SemStreams' hierarchy inference auto-creates container entities for the shared `semsage.default.agentic.orchestrator.loop` prefix, enabling group queries.

### Relationship Predicates

| Predicate | From | To | Meaning |
|-----------|------|----|---------|
| `agentic.loop.spawned` | parent loop | child loop | Parent spawned child |
| `agentic.loop.task` | loop | task | Loop executes task |
| `agentic.task.depends_on` | task | task | DAG dependency edge |

### What This Gives Us for Free

| Need | Graph capability |
|------|-----------------|
| "Get all children of loop X" | `GetOutgoingRelationships(ctx, loopID, "agentic.loop.spawned")` |
| "Get full agent tree" | `ExecutePathQuery` with `MaxDepth` and predicate filter |
| "Get all active loops" | `ListWithPrefix(ctx, "semsage.default.agentic.orchestrator.loop")` |
| DAG dependency traversal | `GetOutgoingRelationships(ctx, taskID, "agentic.task.depends_on")` |
| Sibling discovery | Hierarchy inference auto-creates `.group` container entities |
| Cross-domain queries | Agent entities queryable alongside domain entities via GraphQL/MCP |

### Graph Operations in spawn_agent

When `spawn_agent` creates a child loop, it also:

1. Creates a lightweight graph entity for the child: `semsage.default.agentic.orchestrator.loop.<childID>`
2. Creates a relationship triple: `{parent_entity} --agentic.loop.spawned--> {child_entity}`

The existing `AGENT_LOOPS` KV handles the mutable state (iterations, signals, completion). The graph entity holds only identity and relationship triples.

## NATS Resources

No new KV buckets. No new streams. Uses existing infrastructure:

| Resource | Owner | Purpose |
|----------|-------|---------|
| `AGENT_LOOPS` | SemStreams | Mutable loop state (existing) |
| `ENTITY_STATES` | SemStreams | Graph entities including agent hierarchy references (existing) |
| `AGENT` stream | SemStreams | JetStream subjects `agent.task.*`, `agent.complete.*` (existing) |

## Failure and Cancellation

- **Child failure** — error `ToolResult` returned to parent LLM, which decides: retry, skip, or abort
- **Timeout** — cascades via Go context propagation
- **Cancellation** — propagates down the tree via `UserSignal`
- **Cleanup** — `create_tool` artifacts scoped to agent tree; removed when root loop terminates

## Design Decisions

### Flow Composition Over Code Sandbox

The old spec's `create_tool` with Starlark/WASM is replaced. Instead of a code sandbox, `create_tool` lets the LLM compose existing processors into named flows (reactive definitions) at runtime — wiring existing pieces, not writing new code. This is safer (no arbitrary code execution), leverages the existing governance filter chain, and stays within SemStreams' extension model.

### Subscribe Before Publish

`spawn_agent` subscribes to the completion subject before publishing the task message. This eliminates the race condition where a fast child could complete before the parent starts listening.

### Tool Scoping

Dynamically created tools are scoped to the agent tree that created them, keyed by root loop ID. When the root loop terminates, its tools are cleaned up. This prevents tool namespace pollution across independent agent hierarchies.

## Phasing

### Phase 1 (MVP)

- Graph entity helpers — entity ID mapping, relationship predicates, graph operations
- `spawn_agent` executor — child agent orchestration + graph relationship creation
- `create_tool` executor — flow composition
- `query_agent_tree` executor — hierarchy inspection via graph queries
- Service composition (`cmd/semsage/main.go`)

### Phase 2

- `decompose_task` executor — DAG generation
- DAG execution reactive definition — automated DAG execution with dependency ordering

## Project Structure

```
semsage/
  go.mod
  cmd/semsage/main.go
  agentgraph/                              # Graph entity helpers for agent hierarchy
    entities.go, entities_test.go          # Entity ID mapping, relationship predicates
  tools/
    register.go                            # Tool registration helper
    spawn/executor.go, executor_test.go
    create/executor.go, executor_test.go, types.go
    decompose/executor.go, executor_test.go, types.go
    tree/executor.go, executor_test.go     # query_agent_tree (wraps graph queries)
  workflow/dag/                            # Phase 2
    definition.go, definition_test.go, state.go
  configs/semsage.yaml
```
