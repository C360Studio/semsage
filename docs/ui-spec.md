# Semsage UI Specification

What to reuse from [semspec](https://github.com/C360Studio/semspec)'s UI, what to skip, and what semsage needs on its own.

## Stack (reuse from semspec)

Identical stack — no reason to diverge:

- **Svelte 5** + **SvelteKit 2** (static adapter)
- **TypeScript** (strict)
- **Vite 6** (build)
- **Vitest** (unit tests) + **Playwright** (E2E)
- **Lucide Svelte** (icons)
- **openapi-typescript** (API type generation)
- Vanilla CSS with custom properties (no framework)

## Design System (copy from semspec)

Copy `app.css` design tokens directly — dark/light themes, typography scale, spacing scale, semantic colors, shadows, transitions, layout constants. No changes needed.

## What to Reuse from Semspec

### Copy directly (minimal changes)

| Semspec component/file | Why |
|----------------------|-----|
| `app.css` | Design tokens, theme system |
| `api/client.ts` (request/toQueryString pattern) | HTTP client foundation — strip semspec-specific endpoints, keep `request<T>()` and `api` namespace pattern |
| `stores/activity.svelte.ts` | SSE activity stream — same pattern, different endpoint |
| `stores/trajectory.svelte.ts` | Trajectory cache per loop_id |
| `stores/loops.svelte.ts` | Active loop list + signal controls |
| `stores/settings.svelte.ts` | Theme, reduced motion, activity limit |
| `stores/sidebar.svelte.ts` | Mobile sidebar state |
| `stores/system.svelte.ts` | Backend health/connectivity |
| `+layout.svelte` | App shell — sidebar, header, mobile hamburger, chat drawer, keyboard shortcuts |
| `components/shared/Sidebar.svelte` | Navigation sidebar (simplify nav items) |
| `components/shared/Header.svelte` | Page header |
| `components/shared/Icon.svelte` | Lucide icon wrapper |
| `components/shared/Modal.svelte` | Generic modal |
| `components/shared/CollapsiblePanel.svelte` | Collapsible section |
| `components/shared/ResizableSplit.svelte` | Draggable split panes |
| `components/chat/ChatDrawer.svelte` | Cmd+K chat drawer |
| `components/trajectory/TrajectoryPanel.svelte` | Loop execution details |
| `components/trajectory/TrajectoryEntryCard.svelte` | Model/tool call entries |
| `components/loops/LoopCard.svelte` | Loop status + pause/resume/cancel (add depth indicator, parent link) |
| `components/timeline/AgentTimeline.svelte` | Visual timeline swim lanes |
| `components/timeline/TimelineTrack.svelte` | Per-agent track |
| `components/activity/ActivityFeed.svelte` | Event stream display |
| `components/entities/EntityCard.svelte` | Entity preview card |
| `components/entities/RelationshipList.svelte` | Entity relationships |

### Skip (semspec-specific)

| Semspec feature | Why skip |
|----------------|---------|
| Plan management (PlanCard, PlanDetailPanel, PlanNavTree, pipeline stages) | Semsage doesn't have a formal plan lifecycle |
| Phase/task approval workflow (PhaseDetail, TaskDetail, TaskList, approval modals) | No approval gates in semsage |
| Review dashboard (ReviewDashboard, SpecGate, ReviewerCard, FindingsList) | No multi-stage review process |
| Source management (SourceCard, RepositoryCard, UploadModal, etc.) | No document/repo ingestion UI |
| Setup wizard (SetupWizard, detection steps) | Semsage has different initialization |
| Question queue (QuestionQueue, QuestionCard) | May add later, not needed initially |
| Context budget tracking (context store, provenance) | Semsage handles context differently |

## What Semsage Needs (new or adapted)

### Routes

Simpler than semspec — fewer pages, focused on observing agent hierarchies:

| Route | Purpose | Source |
|-------|---------|--------|
| `/` | Redirect to `/activity` | New (simple redirect) |
| `/activity` | Live event stream + active agent trees | Adapt from semspec `/activity` |
| `/loops/[id]` | Loop detail with trajectory, children, tool calls | Adapt from semspec trajectory panel + new tree view |
| `/entities` | Entity browser — search knowledge graph | Copy from semspec `/entities` |
| `/entities/[id]` | Entity detail with relationships | Copy from semspec `/entities/[id]` |
| `/settings` | Theme, preferences | Copy from semspec `/settings` |

### Core pages

**Activity (home page)**
- Left: activity event stream (SSE)
- Right: active agent trees — root loops with expandable child hierarchies
  - Each root loop shows: role, state, depth indicator, token totals
  - Expandable to reveal children spawned via `spawn_agent`
  - Click any loop → navigate to `/loops/[id]`
  - Pause/resume/cancel on any loop (cancellation cascades to children per spec)

**Loop detail**
- Header: loop metadata — role, model, state, iteration count, duration, tokens, depth/maxDepth
- Parentage: breadcrumb showing parent chain (root → ... → parent → this loop), clickable
- Children: list of child loops spawned by this loop (`agentic.loop.spawned` relationships)
  - Each child: role, state, result summary (if completed)
  - Click to navigate to child's detail page
- Trajectory: chronological list of model calls and tool calls
  - Model call: model name, tokens in/out, finish reason, expandable request/response
  - Tool call: tool name, status, duration, expandable input/output
  - `spawn_agent` calls: highlighted distinctly — show child loop ID, link to child detail, inline status (waiting/completed/failed)
  - `create_tool` calls: show tool name + description + processors composed
  - `decompose_task` calls: show returned DAG inline (nodes + dependency edges)
- Controls: pause, resume, cancel (if active)
- Dynamic tools: list of tools created by this agent tree via `create_tool` (scoped to root loop)

**Entity browser**
- Search + type filter
- Entity cards with key predicates
- Detail view with relationship graph
- Agent entities (`semsage.default.agentic.orchestrator.loop.*`) are browsable here too — shows spawned relationships as graph edges

### API surface

Semsage backend needs these HTTP endpoints for the UI:

```
GET  /api/health                              # System health
GET  /api/activity                            # SSE event stream
GET  /api/loops                               # List active/recent loops (flat)
GET  /api/loops/{id}                          # Loop detail (includes parent_loop_id, depth)
POST /api/loops/{id}/signal                   # Pause/resume/cancel (cascades to children)
GET  /api/loops/{id}/children                 # Direct children of a loop (via graph: agentic.loop.spawned)
GET  /api/loops/{id}/tree                     # Full agent tree from this loop down (via graph: ExecutePathQuery)
GET  /api/trajectory/loops/{id}               # Trajectory for loop
GET  /api/trajectory/loops/{id}/calls/{req_id}  # Full LLM call record
GET  /api/tools?root_loop_id={id}             # Dynamic tools created by this agent tree
POST /api/chat                                # Send chat message
GET  /graphql/                                # Entity queries
```

The `/children` and `/tree` endpoints are thin wrappers around `query_agent_tree` operations (`get_children` and `get_tree`). The `/tools` endpoint queries tools scoped to a root loop ID (per the tool scoping design decision).

### Stores (Svelte 5 runes)

| Store | Based on | Changes |
|-------|----------|---------|
| `activity.svelte.ts` | semspec | Change SSE endpoint to `/api/activity` |
| `loops.svelte.ts` | semspec | Change REST endpoints to `/api/loops`, add `fetchChildren`, `fetchTree` |
| `trajectory.svelte.ts` | semspec | Change endpoints to `/api/trajectory/loops/*` |
| `agentTree.svelte.ts` | **new** | Manages expanded/collapsed tree state, caches children per loop |
| `dynamicTools.svelte.ts` | **new** | Fetches tools scoped to a root loop ID |
| `settings.svelte.ts` | semspec | Same |
| `system.svelte.ts` | semspec | Change health endpoint to `/api/health` |
| `sidebar.svelte.ts` | semspec | Same |
| `chatDrawer.svelte.ts` | semspec | Same |
| `messages.svelte.ts` | semspec | Change send endpoint to `/api/chat` |

### API client

Same `request<T>()` pattern with simplified namespace:

```typescript
export const api = {
    system: { getHealth },
    loops:  { list, get, sendSignal, getChildren, getTree },
    trajectory: { getByLoop, getCall },
    tools:  { listByTree },  // dynamic tools scoped to root loop
    chat:   { send },
    entities: { list, get, relationships, count }  // GraphQL
};
```

## Implementation Order

1. **Scaffold** — SvelteKit project, copy design tokens, shared components, API client pattern
2. **Activity page** — SSE store, loop list store, activity feed + loop cards
3. **Loop detail page** — Trajectory store, trajectory panel, entry cards
4. **Chat drawer** — Chat store, Cmd+K shortcut
5. **Entity browser** — GraphQL client, entity cards, relationship list
6. **Settings** — Theme toggle, preferences
7. **E2E tests** — Playwright with mock mode

## Semsage-Specific Components (new)

These don't exist in semspec and need to be built for semsage's agent hierarchy model:

| Component | Purpose |
|-----------|---------|
| `components/tree/AgentTreeView.svelte` | Expandable tree of agent loops (root → children), used on activity page |
| `components/tree/AgentTreeNode.svelte` | Single node in tree — recursive, shows role/state/tokens, expand to reveal children |
| `components/loops/LoopBreadcrumb.svelte` | Parent chain breadcrumb on loop detail page (root → ... → parent → current) |
| `components/loops/ChildLoopList.svelte` | List of child loops on loop detail page |
| `components/trajectory/SpawnAgentEntry.svelte` | Trajectory entry for `spawn_agent` calls — child link, inline status |
| `components/trajectory/CreateToolEntry.svelte` | Trajectory entry for `create_tool` calls — tool name, processors composed |
| `components/trajectory/DecomposeTaskEntry.svelte` | Trajectory entry for `decompose_task` — inline DAG visualization |
| `components/trajectory/DagView.svelte` | Simple DAG node-and-edge visualization for decompose_task results |
| `components/tools/DynamicToolList.svelte` | List of tools created by `create_tool`, scoped to root loop |

## File Structure

```
ui/
├── src/
│   ├── lib/
│   │   ├── api/
│   │   │   ├── client.ts          # request<T>(), api namespace
│   │   │   ├── graphql.ts         # GraphQL query helper
│   │   │   └── mock.ts            # Mock mode for dev
│   │   ├── stores/                # Svelte 5 runes stores (*.svelte.ts)
│   │   ├── components/
│   │   │   ├── activity/          # ActivityFeed
│   │   │   ├── chat/              # ChatDrawer, ChatPanel
│   │   │   ├── entities/          # EntityCard, RelationshipList
│   │   │   ├── loops/             # LoopCard, LoopBreadcrumb, ChildLoopList
│   │   │   ├── shared/            # Sidebar, Header, Icon, Modal, CollapsiblePanel, ResizableSplit
│   │   │   ├── timeline/          # AgentTimeline, TimelineTrack
│   │   │   ├── tools/             # DynamicToolList
│   │   │   ├── trajectory/        # TrajectoryPanel, TrajectoryEntryCard, SpawnAgentEntry, CreateToolEntry, DecomposeTaskEntry, DagView
│   │   │   └── tree/              # AgentTreeView, AgentTreeNode
│   │   └── types/                 # TypeScript interfaces
│   ├── routes/
│   │   ├── +layout.svelte
│   │   ├── +page.svelte           # Redirect to /activity
│   │   ├── activity/+page.svelte
│   │   ├── loops/[id]/+page.svelte
│   │   ├── entities/+page.svelte
│   │   ├── entities/[id]/+page.svelte
│   │   └── settings/+page.svelte
│   └── app.css
├── e2e/
├── static/
└── package.json
```
