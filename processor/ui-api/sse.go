package uiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// handleActivityStream streams real-time loop activity over Server-Sent Events.
// It watches the AGENT_LOOPS KV bucket and emits one SSE event per KV change.
//
// SSE event types:
//
//	connected      – sent immediately on connect
//	sync_complete  – sent after the initial key snapshot is delivered
//	activity       – sent for each loop_created / loop_updated / loop_deleted change
//	error          – sent when the watcher fails; client should reconnect
//
// Clients must reconnect on error or unexpected close.
func (c *Component) handleActivityStream(w http.ResponseWriter, r *http.Request) {
	// Merge the request context (cancelled when the client disconnects) with
	// the component context (cancelled on Stop()) so that either event tears
	// down the SSE stream.
	c.mu.RLock()
	compCtx := c.componentCtx
	c.mu.RUnlock()

	ctx := r.Context()
	if compCtx != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-compCtx.Done():
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	bucket, err := c.getLoopsBucket(ctx)
	if err != nil {
		c.logger.ErrorContext(ctx, "ui-api: activity stream — KV bucket unavailable",
			slog.String("error", err.Error()),
		)
		c.sseError(w, flusher, "AGENT_LOOPS bucket unavailable", err)
		return
	}

	watcher, err := bucket.WatchAll(ctx)
	if err != nil {
		c.logger.ErrorContext(ctx, "ui-api: activity stream — watcher create failed",
			slog.String("error", err.Error()),
		)
		c.sseError(w, flusher, "failed to create KV watcher", err)
		return
	}
	defer func() {
		if stopErr := watcher.Stop(); stopErr != nil {
			c.logger.Warn("ui-api: activity watcher stop error",
				slog.String("error", stopErr.Error()),
			)
		}
	}()

	// Instruct the browser client to retry after 5 s on disconnect.
	fmt.Fprintf(w, "retry: 5000\n\n")
	flusher.Flush()

	c.sseEvent(w, flusher, "connected", map[string]string{
		"message": "connected to activity stream",
	})

	c.logger.Info("ui-api: SSE client connected",
		slog.String("remote_addr", r.RemoteAddr),
	)

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("ui-api: SSE client disconnected",
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("reason", ctx.Err().Error()),
			)
			return

		case <-heartbeat.C:
			// SSE comment — keeps the TCP connection alive through proxies.
			if _, err := fmt.Fprintf(w, ":heartbeat %d\n\n", time.Now().Unix()); err != nil {
				c.logger.Info("ui-api: SSE heartbeat write failed, closing stream",
					slog.String("remote_addr", r.RemoteAddr),
					slog.String("error", err.Error()),
				)
				return
			}
			flusher.Flush()

		case entry, ok := <-watcher.Updates():
			if !ok {
				c.logger.Warn("ui-api: activity watcher closed unexpectedly")
				c.sseError(w, flusher, "watcher closed unexpectedly", nil)
				return
			}

			if entry == nil {
				// nil sentinel signals end of initial key snapshot.
				c.sseEvent(w, flusher, "sync_complete", map[string]string{
					"message": "initial sync complete",
				})
				continue
			}

			eventType := kvOpToEventType(entry.Operation(), entry.Revision())

			event := ActivityEvent{
				Type:      eventType,
				LoopID:    entry.Key(),
				Timestamp: entry.Created(),
			}

			if entry.Operation() != jetstream.KeyValueDelete && len(entry.Value()) > 0 {
				if json.Valid(entry.Value()) {
					event.Data = json.RawMessage(entry.Value())
				} else {
					event.Data, _ = json.Marshal(string(entry.Value()))
				}
			}

			data, err := json.Marshal(event)
			if err != nil {
				c.logger.ErrorContext(ctx, "ui-api: failed to marshal activity event",
					slog.String("loop_id", entry.Key()),
					slog.String("error", err.Error()),
				)
				continue
			}

			fmt.Fprintf(w, "event: activity\ndata: %s\n\n", data)
			flusher.Flush()

			c.logger.DebugContext(ctx, "ui-api: activity event sent",
				slog.String("loop_id", entry.Key()),
				slog.String("event_type", eventType),
			)
		}
	}
}

// kvOpToEventType maps a NATS KV operation to a UI activity event type string.
// revision == 1 means the key was just created for the first time.
func kvOpToEventType(op jetstream.KeyValueOp, revision uint64) string {
	switch op {
	case jetstream.KeyValuePut:
		if revision == 1 {
			return "loop_created"
		}
		return "loop_updated"
	case jetstream.KeyValueDelete:
		return "loop_deleted"
	default:
		return "loop_updated"
	}
}

// sseEvent serialises data as JSON and writes a named SSE event to w.
func (c *Component) sseEvent(w http.ResponseWriter, f http.Flusher, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		c.logger.Error("ui-api: failed to marshal SSE event",
			slog.String("event", event),
			slog.String("error", err.Error()),
		)
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
	f.Flush()
}

// sseError sends an error event to the SSE client.
func (c *Component) sseError(w http.ResponseWriter, f http.Flusher, msg string, cause error) {
	errData := map[string]string{"error": msg}
	if cause != nil {
		errData["details"] = cause.Error()
	}
	c.sseEvent(w, f, "error", errData)
}
