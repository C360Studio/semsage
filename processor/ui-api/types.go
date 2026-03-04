// Package uiapi provides HTTP endpoints for the Semsage UI.
// It reads loop state from the AGENT_LOOPS KV bucket, queries agent hierarchy
// via the graph layer, and streams real-time activity over Server-Sent Events.
package uiapi

import (
	"encoding/json"
	"time"
)

// LoopResponse is the JSON shape returned for a single loop.
// Fields mirror agentic.LoopEntity with UI-friendly additions.
type LoopResponse struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id,omitempty"`
	State         string     `json:"state"`
	Role          string     `json:"role,omitempty"`
	Model         string     `json:"model,omitempty"`
	Iterations    int        `json:"iterations"`
	MaxIterations int        `json:"max_iterations"`
	Depth         int        `json:"depth"`
	MaxDepth      int        `json:"max_depth,omitempty"`
	ParentLoopID  string     `json:"parent_loop_id,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Outcome       string     `json:"outcome,omitempty"`
	Error         string     `json:"error,omitempty"`
}

// ChildrenResponse is the JSON shape returned for loop children queries.
type ChildrenResponse struct {
	LoopID   string   `json:"loop_id"`
	Children []string `json:"children"`
}

// TreeResponse is the JSON shape returned for full agent tree queries.
type TreeResponse struct {
	RootLoopID string   `json:"root_loop_id"`
	EntityIDs  []string `json:"entity_ids"`
}

// SignalRequest is the request body for POST /api/loops/{id}/signal.
type SignalRequest struct {
	// Type is the signal type: pause, resume, or cancel.
	Type string `json:"type"`
	// Reason is an optional human-readable explanation.
	Reason string `json:"reason,omitempty"`
}

// SignalResponse is the response for a successful signal submission.
type SignalResponse struct {
	LoopID    string `json:"loop_id"`
	Signal    string `json:"signal"`
	Accepted  bool   `json:"accepted"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// TrajectoryEntry is a single step in a loop's trajectory, covering both model
// calls and tool calls. The Type field discriminates the entry kind.
type TrajectoryEntry struct {
	Type          string    `json:"type"` // "model_call" or "tool_call"
	Timestamp     time.Time `json:"timestamp"`
	DurationMs    int64     `json:"duration_ms,omitempty"`
	RequestID     string    `json:"request_id,omitempty"`
	Model         string    `json:"model,omitempty"`
	TokensIn      int       `json:"tokens_in,omitempty"`
	TokensOut     int       `json:"tokens_out,omitempty"`
	FinishReason  string    `json:"finish_reason,omitempty"`
	Error         string    `json:"error,omitempty"`
	ToolName      string    `json:"tool_name,omitempty"`
	Status        string    `json:"status,omitempty"`
	ResultPreview string    `json:"result_preview,omitempty"`
}

// TrajectoryResponse is the aggregated trajectory for a loop.
// Entries is populated only when ?format=json is specified.
type TrajectoryResponse struct {
	LoopID     string            `json:"loop_id"`
	Steps      int               `json:"steps"`
	ModelCalls int               `json:"model_calls"`
	ToolCalls  int               `json:"tool_calls"`
	TokensIn   int               `json:"tokens_in"`
	TokensOut  int               `json:"tokens_out"`
	DurationMs int64             `json:"duration_ms"`
	Status     string            `json:"status,omitempty"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	EndedAt    *time.Time        `json:"ended_at,omitempty"`
	Entries    []TrajectoryEntry `json:"entries,omitempty"`
}

// ToolResponse represents a single dynamically-created tool visible to the UI.
type ToolResponse struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	RootLoopID  string         `json:"root_loop_id,omitempty"`
}

// ChatRequest is the request body for POST /api/chat.
type ChatRequest struct {
	Content string `json:"content"`
	// UserID and ChannelID are optional; defaults are applied when absent.
	UserID    string `json:"user_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

// ChatResponse is the response for a successful chat submission.
type ChatResponse struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// HealthResponse is the response body for GET /api/health.
type HealthResponse struct {
	Status     string            `json:"status"`
	Timestamp  string            `json:"timestamp"`
	Components map[string]string `json:"components,omitempty"`
}

// ErrorResponse is the standard error body for all 4xx/5xx responses.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// ActivityEvent is streamed over SSE for GET /api/activity.
type ActivityEvent struct {
	// Type is one of: loop_created, loop_updated, loop_deleted.
	Type      string          `json:"type"`
	LoopID    string          `json:"loop_id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}
