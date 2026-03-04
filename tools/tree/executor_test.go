package tree_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/c360studio/semstreams/agentic"

	"github.com/c360studio/semsage/tools/tree"
)

// -- mock --

// mockGraph is a test double for the graphQuerier interface.
type mockGraph struct {
	children    []string
	treeIDs     []string
	status      string
	childrenErr error
	treeErr     error
	statusErr   error

	// capture arguments for assertion
	lastGetChildrenLoopID string
	lastGetTreeLoopID     string
	lastGetTreeMaxDepth   int
	lastGetStatusLoopID   string
}

func (m *mockGraph) GetChildren(_ context.Context, loopID string) ([]string, error) {
	m.lastGetChildrenLoopID = loopID
	if m.childrenErr != nil {
		return nil, m.childrenErr
	}
	if m.children == nil {
		return []string{}, nil
	}
	return m.children, nil
}

func (m *mockGraph) GetTree(_ context.Context, rootLoopID string, maxDepth int) ([]string, error) {
	m.lastGetTreeLoopID = rootLoopID
	m.lastGetTreeMaxDepth = maxDepth
	if m.treeErr != nil {
		return nil, m.treeErr
	}
	if m.treeIDs == nil {
		return []string{}, nil
	}
	return m.treeIDs, nil
}

func (m *mockGraph) GetStatus(_ context.Context, loopID string) (string, error) {
	m.lastGetStatusLoopID = loopID
	if m.statusErr != nil {
		return "", m.statusErr
	}
	return m.status, nil
}

// -- helpers --

func makeCall(id, loopID, traceID string, args map[string]any) agentic.ToolCall {
	return agentic.ToolCall{
		ID:        id,
		Name:      "query_agent_tree",
		Arguments: args,
		LoopID:    loopID,
		TraceID:   traceID,
	}
}

func mustUnmarshalStrings(t *testing.T, content string) []string {
	t.Helper()
	var result []string
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		t.Fatalf("unmarshal []string from %q: %v", content, err)
	}
	return result
}

// -- tests --

