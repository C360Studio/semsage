// Package uiapi provides HTTP endpoints for the Semsage UI.
// It reads loop state from the AGENT_LOOPS KV bucket, queries agent hierarchy
// via the graph layer, and streams real-time activity over Server-Sent Events.
package uiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	// Blank import removed — httputil and url used in Start().
	"sync"
	"sync/atomic"
	"time"

	"github.com/c360studio/semstreams/component"
	"github.com/nats-io/nats.go/jetstream"
)

// natsPublisher is the subset of natsclient.Client used by this component.
// Using an interface rather than the concrete type improves testability and
// reduces coupling to the NATS client implementation details.
type natsPublisher interface {
	PublishToStream(ctx context.Context, subject string, data []byte) error
	GetKeyValueBucket(ctx context.Context, name string) (jetstream.KeyValue, error)
}

// graphQuerier is the subset of agentgraph.Helper used by this component.
// Defined locally so tests can inject a fake without importing agentgraph.
type graphQuerier interface {
	GetChildren(ctx context.Context, loopID string) ([]string, error)
	GetTree(ctx context.Context, rootLoopID string, maxDepth int) ([]string, error)
	GetStatus(ctx context.Context, loopID string) (string, error)
}

// Lifecycle state constants — stored in Component.state as int32.
const (
	stateStopped  int32 = 0
	stateStarting int32 = 1
	stateRunning  int32 = 2
	stateStopping int32 = 3
)

// Component implements component.LifecycleComponent for the ui-api layer.
// It exposes HTTP endpoints consumed by the Semsage UI and an SSE stream
// backed by the AGENT_LOOPS KV bucket.
type Component struct {
	name   string
	config Config
	logger *slog.Logger

	natsClient  natsPublisher
	graphHelper graphQuerier // may be nil if no graph client is wired

	// loopsBucket is lazily initialised on the first request that needs it.
	// Access is guarded by loopsBucketMu.
	loopsBucketMu sync.RWMutex
	loopsBucket   jetstream.KeyValue

	// componentCtx is a cancellable context derived from the Start() parent.
	// SSE handlers derive their contexts from this so Stop() can terminate
	// all active streams.
	componentCtx context.Context

	// graphQLProxy is initialised once when GraphQLProxyURL is configured,
	// and reused across requests to allow HTTP connection pooling.
	graphQLProxy http.Handler

	// state drives the lifecycle state machine.
	state     atomic.Int32
	startTime time.Time
	cancel    context.CancelFunc
	mu        sync.RWMutex
}

// NewComponent constructs a ui-api Component from raw JSON config and platform
// dependencies.  It is the component.Factory function registered in the
// component registry.
func NewComponent(rawConfig json.RawMessage, deps component.Dependencies) (component.Discoverable, error) {
	cfg := DefaultConfig()
	if len(rawConfig) > 0 && string(rawConfig) != "null" {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("ui-api: parse config: %w", err)
		}
		// Re-apply defaults for fields that were omitted in the JSON.
		defaults := DefaultConfig()
		if cfg.LoopsBucket == "" {
			cfg.LoopsBucket = defaults.LoopsBucket
		}
		if cfg.DispatchSubject == "" {
			cfg.DispatchSubject = defaults.DispatchSubject
		}
		if cfg.SignalSubject == "" {
			cfg.SignalSubject = defaults.SignalSubject
		}
	}

	logger := deps.GetLogger().With("component", "ui-api")

	// *natsclient.Client satisfies the natsPublisher interface implicitly.
	var pub natsPublisher
	if deps.NATSClient != nil {
		pub = deps.NATSClient
	}

	return &Component{
		name:       "ui-api",
		config:     cfg,
		logger:     logger,
		natsClient: pub,
	}, nil
}

// --- component.LifecycleComponent ---

// Initialize prepares the component. For ui-api this is a no-op because the
// KV bucket and HTTP routes are wired lazily at request time.
func (c *Component) Initialize() error {
	c.state.Store(stateStopped)
	return nil
}

// Start transitions the component to running state.
// The component context is stored so SSE handlers can derive from it and
// Stop() can cancel all active streams.
func (c *Component) Start(ctx context.Context) error {
	if !c.state.CompareAndSwap(stateStopped, stateStarting) {
		return fmt.Errorf("ui-api: Start called in unexpected state %d", c.state.Load())
	}

	child, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancel = cancel
	c.componentCtx = child
	c.startTime = time.Now()
	c.mu.Unlock()

	// Initialize GraphQL reverse proxy once if configured.
	if c.config.GraphQLProxyURL != "" {
		target, err := url.Parse(c.config.GraphQLProxyURL)
		if err != nil {
			c.logger.Warn("ui-api: invalid GraphQL proxy URL",
				slog.String("url", c.config.GraphQLProxyURL),
				slog.String("error", err.Error()),
			)
		} else {
			c.graphQLProxy = httputil.NewSingleHostReverseProxy(target)
		}
	}

	// Attempt to pre-bind the KV bucket. A failure here is non-fatal — the
	// handler will retry on the first request via getLoopsBucket().
	if c.natsClient != nil {
		if bucket, err := c.natsClient.GetKeyValueBucket(child, c.config.LoopsBucket); err != nil {
			c.logger.Warn("ui-api: could not pre-bind AGENT_LOOPS bucket at startup",
				slog.String("bucket", c.config.LoopsBucket),
				slog.String("error", err.Error()),
			)
		} else {
			c.loopsBucketMu.Lock()
			c.loopsBucket = bucket
			c.loopsBucketMu.Unlock()
		}
	}

	c.state.Store(stateRunning)
	c.logger.Info("ui-api: started",
		slog.String("loops_bucket", c.config.LoopsBucket),
		slog.String("dispatch_subject", c.config.DispatchSubject),
	)
	return nil
}

