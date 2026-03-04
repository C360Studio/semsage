package spawn_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/c360studio/semstreams/agentic"
	"github.com/c360studio/semstreams/natsclient"

	"github.com/c360studio/semsage/tools/spawn"
)

// -- mock implementations --

// mockNATSClient records publish calls and allows tests to inject handler
// references so they can drive message delivery synchronously.
type mockNATSClient struct {
	mu           sync.Mutex
	published    []publishedMsg
	publishErr   error
	subscriptions []*capturedSubscription
}

type publishedMsg struct {
	subject string
	data    []byte
}

// capturedSubscription records a subscription and retains the handler so
// tests can fire messages directly.
type capturedSubscription struct {
	subject string
	handler func(context.Context, *nats.Msg)
	subErr  error // non-nil causes Subscribe to return this error
	unsub   bool  // true after Unsubscribe
}

func (s *capturedSubscription) fire(ctx context.Context, data []byte) {
	if s.handler != nil {
		s.handler(ctx, &nats.Msg{Subject: s.subject, Data: data})
	}
}

// stubSubscription wraps a capturedSubscription to satisfy the
// *natsclient.Subscription return type expectation via the NATSClient
// interface. Because natsclient.Subscription is a struct with an
// Unsubscribe method, we track state via the captured pointer.

// newMockNATSClient creates a mock with optional per-subscription error hooks.
func newMockNATSClient() *mockNATSClient {
	return &mockNATSClient{}
}

func (m *mockNATSClient) PublishToStream(_ context.Context, subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishErr != nil {
		return m.publishErr
	}
	m.published = append(m.published, publishedMsg{subject: subject, data: data})
	return nil
}

func (m *mockNATSClient) Subscribe(
	_ context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (*natsclient.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs := &capturedSubscription{subject: subject, handler: handler}
	m.subscriptions = append(m.subscriptions, cs)
	if cs.subErr != nil {
		return nil, cs.subErr
	}
	// natsclient.Subscription has no exported constructor; we cannot construct
	// one in test code because the sub field is unexported. Instead we return
	// nil here — Executor only calls Unsubscribe via defer, so a nil pointer
	// dereference is prevented by the Unsubscribe implementation on Subscription
	// checking s.sub == nil. We use a real subscription proxy below.
	return nil, nil
}

// subscriptionForSubject returns the capturedSubscription whose subject
// matches the given pattern, or nil if not found. Thread-safe.
func (m *mockNATSClient) subscriptionForSubject(subject string) *capturedSubscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.subscriptions {
		if s.subject == subject {
			return s
		}
	}
	return nil
}

// mockGraphHelper records RecordSpawn calls.
type mockGraphHelper struct {
	mu      sync.Mutex
	spawns  []spawnRecord
	spawnErr error
}

type spawnRecord struct {
	parentLoopID string
	childLoopID  string
	role         string
	model        string
}

func (g *mockGraphHelper) RecordSpawn(_ context.Context, parentLoopID, childLoopID, role, model string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.spawnErr != nil {
		return g.spawnErr
	}
	g.spawns = append(g.spawns, spawnRecord{
		parentLoopID: parentLoopID,
		childLoopID:  childLoopID,
		role:         role,
		model:        model,
	})
	return nil
}

// -- helpers --

// buildCompletedPayload constructs the JSON that the executor expects on
// agent.complete.<loopID>: a BaseMessage envelope with a LoopCompletedEvent
// payload.
func buildCompletedPayload(t *testing.T, loopID, taskID, result string) []byte {
	t.Helper()
	event := agentic.LoopCompletedEvent{
		LoopID:  loopID,
		TaskID:  taskID,
		Outcome: agentic.OutcomeSuccess,
		Result:  result,
	}
	return wrapPayload(t, event)
}

// buildFailedPayload constructs the JSON for a LoopFailedEvent envelope.
func buildFailedPayload(t *testing.T, loopID, taskID, reason, errMsg string) []byte {
	t.Helper()
	event := agentic.LoopFailedEvent{
		LoopID:  loopID,
		TaskID:  taskID,
		Outcome: agentic.OutcomeFailed,
		Reason:  reason,
		Error:   errMsg,
	}
	return wrapPayload(t, event)
}

// wrapPayload encodes a payload as a minimal BaseMessage JSON envelope that
// unmarshalPayload (the private helper inside executor.go) can decode. We
// only need the "payload" key because the executor reads only that field.
func wrapPayload(t *testing.T, payload any) []byte {
	t.Helper()
	inner, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("wrapPayload: marshal inner: %v", err)
	}
	envelope := map[string]json.RawMessage{
		"payload": json.RawMessage(inner),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("wrapPayload: marshal envelope: %v", err)
	}
	return data
}

