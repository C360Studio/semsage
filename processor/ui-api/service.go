package uiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/c360studio/semstreams/health"
	"github.com/c360studio/semstreams/metric"
	"github.com/c360studio/semstreams/service"
)

// Compile-time check that UIAPIService satisfies the service interfaces.
var (
	_ service.Service     = (*UIAPIService)(nil)
	_ service.HTTPHandler = (*UIAPIService)(nil)
)

// UIAPIService adapts the ui-api Component to the service.Service and
// service.HTTPHandler interfaces so it can be managed by the SemStreams
// service manager.
type UIAPIService struct {
	*service.BaseService
	component *Component
}

// NewUIAPIService is the service.Constructor for the ui-api service.
// It creates the underlying Component from raw JSON config and wires NATS
// from the service dependencies.
func NewUIAPIService(rawConfig json.RawMessage, deps *service.Dependencies) (service.Service, error) {
	cfg := DefaultConfig()
	if len(rawConfig) > 0 && string(rawConfig) != "null" {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("ui-api: parse config: %w", err)
		}
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

	logger := deps.Logger.With("service", "ui-api")

	var pub natsPublisher
	if deps.NATSClient != nil {
		pub = deps.NATSClient
	}

	comp := &Component{
		name:       "ui-api",
		config:     cfg,
		logger:     logger,
		natsClient: pub,
	}

	base := service.NewBaseServiceWithOptions("ui-api", nil,
		service.WithLogger(logger),
	)
	if deps.NATSClient != nil {
		base = service.NewBaseServiceWithOptions("ui-api", nil,
			service.WithNATS(deps.NATSClient),
			service.WithLogger(logger),
		)
	}

	return &UIAPIService{
		BaseService: base,
		component:   comp,
	}, nil
}

// Start starts the base service health monitoring and then the component.
func (s *UIAPIService) Start(ctx context.Context) error {
	if err := s.BaseService.Start(ctx); err != nil {
		return fmt.Errorf("ui-api: start base: %w", err)
	}
	if err := s.component.Start(ctx); err != nil {
		return fmt.Errorf("ui-api: start component: %w", err)
	}
	return nil
}

// Stop stops the component and then the base service.
func (s *UIAPIService) Stop(timeout time.Duration) error {
	if err := s.component.Stop(timeout); err != nil {
		return fmt.Errorf("ui-api: stop component: %w", err)
	}
	return s.BaseService.Stop(timeout)
}

// Health returns health status derived from the component state.
func (s *UIAPIService) Health() health.Status {
	ch := s.component.Health()
	if ch.Healthy {
		return health.NewHealthy("ui-api", "Component running")
	}
	return health.NewUnhealthy("ui-api", ch.Status)
}

// RegisterMetrics is a no-op for now.
func (s *UIAPIService) RegisterMetrics(_ metric.MetricsRegistrar) error {
	return nil
}

// RegisterHTTPHandlers delegates to the underlying component.
func (s *UIAPIService) RegisterHTTPHandlers(prefix string, mux *http.ServeMux) {
	s.component.RegisterHTTPHandlers(prefix, mux)
}

