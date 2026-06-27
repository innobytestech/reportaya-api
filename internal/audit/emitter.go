package audit

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Emitter writes audit events to outbox.
type Emitter struct {
	outbox *OutboxRepository
}

func NewEmitter(outbox *OutboxRepository) *Emitter {
	return &Emitter{outbox: outbox}
}

func (e *Emitter) Emit(ctx context.Context, evt Event) error {
	if e == nil || e.outbox == nil {
		return nil
	}
	if evt.EventID == "" {
		evt.EventID = uuid.NewString()
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	if evt.Source == "" {
		evt.Source = "api"
	}
	evt.Before = sanitizeMap(evt.Before)
	evt.After = sanitizeMap(evt.After)
	evt.Metadata = sanitizeMap(evt.Metadata)

	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return e.outbox.Enqueue(ctx, evt.EventID, evt.OccurredAt, evt.Action, b)
}

func sanitizeMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		if isSensitiveKey(k) {
			out[k] = "[REDACTED]"
			continue
		}
		switch child := v.(type) {
		case map[string]interface{}:
			out[k] = sanitizeMap(child)
		default:
			out[k] = v
		}
	}
	return out
}

// sensitiveSubstrings are checked against a normalized (lowercase, no separators) key.
var sensitiveSubstrings = []string{
	"password", "secret", "token", "apikey", "jwt", "credential",
}

func isSensitiveKey(k string) bool {
	k = strings.ToLower(strings.TrimSpace(k))
	// Flatten separators so "api_key", "apiKey", "api-key" all match "apikey"
	flat := strings.NewReplacer("_", "", "-", "").Replace(k)
	for _, sub := range sensitiveSubstrings {
		if strings.Contains(flat, sub) {
			return true
		}
	}
	return false
}
