package create

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/c360studio/semstreams/agentic"
	agentictools "github.com/c360studio/semstreams/processor/agentic-tools"
)

const toolName = "create_tool"

// ToolRegistry is the subset of the global agentic-tools registry that
// Executor needs. Using a local interface keeps the executor testable without
// the real global registry.
type ToolRegistry interface {
	RegisterTool(name string, executor agentictools.ToolExecutor) error
}

// Executor implements agentictools.ToolExecutor for the create_tool tool.
// It is safe for concurrent use.
type Executor struct {
	registry  ToolRegistry
	mu        sync.RWMutex
	specs     map[string]*FlowSpec // active specs, keyed by tool name
	treeScope map[string][]string  // root loop ID -> registered tool names
}

// NewExecutor constructs an Executor backed by the given ToolRegistry.
// registry must not be nil; a nil registry will cause panics at call time.
func NewExecutor(registry ToolRegistry) *Executor {
	if registry == nil {
		panic("create.NewExecutor: registry must not be nil")
	}
	return &Executor{
		registry:  registry,
		specs:     make(map[string]*FlowSpec),
		treeScope: make(map[string][]string),
	}
}

// Execute processes a create_tool call. It:
//  1. Parses the FlowSpec from call.Arguments.
//  2. Validates the spec (non-empty name, at least one processor, consistent wiring).
//  3. Checks for duplicate tool names within the active spec set.
//  4. Stores the spec and registers a flowToolExecutor for it.
//  5. Tracks the new tool name under the calling loop's root tree ID.
//  6. Returns a success ToolResult describing the registered tool.
func (e *Executor) Execute(ctx context.Context, call agentic.ToolCall) (agentic.ToolResult, error) {
	spec, err := parseFlowSpec(call.Arguments)
	if err != nil {
		return errorResult(call, err.Error()), nil
	}

	if valErr := validateSpec(spec); valErr != nil {
		return errorResult(call, valErr.Error()), nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.specs[spec.Name]; exists {
		return errorResult(call, fmt.Sprintf("create_tool: tool %q is already registered in this tree", spec.Name)), nil
	}

	flowExec := &flowToolExecutor{spec: spec}
	if regErr := e.registry.RegisterTool(spec.Name, flowExec); regErr != nil {
		return errorResult(call, fmt.Sprintf("create_tool: register tool %q: %s", spec.Name, regErr)), nil
	}

	e.specs[spec.Name] = spec

	// Associate the tool with the root tree ID so CleanupTree can remove it.
	rootID := rootLoopID(call)
	e.treeScope[rootID] = append(e.treeScope[rootID], spec.Name)

	return agentic.ToolResult{
		CallID: call.ID,
		Content: fmt.Sprintf(
			"tool %q created with %d processor(s) and %d wiring rule(s)",
			spec.Name, len(spec.Processors), len(spec.Wiring),
		),
		Metadata: map[string]any{
			"tool_name":       spec.Name,
			"processor_count": len(spec.Processors),
			"wiring_count":    len(spec.Wiring),
		},
		LoopID:  call.LoopID,
		TraceID: call.TraceID,
	}, nil
}

// ListTools returns the single tool definition for create_tool.
func (e *Executor) ListTools() []agentic.ToolDefinition {
	return []agentic.ToolDefinition{{
		Name:        toolName,
		Description: "Create a new tool by composing existing processors into a named flow. The tool becomes available for use by agents in the current tree.",
		Parameters: map[string]any{
			"type":     "object",
			"required": []string{"name", "description", "processors", "wiring"},
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Unique tool name",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Human-readable description",
				},
				"processors": map[string]any{
					"type":        "array",
					"description": "Processors to compose",
					"items":       map[string]any{"type": "object"},
				},
				"wiring": map[string]any{
					"type":        "array",
					"description": "How to wire processor outputs to inputs",
					"items":       map[string]any{"type": "object"},
				},
				"parameters": map[string]any{
					"type":        "object",
					"description": "Parameter schema for the new tool",
				},
			},
		},
	}}
}

// CleanupTree removes all tool registrations associated with rootLoopID from
// the executor's internal tracking. It does not unregister tools from the
// global registry (which has no remove API in the current SemStreams version);
// callers that need hard removal should replace the registry for the affected
// agent tree.
func (e *Executor) CleanupTree(rootLoopID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	names := e.treeScope[rootLoopID]
	for _, name := range names {
		delete(e.specs, name)
	}
	delete(e.treeScope, rootLoopID)
}

// -- flowToolExecutor --

