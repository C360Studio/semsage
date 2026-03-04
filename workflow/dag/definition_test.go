package dag

import (
	"testing"

	"github.com/c360studio/semstreams/agentic"
	"github.com/c360studio/semstreams/processor/reactive"
)

func TestNewDefinition_Validates(t *testing.T) {
	def := NewDefinition()
	if def == nil {
		t.Fatal("NewDefinition() returned nil")
	}
	if def.ID != WorkflowID {
		t.Errorf("ID = %q, want %q", def.ID, WorkflowID)
	}
	if def.StateBucket != StateBucket {
		t.Errorf("StateBucket = %q, want %q", def.StateBucket, StateBucket)
	}
	if len(def.Rules) != 5 {
		t.Errorf("got %d rules, want 5", len(def.Rules))
	}
}

func TestNewDefinition_RuleIDs(t *testing.T) {
	def := NewDefinition()
	wantIDs := []string{
		"init-dag",
		"spawn-ready-nodes",
		"handle-completion",
		"handle-failure",
		"check-dag-complete",
	}
	for i, want := range wantIDs {
		if def.Rules[i].ID != want {
			t.Errorf("Rules[%d].ID = %q, want %q", i, def.Rules[i].ID, want)
		}
	}
}

func TestNewDefinition_StateFactory(t *testing.T) {
	def := NewDefinition()
	state := def.StateFactory()
	dagState, ok := state.(*State)
	if !ok {
		t.Fatalf("StateFactory returned %T, want *State", state)
	}
	// Verify StateAccessor works.
	es := dagState.GetExecutionState()
	if es == nil {
		t.Fatal("GetExecutionState() returned nil")
	}
}

func TestMakeTaskID_ParseTaskID_Roundtrip(t *testing.T) {
	taskID := MakeTaskID("exec-123", "node-a")
	execID, nodeID, ok := ParseTaskID(taskID)
	if !ok {
		t.Fatal("ParseTaskID returned false")
	}
	if execID != "exec-123" {
		t.Errorf("execID = %q, want exec-123", execID)
	}
	if nodeID != "node-a" {
		t.Errorf("nodeID = %q, want node-a", nodeID)
	}
}

func TestParseTaskID_InvalidFormat(t *testing.T) {
	tests := []string{
		"",
		"plain-task-id",
		"wrong:exec:node",
		"dagexec:only-one-part",
		"dagexec:",
	}
	for _, id := range tests {
		_, _, ok := ParseTaskID(id)
		if ok {
			t.Errorf("ParseTaskID(%q) = ok, want not ok", id)
		}
	}
}

func TestDAGTrigger_Schema(t *testing.T) {
	trigger := &DAGTrigger{}
	schema := trigger.Schema()
	if schema.Domain != "semsage" {
		t.Errorf("Schema().Domain = %q, want semsage", schema.Domain)
	}
	if schema.Category != "dag-trigger" {
		t.Errorf("Schema().Category = %q, want dag-trigger", schema.Category)
	}
}

