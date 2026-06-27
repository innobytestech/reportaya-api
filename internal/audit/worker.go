package audit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog"
)

// Worker drains outbox and writes to sink.
type Worker struct {
	repo        *OutboxRepository
	sink        Sink
	log         *zerolog.Logger
	pollEvery   time.Duration
	batchSize   int
	maxAttempts int
}

func NewWorker(repo *OutboxRepository, sink Sink, log *zerolog.Logger, pollEvery time.Duration, batchSize, maxAttempts int) *Worker {
	if pollEvery <= 0 {
		pollEvery = 2 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	return &Worker{repo: repo, sink: sink, log: log, pollEvery: pollEvery, batchSize: batchSize, maxAttempts: maxAttempts}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.repo == nil || w.sink == nil {
		return
	}
	ticker := time.NewTicker(w.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.drainOnce(ctx)
		}
	}
}

func (w *Worker) drainOnce(ctx context.Context) {
	startedAt := time.Now()
	messages, err := w.repo.ClaimBatch(ctx, w.batchSize)
	if err != nil {
		if w.log != nil {
			w.log.Error().Err(err).Msg("audit outbox claim failed")
		}
		return
	}
	if len(messages) == 0 {
		return
	}

	processed := 0
	succeeded := 0
	failed := 0
	dead := 0
	for i := range messages {
		msg := messages[i]
		processed++
		if err := w.processOne(ctx, msg); err != nil {
			failed++
			if msg.Attempts+1 >= w.maxAttempts {
				dead++
			}
			if w.log != nil {
				w.log.Error().Err(err).Str("outbox_id", msg.ID).Msg("audit outbox process failed")
			}
			continue
		}
		succeeded++
	}

	if w.log != nil {
		w.log.Info().
			Int("claimed", len(messages)).
			Int("processed", processed).
			Int("succeeded", succeeded).
			Int("failed", failed).
			Int("dead", dead).
			Dur("elapsed", time.Since(startedAt)).
			Msg("audit outbox batch processed")
	}
}

func (w *Worker) processOne(ctx context.Context, msg OutboxMessage) error {
	var evt Event
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		_ = w.repo.MarkRetry(ctx, msg.ID, msg.Attempts+1, time.Now().UTC().Add(2*time.Minute), "invalid payload", true)
		return err
	}
	if evt.EventID == "" {
		evt.EventID = msg.ID
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = msg.OccurredAt
	}
	err := w.sink.Write(ctx, evt)
	if err == nil {
		return w.repo.MarkProcessed(ctx, msg.ID)
	}
	attempts := msg.Attempts + 1
	dead := attempts >= w.maxAttempts
	nextRetry := time.Now().UTC().Add(retryBackoff(attempts))
	markErr := w.repo.MarkRetry(ctx, msg.ID, attempts, nextRetry, err.Error(), dead)
	if markErr != nil {
		return errors.Join(err, markErr)
	}
	return err
}

func retryBackoff(attempt int) time.Duration {
	// attempt is bounded to [1,6]; shift is safe within int range.
	var shift uint
	switch {
	case attempt < 1:
		shift = 1
	case attempt > 6:
		shift = 6
	default:
		shift = uint(attempt) //nolint:gosec // bounded above to 6
	}
	return time.Duration(int64(1)<<shift) * time.Second
}
