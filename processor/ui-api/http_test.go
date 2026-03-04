package uiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// --- fakes ---

// fakeMsg records a single PublishToStream call.
type fakeMsg struct {
	subject string
	data    []byte
}

// fakeNATS is a natsPublisher that records published messages and optionally
// returns a provided KV bucket.
type fakeNATS struct {
	published []fakeMsg
	kv        jetstream.KeyValue
	kvErr     error
}

func (f *fakeNATS) PublishToStream(_ context.Context, subject string, data []byte) error {
	f.published = append(f.published, fakeMsg{subject: subject, data: data})
	return nil
}

func (f *fakeNATS) GetKeyValueBucket(_ context.Context, _ string) (jetstream.KeyValue, error) {
	if f.kvErr != nil {
		return nil, f.kvErr
	}
	if f.kv == nil {
		return nil, fmt.Errorf("no KV bucket configured in fake")
	}
	return f.kv, nil
}

// fakeGraphQuerier returns canned results for graph queries.
type fakeGraphQuerier struct {
	children map[string][]string
	tree     map[string][]string
	status   map[string]string
	err      error
}

func (f *fakeGraphQuerier) GetChildren(_ context.Context, loopID string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.children[loopID], nil
}

func (f *fakeGraphQuerier) GetTree(_ context.Context, rootLoopID string, _ int) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tree[rootLoopID], nil
}

func (f *fakeGraphQuerier) GetStatus(_ context.Context, loopID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.status[loopID], nil
}

// --- helpers ---

// newTestComponent constructs a Component wired for unit tests.
func newTestComponent(nats natsPublisher) *Component {
	c := &Component{
		name:       "ui-api-test",
		config:     DefaultConfig(),
		logger:     slog.Default(),
		natsClient: nats,
	}
	return c
}

// newMux builds a ServeMux with all routes registered.
func newMux(c *Component) *http.ServeMux {
	mux := http.NewServeMux()
	c.RegisterHTTPHandlers("", mux)
	return mux
}

// req fires a request against handler and returns the recorder.
func req(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	r := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// --- tests ---

func TestHandleHealth_NATSAbsent(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/health", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "degraded" {
		t.Errorf("expected degraded status, got %q", resp.Status)
	}
}

func TestHandleListLoops_BucketUnavailable(t *testing.T) {
	fake := &fakeNATS{} // no KV configured → returns error
	c := newTestComponent(fake)
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/loops", "")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleGetLoop_BucketUnavailable(t *testing.T) {
	fake := &fakeNATS{}
	c := newTestComponent(fake)
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/loops/abc123", "")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleGetTrajectory_BucketUnavailable(t *testing.T) {
	fake := &fakeNATS{}
	c := newTestComponent(fake)
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/trajectory/loops/abc123", "")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleGetCall_MissingReqID(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	// req_id is present in the URL but handler returns 501 (not yet implemented).
	w := req(t, mux, "GET", "/api/trajectory/loops/abc123/calls/req-001", "")

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestHandleLoopSignal_NATSAbsent(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	body := `{"type":"pause"}`
	w := req(t, mux, "POST", "/api/loops/abc123/signal", body)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleLoopSignal_InvalidType(t *testing.T) {
	fake := &fakeNATS{}
	c := newTestComponent(fake)
	mux := newMux(c)
	body := `{"type":"delete"}`
	w := req(t, mux, "POST", "/api/loops/abc123/signal", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleLoopSignal_PublishesMessage(t *testing.T) {
	fake := &fakeNATS{}
	c := newTestComponent(fake)
	mux := newMux(c)

	for _, sigType := range []string{"pause", "resume", "cancel"} {
		fake.published = nil
		body := `{"type":"` + sigType + `","reason":"test"}`
		w := req(t, mux, "POST", "/api/loops/loop-001/signal", body)

		if w.Code != http.StatusOK {
			t.Fatalf("signal %q: expected 200, got %d: %s", sigType, w.Code, w.Body.String())
		}
		if len(fake.published) != 1 {
			t.Fatalf("signal %q: expected 1 published message, got %d", sigType, len(fake.published))
		}
		if !strings.HasPrefix(fake.published[0].subject, "agent.signal.") {
			t.Errorf("signal %q: unexpected subject %q", sigType, fake.published[0].subject)
		}

		var resp SignalResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("signal %q: decode response: %v", sigType, err)
		}
		if !resp.Accepted {
			t.Errorf("signal %q: expected accepted=true", sigType)
		}
	}
}

func TestHandleLoopChildren_GraphAbsent(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/loops/loop-001/children", "")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleLoopChildren_ReturnsChildren(t *testing.T) {
	c := newTestComponent(nil)
	c.graphHelper = &fakeGraphQuerier{
		children: map[string][]string{"loop-001": {"loop-002", "loop-003"}},
	}
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/loops/loop-001/children", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ChildrenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.LoopID != "loop-001" {
		t.Errorf("expected loop_id=loop-001, got %q", resp.LoopID)
	}
	if len(resp.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(resp.Children))
	}
}

func TestHandleLoopTree_GraphAbsent(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/loops/loop-001/tree", "")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleLoopTree_ReturnsEntityIDs(t *testing.T) {
	c := newTestComponent(nil)
	c.graphHelper = &fakeGraphQuerier{
		tree: map[string][]string{
			"loop-001": {"entity-1", "entity-2"},
		},
	}
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/loops/loop-001/tree", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp TreeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.EntityIDs) != 2 {
		t.Errorf("expected 2 entity IDs, got %d", len(resp.EntityIDs))
	}
}