// OpenAPISpec returns the OpenAPI fragment for the ui-api service.
func (s *UIAPIService) OpenAPISpec() *service.OpenAPISpec {
	spec := service.NewOpenAPISpec()
	spec.AddTag("ui-api", "Semsage UI API — loop state, hierarchy, trajectory, SSE activity stream")

	spec.ResponseTypes = []reflect.Type{
		reflect.TypeOf(HealthResponse{}),
		reflect.TypeOf(LoopResponse{}),
		reflect.TypeOf(SignalResponse{}),
		reflect.TypeOf(ChildrenResponse{}),
		reflect.TypeOf(TreeResponse{}),
		reflect.TypeOf(TrajectoryResponse{}),
		reflect.TypeOf(ToolResponse{}),
		reflect.TypeOf(ChatResponse{}),
		reflect.TypeOf(ErrorResponse{}),
	}
	spec.RequestBodyTypes = []reflect.Type{
		reflect.TypeOf(SignalRequest{}),
		reflect.TypeOf(ChatRequest{}),
	}

	tags := []string{"ui-api"}

	spec.AddPath("/api/health", service.PathSpec{
		GET: &service.OperationSpec{
			Summary:   "System health",
			Responses: map[string]service.ResponseSpec{"200": {Description: "Health status", SchemaRef: "#/components/schemas/HealthResponse"}},
			Tags:      tags,
		},
	})
	spec.AddPath("/api/loops", service.PathSpec{
		GET: &service.OperationSpec{
			Summary: "List loops",
			Parameters: []service.ParameterSpec{
				{Name: "state", In: "query", Description: "Filter by loop state", Schema: service.Schema{Type: "string"}},
			},
			Responses: map[string]service.ResponseSpec{"200": {Description: "Loop list", SchemaRef: "#/components/schemas/LoopResponse", IsArray: true}},
			Tags:      tags,
		},
	})
	spec.AddPath("/api/loops/{id}", service.PathSpec{
		GET: &service.OperationSpec{
			Summary: "Get loop detail",
			Parameters: []service.ParameterSpec{
				{Name: "id", In: "path", Required: true, Schema: service.Schema{Type: "string"}},
			},
			Responses: map[string]service.ResponseSpec{"200": {Description: "Loop detail", SchemaRef: "#/components/schemas/LoopResponse"}},
			Tags:      tags,
		},
	})
	spec.AddPath("/api/loops/{id}/signal", service.PathSpec{
		POST: &service.OperationSpec{
			Summary:     "Send signal to loop",
			RequestBody: &service.RequestBodySpec{SchemaRef: "#/components/schemas/SignalRequest", Required: true},
			Responses:   map[string]service.ResponseSpec{"200": {Description: "Signal accepted", SchemaRef: "#/components/schemas/SignalResponse"}},
			Tags:        tags,
		},
	})
	spec.AddPath("/api/loops/{id}/children", service.PathSpec{
		GET: &service.OperationSpec{
			Summary:   "Get loop children",
			Responses: map[string]service.ResponseSpec{"200": {Description: "Children list", SchemaRef: "#/components/schemas/ChildrenResponse"}},
			Tags:      tags,
		},
	})
	spec.AddPath("/api/loops/{id}/tree", service.PathSpec{
		GET: &service.OperationSpec{
			Summary:   "Get loop subtree",
			Responses: map[string]service.ResponseSpec{"200": {Description: "Entity IDs", SchemaRef: "#/components/schemas/TreeResponse"}},
			Tags:      tags,
		},
	})
	spec.AddPath("/api/trajectory/loops/{id}", service.PathSpec{
		GET: &service.OperationSpec{
			Summary:   "Get trajectory for loop",
			Responses: map[string]service.ResponseSpec{"200": {Description: "Trajectory", SchemaRef: "#/components/schemas/TrajectoryResponse"}},
			Tags:      tags,
		},
	})
	spec.AddPath("/api/tools", service.PathSpec{
		GET: &service.OperationSpec{
			Summary:   "List registered tools",
			Responses: map[string]service.ResponseSpec{"200": {Description: "Tool list", SchemaRef: "#/components/schemas/ToolResponse", IsArray: true}},
			Tags:      tags,
		},
	})
	spec.AddPath("/api/chat", service.PathSpec{
		POST: &service.OperationSpec{
			Summary:     "Send chat message",
			RequestBody: &service.RequestBodySpec{SchemaRef: "#/components/schemas/ChatRequest", Required: true},
			Responses:   map[string]service.ResponseSpec{"200": {Description: "Chat accepted", SchemaRef: "#/components/schemas/ChatResponse"}},
			Tags:        tags,
		},
	})
	spec.AddPath("/api/activity", service.PathSpec{
		GET: &service.OperationSpec{
			Summary:   "SSE activity stream",
			Responses: map[string]service.ResponseSpec{"200": {Description: "Server-Sent Events stream"}},
			Tags:      tags,
		},
	})

	return spec
}

// Component returns the underlying ui-api Component for wiring graph helpers
// or other post-construction dependencies.
func (s *UIAPIService) Component() *Component {
	return s.component
}
