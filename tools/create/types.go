// Package create implements the create_tool executor.
//
// create_tool lets an LLM compose existing processors into a named, reusable
// tool at runtime. For the MVP the executor validates the provided FlowSpec,
// stores it, and registers a flowToolExecutor that reports the spec back to
// the LLM. Full reactive-engine wiring is deferred to a future phase once the
// SemStreams processor registry exposes a dynamic-registration API.
package create

// FlowSpec describes a composed tool that wires existing processors.
// The LLM supplies a FlowSpec when calling create_tool; the executor
// validates it and registers a new tool in the active agent tree.
type FlowSpec struct {
	// Name is the unique identifier for the new tool within the current tree.
	Name string `json:"name"`

	// Description is a human-readable explanation of what the tool does.
	Description string `json:"description"`

	// Processors lists the existing processor instances that form the flow.
	Processors []ProcessorRef `json:"processors"`

	// Wiring describes how outputs of one processor feed inputs of another.
	Wiring []WiringRule `json:"wiring"`

	// Parameters is an optional JSON Schema describing the tool's input parameters.
	Parameters map[string]any `json:"parameters,omitempty"`
}

// ProcessorRef identifies an existing processor in the SemStreams component
// registry and assigns it a local instance ID within the flow.
type ProcessorRef struct {
	// ID is the instance identifier within this flow (e.g. "step-1").
	ID string `json:"id"`

	// Type is the processor type name from the component registry
	// (e.g. "agentic-model", "agentic-tools").
	Type string `json:"type"`

	// Config holds optional per-instance configuration overrides.
	Config map[string]any `json:"config,omitempty"`
}

// WiringRule describes how the output of one processor feeds the input of another.
type WiringRule struct {
	// From is the source processor instance ID.
	From string `json:"from"`

	// FromPort is the output port name on the source processor.
	FromPort string `json:"from_port"`

	// To is the target processor instance ID.
	To string `json:"to"`

	// ToPort is the input port name on the target processor.
	ToPort string `json:"to_port"`
}