func TestHandleListTools(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	w := req(t, mux, "GET", "/api/tools", "")

	// Even with no tools registered, the endpoint should return 200 + empty array.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Response must be valid JSON array.
	var tools []ToolResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tools); err != nil {
		t.Fatalf("decode tools response: %v", err)
	}
}

func TestHandleChat_ContentRequired(t *testing.T) {
	fake := &fakeNATS{}
	c := newTestComponent(fake)
	mux := newMux(c)
	w := req(t, mux, "POST", "/api/chat", `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleChat_NATSAbsent(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	w := req(t, mux, "POST", "/api/chat", `{"content":"hello"}`)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleChat_DispatchesTask(t *testing.T) {
	fake := &fakeNATS{}
	c := newTestComponent(fake)
	mux := newMux(c)
	w := req(t, mux, "POST", "/api/chat", `{"content":"hello semsage","user_id":"u1"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(fake.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(fake.published))
	}
	if !strings.HasPrefix(fake.published[0].subject, "agent.task.") {
		t.Errorf("unexpected subject %q", fake.published[0].subject)
	}

	var resp ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MessageID == "" {
		t.Error("expected non-empty message_id")
	}
}

func TestHandleGraphQL_NotConfigured(t *testing.T) {
	c := newTestComponent(nil)
	mux := newMux(c)
	w := req(t, mux, "GET", "/graphql/", "")

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestParseLoopEntity_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	entity := struct {
		ID            string    `json:"id"`
		TaskID        string    `json:"task_id"`
		State         string    `json:"state"`
		Role          string    `json:"role"`
		Model         string    `json:"model"`
		Iterations    int       `json:"iterations"`
		MaxIterations int       `json:"max_iterations"`
		Depth         int       `json:"depth"`
		MaxDepth      int       `json:"max_depth"`
		ParentLoopID  string    `json:"parent_loop_id"`
		StartedAt     time.Time `json:"started_at"`
		Outcome       string    `json:"outcome"`
		Error         string    `json:"error"`
	}{
		ID:            "loop-abc",
		TaskID:        "task-xyz",
		State:         "executing",
		Role:          "orchestrator",
		Model:         "claude-sonnet-4-20250514",
		Iterations:    3,
		MaxIterations: 20,
		Depth:         1,
		MaxDepth:      5,
		ParentLoopID:  "loop-parent",
		StartedAt:     now,
		Outcome:       "",
		Error:         "",
	}

	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("marshal entity: %v", err)
	}

	lr, err := parseLoopEntity(data)
	if err != nil {
		t.Fatalf("parseLoopEntity: %v", err)
	}

	if lr.ID != "loop-abc" {
		t.Errorf("ID: got %q", lr.ID)
	}
	if lr.State != "executing" {
		t.Errorf("State: got %q", lr.State)
	}
	if lr.Depth != 1 {
		t.Errorf("Depth: got %d", lr.Depth)
	}
	if lr.StartedAt == nil {
		t.Error("StartedAt should not be nil")
	}
}

func TestParseLoopEntity_InvalidJSON(t *testing.T) {
	_, err := parseLoopEntity([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %q", ct)
	}
	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key=value, got %q", result["key"])
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		msg    string
		cause  error
	}{
		{"no cause", http.StatusBadRequest, "bad input", nil},
		{"with cause", http.StatusInternalServerError, "internal", fmt.Errorf("disk full")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, tt.status, tt.msg, tt.cause)

			if w.Code != tt.status {
				t.Errorf("expected %d, got %d", tt.status, w.Code)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Error != tt.msg {
				t.Errorf("error field: got %q, want %q", resp.Error, tt.msg)
			}
			// Cause is logged server-side, never leaked to client.
			if resp.Details != "" {
				t.Errorf("expected empty details (cause not exposed), got %q", resp.Details)
			}
		})
	}
}

func TestKvOpToEventType(t *testing.T) {
	tests := []struct {
		op       jetstream.KeyValueOp
		revision uint64
		want     string
	}{
		{jetstream.KeyValuePut, 1, "loop_created"},
		{jetstream.KeyValuePut, 2, "loop_updated"},
		{jetstream.KeyValuePut, 99, "loop_updated"},
		{jetstream.KeyValueDelete, 5, "loop_deleted"},
	}

	for _, tt := range tests {
		got := kvOpToEventType(tt.op, tt.revision)
		if got != tt.want {
			t.Errorf("kvOpToEventType(%v, %d) = %q, want %q", tt.op, tt.revision, got, tt.want)
		}
	}
}
