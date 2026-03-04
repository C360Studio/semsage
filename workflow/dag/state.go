package dag

import (
	"github.com/c360studio/semstreams/processor/reactive"
)

// Phases for DAG execution.
const (
	PhaseInit       = "init"
	PhaseExecuting  = "executing"
	PhaseCompleted  = "completed"
	PhaseFailed     = "failed"
)

// NodeStatus tracks the execution state of a single DAG node.
const (
	NodePending   = "pending"
	NodeReady     = "ready"
	NodeRunning   = "running"
	NodeCompleted = "completed"
	NodeFailed    = "failed"
)

// State is the reactive execution state for a DAG workflow.
// It embeds reactive.ExecutionState and adds DAG-specific tracking.
type State struct {
	reactive.ExecutionState

	// Goal is the high-level objective this DAG is executing.
	Goal string `json:"goal"`

	// Nodes is the ordered list of DAG nodes to execute.
	Nodes []NodeSpec `json:"nodes"`

	// NodeStates tracks per-node execution progress, keyed by node ID.
	NodeStates map[string]*NodeState `json:"node_states"`

	// ParentLoopID is the agent loop that triggered this DAG execution.
	ParentLoopID string `json:"parent_loop_id,omitempty"`
}

// GetExecutionState implements reactive.StateAccessor to avoid reflection.
func (s *State) GetExecutionState() *reactive.ExecutionState {
	return &s.ExecutionState
}

// NodeSpec describes a single node in the DAG (mirrors decompose.TaskNode).
type NodeSpec struct {
	ID        string   `json:"id"`
	Prompt    string   `json:"prompt"`
	Role      string   `json:"role"`
	DependsOn []string `json:"depends_on"`
}

// NodeState tracks the runtime state of a single node.
type NodeState struct {
	Status string `json:"status"`
	LoopID string `json:"loop_id,omitempty"` // agent loop ID once spawned
	Result string `json:"result,omitempty"`  // completion result
	Error  string `json:"error,omitempty"`   // failure reason
}

// ReadyNodes returns the IDs of nodes that are in "ready" status.
// Nodes transition to ready when their dependencies are all completed
// (done by the init rule for root nodes, and by the completion handler
// for dependent nodes).
func (s *State) ReadyNodes() []string {
	var ready []string
	for _, node := range s.Nodes {
		ns := s.NodeStates[node.ID]
		if ns != nil && ns.Status == NodeReady {
			ready = append(ready, node.ID)
		}
	}
	return ready
}

// AllTerminal returns true when every node is in a terminal state
// (completed or failed).
func (s *State) AllTerminal() bool {
	for _, node := range s.Nodes {
		ns := s.NodeStates[node.ID]
		if ns == nil {
			return false
		}
		if ns.Status != NodeCompleted && ns.Status != NodeFailed {
			return false
		}
	}
	return len(s.Nodes) > 0
}

// AnyFailed returns true if at least one node has failed.
func (s *State) AnyFailed() bool {
	for _, ns := range s.NodeStates {
		if ns.Status == NodeFailed {
			return true
		}
	}
	return false
}

// NodeByLoopID finds the node ID associated with a given agent loop ID.
// Returns ("", false) if no match is found.
func (s *State) NodeByLoopID(loopID string) (string, bool) {
	for id, ns := range s.NodeStates {
		if ns.LoopID == loopID {
			return id, true
		}
	}
	return "", false
}

// GetNode returns the NodeSpec for the given ID, or nil if not found.
func (s *State) GetNode(id string) *NodeSpec {
	for i := range s.Nodes {
		if s.Nodes[i].ID == id {
			return &s.Nodes[i]
		}
	}
	return nil
}

func (s *State) allDepsCompleted(deps []string) bool {
	for _, depID := range deps {
		ns := s.NodeStates[depID]
		if ns == nil || ns.Status != NodeCompleted {
			return false
		}
	}
	return true
}

// InitNodeStates populates NodeStates from the Nodes spec list,
// setting all nodes to pending status.
func (s *State) InitNodeStates() {
	s.NodeStates = make(map[string]*NodeState, len(s.Nodes))
	for _, node := range s.Nodes {
		s.NodeStates[node.ID] = &NodeState{Status: NodePending}
	}
}
