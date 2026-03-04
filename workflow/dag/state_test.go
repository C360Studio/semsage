package dag

import (
	"testing"
)

func TestState_InitNodeStates(t *testing.T) {
	s := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker", DependsOn: []string{"a"}},
		},
	}

	s.InitNodeStates()

	if len(s.NodeStates) != 2 {
		t.Fatalf("got %d node states, want 2", len(s.NodeStates))
	}
	for _, id := range []string{"a", "b"} {
		ns := s.NodeStates[id]
		if ns == nil {
			t.Fatalf("missing node state for %q", id)
		}
		if ns.Status != NodePending {
			t.Errorf("node %q status = %q, want %q", id, ns.Status, NodePending)
		}
	}
}

func TestState_ReadyNodes_ReturnsReadyStatus(t *testing.T) {
	s := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker", DependsOn: []string{"a"}},
			{ID: "c", Prompt: "do c", Role: "worker"},
		},
	}
	s.InitNodeStates()
	// Mark root nodes as ready (as the init rule would).
	s.NodeStates["a"].Status = NodeReady
	s.NodeStates["c"].Status = NodeReady

	ready := s.ReadyNodes()
	if len(ready) != 2 {
		t.Fatalf("got %d ready nodes, want 2", len(ready))
	}
	wantReady := map[string]bool{"a": true, "c": true}
	for _, id := range ready {
		if !wantReady[id] {
			t.Errorf("unexpected ready node %q", id)
		}
	}
}

func TestState_ReadyNodes_OnlyReady(t *testing.T) {
	s := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker", DependsOn: []string{"a"}},
		},
	}
	s.InitNodeStates()
	s.NodeStates["a"].Status = NodeCompleted
	s.NodeStates["b"].Status = NodeReady // promoted by completion handler

	ready := s.ReadyNodes()
	if len(ready) != 1 || ready[0] != "b" {
		t.Fatalf("got ready %v, want [b]", ready)
	}
}

func TestState_ReadyNodes_NoneWhenRunning(t *testing.T) {
	s := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
		},
	}
	s.InitNodeStates()
	s.NodeStates["a"].Status = NodeRunning

	ready := s.ReadyNodes()
	if len(ready) != 0 {
		t.Fatalf("got ready %v, want none", ready)
	}
}

func TestState_AllTerminal(t *testing.T) {
	tests := []struct {
		name     string
		statuses map[string]string
		want     bool
	}{
		{
			name:     "all completed",
			statuses: map[string]string{"a": NodeCompleted, "b": NodeCompleted},
			want:     true,
		},
		{
			name:     "mixed terminal",
			statuses: map[string]string{"a": NodeCompleted, "b": NodeFailed},
			want:     true,
		},
		{
			name:     "one running",
			statuses: map[string]string{"a": NodeCompleted, "b": NodeRunning},
			want:     false,
		},
		{
			name:     "one pending",
			statuses: map[string]string{"a": NodeCompleted, "b": NodePending},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Nodes: []NodeSpec{
					{ID: "a", Prompt: "do a", Role: "worker"},
					{ID: "b", Prompt: "do b", Role: "worker"},
				},
			}
			s.InitNodeStates()
			for id, status := range tt.statuses {
				s.NodeStates[id].Status = status
			}

			got := s.AllTerminal()
			if got != tt.want {
				t.Errorf("AllTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_AllTerminal_EmptyNodes(t *testing.T) {
	s := &State{}
	if s.AllTerminal() {
		t.Error("AllTerminal() = true for empty nodes, want false")
	}
}

func TestState_AnyFailed(t *testing.T) {
	s := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker"},
		},
	}
	s.InitNodeStates()

	if s.AnyFailed() {
		t.Error("AnyFailed() = true with all pending, want false")
	}

	s.NodeStates["b"].Status = NodeFailed
	if !s.AnyFailed() {
		t.Error("AnyFailed() = false with one failed, want true")
	}
}

func TestState_NodeByLoopID(t *testing.T) {
	s := &State{
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeRunning, LoopID: "loop-1"},
			"b": {Status: NodeRunning, LoopID: "loop-2"},
		},
	}

	nodeID, found := s.NodeByLoopID("loop-2")
	if !found || nodeID != "b" {
		t.Errorf("NodeByLoopID(loop-2) = (%q, %v), want (b, true)", nodeID, found)
	}

	_, found = s.NodeByLoopID("loop-999")
	if found {
		t.Error("NodeByLoopID(loop-999) found, want not found")
	}
}

func TestState_GetNode(t *testing.T) {
	s := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "analyst"},
		},
	}

	node := s.GetNode("b")
	if node == nil {
		t.Fatal("GetNode(b) = nil")
	}
	if node.Role != "analyst" {
		t.Errorf("GetNode(b).Role = %q, want analyst", node.Role)
	}

	if s.GetNode("z") != nil {
		t.Error("GetNode(z) != nil, want nil")
	}
}

func TestState_GetExecutionState(t *testing.T) {
	s := &State{}
	s.Phase = "test-phase"

	es := s.GetExecutionState()
	if es == nil {
		t.Fatal("GetExecutionState() = nil")
	}
	if es.Phase != "test-phase" {
		t.Errorf("Phase = %q, want test-phase", es.Phase)
	}
}
