# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Semsage is a Go application built on top of [SemStreams](https://github.com/C360Studio/semstreams), a stream processor that builds semantic knowledge graphs from event data using NATS JetStream. Semsage uses SemStreams' component architecture, payload registry, and NATS primitives (KV Watch + JetStream Streams) to implement agentic AI workflows — LLM task orchestration, tool execution, and multi-step workflow processing.

See `docs/semsage-spec.md` for the architectural specification.

## Relationship to SemStreams

SemStreams (available locally at `../semstreams`) provides the core framework:

| SemStreams provides | Semsage uses it for |
|--------------------|--------------------|
| `component/` — base types, lifecycle, ports, payload registry | Building custom components |
| `message/` — Graphable interface, Triple, BaseMessage | Domain entity modeling |
| `graph/` — EntityState, DataManager, query client, hierarchy inference | Agent hierarchy as graph entities with relationship triples |
| `natsclient/` — NATS connection, KV buckets, JetStream | All NATS communication |
| `agentic/` — types, state machine, payload registrations | Agentic loop state and payloads |
| `processor/agentic-*` — loop, model, tools, dispatch | Core agentic processing pipeline |
| `config/` — configuration loading | Application configuration |

When designing new components or message flows, consult SemStreams docs at `../semstreams/docs/`.

## Build & Test Commands

```bash
go build ./...
go test ./...
go test ./... -race
go test -run TestName ./path/to/package  # single test
go vet ./...
```

## Architecture

### SemStreams Components (used by Semsage)

| Component | Role | Input | State |
|-----------|------|-------|-------|
| `agentic-loop` | Task orchestration | JetStream: `agent.task.*`, `agent.response.*`, `tool.result.*` | KV: `AGENT_LOOPS`, `AGENT_TRAJECTORIES` |
| `agentic-model` | LLM dispatch | JetStream: `agent.request.*` | — |
| `agentic-tools` | Tool execution | JetStream: `tool.execute.*` | — |
| `workflow-processor` | Workflow orchestration | JetStream: `workflow.trigger.*` | KV: `WORKFLOW_EXECUTIONS`, `WORKFLOW_TIMERS` |

### Semsage Tools (registered as ToolExecutors)

| Tool | Role |
|------|------|
| `spawn_agent` | Publishes TaskMessage, blocks until child completes, creates graph relationship |
| `create_tool` | Composes existing processors into named reactive definitions at runtime |
| `decompose_task` | Returns DAG of subtasks as structured JSON |
| `query_agent_tree` | Queries agent hierarchy via graph query client |

### Agent Hierarchy — Hybrid Storage Model

- **`AGENT_LOOPS` KV** (SemStreams, existing) — mutable loop state machine, KV Watch for real-time SSE
- **`ENTITY_STATES` KV** (SemStreams, existing) — lightweight graph entity references with relationship triples (`agentic.loop.spawned`, `agentic.task.depends_on`)
- Entity IDs for agents: `semsage.default.agentic.orchestrator.loop.<loop-id>`
- No new KV buckets — hierarchy queries use existing graph infrastructure

### Key Design Decisions

**KV Watch vs JetStream Streams** — Use the restart test: if replaying all messages since the beginning of time would be correct, use KV Watch. If it would be catastrophic, use JetStream Stream. Use `/kv-or-stream` for the full 4-test heuristic.

**Orchestration boundaries** — Rules trigger (single action), workflows coordinate (loops with limits), components execute (workflow-agnostic). State ownership is exclusive. Use `/orchestration-check` for boundary decisions.

**Payload types** — Polymorphic JSON via type-discriminated `BaseMessage` envelopes. New types need `init()` registration, `MarshalJSON` with type alias, and a blank import. Use `/new-payload` for the checklist.

**Query access** — GraphQL for external apps, MCP for AI agents, NATS Direct for internal services. Use `/query-pattern` for selection guidance.

## Conventions

- **Go context**: Always pass `context.Context` as first parameter for I/O operations
- **Error handling**: Return errors with context wrapping (`fmt.Errorf("operation: %w", err)`), don't log-and-return
- **Concurrency**: Prefer channels over shared memory; defer mutex unlock immediately after lock
- **Testing**: Table-driven tests, test behavior not implementation, explicit synchronization (no arbitrary sleeps)
- **Commits**: Conventional commits format `<type>(scope): subject`
- **Entity IDs**: 6-part hierarchical format `org.platform.domain.system.type.instance`
- **Configs**: JSON only (not YAML) — follows SemStreams convention. See `configs/semsage.json`
- **Payloads**: Any payload crossing NATS must use the payload registry (`Schema()`, `MarshalJSON` with type alias, `init()` registration). No ad-hoc structs for wire format.