// baseCall returns a minimal ToolCall for the spawn_agent tool.
func baseCall(prompt, role string) agentic.ToolCall {
	return agentic.ToolCall{
		ID:     "call-1",
		Name:   "spawn_agent",
		LoopID: "parent-loop",
		Arguments: map[string]any{
			"prompt":  prompt,
			"role":    role,
			"timeout": "100ms", // short timeout for tests
		},
	}
}

// -- tests --

func TestExecutor_ListTools(t *testing.T) {
	t.Parallel()

	e := spawn.NewExecutor(newMockNATSClient(), &mockGraphHelper{})
	tools := e.ListTools()

	if len(tools) != 1 {
		t.Fatalf("ListTools() returned %d tools, want 1", len(tools))
	}
	tool := tools[0]
	if tool.Name != "spawn_agent" {
		t.Errorf("tool name = %q, want %q", tool.Name, "spawn_agent")
	}
	if tool.Description == "" {
		t.Error("tool description must not be empty")
	}
	params, ok := tool.Parameters["required"].([]string)
	if !ok {
		t.Fatal("tool parameters missing 'required' slice")
	}
	requiredSet := make(map[string]bool, len(params))
	for _, p := range params {
		requiredSet[p] = true
	}
	if !requiredSet["prompt"] {
		t.Error("'prompt' must be in the required list")
	}
	if !requiredSet["role"] {
		t.Error("'role' must be in the required list")
	}
}

