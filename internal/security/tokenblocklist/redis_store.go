package tokenblocklist

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a Redis-backed token blacklist store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Add stores the token ID with TTL.
func (s *RedisStore) Add(ctx context.Context, tokenID string, ttl time.Duration) error {
	return s.client.Set(ctx, BuildKey(tokenID), "1", ttl).Err()
}

// Exists checks if token ID is blacklisted.
func (s *RedisStore) Exists(ctx context.Context, tokenID string) (bool, error) {
	count, err := s.client.Exists(ctx, BuildKey(tokenID)).Result()
	return count > 0, err
}
