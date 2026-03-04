package spawn

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/c360studio/semstreams/agentic"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
)

const (
	// defaultTimeout is used when the caller does not specify a timeout.
	defaultTimeout = 5 * time.Minute

	// defaultMaxDepth caps how many levels of nested agents may be spawned.
	defaultMaxDepth = 5

	// sourceSpawn is the source identifier stamped on BaseMessage envelopes
	// published by this executor.
	sourceSpawn = "semsage.spawn"
)

// NATSClient is the subset of natsclient.Client that Executor needs.
// Depending on this interface rather than the concrete struct keeps the
// executor testable without a live NATS connection.
type NATSClient interface {
	PublishToStream(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler func(context.Context, *nats.Msg)) (*natsclient.Subscription, error)
}

// GraphHelper is the subset of agentgraph.Helper that Executor needs.
type GraphHelper interface {
	RecordSpawn(ctx context.Context, parentLoopID, childLoopID, role, model string) error
}

// childResult carries the outcome of a child agent loop back to the waiting
// Execute call through an unbuffered or buffered channel.
type childResult struct {
	content string // non-empty on success
	err     string // non-empty on failure
}

// Executor implements the ToolExecutor interface for the spawn_agent tool.
// It publishes a TaskMessage to start a child agentic loop, waits for the
// loop's completion or failure event, and returns the result as a ToolResult.
type Executor struct {
	nats         NATSClient
	graph        GraphHelper
	defaultModel string
	maxDepth     int
}

// Option is a functional option for configuring an Executor.
type Option func(*Executor)

// WithDefaultModel sets the fallback model used when the caller does not
// provide one in the tool arguments.
func WithDefaultModel(model string) Option {
	return func(e *Executor) {
		e.defaultModel = model
	}
}

// WithMaxDepth sets the maximum spawn depth. The default is 5.
func WithMaxDepth(depth int) Option {
	return func(e *Executor) {
		e.maxDepth = depth
	}
}

// NewExecutor constructs an Executor with the given NATS client and graph
// helper. Pass functional options to override defaults.
func NewExecutor(n NATSClient, g GraphHelper, opts ...Option) *Executor {
	e := &Executor{
		nats:     n,
		graph:    g,
		maxDepth: defaultMaxDepth,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ListTools returns the tool definitions that this executor exposes to an
// agentic loop's tool registry.
func (e *Executor) ListTools() []agentic.ToolDefinition {
	return []agentic.ToolDefinition{{
		Name:        "spawn_agent",
		Description: "Spawn a child agent to perform a subtask. The child runs as an independent agentic loop and returns its result when complete.",
		Parameters: map[string]any{
			"type":     "object",
			"required": []string{"prompt", "role"},
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "Task prompt for the child agent",
				},
				"role": map[string]any{
					"type":        "string",
					"description": "System role for the child agent",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "LLM model (defaults to parent's model)",
				},
				"tools": map[string]any{
					"type":        "array",
					"description": "Tool subset for the child",
					"items":       map[string]any{"type": "object"},
				},
				"timeout": map[string]any{
					"type":        "string",
					"description": "Timeout duration (e.g. '5m', '30s')",
					"default":     "5m",
				},
				"metadata": map[string]any{
					"type":        "object",
					"description": "Additional context passed to child",
				},
			},
		},
	}}
}