func TestDAGTrigger_Validate(t *testing.T) {
	tests := []struct {
		name    string
		trigger DAGTrigger
		wantErr bool
	}{
		{
			name: "valid",
			trigger: DAGTrigger{
				ExecutionID: "exec-1",
				Goal:        "test goal",
				Nodes:       []NodeSpec{{ID: "a", Prompt: "do a", Role: "worker"}},
			},
			wantErr: false,
		},
		{
			name:    "missing execution_id",
			trigger: DAGTrigger{Goal: "test", Nodes: []NodeSpec{{ID: "a"}}},
			wantErr: true,
		},
		{
			name:    "missing goal",
			trigger: DAGTrigger{ExecutionID: "e", Nodes: []NodeSpec{{ID: "a"}}},
			wantErr: true,
		},
		{
			name:    "no nodes",
			trigger: DAGTrigger{ExecutionID: "e", Goal: "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitRule_InitializesState(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[0] // init-dag

	trigger := &DAGTrigger{
		ExecutionID:  "exec-1",
		Goal:         "build something",
		ParentLoopID: "parent-loop",
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "analyst", DependsOn: []string{"a"}},
			{ID: "c", Prompt: "do c", Role: "worker"},
		},
	}

	state := &State{}
	ctx := &reactive.RuleContext{
		State:   state,
		Message: trigger,
	}

	// Verify condition fires.
	for _, cond := range rule.Conditions {
		if !cond.Evaluate(ctx) {
			t.Fatalf("condition %q failed", cond.Description)
		}
	}

	// Execute the mutation.
	if err := rule.Action.MutateState(ctx, nil); err != nil {
		t.Fatalf("MutateState error: %v", err)
	}

	if state.ID != "exec-1" {
		t.Errorf("state.ID = %q, want exec-1", state.ID)
	}
	if state.Goal != "build something" {
		t.Errorf("state.Goal = %q, want build something", state.Goal)
	}
	if state.ParentLoopID != "parent-loop" {
		t.Errorf("state.ParentLoopID = %q, want parent-loop", state.ParentLoopID)
	}
	if state.Phase != PhaseExecuting {
		t.Errorf("state.Phase = %q, want %q", state.Phase, PhaseExecuting)
	}
	if state.Status != reactive.StatusRunning {
		t.Errorf("state.Status = %q, want %q", state.Status, reactive.StatusRunning)
	}
	if len(state.NodeStates) != 3 {
		t.Fatalf("got %d node states, want 3", len(state.NodeStates))
	}

	// Root nodes (a, c) should be ready; b depends on a so stays pending.
	if state.NodeStates["a"].Status != NodeReady {
		t.Errorf("node a status = %q, want ready", state.NodeStates["a"].Status)
	}
	if state.NodeStates["b"].Status != NodePending {
		t.Errorf("node b status = %q, want pending", state.NodeStates["b"].Status)
	}
	if state.NodeStates["c"].Status != NodeReady {
		t.Errorf("node c status = %q, want ready", state.NodeStates["c"].Status)
	}
}

func TestSpawnReadyRule_TransitionsToRunning(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[1] // spawn-ready-nodes

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker", DependsOn: []string{"a"}},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeReady},
			"b": {Status: NodePending},
		},
	}
	state.Phase = PhaseExecuting

	ctx := &reactive.RuleContext{State: state}

	// Verify conditions fire.
	for _, cond := range rule.Conditions {
		if !cond.Evaluate(ctx) {
			t.Fatalf("condition %q failed", cond.Description)
		}
	}

	// Execute mutation.
	if err := rule.Action.MutateState(ctx, nil); err != nil {
		t.Fatalf("MutateState error: %v", err)
	}

	if state.NodeStates["a"].Status != NodeRunning {
		t.Errorf("node a status = %q, want running", state.NodeStates["a"].Status)
	}
	if state.NodeStates["b"].Status != NodePending {
		t.Errorf("node b status = %q, want pending (dep not met)", state.NodeStates["b"].Status)
	}
}

func TestSpawnReadyRule_NoFireWhenNoReadyNodes(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[1]

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeRunning},
		},
	}
	state.Phase = PhaseExecuting

	ctx := &reactive.RuleContext{State: state}

	// First condition checks for ready nodes — should fail.
	if rule.Conditions[0].Evaluate(ctx) {
		t.Error("expected condition to be false when no ready nodes")
	}
}

func TestHandleCompletionRule_MarksCompletedAndPromotesDependents(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[2] // handle-completion

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker", DependsOn: []string{"a"}},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeRunning, LoopID: "loop-1"},
			"b": {Status: NodePending},
		},
	}
	state.Phase = PhaseExecuting

	event := &agentic.LoopCompletedEvent{
		LoopID: "loop-1",
		TaskID: MakeTaskID("exec-1", "a"),
		Result: "task a done",
	}

	ctx := &reactive.RuleContext{
		State:   state,
		Message: event,
	}

	// Verify conditions.
	for _, cond := range rule.Conditions {
		if !cond.Evaluate(ctx) {
			t.Fatalf("condition %q failed", cond.Description)
		}
	}

	// Execute mutation.
	if err := rule.Action.MutateState(ctx, nil); err != nil {
		t.Fatalf("MutateState error: %v", err)
	}

	if state.NodeStates["a"].Status != NodeCompleted {
		t.Errorf("node a status = %q, want completed", state.NodeStates["a"].Status)
	}
	if state.NodeStates["a"].Result != "task a done" {
		t.Errorf("node a result = %q, want 'task a done'", state.NodeStates["a"].Result)
	}
	// b's dep (a) is now completed, so b should be promoted to ready.
	if state.NodeStates["b"].Status != NodeReady {
		t.Errorf("node b status = %q, want ready (dep completed)", state.NodeStates["b"].Status)
	}
}