func TestExecutor_GetChildren_ReturnsChildIDs(t *testing.T) {
	t.Parallel()

	mock := &mockGraph{children: []string{"child-a", "child-b"}}
	exec := tree.NewExecutor(mock)

	call := makeCall("call-1", "loop-parent", "trace-1", map[string]any{
		"operation": "get_children",
		"loop_id":   "loop-parent",
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("Execute() result.Error = %q, want empty", result.Error)
	}
	if result.CallID != "call-1" {
		t.Errorf("CallID = %q, want %q", result.CallID, "call-1")
	}
	if mock.lastGetChildrenLoopID != "loop-parent" {
		t.Errorf("GetChildren called with loopID = %q, want %q", mock.lastGetChildrenLoopID, "loop-parent")
	}

	got := mustUnmarshalStrings(t, result.Content)
	if len(got) != 2 || got[0] != "child-a" || got[1] != "child-b" {
		t.Errorf("Content = %v, want [child-a child-b]", got)
	}
}

func TestExecutor_GetChildren_NoChildren_ReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	mock := &mockGraph{} // nil children → returns []string{}
	exec := tree.NewExecutor(mock)

	call := makeCall("call-2", "loop-1", "", map[string]any{
		"operation": "get_children",
		"loop_id":   "loop-1",
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("Execute() result.Error = %q, want empty", result.Error)
	}

	got := mustUnmarshalStrings(t, result.Content)
	if len(got) != 0 {
		t.Errorf("Content = %v, want empty array", got)
	}
}

func TestExecutor_GetTree_ReturnsAllEntityIDs(t *testing.T) {
	t.Parallel()

	wantIDs := []string{
		"semsage.default.agentic.orchestrator.loop.root",
		"semsage.default.agentic.orchestrator.loop.child-1",
	}
	mock := &mockGraph{treeIDs: wantIDs}
	exec := tree.NewExecutor(mock)

	call := makeCall("call-3", "loop-root", "trace-3", map[string]any{
		"operation": "get_tree",
		"loop_id":   "loop-root",
		"max_depth": float64(5), // JSON numbers come in as float64
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("Execute() result.Error = %q, want empty", result.Error)
	}
	if mock.lastGetTreeLoopID != "loop-root" {
		t.Errorf("GetTree loopID = %q, want %q", mock.lastGetTreeLoopID, "loop-root")
	}
	if mock.lastGetTreeMaxDepth != 5 {
		t.Errorf("GetTree maxDepth = %d, want 5", mock.lastGetTreeMaxDepth)
	}

	got := mustUnmarshalStrings(t, result.Content)
	if len(got) != len(wantIDs) {
		t.Fatalf("Content len = %d, want %d: %v", len(got), len(wantIDs), got)
	}
	for i, want := range wantIDs {
		if got[i] != want {
			t.Errorf("Content[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestExecutor_GetTree_DefaultsToCallLoopID(t *testing.T) {
	t.Parallel()

	mock := &mockGraph{treeIDs: []string{"semsage.default.agentic.orchestrator.loop.self"}}
	exec := tree.NewExecutor(mock)

	// No loop_id argument — should default to call.LoopID.
	call := makeCall("call-4", "loop-self", "trace-4", map[string]any{
		"operation": "get_tree",
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("Execute() result.Error = %q, want empty", result.Error)
	}
	if mock.lastGetTreeLoopID != "loop-self" {
		t.Errorf("GetTree loopID = %q, want %q (call.LoopID)", mock.lastGetTreeLoopID, "loop-self")
	}
	// max_depth should default to 10.
	if mock.lastGetTreeMaxDepth != 10 {
		t.Errorf("GetTree maxDepth = %d, want 10 (default)", mock.lastGetTreeMaxDepth)
	}
}

func TestExecutor_GetStatus_ReturnsStatus(t *testing.T) {
	t.Parallel()

	mock := &mockGraph{status: "running"}
	exec := tree.NewExecutor(mock)

	call := makeCall("call-5", "loop-1", "trace-5", map[string]any{
		"operation": "get_status",
		"loop_id":   "loop-abc",
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("Execute() result.Error = %q, want empty", result.Error)
	}
	if mock.lastGetStatusLoopID != "loop-abc" {
		t.Errorf("GetStatus loopID = %q, want %q", mock.lastGetStatusLoopID, "loop-abc")
	}

	var body map[string]string
	if err := json.Unmarshal([]byte(result.Content), &body); err != nil {
		t.Fatalf("unmarshal status body: %v", err)
	}
	if body["loop_id"] != "loop-abc" {
		t.Errorf("body[loop_id] = %q, want %q", body["loop_id"], "loop-abc")
	}
	if body["status"] != "running" {
		t.Errorf("body[status] = %q, want %q", body["status"], "running")
	}
}

func TestExecutor_MissingOperation_ReturnsError(t *testing.T) {
	t.Parallel()

	exec := tree.NewExecutor(&mockGraph{})

	call := makeCall("call-6", "loop-1", "", map[string]any{})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error == "" {
		t.Error("Execute() result.Error is empty, want non-empty error message")
	}
	if result.Content != "" {
		t.Errorf("Execute() result.Content = %q, want empty on error", result.Content)
	}
}

func TestExecutor_UnknownOperation_ReturnsError(t *testing.T) {
	t.Parallel()

	exec := tree.NewExecutor(&mockGraph{})

	call := makeCall("call-7", "loop-1", "", map[string]any{
		"operation": "delete_all_agents",
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error == "" {
		t.Error("Execute() result.Error is empty, want non-empty error for unknown operation")
	}
}

func TestExecutor_GetChildren_MissingLoopID_ReturnsError(t *testing.T) {
	t.Parallel()

	exec := tree.NewExecutor(&mockGraph{})

	call := makeCall("call-8", "loop-1", "", map[string]any{
		"operation": "get_children",
		// no loop_id
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.Error == "" {
		t.Error("Execute() result.Error is empty, want error about missing loop_id")
	}
}

func TestExecutor_GraphError_PropagatesAsResultError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call agentic.ToolCall
		mock *mockGraph
	}{
		{
			name: "get_children graph error",
			call: makeCall("call-9a", "loop-1", "", map[string]any{
				"operation": "get_children",
				"loop_id":   "loop-1",
			}),
			mock: &mockGraph{childrenErr: errors.New("nats: timeout")},
		},
		{
			name: "get_tree graph error",
			call: makeCall("call-9b", "loop-1", "", map[string]any{
				"operation": "get_tree",
				"loop_id":   "loop-1",
			}),
			mock: &mockGraph{treeErr: errors.New("query: depth exceeded")},
		},
		{
			name: "get_status graph error",
			call: makeCall("call-9c", "loop-1", "", map[string]any{
				"operation": "get_status",
				"loop_id":   "loop-1",
			}),
			mock: &mockGraph{statusErr: errors.New("entity not found")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exec := tree.NewExecutor(tc.mock)
			result, err := exec.Execute(context.Background(), tc.call)

			if err != nil {
				t.Fatalf("Execute() unexpected Go error: %v", err)
			}
			if result.Error == "" {
				t.Error("Execute() result.Error is empty, want wrapped graph error")
			}
		})
	}
}

func TestExecutor_ResultCarriesLoopAndTraceIDs(t *testing.T) {
	t.Parallel()

	mock := &mockGraph{children: []string{"c1"}}
	exec := tree.NewExecutor(mock)

	call := makeCall("call-10", "loop-xyz", "trace-abc", map[string]any{
		"operation": "get_children",
		"loop_id":   "loop-xyz",
	})

	result, err := exec.Execute(context.Background(), call)

	if err != nil {
		t.Fatalf("Execute() unexpected Go error: %v", err)
	}
	if result.LoopID != "loop-xyz" {
		t.Errorf("LoopID = %q, want %q", result.LoopID, "loop-xyz")
	}
	if result.TraceID != "trace-abc" {
		t.Errorf("TraceID = %q, want %q", result.TraceID, "trace-abc")
	}
}

func TestExecutor_ListTools_ReturnsOneDefinition(t *testing.T) {
	t.Parallel()

	exec := tree.NewExecutor(&mockGraph{})
	tools := exec.ListTools()

	if len(tools) != 1 {
		t.Fatalf("ListTools() returned %d definitions, want 1", len(tools))
	}

	def := tools[0]
	if def.Name != "query_agent_tree" {
		t.Errorf("tool Name = %q, want %q", def.Name, "query_agent_tree")
	}
	if def.Description == "" {
		t.Error("tool Description is empty")
	}
	if def.Parameters == nil {
		t.Fatal("tool Parameters is nil")
	}
	// Verify the required field declares "operation".
	required, ok := def.Parameters["required"].([]string)
	if !ok {
		t.Fatalf("Parameters[required] type = %T, want []string", def.Parameters["required"])
	}
	if len(required) != 1 || required[0] != "operation" {
		t.Errorf("required = %v, want [operation]", required)
	}
}
