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
| `natsclient/` — NATS connection, KV buckets, JetStream | All NATS communication |
| `agentic/` — types, state machine, payload registrations | Agentic loop state and payloads |
| `processor/agentic-*` — loop, model, tools, dispatch | Core agentic processing pipeline |
| `config/` — configuration loading | Application configuration |

When designing new components or message flows, consult SemStreams docs at `../semstreams/docs/`.

## Build & Test Commands

*To be updated once Go modules are initialized.*

```bash
go build ./...
go test ./...
go test ./... -race
go test -run TestName ./path/to/package  # single test
go vet ./...
```

## Architecture

### Core Components

| Component | Role | Input | State |
|-----------|------|-------|-------|
| `agentic-loop` | Task orchestration | JetStream: `agent.task.*`, `agent.response.*`, `tool.result.*` | KV: `AGENT_LOOPS`, `AGENT_TRAJECTORIES` |
| `agentic-model` | LLM dispatch | JetStream: `agent.request.*` | — |
| `agentic-tools` | Tool execution | JetStream: `tool.execute.*` | — |
| `workflow-processor` | Workflow orchestration | JetStream: `workflow.trigger.*` | KV: `WORKFLOW_EXECUTIONS`, `WORKFLOW_TIMERS` |

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
