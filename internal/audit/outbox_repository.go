package audit

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const (
	outboxStatusPending    = "pending"
	outboxStatusProcessing = "processing"
	outboxStatusProcessed  = "processed"
	outboxStatusDead       = "dead"
)

// OutboxMessage is a persisted outbox row.
type OutboxMessage struct {
	ID         string    `gorm:"column:id"`
	OccurredAt time.Time `gorm:"column:occurred_at"`
	EventType  string    `gorm:"column:event_type"`
	Payload    []byte    `gorm:"column:payload_json"`
	Status     string    `gorm:"column:status"`
	Attempts   int       `gorm:"column:attempts"`
}

func (OutboxMessage) TableName() string { return "audit_outbox" }

// OutboxRepository handles audit outbox persistence.
type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Enqueue(ctx context.Context, id string, occurredAt time.Time, eventType string, payload []byte) error {
	q := `
		INSERT INTO audit_outbox (id, occurred_at, event_type, payload_json, status, attempts, next_retry_at, created_at, updated_at)
		VALUES (?, ?, ?, ?::jsonb, 'pending', 0, NOW(), NOW(), NOW())`
	return r.db.WithContext(ctx).Exec(q, id, occurredAt, eventType, string(payload)).Error
}

func (r *OutboxRepository) ClaimBatch(ctx context.Context, batchSize int) ([]OutboxMessage, error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	var rows []OutboxMessage
	q := `
		WITH picked AS (
			SELECT id
			FROM audit_outbox
			WHERE status = 'pending'
			  AND next_retry_at <= NOW()
			ORDER BY created_at ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		UPDATE audit_outbox o
		SET status = 'processing',
		    updated_at = NOW()
		FROM picked
		WHERE o.id = picked.id
		RETURNING o.id, o.occurred_at, o.event_type, o.payload_json, o.status, o.attempts`
	err := r.db.WithContext(ctx).Raw(q, batchSize).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id string) error {
	q := `
		UPDATE audit_outbox
		SET status = 'processed',
		    processed_at = NOW(),
		    updated_at = NOW()
		WHERE id = ?`
	return r.db.WithContext(ctx).Exec(q, id).Error
}

func (r *OutboxRepository) MarkRetry(ctx context.Context, id string, attempts int, nextRetryAt time.Time, errMsg string, dead bool) error {
	status := outboxStatusPending
	if dead {
		status = outboxStatusDead
	}
	q := `
		UPDATE audit_outbox
		SET status = ?,
		    attempts = ?,
		    next_retry_at = ?,
		    last_error = ?,
		    updated_at = NOW()
		WHERE id = ?`
	return r.db.WithContext(ctx).Exec(q, status, attempts, nextRetryAt, errMsg, id).Error
}