func TestHandleCompletionRule_NoFireForUnknownTask(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[2]

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeRunning, LoopID: "loop-1"},
		},
	}
	state.Phase = PhaseExecuting

	// Event with a task ID that doesn't match any node.
	event := &agentic.LoopCompletedEvent{
		LoopID: "loop-999",
		TaskID: "plain-task-id", // not a dagexec: prefixed ID
	}

	ctx := &reactive.RuleContext{
		State:   state,
		Message: event,
	}

	// Condition should not fire.
	for _, cond := range rule.Conditions {
		if cond.Evaluate(ctx) {
			t.Error("expected condition to be false for non-DAG task ID")
			return
		}
	}
}

func TestHandleFailureRule_MarksNodeFailed(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[3] // handle-failure

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeRunning, LoopID: "loop-1"},
		},
	}
	state.Phase = PhaseExecuting

	event := &agentic.LoopFailedEvent{
		LoopID: "loop-1",
		TaskID: MakeTaskID("exec-1", "a"),
		Error:  "something went wrong",
	}

	ctx := &reactive.RuleContext{
		State:   state,
		Message: event,
	}

	// Verify conditions.
	for _, cond := range rule.Conditions {
		if !cond.Evaluate(ctx) {
			t.Fatalf("condition %q failed", cond.Description)
		}
	}

	// Execute mutation.
	if err := rule.Action.MutateState(ctx, nil); err != nil {
		t.Fatalf("MutateState error: %v", err)
	}

	if state.NodeStates["a"].Status != NodeFailed {
		t.Errorf("node a status = %q, want failed", state.NodeStates["a"].Status)
	}
	if state.NodeStates["a"].Error != "something went wrong" {
		t.Errorf("node a error = %q, want 'something went wrong'", state.NodeStates["a"].Error)
	}
}

func TestCheckDAGCompleteRule_CompletesWhenAllDone(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[4] // check-dag-complete

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker"},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeCompleted},
			"b": {Status: NodeCompleted},
		},
	}
	state.Phase = PhaseExecuting

	ctx := &reactive.RuleContext{State: state}

	for _, cond := range rule.Conditions {
		if !cond.Evaluate(ctx) {
			t.Fatalf("condition %q failed", cond.Description)
		}
	}

	if err := rule.Action.MutateState(ctx, nil); err != nil {
		t.Fatalf("MutateState error: %v", err)
	}

	if state.Phase != PhaseCompleted {
		t.Errorf("phase = %q, want %q", state.Phase, PhaseCompleted)
	}
}

func TestCheckDAGCompleteRule_FailsWhenNodeFailed(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[4]

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker"},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeCompleted},
			"b": {Status: NodeFailed},
		},
	}
	state.Phase = PhaseExecuting

	ctx := &reactive.RuleContext{State: state}

	for _, cond := range rule.Conditions {
		if !cond.Evaluate(ctx) {
			t.Fatalf("condition %q failed", cond.Description)
		}
	}

	if err := rule.Action.MutateState(ctx, nil); err != nil {
		t.Fatalf("MutateState error: %v", err)
	}

	if state.Phase != PhaseFailed {
		t.Errorf("phase = %q, want %q", state.Phase, PhaseFailed)
	}
	if state.Error == "" {
		t.Error("expected error message when node failed")
	}
}

func TestCheckDAGCompleteRule_NoFireWhenNotAllTerminal(t *testing.T) {
	def := NewDefinition()
	rule := def.Rules[4]

	state := &State{
		Nodes: []NodeSpec{
			{ID: "a", Prompt: "do a", Role: "worker"},
			{ID: "b", Prompt: "do b", Role: "worker"},
		},
		NodeStates: map[string]*NodeState{
			"a": {Status: NodeCompleted},
			"b": {Status: NodeRunning},
		},
	}
	state.Phase = PhaseExecuting

	ctx := &reactive.RuleContext{State: state}

	if rule.Conditions[0].Evaluate(ctx) {
		t.Error("expected condition to be false when not all terminal")
	}
}
