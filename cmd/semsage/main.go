package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c360studio/semstreams/config"
	gtypes "github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/graph/datamanager"
	"github.com/c360studio/semstreams/graph/query"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/metric"
	"github.com/c360studio/semstreams/natsclient"
	agentictools "github.com/c360studio/semstreams/processor/agentic-tools"
	"github.com/c360studio/semstreams/service"
	"github.com/c360studio/semstreams/types"

	uiapi "github.com/c360studio/semsage/processor/ui-api"
	"github.com/c360studio/semsage/tools"
	"github.com/c360studio/semsage/tools/spawn"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/semsage.json", "Path to configuration file")
	flag.Parse()

	// --- Load configuration ---
	loader := config.NewLoader()
	cfg, err := loader.LoadFile(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// --- Connect to NATS ---
	natsURL := natsURLFromConfig(cfg)
	natsClient, err := natsclient.NewClient(natsURL)
	if err != nil {
		return fmt.Errorf("create NATS client: %w", err)
	}

	ctx := context.Background()
	if err := natsClient.Connect(ctx); err != nil {
		return fmt.Errorf("connect to NATS %s: %w", natsURL, err)
	}
	defer natsClient.Close(ctx)
	logger.Info("connected to NATS", "url", natsURL)

	// --- Query client ---
	queryClient, err := query.NewClient(ctx, natsClient, query.DefaultConfig())
	if err != nil {
		return fmt.Errorf("create query client: %w", err)
	}

	// --- Register semsage tool executors ---
	semsageCfg := semsageConfig(cfg)
	if err := tools.RegisterAll(
		tools.Dependencies{
			NATS:        natsClient,
			EntityStore: &noopEntityStore{},
			QueryClient: queryClient,
		},
		spawn.WithDefaultModel(semsageCfg.defaultModel),
		spawn.WithMaxDepth(semsageCfg.maxSpawnDepth),
	); err != nil {
		return fmt.Errorf("register tools: %w", err)
	}

	registered := agentictools.ListRegisteredTools()
	names := make([]string, 0, len(registered))
	for _, def := range registered {
		names = append(names, def.Name)
	}
	logger.Info("semsage tools registered", "tools", names)

	// --- Setup service registry and manager ---
	serviceRegistry := service.NewServiceRegistry()
	if err := service.RegisterAll(serviceRegistry); err != nil {
		return fmt.Errorf("register built-in services: %w", err)
	}

	// Register semsage-specific services.
	if err := serviceRegistry.Register("ui-api", uiapi.NewUIAPIService); err != nil {
		return fmt.Errorf("register ui-api service: %w", err)
	}

	manager := service.NewServiceManager(serviceRegistry)
	ensureServiceManagerConfig(cfg)

	// --- Create service dependencies ---
	metricsRegistry := metric.NewMetricsRegistry()
	platform := extractPlatformMeta(cfg)

	svcDeps := &service.Dependencies{
		NATSClient:      natsClient,
		MetricsRegistry: metricsRegistry,
		Logger:          logger,
		Platform:        platform,
	}

	// --- Configure and create services ---
	if err := manager.ConfigureFromServices(cfg.Services, svcDeps); err != nil {
		return fmt.Errorf("configure service manager: %w", err)
	}

	for name, svcConfig := range cfg.Services {
		if name == "service-manager" {
			continue
		}
		if !svcConfig.Enabled {
			logger.Info("service disabled", "name", name)
			continue
		}
		if !manager.HasConstructor(name) {
			logger.Warn("service not registered", "name", name)
			continue
		}
		if _, err := manager.CreateService(name, svcConfig.Config, svcDeps); err != nil {
			return fmt.Errorf("create service %s: %w", name, err)
		}
		logger.Info("created service", "name", name)
	}

	// --- Start all services ---
	signalCtx, signalCancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()

	if err := manager.StartAll(signalCtx); err != nil {
		return fmt.Errorf("start services: %w", err)
	}
	logger.Info("semsage ready — all services started")

	// --- Block until signal ---
	<-signalCtx.Done()
	logger.Info("shutdown signal received")

	shutdownTimeout := 10 * time.Second
	if err := manager.StopAll(shutdownTimeout); err != nil {
		logger.Error("error stopping services", "error", err)
		return err
	}

	logger.Info("semsage shutdown complete")
	return nil
}

// natsURLFromConfig extracts the NATS URL, preferring env var over config.
func natsURLFromConfig(cfg *config.Config) string {
	if v := os.Getenv("NATS_URL"); v != "" {
		return v
	}
	if len(cfg.NATS.URLs) > 0 {
		return cfg.NATS.URLs[0]
	}
	return "nats://localhost:4222"
}

// semsageSettings holds semsage-specific configuration extracted from the
// top-level config "semsage" key.
type semsageSettings struct {
	defaultModel  string
	maxSpawnDepth int
}

// semsageConfig extracts semsage-specific settings from the config.
func semsageConfig(cfg *config.Config) semsageSettings {
	s := semsageSettings{
		defaultModel:  "claude-sonnet-4-20250514",
		maxSpawnDepth: 5,
	}

	// The semsage-specific config lives in the top-level JSON under "semsage".
	// config.Config doesn't have a Semsage field, so we access it via the
	// raw services config or use environment variables.
	if v := os.Getenv("SEMSAGE_DEFAULT_MODEL"); v != "" {
		s.defaultModel = v
	}
	if v := os.Getenv("SEMSAGE_MAX_SPAWN_DEPTH"); v != "" {
		var n int
		if _, scanErr := fmt.Sscanf(v, "%d", &n); scanErr == nil && n > 0 {
			s.maxSpawnDepth = n
		}
	}
	return s
}

// ensureServiceManagerConfig ensures the service-manager config exists with defaults.
func ensureServiceManagerConfig(cfg *config.Config) {
	if cfg.Services == nil {
		cfg.Services = make(types.ServiceConfigs)
	}
	if _, exists := cfg.Services["service-manager"]; !exists {
		defaultConfig := map[string]any{
			"http_port":  8090,
			"swagger_ui": true,
			"server_info": map[string]string{
				"title":   "Semsage API",
				"version": "0.1.0",
			},
		}
		raw, _ := json.Marshal(defaultConfig)
		cfg.Services["service-manager"] = types.ServiceConfig{
			Name:    "service-manager",
			Enabled: true,
			Config:  raw,
		}
	}
}

// extractPlatformMeta extracts platform identity from config.
func extractPlatformMeta(cfg *config.Config) types.PlatformMeta {
	platformID := cfg.Platform.InstanceID
	if platformID == "" {
		platformID = cfg.Platform.ID
	}
	return types.PlatformMeta{
		Org:      cfg.Platform.Org,
		Platform: platformID,
	}
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
