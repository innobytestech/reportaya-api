package tokenblocklist

import (
	"context"
	"fmt"
	"time"
)

// Store defines a token blacklist store.
type Store interface {
	Add(ctx context.Context, tokenID string, ttl time.Duration) error
	Exists(ctx context.Context, tokenID string) (bool, error)
}

// BuildKey builds a redis key for a blacklisted token.
func BuildKey(tokenID string) string {
	return fmt.Sprintf("token:blacklist:%s", tokenID)
}
