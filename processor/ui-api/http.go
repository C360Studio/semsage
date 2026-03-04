package uiapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/c360studio/semstreams/agentic"
	"github.com/c360studio/semstreams/message"
	agentictools "github.com/c360studio/semstreams/processor/agentic-tools"
	"github.com/google/uuid"
)

// Compile-time check that Component satisfies the HTTP handler interface.
var _ interface {
	RegisterHTTPHandlers(prefix string, mux *http.ServeMux)
} = (*Component)(nil)

// RegisterHTTPHandlers mounts all ui-api routes onto mux under the given prefix.
// The prefix should be a path without a trailing slash (e.g. "/api"); routes
// are registered using Go 1.22+ method+path patterns.
func (c *Component) RegisterHTTPHandlers(prefix string, mux *http.ServeMux) {
	p := strings.TrimSuffix(prefix, "/")

	mux.HandleFunc("GET "+p+"/api/health", c.handleHealth)
	mux.HandleFunc("GET "+p+"/api/loops", c.handleListLoops)
	mux.HandleFunc("GET "+p+"/api/loops/{id}", c.handleGetLoop)
	mux.HandleFunc("POST "+p+"/api/loops/{id}/signal", c.handleLoopSignal)
	mux.HandleFunc("GET "+p+"/api/loops/{id}/children", c.handleLoopChildren)
	mux.HandleFunc("GET "+p+"/api/loops/{id}/tree", c.handleLoopTree)
	mux.HandleFunc("GET "+p+"/api/trajectory/loops/{id}", c.handleGetTrajectory)
	mux.HandleFunc("GET "+p+"/api/trajectory/loops/{id}/calls/{req_id}", c.handleGetCall)
	mux.HandleFunc("GET "+p+"/api/tools", c.handleListTools)
	mux.HandleFunc("POST "+p+"/api/chat", c.handleChat)
	mux.HandleFunc("GET "+p+"/api/activity", c.handleActivityStream)
	mux.HandleFunc("GET "+p+"/graphql/", c.handleGraphQL)

	c.logger.Info("ui-api HTTP handlers registered", slog.String("prefix", p))
}

// handleHealth returns system health including NATS / KV connectivity.
func (c *Component) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	components := make(map[string]string)

	// Probe the AGENT_LOOPS bucket.
	if _, err := c.getLoopsBucket(ctx); err != nil {
		components["agent_loops_kv"] = "unavailable"
	} else {
		components["agent_loops_kv"] = "ok"
	}

	status := "ok"
	httpStatus := http.StatusOK
	for _, v := range components {
		if v != "ok" {
			status = "degraded"
			break
		}
	}

	writeJSON(w, httpStatus, HealthResponse{
		Status:     status,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Components: components,
	})
}

// handleListLoops lists all loops stored in the AGENT_LOOPS KV bucket.
// Optional query param: ?state=<state> to filter by loop state.
func (c *Component) handleListLoops(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bucket, err := c.getLoopsBucket(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "AGENT_LOOPS bucket unavailable", err)
		return
	}

	keys, err := bucket.Keys(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list loop keys", err)
		return
	}

	stateFilter := r.URL.Query().Get("state")

	maxLoops := c.config.MaxLoopsPerPage
	if maxLoops <= 0 {
		maxLoops = 200
	}

	loops := make([]LoopResponse, 0, len(keys))
	for _, key := range keys {
		entry, getErr := bucket.Get(ctx, key)
		if getErr != nil {
			c.logger.Warn("ui-api: failed to get loop entry",
				slog.String("key", key),
				slog.String("error", getErr.Error()),
			)
			continue
		}
		lr, parseErr := parseLoopEntity(entry.Value())
		if parseErr != nil {
			c.logger.Warn("ui-api: failed to parse loop entity",
				slog.String("key", key),
				slog.String("error", parseErr.Error()),
			)
			continue
		}
		if stateFilter != "" && lr.State != stateFilter {
			continue
		}
		loops = append(loops, lr)
		if len(loops) >= maxLoops {
			break
		}
	}

	writeJSON(w, http.StatusOK, loops)
}

// handleGetLoop returns the detail of a single loop by ID.
func (c *Component) handleGetLoop(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	loopID := r.PathValue("id")
	if loopID == "" {
		writeError(w, http.StatusBadRequest, "loop id required", nil)
		return
	}

	bucket, err := c.getLoopsBucket(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "AGENT_LOOPS bucket unavailable", err)
		return
	}

	entry, err := bucket.Get(ctx, loopID)
	if err != nil {
		if isKeyNotFound(err) {
			writeError(w, http.StatusNotFound, "loop not found", nil)
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get loop", err)
		}
		return
	}

	lr, err := parseLoopEntity(entry.Value())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse loop", err)
		return
	}

	writeJSON(w, http.StatusOK, lr)
}

