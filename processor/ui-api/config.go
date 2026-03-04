package uiapi

// Config holds the configuration for the ui-api component.
// All fields have sensible defaults (see DefaultConfig).
type Config struct {
	// LoopsBucket is the NATS KV bucket name for AGENT_LOOPS state.
	// Defaults to "AGENT_LOOPS".
	LoopsBucket string `json:"loops_bucket"`

	// DispatchSubject is the NATS JetStream subject prefix used when publishing
	// user messages to agentic-dispatch.
	// The published subject is: "<DispatchSubject>.<taskID>"
	// Defaults to "agent.task".
	DispatchSubject string `json:"dispatch_subject"`

	// SignalSubject is the NATS JetStream subject prefix used when publishing
	// user control signals (pause/resume/cancel) to the agentic loop.
	// The published subject is: "<SignalSubject>.<loopID>"
	// Defaults to "agent.signal".
	SignalSubject string `json:"signal_subject"`

	// GraphQLProxyURL is the base URL of the semstreams GraphQL gateway.
	// When set, GET /graphql/ requests are proxied there.
	// When empty, /graphql/ returns 501 Not Implemented.
	GraphQLProxyURL string `json:"graphql_proxy_url"`

	// DefaultModel is the LLM model used for chat task dispatch.
	// Defaults to "claude-sonnet-4-20250514".
	DefaultModel string `json:"default_model"`

	// MaxBodyBytes is the maximum allowed request body size for POST endpoints.
	// Defaults to 1 MB.
	MaxBodyBytes int64 `json:"max_body_bytes"`

	// MaxLoopsPerPage is the maximum number of loops returned per list request.
	// Defaults to 200.
	MaxLoopsPerPage int `json:"max_loops_per_page"`
}

// DefaultConfig returns a Config populated with production-safe defaults.
func DefaultConfig() Config {
	return Config{
		LoopsBucket:     "AGENT_LOOPS",
		DispatchSubject: "agent.task",
		SignalSubject:   "agent.signal",
		GraphQLProxyURL: "",
		DefaultModel:    "claude-sonnet-4-20250514",
		MaxBodyBytes:    1 << 20, // 1 MB
		MaxLoopsPerPage: 200,
	}
}
