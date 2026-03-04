package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	gtypes "github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/graph/datamanager"
	"github.com/c360studio/semstreams/graph/query"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	agentictools "github.com/c360studio/semstreams/processor/agentic-tools"

	"github.com/c360studio/semsage/tools"
	"github.com/c360studio/semsage/tools/spawn"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// --- Configuration ---
	natsURL := flag.String("nats-url", envOr("NATS_URL", "nats://localhost:4222"), "NATS server URL")
	defaultModel := flag.String(
		"default-model",
		envOr("SEMSAGE_DEFAULT_MODEL", "claude-sonnet-4-20250514"),
		"Default LLM model for spawned agents",
	)
	maxSpawnDepth := flag.Int("max-spawn-depth", 5, "Maximum agent spawn depth")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- NATS connection ---
	natsClient, err := natsclient.NewClient(*natsURL)
	if err != nil {
		logger.Error("failed to create NATS client", "error", err)
		os.Exit(1)
	}

	if err := natsClient.Connect(ctx); err != nil {
		logger.Error("failed to connect to NATS", "url", *natsURL, "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := natsClient.Close(context.Background()); closeErr != nil {
			logger.Warn("NATS close returned error", "error", closeErr)
		}
	}()

	logger.Info("connected to NATS", "url", *natsURL)

	// --- Query client ---
	// query.NewClient lazily initialises the KV bucket handles on first use,
	// so it does not need running JetStream streams at construction time.
	queryClient, err := query.NewClient(ctx, natsClient, query.DefaultConfig())
	if err != nil {
		logger.Error("failed to create query client", "error", err)
		os.Exit(1)
	}

	// --- Tool registration ---
	// Semsage's role in the SemStreams ecosystem is to register its tools into
	// the global registry. The agentic-tools component (running inside the SemStreams
	// service) dispatches incoming tool.execute.* messages to these executors.
	//
	// TODO: Replace noopEntityStore with a real *datamanager.Manager once the
	// graph-processor KV bucket is provisioned. Until then, RecordSpawn and
	// RecordLoopStatus calls are silently no-ops; spawn_agent still works
	// end-to-end, just without graph hierarchy recording.
	if err := tools.RegisterAll(
		tools.Dependencies{
			NATS:        natsClient,
			EntityStore: &noopEntityStore{},
			QueryClient: queryClient,
		},
		spawn.WithDefaultModel(*defaultModel),
		spawn.WithMaxDepth(*maxSpawnDepth),
	); err != nil {
		logger.Error("failed to register tools", "error", err)
		os.Exit(1)
	}

	// Log registered tools for operator visibility.
	registered := agentictools.ListRegisteredTools()
	names := make([]string, 0, len(registered))
	for _, def := range registered {
		names = append(names, def.Name)
	}
	logger.Info("semsage ready", "registered_tools", names)

	// --- Block until signal ---
	<-ctx.Done()
	logger.Info("shutdown signal received, draining...")
}

// envOr returns the value of the named environment variable, or fallback if
// the variable is unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// noopEntityStore is a compile-time placeholder that satisfies agentgraph.EntityStore
// (datamanager.EntityManager + datamanager.TripleManager) with no-op implementations.
// All write operations silently succeed; reads return not-found errors.
//
// Replace with a real *datamanager.Manager once the KV bucket is provisioned.
type noopEntityStore struct{}

// -- datamanager.EntityReader --

func (n *noopEntityStore) GetEntity(_ context.Context, _ string) (*gtypes.EntityState, error) {
	return nil, nil
}

func (n *noopEntityStore) ExistsEntity(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (n *noopEntityStore) BatchGet(_ context.Context, _ []string) ([]*gtypes.EntityState, error) {
	return nil, nil
}

func (n *noopEntityStore) ListWithPrefix(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

// -- datamanager.EntityWriter --

func (n *noopEntityStore) CreateEntity(_ context.Context, e *gtypes.EntityState) (*gtypes.EntityState, error) {
	return e, nil
}

func (n *noopEntityStore) UpdateEntity(_ context.Context, e *gtypes.EntityState) (*gtypes.EntityState, error) {
	return e, nil
}

func (n *noopEntityStore) UpsertEntity(_ context.Context, e *gtypes.EntityState) (*gtypes.EntityState, error) {
	return e, nil
}

func (n *noopEntityStore) DeleteEntity(_ context.Context, _ string) error {
	return nil
}

// -- datamanager.EntityManager (additional methods) --

func (n *noopEntityStore) CreateEntityWithTriples(
	_ context.Context,
	e *gtypes.EntityState,
	_ []message.Triple,
) (*gtypes.EntityState, error) {
	return e, nil
}

func (n *noopEntityStore) UpdateEntityWithTriples(
	_ context.Context,
	e *gtypes.EntityState,
	_ []message.Triple,
	_ []string,
) (*gtypes.EntityState, error) {
	return e, nil
}

func (n *noopEntityStore) BatchWrite(_ context.Context, _ []datamanager.EntityWrite) error {
	return nil
}

func (n *noopEntityStore) List(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

// -- datamanager.TripleManager --

func (n *noopEntityStore) AddTriple(_ context.Context, _ message.Triple) error {
	return nil
}

func (n *noopEntityStore) RemoveTriple(_ context.Context, _, _ string) error {
	return nil
}

func (n *noopEntityStore) CreateRelationship(
	_ context.Context,
	_, _, _ string,
	_ map[string]any,
) error {
	return nil
}

func (n *noopEntityStore) DeleteRelationship(
	_ context.Context,
	_, _, _ string,
) error {
	return nil
}