// isKeyNotFound returns true if the error indicates a missing KV key.
func isKeyNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "key not found")
}

// handleLoopSignal publishes a pause/resume/cancel signal for a loop.
func (c *Component) handleLoopSignal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	loopID := r.PathValue("id")
	if loopID == "" {
		writeError(w, http.StatusBadRequest, "loop id required", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, c.config.MaxBodyBytes)
	var req SignalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	switch req.Type {
	case agentic.SignalPause, agentic.SignalResume, agentic.SignalCancel:
		// Valid signal types.
	default:
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("invalid signal type %q: must be pause, resume, or cancel", req.Type), nil)
		return
	}

	if c.natsClient == nil {
		writeError(w, http.StatusServiceUnavailable, "NATS not configured", nil)
		return
	}

	signal := agentic.UserSignal{
		SignalID:    uuid.New().String(),
		Type:        req.Type,
		LoopID:      loopID,
		UserID:      "ui",
		ChannelType: "http",
		ChannelID:   "ui-api",
		Timestamp:   time.Now(),
	}
	if req.Reason != "" {
		signal.Payload = map[string]string{"reason": req.Reason}
	}

	baseMsg := message.NewBaseMessage(signal.Schema(), &signal, "ui-api")
	data, err := json.Marshal(baseMsg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal signal", err)
		return
	}

	subject := fmt.Sprintf("%s.%s", c.config.SignalSubject, loopID)
	if err := c.natsClient.PublishToStream(ctx, subject, data); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to publish signal", err)
		return
	}

	c.logger.Info("ui-api: signal sent",
		slog.String("loop_id", loopID),
		slog.String("signal", req.Type),
	)

	writeJSON(w, http.StatusOK, SignalResponse{
		LoopID:    loopID,
		Signal:    req.Type,
		Accepted:  true,
		Message:   fmt.Sprintf("signal %q sent to loop %s", req.Type, loopID),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleLoopChildren returns the direct children of a loop via the graph layer.
func (c *Component) handleLoopChildren(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	loopID := r.PathValue("id")
	if loopID == "" {
		writeError(w, http.StatusBadRequest, "loop id required", nil)
		return
	}

	c.mu.RLock()
	g := c.graphHelper
	c.mu.RUnlock()

	if g == nil {
		writeError(w, http.StatusServiceUnavailable, "graph layer not configured", nil)
		return
	}

	children, err := g.GetChildren(ctx, loopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query children", err)
		return
	}

	writeJSON(w, http.StatusOK, ChildrenResponse{
		LoopID:   loopID,
		Children: children,
	})
}

// handleLoopTree returns all loop entity IDs reachable from a root loop.
func (c *Component) handleLoopTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	loopID := r.PathValue("id")
	if loopID == "" {
		writeError(w, http.StatusBadRequest, "loop id required", nil)
		return
	}

	c.mu.RLock()
	g := c.graphHelper
	c.mu.RUnlock()

	if g == nil {
		writeError(w, http.StatusServiceUnavailable, "graph layer not configured", nil)
		return
	}

	const defaultMaxDepth = 10
	entityIDs, err := g.GetTree(ctx, loopID, defaultMaxDepth)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query tree", err)
		return
	}

	writeJSON(w, http.StatusOK, TreeResponse{
		RootLoopID: loopID,
		EntityIDs:  entityIDs,
	})
}

// handleGetTrajectory returns the trajectory summary for a loop.
// Trajectory data is read from the AGENT_LOOPS KV bucket where model and
// tool call records are stored alongside loop state.
// Pass ?format=json to include the full entry list.
func (c *Component) handleGetTrajectory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	loopID := r.PathValue("id")
	if loopID == "" {
		writeError(w, http.StatusBadRequest, "loop id required", nil)
		return
	}

	bucket, err := c.getLoopsBucket(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "AGENT_LOOPS bucket unavailable", err)
		return
	}

	entry, err := bucket.Get(ctx, loopID)
	if err != nil {
		writeError(w, http.StatusNotFound, "loop not found", err)
		return
	}

	var loop agentic.LoopEntity
	if err := json.Unmarshal(entry.Value(), &loop); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse loop entity", err)
		return
	}

	var startedAt *time.Time
	if !loop.StartedAt.IsZero() {
		t := loop.StartedAt
		startedAt = &t
	}

	var endedAt *time.Time
	if !loop.CompletedAt.IsZero() {
		t := loop.CompletedAt
		endedAt = &t
	}

	var durationMs int64
	if startedAt != nil && endedAt != nil {
		durationMs = endedAt.Sub(*startedAt).Milliseconds()
	}

	resp := TrajectoryResponse{
		LoopID:    loopID,
		Status:    string(loop.State),
		StartedAt: startedAt,
		EndedAt:   endedAt,
		DurationMs: durationMs,
	}

	// When ?format=json is requested, include the full entry list.
	// Trajectory entries are not yet stored in the KV bucket; this is a
	// placeholder until a dedicated trajectory store is wired.
	if r.URL.Query().Get("format") == "json" {
		resp.Entries = []TrajectoryEntry{}
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleGetCall returns the full LLM call record for a given request ID.
// Not yet implemented — trajectory storage is a follow-up.
func (c *Component) handleGetCall(w http.ResponseWriter, r *http.Request) {
	reqID := r.PathValue("req_id")
	if reqID == "" {
		writeError(w, http.StatusBadRequest, "req_id required", nil)
		return
	}
	writeError(w, http.StatusNotImplemented, "trajectory call detail not yet implemented", nil)
}

// handleListTools lists the tools registered in the global agentic-tools registry.
// Optional query param: ?root_loop_id=<id> (reserved for future scoping; currently ignored).
func (c *Component) handleListTools(w http.ResponseWriter, r *http.Request) {
	defs := agentictools.ListRegisteredTools()

	tools := make([]ToolResponse, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, ToolResponse{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  d.Parameters,
			RootLoopID:  r.URL.Query().Get("root_loop_id"),
		})
	}

	writeJSON(w, http.StatusOK, tools)
}

