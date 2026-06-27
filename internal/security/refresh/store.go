package refresh

import (
	"context"
	"fmt"
	"time"
)

// Store defines refresh token persistence.
type Store interface {
	Save(ctx context.Context, tokenID string, ttl time.Duration, value string) error
	Exists(ctx context.Context, tokenID string) (bool, error)
	Delete(ctx context.Context, tokenID string) error
	// ConsumeAndRotate atomically deletes oldID and creates newID.
	// Returns true if oldID existed and was consumed; false if it was already consumed (replay).
	ConsumeAndRotate(ctx context.Context, oldID, newID string, newTTL time.Duration, value string) (bool, error)
}

// BuildKey builds a redis key for a refresh token.
func BuildKey(tokenID string) string {
	return fmt.Sprintf("refresh:%s", tokenID)
}
