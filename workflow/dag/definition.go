package dag

import (
	"fmt"
	"strings"
	"time"

	"github.com/c360studio/semstreams/agentic"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/processor/reactive"
)

const (
	// WorkflowID is the unique identifier for the DAG execution workflow.
	WorkflowID = "dag-execution"

	// StateBucket is the KV bucket that stores DAG execution state.
	StateBucket = "DAG_EXECUTIONS"

	// TriggerSubject is the NATS subject that triggers DAG execution.
	// Publish a DAGTrigger message here to start a new DAG.
	TriggerSubject = "workflow.dag.trigger"

	// AgentStream is the JetStream stream for agent lifecycle events.
	AgentStream = "AGENT"

	// CompletionSubject is the NATS subject pattern for agent loop completions.
	CompletionSubject = "agent.complete.>"

	// FailureSubject is the NATS subject pattern for agent loop failures.
	FailureSubject = "agent.failed.>"
)

// TaskIDPrefix is prepended to TaskIDs for agent loops spawned by a DAG.
// Format: "dagexec:<executionID>:<nodeID>". This convention lets the
// StateKeyFunc extract the execution ID from completion/failure events
// without requiring a Metadata field on the event types.
const TaskIDPrefix = "dagexec"

// MakeTaskID creates a task ID that encodes the DAG execution ID and node ID.
func MakeTaskID(executionID, nodeID string) string {
	return TaskIDPrefix + ":" + executionID + ":" + nodeID
}