// Stop cancels any outstanding SSE streams and transitions to stopped.
func (c *Component) Stop(timeout time.Duration) error {
	if !c.state.CompareAndSwap(stateRunning, stateStopping) {
		return nil // Already stopped or not running.
	}

	c.mu.RLock()
	cancel := c.cancel
	c.mu.RUnlock()

	if cancel != nil {
		cancel()
	}

	c.state.Store(stateStopped)
	c.logger.Info("ui-api: stopped")
	return nil
}

// --- component.Discoverable ---

// Meta returns static metadata about this component.
func (c *Component) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.name,
		Type:        "processor",
		Description: "HTTP API layer for the Semsage UI: loop state, hierarchy, trajectory, SSE activity stream",
		Version:     "v1",
	}
}

// InputPorts returns the NATS KV watch port consumed by this component.
func (c *Component) InputPorts() []component.Port {
	return []component.Port{
		{
			Name:        "agent_loops_kv",
			Direction:   component.DirectionInput,
			Required:    false,
			Description: "AGENT_LOOPS KV bucket — read for loop state and SSE activity",
		},
	}
}

// OutputPorts returns the JetStream subjects this component publishes to.
func (c *Component) OutputPorts() []component.Port {
	return []component.Port{
		{
			Name:        "signal_subject",
			Direction:   component.DirectionOutput,
			Required:    false,
			Description: "NATS JetStream subject for user control signals (pause/resume/cancel)",
		},
		{
			Name:        "dispatch_subject",
			Direction:   component.DirectionOutput,
			Required:    false,
			Description: "NATS JetStream subject for chat task dispatch",
		},
	}
}

// ConfigSchema describes the configuration properties accepted by NewComponent.
func (c *Component) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"loops_bucket": {
				Type:        "string",
				Description: "NATS KV bucket name for AGENT_LOOPS state",
				Default:     "AGENT_LOOPS",
			},
			"dispatch_subject": {
				Type:        "string",
				Description: "NATS JetStream subject prefix for task dispatch",
				Default:     "agent.task",
			},
			"signal_subject": {
				Type:        "string",
				Description: "NATS JetStream subject prefix for user control signals",
				Default:     "agent.signal",
			},
			"graphql_proxy_url": {
				Type:        "string",
				Description: "Base URL of the semstreams GraphQL gateway; empty disables the proxy",
				Default:     "",
			},
		},
	}
}

// Health returns the current health of the component.
func (c *Component) Health() component.HealthStatus {
	healthy := c.state.Load() == stateRunning

	c.mu.RLock()
	startTime := c.startTime
	c.mu.RUnlock()

	var uptime time.Duration
	if healthy {
		uptime = time.Since(startTime)
	}

	status := "stopped"
	switch c.state.Load() {
	case stateRunning:
		status = "running"
	case stateStarting:
		status = "starting"
	case stateStopping:
		status = "stopping"
	}

	return component.HealthStatus{
		Healthy:   healthy,
		LastCheck: time.Now(),
		Uptime:    uptime,
		Status:    status,
	}
}

// DataFlow returns stub flow metrics (ui-api does not track throughput).
func (c *Component) DataFlow() component.FlowMetrics {
	return component.FlowMetrics{LastActivity: time.Now()}
}

// --- internal helpers ---

// getLoopsBucket returns the cached AGENT_LOOPS KV bucket, binding it lazily
// on the first call. Thread-safe via double-checked locking.
func (c *Component) getLoopsBucket(ctx context.Context) (jetstream.KeyValue, error) {
	// Fast path — bucket already bound.
	c.loopsBucketMu.RLock()
	bucket := c.loopsBucket
	c.loopsBucketMu.RUnlock()
	if bucket != nil {
		return bucket, nil
	}

	// Slow path — attempt to bind.
	c.loopsBucketMu.Lock()
	defer c.loopsBucketMu.Unlock()
	if c.loopsBucket != nil {
		return c.loopsBucket, nil // Another goroutine beat us here.
	}

	if c.natsClient == nil {
		return nil, fmt.Errorf("ui-api: NATS client not configured")
	}

	var err error
	bucket, err = c.natsClient.GetKeyValueBucket(ctx, c.config.LoopsBucket)
	if err != nil {
		return nil, fmt.Errorf("ui-api: get AGENT_LOOPS bucket %q: %w", c.config.LoopsBucket, err)
	}
	c.loopsBucket = bucket
	return bucket, nil
}

// WithGraphHelper wires a graphQuerier implementation into the component.
// Call this after NewComponent when a real agentgraph.Helper is available.
func (c *Component) WithGraphHelper(g graphQuerier) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.graphHelper = g
}
