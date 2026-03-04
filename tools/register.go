// Package tools wires Semsage's tool executors to the global agentic-tools registry.
// Callers invoke RegisterAll once at startup with the required infrastructure
// dependencies; tool availability is then visible to every agentic loop that
// runs in the same process.
package tools

import (
	"fmt"

	"github.com/c360studio/semstreams/graph/query"
	agentictools "github.com/c360studio/semstreams/processor/agentic-tools"

	"github.com/c360studio/semsage/agentgraph"
	"github.com/c360studio/semsage/tools/create"
	"github.com/c360studio/semsage/tools/decompose"
	"github.com/c360studio/semsage/tools/spawn"
	"github.com/c360studio/semsage/tools/tree"
)

// Dependencies holds the infrastructure needed to create and register Semsage tools.
// Each field maps to a concrete type from the semstreams framework, expressed as the
// minimal interface the tools actually require, so the caller can supply real
// infrastructure or a test double without pulling in heavyweight concrete types.
type Dependencies struct {
	// NATS is used by the spawn executor to publish task messages and subscribe
	// to completion / failure events. *natsclient.Client satisfies this interface.
	NATS spawn.NATSClient

	// EntityStore persists agent hierarchy entities and relationship triples.
	// *datamanager.Manager satisfies this interface.
	EntityStore agentgraph.EntityStore

	// QueryClient reads the graph for tree traversal queries.
	// Obtain via query.NewClient(ctx, natsClient, query.DefaultConfig()).
	QueryClient query.Client
}

// RegisterAll creates tool executors from deps and registers them with the
// global agentic-tools registry. Pass spawn.Option values (e.g.
// spawn.WithDefaultModel, spawn.WithMaxDepth) to configure the spawn executor.
//
// RegisterAll returns the first error encountered during registration. Because
// the global registry rejects duplicate names, calling RegisterAll more than
// once in the same process will return an error on the second call.
//
// NOTE: Registration is not transactional. If a later registration fails,
// earlier tools remain in the global registry. This is acceptable because
// RegisterAll is called once at startup and failure is fatal.
func RegisterAll(deps Dependencies, opts ...spawn.Option) error {
	helper := agentgraph.NewHelper(deps.EntityStore, deps.QueryClient)

	spawnExec := spawn.NewExecutor(deps.NATS, helper, opts...)
	if err := agentictools.RegisterTool("spawn_agent", spawnExec); err != nil {
		return fmt.Errorf("tools: register spawn_agent: %w", err)
	}

	treeExec := tree.NewExecutor(helper)
	if err := agentictools.RegisterTool("query_agent_tree", treeExec); err != nil {
		return fmt.Errorf("tools: register query_agent_tree: %w", err)
	}

	createExec := create.NewExecutor(agentictools.GetGlobalRegistry())
	if err := agentictools.RegisterTool("create_tool", createExec); err != nil {
		return fmt.Errorf("tools: register create_tool: %w", err)
	}

	decomposeExec := decompose.NewExecutor()
	if err := agentictools.RegisterTool("decompose_task", decomposeExec); err != nil {
		return fmt.Errorf("tools: register decompose_task: %w", err)
	}

	return nil
}