// Execute runs the spawn_agent tool call. It:
//  1. Parses and validates arguments from call.Arguments.
//  2. Checks the spawn depth against the configured limit.
//  3. Subscribes to child completion and failure subjects before publishing
//     (critical — prevents the race where the child completes before we listen).
//  4. Publishes a TaskMessage to agent.task.<taskID>.
//  5. Records the parent→child relationship in the graph.
//  6. Blocks until the child completes, fails, the context is cancelled, or
//     the timeout expires.
func (e *Executor) Execute(ctx context.Context, call agentic.ToolCall) (agentic.ToolResult, error) {
	args, parseErr := parseArguments(call.Arguments)
	if parseErr != nil {
		return errorResult(call.ID, call.LoopID, call.TraceID, parseErr.Error()), nil
	}

	// Determine current depth and enforce the limit.
	currentDepth := 0
	if call.Metadata != nil {
		if d, ok := call.Metadata["depth"]; ok {
			switch v := d.(type) {
			case int:
				currentDepth = v
			case float64:
				// JSON numbers unmarshal to float64.
				currentDepth = int(v)
			}
		}
	}
	if currentDepth+1 >= e.maxDepth {
		return errorResult(call.ID, call.LoopID, call.TraceID,
			fmt.Sprintf("spawn depth limit reached: current depth %d, max depth %d",
				currentDepth, e.maxDepth)), nil
	}

	// Resolve model: prefer argument, fall back to executor default.
	model := args.model
	if model == "" {
		model = e.defaultModel
	}
	if model == "" {
		return errorResult(call.ID, call.LoopID, call.TraceID,
			"spawn_agent: no model specified and no default model configured"), nil
	}

	childLoopID := uuid.New().String()
	taskID := uuid.New().String()

	// Subscribe BEFORE publishing to avoid losing the completion event.
	resultCh := make(chan childResult, 1)

	completeSub, subErr := e.nats.Subscribe(
		ctx,
		fmt.Sprintf("agent.complete.%s", childLoopID),
		completionHandler(resultCh),
	)
	if subErr != nil {
		return agentic.ToolResult{}, fmt.Errorf("spawn_agent: subscribe to completion subject: %w", subErr)
	}
	defer func() {
		if completeSub != nil {
			_ = completeSub.Unsubscribe()
		}
	}()

	failedSub, subErr := e.nats.Subscribe(
		ctx,
		fmt.Sprintf("agent.failed.%s", childLoopID),
		failureHandler(resultCh),
	)
	if subErr != nil {
		return agentic.ToolResult{}, fmt.Errorf("spawn_agent: subscribe to failure subject: %w", subErr)
	}
	defer func() {
		if failedSub != nil {
			_ = failedSub.Unsubscribe()
		}
	}()

	// Build and publish the TaskMessage.
	task := &agentic.TaskMessage{
		LoopID:       childLoopID,
		TaskID:       taskID,
		Role:         args.role,
		Model:        model,
		Prompt:       args.prompt,
		ParentLoopID: call.LoopID,
		Depth:        currentDepth + 1,
		MaxDepth:     e.maxDepth,
		Tools:        args.tools,
		Metadata:     args.metadata,
	}

	msg := message.NewBaseMessage(task.Schema(), task, sourceSpawn)
	data, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		return agentic.ToolResult{}, fmt.Errorf("spawn_agent: marshal task message: %w", marshalErr)
	}

	subject := fmt.Sprintf("agent.task.%s", taskID)
	if pubErr := e.nats.PublishToStream(ctx, subject, data); pubErr != nil {
		return agentic.ToolResult{}, fmt.Errorf("spawn_agent: publish task: %w", pubErr)
	}

	// Record the spawn relationship in the graph. Best-effort — the child
	// loop is already running so we continue waiting regardless of failure.
	var graphWarning string
	if graphErr := e.graph.RecordSpawn(ctx, call.LoopID, childLoopID, args.role, model); graphErr != nil {
		graphWarning = fmt.Sprintf("graph recording failed (non-fatal): %v", graphErr)
	}

	// Wait for the outcome.
	timer := time.NewTimer(args.timeout)
	defer timer.Stop()

	resultMeta := map[string]any{
		"child_loop_id": childLoopID,
		"task_id":       taskID,
	}
	if graphWarning != "" {
		resultMeta["warning"] = graphWarning
	}

	select {
	case result := <-resultCh:
		if result.err != "" {
			return errorResult(call.ID, call.LoopID, call.TraceID, result.err), nil
		}
		return agentic.ToolResult{
			CallID:   call.ID,
			Content:  result.content,
			Metadata: resultMeta,
			LoopID:   call.LoopID,
			TraceID:  call.TraceID,
		}, nil

	case <-timer.C:
		return errorResult(call.ID, call.LoopID, call.TraceID,
			fmt.Sprintf("spawn_agent: child loop %s timed out after %s", childLoopID, args.timeout)), nil

	case <-ctx.Done():
		return errorResult(call.ID, call.LoopID, call.TraceID,
			fmt.Sprintf("spawn_agent: context cancelled: %v", ctx.Err())), nil
	}
}