// handleChat dispatches a user chat message as an agentic TaskMessage.
func (c *Component) handleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, c.config.MaxBodyBytes)
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required", nil)
		return
	}

	if c.natsClient == nil {
		writeError(w, http.StatusServiceUnavailable, "NATS not configured", nil)
		return
	}

	// Apply defaults.
	userID := req.UserID
	if userID == "" {
		userID = "ui"
	}
	channelID := req.ChannelID
	if channelID == "" {
		channelID = fmt.Sprintf("http-%d", time.Now().UnixNano())
	}

	taskID := uuid.New().String()

	task := agentic.TaskMessage{
		TaskID:      taskID,
		Role:        "orchestrator",
		Model:       c.config.DefaultModel,
		Prompt:      req.Content,
		ChannelType: "http",
		ChannelID:   channelID,
		UserID:      userID,
	}

	baseMsg := message.NewBaseMessage(task.Schema(), &task, "ui-api")
	data, err := json.Marshal(baseMsg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal task", err)
		return
	}

	subject := fmt.Sprintf("%s.%s", c.config.DispatchSubject, taskID)
	if err := c.natsClient.PublishToStream(ctx, subject, data); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to dispatch task", err)
		return
	}

	c.logger.Info("ui-api: chat task dispatched",
		slog.String("task_id", taskID),
		slog.String("user_id", userID),
	)

	writeJSON(w, http.StatusOK, ChatResponse{
		MessageID: taskID,
		Content:   req.Content,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGraphQL proxies requests to the semstreams GraphQL gateway.
// Returns 501 when GraphQLProxyURL is not configured.
func (c *Component) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	proxy := c.graphQLProxy
	c.mu.RUnlock()

	if proxy == nil {
		writeError(w, http.StatusNotImplemented, "GraphQL proxy not configured", nil)
		return
	}

	proxy.ServeHTTP(w, r)
}

// --- helpers ---

// parseLoopEntity decodes raw KV entry bytes into a LoopResponse.
func parseLoopEntity(data []byte) (LoopResponse, error) {
	var entity agentic.LoopEntity
	if err := json.Unmarshal(data, &entity); err != nil {
		return LoopResponse{}, fmt.Errorf("ui-api: unmarshal loop entity: %w", err)
	}

	lr := LoopResponse{
		ID:            entity.ID,
		TaskID:        entity.TaskID,
		State:         string(entity.State),
		Role:          entity.Role,
		Model:         entity.Model,
		Iterations:    entity.Iterations,
		MaxIterations: entity.MaxIterations,
		Depth:         entity.Depth,
		MaxDepth:      entity.MaxDepth,
		ParentLoopID:  entity.ParentLoopID,
		Outcome:       entity.Outcome,
		Error:         entity.Error,
	}
	if !entity.StartedAt.IsZero() {
		t := entity.StartedAt
		lr.StartedAt = &t
	}
	if !entity.CompletedAt.IsZero() {
		t := entity.CompletedAt
		lr.CompletedAt = &t
	}
	return lr, nil
}

// writeJSON encodes v as JSON and writes it to w with the given HTTP status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status is already written; we can only log.
		slog.Default().Error("ui-api: failed to encode JSON response", "error", err)
	}
}

// writeError writes a standard ErrorResponse body. The cause is logged
// server-side but NOT exposed to the client to avoid leaking internals.
func writeError(w http.ResponseWriter, status int, msg string, cause error) {
	if cause != nil {
		slog.Default().Error("ui-api: request error",
			"status", status,
			"message", msg,
			"error", cause,
		)
	}
	writeJSON(w, status, ErrorResponse{Error: msg})
}