// flowToolExecutor wraps a validated FlowSpec and satisfies agentictools.ToolExecutor.
// For the MVP its Execute method returns a human-readable stub message because
// the SemStreams reactive engine does not yet expose a dynamic-registration API.
// Full reactive wiring is Phase 2.
type flowToolExecutor struct {
	spec *FlowSpec
}

// Execute satisfies agentictools.ToolExecutor. Returns a stub response that
// confirms invocation; reactive wiring is deferred to Phase 2.
func (f *flowToolExecutor) Execute(_ context.Context, call agentic.ToolCall) (agentic.ToolResult, error) {
	return agentic.ToolResult{
		CallID: call.ID,
		Content: fmt.Sprintf(
			"tool %q invoked — reactive engine wiring pending (Phase 2)",
			f.spec.Name,
		),
		LoopID:  call.LoopID,
		TraceID: call.TraceID,
	}, nil
}

// ListTools returns the ToolDefinition derived from the FlowSpec.
func (f *flowToolExecutor) ListTools() []agentic.ToolDefinition {
	params := f.spec.Parameters
	if params == nil {
		params = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return []agentic.ToolDefinition{{
		Name:        f.spec.Name,
		Description: f.spec.Description,
		Parameters:  params,
	}}
}

// -- validation --

// validateSpec checks that a FlowSpec is structurally sound.
// It verifies:
//   - name is non-empty
//   - description is non-empty
//   - at least one processor is declared
//   - all processor IDs are non-empty and unique within the spec
//   - all wiring from/to IDs reference declared processor IDs
func validateSpec(spec *FlowSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("create_tool: name is required")
	}
	if spec.Description == "" {
		return fmt.Errorf("create_tool: description is required")
	}
	if len(spec.Processors) == 0 {
		return fmt.Errorf("create_tool: at least one processor is required")
	}

	// Build the set of declared processor IDs.
	procIDs := make(map[string]struct{}, len(spec.Processors))
	for i, p := range spec.Processors {
		if p.ID == "" {
			return fmt.Errorf("create_tool: processors[%d].id is required", i)
		}
		if p.Type == "" {
			return fmt.Errorf("create_tool: processors[%d].type is required", i)
		}
		if _, dup := procIDs[p.ID]; dup {
			return fmt.Errorf("create_tool: duplicate processor id %q", p.ID)
		}
		procIDs[p.ID] = struct{}{}
	}

	// Verify all wiring references use declared processor IDs and non-empty ports.
	for i, w := range spec.Wiring {
		if w.FromPort == "" {
			return fmt.Errorf("create_tool: wiring[%d].from_port is required", i)
		}
		if w.ToPort == "" {
			return fmt.Errorf("create_tool: wiring[%d].to_port is required", i)
		}
		if _, ok := procIDs[w.From]; !ok {
			return fmt.Errorf("create_tool: wiring[%d].from %q does not reference a declared processor", i, w.From)
		}
		if _, ok := procIDs[w.To]; !ok {
			return fmt.Errorf("create_tool: wiring[%d].to %q does not reference a declared processor", i, w.To)
		}
	}

	return nil
}

// -- argument parsing --

// parseFlowSpec round-trips the arguments map through JSON to populate a
// FlowSpec. This is the same approach used by spawn's parseTools: marshal the
// map back to JSON bytes and unmarshal into the typed struct, which avoids
// fragile type-assertion chains on map[string]any.
func parseFlowSpec(args map[string]any) (*FlowSpec, error) {
	if args == nil {
		return nil, fmt.Errorf("create_tool: arguments are required")
	}

	data, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("create_tool: marshal arguments: %w", err)
	}

	var spec FlowSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("create_tool: unmarshal FlowSpec: %w", err)
	}

	return &spec, nil
}

// -- helpers --

// rootLoopID extracts the root loop ID from call.Metadata["root_loop_id"].
// When absent it falls back to call.LoopID, which is the conventional
// behaviour: a root loop is its own root.
func rootLoopID(call agentic.ToolCall) string {
	if call.Metadata != nil {
		if v, ok := call.Metadata["root_loop_id"]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return call.LoopID
}

// errorResult constructs a ToolResult that surfaces an error string to the
// LLM without returning a Go error. Returning a ToolResult.Error (rather than
// a Go error) keeps the agentic loop running; Go errors signal infrastructure
// failures that should stop execution.
func errorResult(call agentic.ToolCall, msg string) agentic.ToolResult {
	return agentic.ToolResult{
		CallID:  call.ID,
		Error:   msg,
		LoopID:  call.LoopID,
		TraceID: call.TraceID,
	}
}
