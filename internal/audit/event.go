package audit

import "time"

// Event is the canonical audit payload.
type Event struct {
	EventID      string                 `json:"event_id" bson:"event_id"`
	OccurredAt   time.Time              `json:"occurred_at" bson:"occurred_at"`
	ActorUserID  string                 `json:"actor_user_id,omitempty" bson:"actor_user_id,omitempty"`
	ActorRealm   string                 `json:"actor_realm,omitempty" bson:"actor_realm,omitempty"`
	Action       string                 `json:"action" bson:"action"`
	ResourceType string                 `json:"resource_type" bson:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty" bson:"resource_id,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty" bson:"tenant_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty" bson:"request_id,omitempty"`
	IP           string                 `json:"ip,omitempty" bson:"ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty" bson:"user_agent,omitempty"`
	Status       string                 `json:"status" bson:"status"`
	ErrorCode    string                 `json:"error_code,omitempty" bson:"error_code,omitempty"`
	Before       map[string]interface{} `json:"before,omitempty" bson:"before,omitempty"`
	After        map[string]interface{} `json:"after,omitempty" bson:"after,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	Source       string                 `json:"source" bson:"source"`
}