// ParseTaskID extracts the execution ID and node ID from a DAG task ID.
// Returns ("", "", false) if the task ID does not match the convention.
func ParseTaskID(taskID string) (executionID, nodeID string, ok bool) {
	parts := strings.SplitN(taskID, ":", 3)
	if len(parts) != 3 || parts[0] != TaskIDPrefix {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// DAGTrigger is the message payload that initiates a DAG execution.
// It carries the goal and node specs from decompose_task output.
type DAGTrigger struct {
	// ExecutionID is the unique identifier for this execution.
	// Used as the KV key in StateBucket.
	ExecutionID string `json:"execution_id"`

	// Goal is the high-level objective.
	Goal string `json:"goal"`

	// Nodes defines the DAG structure.
	Nodes []NodeSpec `json:"nodes"`

	// ParentLoopID is the agent loop that initiated this DAG.
	ParentLoopID string `json:"parent_loop_id,omitempty"`

	// Model is the LLM model to use for spawned agents.
	Model string `json:"model,omitempty"`

	// MaxDepth is the maximum spawn depth for child agents.
	MaxDepth int `json:"max_depth,omitempty"`
}

// Schema returns the message type for payload registry.
func (d *DAGTrigger) Schema() message.Type {
	return message.Type{Domain: "semsage", Category: "dag-trigger", Version: "v1"}
}

// Validate checks the trigger payload.
func (d *DAGTrigger) Validate() error {
	if d.ExecutionID == "" {
		return fmt.Errorf("execution_id required")
	}
	if d.Goal == "" {
		return fmt.Errorf("goal required")
	}
	if len(d.Nodes) == 0 {
		return fmt.Errorf("at least one node required")
	}
	return nil
}

// NewDefinition constructs the DAG execution reactive workflow definition.
// The workflow uses four rules:
//
//  1. init-dag: Triggered by a DAGTrigger message. Initializes state,
//     marks root nodes (no dependencies) as ready.
//
//  2. spawn-ready-nodes: Triggered by KV state changes. When nodes are
//     in "ready" status, publishes TaskMessages to spawn agent loops.
//
//  3. handle-completion: Triggered by agent.complete.> messages with
//     state lookup. Marks the corresponding node as completed, promotes
//     newly-ready dependents.
//
//  4. handle-failure: Triggered by agent.failed.> messages with state
//     lookup. Marks the corresponding node as failed.
//
//  5. check-dag-complete: Triggered by KV state changes. When all nodes
//     are terminal, completes the DAG execution.
func NewDefinition() *reactive.Definition {
	return reactive.NewWorkflow(WorkflowID).
		WithDescription("Executes a DAG of subtasks by spawning agent loops for each node and tracking completion").
		WithStateBucket(StateBucket).
		WithStateFactory(func() any { return &State{} }).
		WithTimeout(30 * time.Minute).
		WithMaxIterations(1000).
		WithOnComplete("workflow.dag.complete").
		WithOnFail("workflow.dag.failed").
		AddRule(initRule()).
		AddRule(spawnReadyRule()).
		AddRule(handleCompletionRule()).
		AddRule(handleFailureRule()).
		AddRule(checkDAGCompleteRule()).
		MustBuild()
}

// initRule creates the DAG state from the trigger message.
func initRule() reactive.RuleDef {
	return reactive.NewRule("init-dag").
		OnSubject(TriggerSubject, func() any { return &DAGTrigger{} }).
		When("has trigger message", reactive.HasMessage()).
		Mutate(func(ctx *reactive.RuleContext, _ any) error {
			trigger := ctx.Message.(*DAGTrigger)
			state := ctx.State.(*State)

			state.ID = trigger.ExecutionID
			state.WorkflowID = WorkflowID
			state.Goal = trigger.Goal
			state.Nodes = trigger.Nodes
			state.ParentLoopID = trigger.ParentLoopID
			state.Phase = PhaseExecuting
			state.Status = reactive.StatusRunning
			state.InitNodeStates()

			// Mark root nodes (no dependencies) as ready.
			for _, node := range state.Nodes {
				if len(node.DependsOn) == 0 {
					state.NodeStates[node.ID].Status = NodeReady
				}
			}

			return nil
		}).
		WithMaxFirings(1).
		MustBuild()
}

// spawnReadyRule watches for nodes in "ready" status and spawns agents.
func spawnReadyRule() reactive.RuleDef {
	return reactive.NewRule("spawn-ready-nodes").
		WatchKV(StateBucket, ">").
		When("has ready nodes", func(ctx *reactive.RuleContext) bool {
			state, ok := ctx.State.(*State)
			if !ok {
				return false
			}
			return len(state.ReadyNodes()) > 0
		}).
		When("dag is executing", func(ctx *reactive.RuleContext) bool {
			state, ok := ctx.State.(*State)
			if !ok {
				return false
			}
			return state.Phase == PhaseExecuting
		}).
		Mutate(func(ctx *reactive.RuleContext, _ any) error {
			state := ctx.State.(*State)

			// Transition all ready nodes to running.
			// The actual agent spawning happens via the spawn_agent tool
			// or by publishing TaskMessages — here we record the intent.
			for _, nodeID := range state.ReadyNodes() {
				state.NodeStates[nodeID].Status = NodeRunning
			}

			return nil
		}).
		MustBuild()
}

// handleCompletionRule processes agent loop completion events.
func handleCompletionRule() reactive.RuleDef {
	return reactive.NewRule("handle-completion").
		OnJetStreamSubject(AgentStream, CompletionSubject, func() any {
			return &agentic.LoopCompletedEvent{}
		}).
		WithStateLookup(StateBucket, func(msg any) string {
			event := msg.(*agentic.LoopCompletedEvent)
			execID, _, ok := ParseTaskID(event.TaskID)
			if !ok {
				return ""
			}
			return execID
		}).
		When("event matches a running node", func(ctx *reactive.RuleContext) bool {
			state, ok := ctx.State.(*State)
			if !ok || state.Phase != PhaseExecuting {
				return false
			}
			event := ctx.Message.(*agentic.LoopCompletedEvent)
			_, nodeID, ok := ParseTaskID(event.TaskID)
			if !ok {
				return false
			}
			ns := state.NodeStates[nodeID]
			return ns != nil && ns.Status == NodeRunning
		}).
		Mutate(func(ctx *reactive.RuleContext, _ any) error {
			state := ctx.State.(*State)
			event := ctx.Message.(*agentic.LoopCompletedEvent)

			_, nodeID, _ := ParseTaskID(event.TaskID)
			state.NodeStates[nodeID].Status = NodeCompleted
			state.NodeStates[nodeID].Result = event.Result

			// Promote dependents that are now ready.
			for _, candidate := range state.Nodes {
				ns := state.NodeStates[candidate.ID]
				if ns.Status != NodePending {
					continue
				}
				if state.allDepsCompleted(candidate.DependsOn) {
					ns.Status = NodeReady
				}
			}

			return nil
		}).
		MustBuild()
}

// handleFailureRule processes agent loop failure events.
func handleFailureRule() reactive.RuleDef {
	return reactive.NewRule("handle-failure").
		OnJetStreamSubject(AgentStream, FailureSubject, func() any {
			return &agentic.LoopFailedEvent{}
		}).
		WithStateLookup(StateBucket, func(msg any) string {
			event := msg.(*agentic.LoopFailedEvent)
			execID, _, ok := ParseTaskID(event.TaskID)
			if !ok {
				return ""
			}
			return execID
		}).
		When("event matches a running node", func(ctx *reactive.RuleContext) bool {
			state, ok := ctx.State.(*State)
			if !ok || state.Phase != PhaseExecuting {
				return false
			}
			event := ctx.Message.(*agentic.LoopFailedEvent)
			_, nodeID, ok := ParseTaskID(event.TaskID)
			if !ok {
				return false
			}
			ns := state.NodeStates[nodeID]
			return ns != nil && ns.Status == NodeRunning
		}).
		Mutate(func(ctx *reactive.RuleContext, _ any) error {
			state := ctx.State.(*State)
			event := ctx.Message.(*agentic.LoopFailedEvent)

			_, nodeID, _ := ParseTaskID(event.TaskID)
			state.NodeStates[nodeID].Status = NodeFailed
			state.NodeStates[nodeID].Error = event.Error

			return nil
		}).
		MustBuild()
}

// checkDAGCompleteRule fires when all nodes are terminal.
func checkDAGCompleteRule() reactive.RuleDef {
	return reactive.NewRule("check-dag-complete").
		WatchKV(StateBucket, ">").
		When("all nodes are terminal", func(ctx *reactive.RuleContext) bool {
			state, ok := ctx.State.(*State)
			if !ok {
				return false
			}
			return state.Phase == PhaseExecuting && state.AllTerminal()
		}).
		CompleteWithMutation(func(ctx *reactive.RuleContext, _ any) error {
			state := ctx.State.(*State)
			if state.AnyFailed() {
				state.Phase = PhaseFailed
				state.Error = "one or more DAG nodes failed"
			} else {
				state.Phase = PhaseCompleted
			}
			return nil
		}).
		MustBuild()
}
