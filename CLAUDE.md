# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Semsage is a full-stack application (Go backend + SvelteKit frontend) built on top of [SemStreams](https://github.com/C360Studio/semstreams), a stream processor that builds semantic knowledge graphs from event data using NATS JetStream. Semsage uses SemStreams' component architecture, payload registry, and NATS primitives (KV Watch + JetStream Streams) to implement agentic AI workflows — LLM task orchestration, tool execution, and multi-step workflow processing — with a real-time dashboard for monitoring and control.

See `docs/semsage-spec.md` for the backend architectural specification and `docs/ui-spec.md` for the dashboard specification.

## Relationship to SemStreams

SemStreams (`github.com/c360studio/semstreams`) provides the core framework:

| SemStreams provides | Semsage uses it for |
|--------------------|--------------------|
| `component/` — base types, lifecycle, ports, payload registry | Building custom components (including `ui-api`) |
| `message/` — Graphable interface, Triple, BaseMessage | Domain entity modeling |
| `graph/` — EntityState, DataManager, query client, hierarchy inference | Agent hierarchy as graph entities with relationship triples |
| `natsclient/` — NATS connection, KV buckets, JetStream | All NATS communication |
| `agentic/` — types, state machine, payload registrations | Agentic loop state and payloads |
| `processor/agentic-*` — loop, model, tools, dispatch | Core agentic processing pipeline |
| `config/` — configuration loading | Application configuration |

When designing new components or message flows, consult SemStreams docs (clone `github.com/C360Studio/semstreams` locally if needed).

## Build & Test Commands

### Backend (Go)

```bash
go build ./...
go test ./...
go test ./... -race
go test -run TestName ./path/to/package  # single test
go vet ./...
```

### Frontend (SvelteKit)

```bash
cd ui
npm install
npm run dev              # Dev server (port 5173)
npm run build            # Production build (static adapter)
npm run check            # Svelte/TypeScript validation
npm run test:e2e         # Playwright E2E tests
npm run test:e2e:ui      # Playwright UI mode
```

### Infrastructure

```bash
docker compose up -d     # NATS server (ports 4222, 8222)
```

## Project Structure

```
semsage/
├── cmd/semsage/           # Service entry point
├── agentgraph/            # Graph entity helpers for agent hierarchy
├── processor/ui-api/      # HTTP API + SSE component (SemStreams component)
├── tools/                 # Agentic tool executor registration
│   ├── spawn/             # spawn_agent executor
│   ├── create/            # create_tool executor
│   ├── decompose/         # decompose_task executor
│   └── tree/              # query_agent_tree executor
├── workflow/dag/          # DAG execution reactive definition
├── ui/                    # SvelteKit 2 + Svelte 5 dashboard
│   ├── src/lib/           # Stores, API client, types, components
│   └── e2e/               # Playwright E2E tests (43 tests, route-mocked)
├── configs/               # JSON configuration (semsage.json)
├── docs/                  # Specs (semsage-spec.md, ui-spec.md)
└── docker-compose.yml     # NATS dev environment
```

## Architecture

### Go Packages

| Package | Role |
|---------|------|
| `agentgraph` | Entity ID mapping, relationship predicates, graph helper facade |
| `processor/ui-api` | SemStreams component: HTTP handlers, SSE activity stream, GraphQL proxy |
| `tools/spawn` | spawn_agent — publishes TaskMessage, blocks until child completes, records graph |
| `tools/create` | create_tool — composes processors into named reactive definitions at runtime |
| `tools/decompose` | decompose_task — validates DAG structure, returns as JSON for parent agent |
| `tools/tree` | query_agent_tree — wraps graph queries (get_tree, get_children, get_status) |
| `workflow/dag` | DAG execution reactive definition (workflow orchestration) |
| `cmd/semsage` | Entry point — NATS setup, tool registration, graceful shutdown |

### SemStreams Components (used by Semsage)