func TestExecutor_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		call           agentic.ToolCall
		publishErr     error
		graphErr       error
		noDefaultModel bool // skip WithDefaultModel option
		// drive drives the mock after Execute starts: it receives the mock and
		// the child loop ID extracted from the first subscription subject, then
		// fires a message to simulate the child outcome. A nil drive func means
		// the test waits for the timeout.
		drive       func(t *testing.T, m *mockNATSClient, childLoopID string)
		wantContent string
		wantErrMsg  string // non-empty portion that must appear in ToolResult.Error
		wantGoErr   bool   // true when Execute itself returns a non-nil error
	}{
		{
			name: "successful spawn returns child result",
			call: baseCall("write a hello world program", "developer"),
			drive: func(t *testing.T, m *mockNATSClient, childLoopID string) {
				t.Helper()
				sub := waitForSubscription(t, m, fmt.Sprintf("agent.complete.%s", childLoopID))
				sub.fire(context.Background(), buildCompletedPayload(t, childLoopID, "task-x", "Hello, World!"))
			},
			wantContent: "Hello, World!",
		},
		{
			name: "child failure returns error tool result",
			call: baseCall("do something", "executor"),
			drive: func(t *testing.T, m *mockNATSClient, childLoopID string) {
				t.Helper()
				sub := waitForSubscription(t, m, fmt.Sprintf("agent.failed.%s", childLoopID))
				sub.fire(context.Background(), buildFailedPayload(t, childLoopID, "task-y", "iteration limit", "max iterations reached"))
			},
			wantErrMsg: "iteration limit",
		},
		{
			name: "timeout returns error tool result",
			call: baseCall("slow task", "executor"),
			// no drive — child never responds, short timeout fires
			wantErrMsg: "timed out",
		},
		{
			name: "depth limit exceeded returns error immediately",
			call: agentic.ToolCall{
				ID:     "call-depth",
				Name:   "spawn_agent",
				LoopID: "parent-loop",
				Arguments: map[string]any{
					"prompt": "nested task",
					"role":   "executor",
				},
				Metadata: map[string]any{
					// With maxDepth=5 and current depth=4, next depth would be 5,
					// which equals maxDepth — so it must be rejected.
					"depth": float64(4),
				},
			},
			wantErrMsg: "depth limit reached",
		},
		{
			name: "context cancellation returns error tool result",
			call: baseCall("cancelled task", "executor"),
			drive: func(t *testing.T, m *mockNATSClient, childLoopID string) {
				t.Helper()
				// Subscriptions are established but we cancel the context from
				// the test goroutine before firing any event. The executor's
				// select on ctx.Done() should win.
			},
			wantErrMsg: "context cancelled",
		},
		{
			name: "missing prompt argument returns error tool result",
			call: agentic.ToolCall{
				ID:     "call-no-prompt",
				Name:   "spawn_agent",
				LoopID: "parent-loop",
				Arguments: map[string]any{
					"role": "executor",
				},
			},
			wantErrMsg: "'prompt' is required",
		},
		{
			name: "missing role argument returns error tool result",
			call: agentic.ToolCall{
				ID:     "call-no-role",
				Name:   "spawn_agent",
				LoopID: "parent-loop",
				Arguments: map[string]any{
					"prompt": "do something",
				},
			},
			wantErrMsg: "'role' is required",
		},
		{
			name:       "publish failure returns Go error",
			call:       baseCall("publish fails", "executor"),
			publishErr: errors.New("NATS: stream not found"),
			wantGoErr:  true,
		},
		{
			name:     "graph error is non-fatal, child result still returned",
			call:     baseCall("graph fails", "executor"),
			graphErr: errors.New("bucket unavailable"),
			drive: func(t *testing.T, m *mockNATSClient, childLoopID string) {
				t.Helper()
				sub := waitForSubscription(t, m, fmt.Sprintf("agent.complete.%s", childLoopID))
				sub.fire(context.Background(), buildCompletedPayload(t, childLoopID, "task-g", "graph-fail-result"))
			},
			wantContent: "graph-fail-result",
		},
		{
			name:           "no model and no default returns error",
			noDefaultModel: true,
			call: agentic.ToolCall{
				ID:     "call-no-model",
				Name:   "spawn_agent",
				LoopID: "parent-loop",
				Arguments: map[string]any{
					"prompt":  "some task",
					"role":    "developer",
					"timeout": "100ms",
				},
			},
			wantErrMsg: "no model specified",
		},
		{
			name: "default model used when model arg omitted",
			call: agentic.ToolCall{
				ID:     "call-default-model",
				Name:   "spawn_agent",
				LoopID: "parent-loop",
				Arguments: map[string]any{
					"prompt":  "some task",
					"role":    "developer",
					"timeout": "100ms",
				},
			},
			drive: func(t *testing.T, m *mockNATSClient, childLoopID string) {
				t.Helper()
				sub := waitForSubscription(t, m, fmt.Sprintf("agent.complete.%s", childLoopID))
				sub.fire(context.Background(), buildCompletedPayload(t, childLoopID, "task-z", "done"))
			},
			wantContent: "done",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockNATS := newMockNATSClient()
			mockNATS.publishErr = tc.publishErr

			mockGraph := &mockGraphHelper{spawnErr: tc.graphErr}

			opts := []spawn.Option{spawn.WithMaxDepth(5)}
			if !tc.noDefaultModel {
				opts = append(opts, spawn.WithDefaultModel("claude-3-5-sonnet"))
			}
			e := spawn.NewExecutor(mockNATS, mockGraph, opts...)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// For context-cancellation test: cancel shortly after Execute is
			// called. We do this before the goroutine so the cancel function is
			// in scope.
			var childLoopIDCh chan string
			if tc.drive != nil {
				childLoopIDCh = make(chan string, 1)
			}

			var (
				result agentic.ToolResult
				goErr  error
				wg     sync.WaitGroup
			)

			wg.Add(1)
			go func() {
				defer wg.Done()
				result, goErr = e.Execute(ctx, tc.call)
			}()

			// If the test case has a drive function, extract the child loop ID
			// from the first subscription (agent.complete.<childLoopID>) once
			// it appears, then invoke drive.
			if tc.drive != nil {
				go func() {
					if tc.name == "context cancellation returns error tool result" {
						// Wait until both subscriptions are registered — meaning
						// Execute has progressed past Subscribe calls and is
						// about to enter the select — then cancel the context.
						deadline := time.Now().Add(500 * time.Millisecond)
						for time.Now().Before(deadline) {
							mockNATS.mu.Lock()
							n := len(mockNATS.subscriptions)
							mockNATS.mu.Unlock()
							if n >= 2 {
								break
							}
							time.Sleep(1 * time.Millisecond)
						}
						cancel()
						return
					}
					childLoopID := extractChildLoopID(t, mockNATS)
					if childLoopIDCh != nil {
						childLoopIDCh <- childLoopID
					}
					tc.drive(t, mockNATS, childLoopID)
				}()
			}

			// Wait for Execute to return.
			wg.Wait()

			// Assertions.
			if tc.wantGoErr {
				if goErr == nil {
					t.Fatalf("expected Execute to return a Go error, got nil")
				}
				return
			}
			if goErr != nil {
				t.Fatalf("Execute returned unexpected Go error: %v", goErr)
			}

			if tc.wantContent != "" {
				if result.Content != tc.wantContent {
					t.Errorf("ToolResult.Content = %q, want %q", result.Content, tc.wantContent)
				}
				if result.Error != "" {
					t.Errorf("ToolResult.Error should be empty, got %q", result.Error)
				}
			}

			if tc.wantErrMsg != "" {
				if result.Error == "" {
					t.Fatalf("expected ToolResult.Error containing %q, got empty string", tc.wantErrMsg)
				}
				if !strings.Contains(result.Error, tc.wantErrMsg) {
					t.Errorf("ToolResult.Error = %q, want it to contain %q", result.Error, tc.wantErrMsg)
				}
			}

			// Verify CallID propagation for non-argument-error cases.
			if tc.call.ID != "" && !tc.wantGoErr {
				if result.CallID != tc.call.ID {
					t.Errorf("ToolResult.CallID = %q, want %q", result.CallID, tc.call.ID)
				}
			}
		})
	}
}