// completionHandler returns a NATS message handler that decodes a
// LoopCompletedEvent from the wire and sends its Result into resultCh.
// The channel is buffered (capacity 1) so the send never blocks even if
// both completion and failure events arrive simultaneously.
func completionHandler(resultCh chan<- childResult) func(context.Context, *nats.Msg) {
	return func(_ context.Context, msg *nats.Msg) {
		var event agentic.LoopCompletedEvent
		if err := unmarshalPayload(msg.Data, &event); err != nil {
			// Malformed message — do not send; the timeout will fire instead.
			return
		}
		// Non-blocking send: if the channel is already full (e.g. a duplicate
		// delivery), the second message is silently discarded.
		select {
		case resultCh <- childResult{content: event.Result}:
		default:
		}
	}
}

// failureHandler returns a NATS message handler that decodes a
// LoopFailedEvent and sends an error childResult into resultCh.
func failureHandler(resultCh chan<- childResult) func(context.Context, *nats.Msg) {
	return func(_ context.Context, msg *nats.Msg) {
		var event agentic.LoopFailedEvent
		if err := unmarshalPayload(msg.Data, &event); err != nil {
			return
		}
		errMsg := event.Error
		if event.Reason != "" {
			errMsg = fmt.Sprintf("%s: %s", event.Reason, errMsg)
		}
		select {
		case resultCh <- childResult{err: errMsg}:
		default:
		}
	}
}

// wireEnvelope is a minimal representation of a BaseMessage used only to
// extract the raw payload bytes without requiring the full registry machinery.
type wireEnvelope struct {
	Payload json.RawMessage `json:"payload"`
}

// unmarshalPayload extracts the payload field from a BaseMessage JSON envelope
// and unmarshals it into dst.
func unmarshalPayload(data []byte, dst any) error {
	var env wireEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if len(env.Payload) == 0 {
		return fmt.Errorf("empty payload")
	}
	return json.Unmarshal(env.Payload, dst)
}

// errorResult constructs a ToolResult that signals an error back to the loop.
// Returning a ToolResult with Error set (rather than a Go error) lets the
// agentic loop decide how to handle the failure; it does not crash the loop.
func errorResult(callID, loopID, traceID, msg string) agentic.ToolResult {
	return agentic.ToolResult{
		CallID:  callID,
		Error:   msg,
		LoopID:  loopID,
		TraceID: traceID,
	}
}

// spawnArgs holds parsed and validated arguments from a spawn_agent tool call.
type spawnArgs struct {
	prompt   string
	role     string
	model    string
	tools    []agentic.ToolDefinition
	timeout  time.Duration
	metadata map[string]any
}

// parseArguments validates the raw arguments map from a ToolCall and returns
// a typed spawnArgs. It returns an error if required fields are absent.
func parseArguments(args map[string]any) (spawnArgs, error) {
	out := spawnArgs{timeout: defaultTimeout}

	prompt, ok := stringArg(args, "prompt")
	if !ok || prompt == "" {
		return spawnArgs{}, fmt.Errorf("spawn_agent: argument 'prompt' is required")
	}
	out.prompt = prompt

	role, ok := stringArg(args, "role")
	if !ok || role == "" {
		return spawnArgs{}, fmt.Errorf("spawn_agent: argument 'role' is required")
	}
	out.role = role

	if model, ok := stringArg(args, "model"); ok {
		out.model = model
	}

	if timeoutStr, ok := stringArg(args, "timeout"); ok && timeoutStr != "" {
		d, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return spawnArgs{}, fmt.Errorf("spawn_agent: invalid timeout %q: %w", timeoutStr, err)
		}
		out.timeout = d
	}

	if rawTools, exists := args["tools"]; exists && rawTools != nil {
		tools, err := parseTools(rawTools)
		if err != nil {
			return spawnArgs{}, fmt.Errorf("spawn_agent: invalid tools: %w", err)
		}
		out.tools = tools
	}

	if rawMeta, exists := args["metadata"]; exists && rawMeta != nil {
		if m, ok := rawMeta.(map[string]any); ok {
			out.metadata = m
		}
	}

	return out, nil
}

// stringArg safely extracts a string value from an arguments map.
func stringArg(args map[string]any, key string) (string, bool) {
	v, exists := args[key]
	if !exists || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// parseTools converts the raw tools argument (slice of maps) into
// []agentic.ToolDefinition using round-trip JSON encoding for safety.
func parseTools(raw any) ([]agentic.ToolDefinition, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var tools []agentic.ToolDefinition
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, err
	}
	return tools, nil
}