| Component | Role | Input | State |
|-----------|------|-------|-------|
| `agentic-loop` | Task orchestration | JetStream: `agent.task.*`, `agent.response.*`, `tool.result.*` | KV: `AGENT_LOOPS`, `AGENT_TRAJECTORIES` |
| `agentic-model` | LLM dispatch | JetStream: `agent.request.*` | — |
| `agentic-tools` | Tool execution | JetStream: `tool.execute.*` | — |
| `workflow-processor` | Workflow orchestration | JetStream: `workflow.trigger.*` | KV: `WORKFLOW_EXECUTIONS`, `WORKFLOW_TIMERS` |

### Semsage-Owned Component

| Component | Role | Input | State |
|-----------|------|-------|-------|
| `ui-api` | HTTP API + SSE for dashboard | KV Watch: `AGENT_LOOPS` | — |

### HTTP API Routes (`processor/ui-api`)

| Method | Route | Purpose |
|--------|-------|---------|
| GET | `/api/health` | System health with component status |
| GET | `/api/loops` | List loops (optional `?state=` filter) |
| GET | `/api/loops/{id}` | Loop detail (parent, depth, status) |
| POST | `/api/loops/{id}/signal` | Pause/resume/cancel (cascades to children) |
| GET | `/api/loops/{id}/children` | Direct children via graph |
| GET | `/api/loops/{id}/tree` | Full subtree from loop |
| GET | `/api/trajectory/loops/{id}` | Trajectory for loop |
| GET | `/api/trajectory/loops/{id}/calls/{req_id}` | Full LLM call record |
| GET | `/api/tools` | Dynamic tools (scoped by `?root_loop_id=`) |
| POST | `/api/chat` | Send chat message |
| GET | `/api/activity` | SSE event stream |
| GET | `/graphql/` | Entity queries (reverse proxy) |

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

## Frontend (SvelteKit Dashboard)

### Stack
- **SvelteKit 2** with static adapter, **Svelte 5** runes system, **TypeScript** (strict)
- **Vite 6** build tool, **Playwright** E2E tests, **Lucide** icons
- No CSS framework — custom design tokens copied from semspec app.css

### Key Directories

| Directory | Contents |
|-----------|----------|
| `ui/src/routes/` | 6 pages: activity, loop detail, entities, entity detail, settings |
| `ui/src/lib/stores/` | 11 Svelte 5 rune-based stores (activity, loops, trajectory, settings, etc.) |
| `ui/src/lib/components/` | 51 components across shared, activity, chat, loops, entities, tree, trajectory, tools |
| `ui/src/lib/api/` | HTTP client, GraphQL helper, data transforms |
| `ui/src/lib/types/` | TypeScript type definitions (Loop, Trajectory, Entity, AgentTreeNode, etc.) |
| `ui/e2e/` | 6 Playwright test files, route mocking fixtures (no backend needed) |

### Frontend Conventions
- **State management**: Svelte 5 runes (`$state()`, `$derived()`, `$effect()`, `$props()`) — no legacy stores
- **Events**: Callback props pattern, not createEventDispatcher
- **API calls**: Use `api` namespace from `src/lib/api/client.ts`
- **Testing**: E2E with Playwright using route mocks (`e2e/fixtures/api-mocks.ts`)
- **Types**: All API responses typed in `src/lib/types/index.ts`

## Conventions

### Go
- **Context**: Always pass `context.Context` as first parameter for I/O operations
- **Error handling**: Return errors with context wrapping (`fmt.Errorf("operation: %w", err)`), don't log-and-return
- **Concurrency**: Prefer channels over shared memory; defer mutex unlock immediately after lock
- **Testing**: Table-driven tests, test behavior not implementation, explicit synchronization (no arbitrary sleeps)
- **Tool executors**: Return `ToolResult.Error` for business errors, Go `error` for infrastructure failures

### General
- **Commits**: Conventional commits format `<type>(scope): subject`
- **Entity IDs**: 6-part hierarchical format `org.platform.domain.system.type.instance`
- **Configs**: JSON only (not YAML) — follows SemStreams convention. See `configs/semsage.json`
- **Payloads**: Any payload crossing NATS must use the payload registry (`Schema()`, `MarshalJSON` with type alias, `init()` registration). No ad-hoc structs for wire format.