// waitForSubscription polls until the mock records a subscription on the
// given subject. It fails the test after a short deadline.
func waitForSubscription(t *testing.T, m *mockNATSClient, subject string) *capturedSubscription {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if sub := m.subscriptionForSubject(subject); sub != nil {
			return sub
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for subscription on %q", subject)
	return nil
}

// extractChildLoopID waits for at least one subscription to appear on the
// mock client and extracts the child loop ID from the subject
// "agent.complete.<childLoopID>".
func extractChildLoopID(t *testing.T, m *mockNATSClient) string {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		subs := m.subscriptions
		m.mu.Unlock()
		for _, s := range subs {
			var childLoopID string
			if n, _ := fmt.Sscanf(s.subject, "agent.complete.%s", &childLoopID); n == 1 {
				return childLoopID
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("timed out waiting for agent.complete subscription to extract child loop ID")
	return ""
}

// TestExecutor_PublishSubjectFormat verifies that the TaskMessage is published
// to the correct subject prefix.
func TestExecutor_PublishSubjectFormat(t *testing.T) {
	t.Parallel()

	mockNATS := newMockNATSClient()
	mockGraph := &mockGraphHelper{}

	e := spawn.NewExecutor(mockNATS, mockGraph,
		spawn.WithDefaultModel("gpt-4o"),
	)

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// We don't care about the result here, just that the publish happened.
		_, _ = e.Execute(ctx, agentic.ToolCall{
			ID:     "call-pub",
			Name:   "spawn_agent",
			LoopID: "parent-loop",
			Arguments: map[string]any{
				"prompt":  "check subject",
				"role":    "executor",
				"timeout": "50ms",
			},
		})
	}()

	wg.Wait()

	mockNATS.mu.Lock()
	published := mockNATS.published
	mockNATS.mu.Unlock()

	if len(published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(published))
	}
	subject := published[0].subject
	var taskID string
	if n, _ := fmt.Sscanf(subject, "agent.task.%s", &taskID); n != 1 || taskID == "" {
		t.Errorf("published subject %q does not match agent.task.<taskID>", subject)
	}
}

// TestExecutor_ChildMetadataInResult verifies that the successful ToolResult
// includes the child_loop_id and task_id in its metadata.
func TestExecutor_ChildMetadataInResult(t *testing.T) {
	t.Parallel()

	mockNATS := newMockNATSClient()
	mockGraph := &mockGraphHelper{}
	e := spawn.NewExecutor(mockNATS, mockGraph,
		spawn.WithDefaultModel("gpt-4o"),
	)

	ctx := context.Background()

	resultCh := make(chan agentic.ToolResult, 1)
	go func() {
		result, _ := e.Execute(ctx, agentic.ToolCall{
			ID:     "call-meta",
			Name:   "spawn_agent",
			LoopID: "parent-loop",
			Arguments: map[string]any{
				"prompt":  "check metadata",
				"role":    "developer",
				"timeout": "500ms",
			},
		})
		resultCh <- result
	}()

	childLoopID := extractChildLoopID(t, mockNATS)
	sub := waitForSubscription(t, mockNATS, fmt.Sprintf("agent.complete.%s", childLoopID))
	sub.fire(ctx, buildCompletedPayload(t, childLoopID, "meta-task", "meta result"))

	result := <-resultCh
	if result.Error != "" {
		t.Fatalf("unexpected error in result: %s", result.Error)
	}

	if result.Metadata == nil {
		t.Fatal("ToolResult.Metadata is nil, expected child_loop_id and task_id")
	}
	if _, ok := result.Metadata["child_loop_id"]; !ok {
		t.Error("ToolResult.Metadata missing 'child_loop_id'")
	}
	if _, ok := result.Metadata["task_id"]; !ok {
		t.Error("ToolResult.Metadata missing 'task_id'")
	}
	if result.Metadata["child_loop_id"] != childLoopID {
		t.Errorf("child_loop_id = %v, want %q", result.Metadata["child_loop_id"], childLoopID)
	}
}
