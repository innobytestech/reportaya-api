package refresh

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a Redis-backed refresh token store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Save stores a refresh token with TTL.
func (s *RedisStore) Save(ctx context.Context, tokenID string, ttl time.Duration, value string) error {
	return s.client.Set(ctx, BuildKey(tokenID), value, ttl).Err()
}

// Exists checks if refresh token exists.
func (s *RedisStore) Exists(ctx context.Context, tokenID string) (bool, error) {
	count, err := s.client.Exists(ctx, BuildKey(tokenID)).Result()
	return count > 0, err
}

// Delete removes refresh token.
func (s *RedisStore) Delete(ctx context.Context, tokenID string) error {
	return s.client.Del(ctx, BuildKey(tokenID)).Err()
}

// consumeAndRotateScript atomically deletes the old key and creates the new one.
// Returns 1 if old key was deleted (consumed), 0 if it did not exist (replay).
var consumeAndRotateScript = redis.NewScript(`
local old_key = KEYS[1]
local new_key = KEYS[2]
local ttl_ms  = tonumber(ARGV[1])
local value   = ARGV[2]
local deleted = redis.call('DEL', old_key)
if deleted == 0 then
  return 0
end
redis.call('SET', new_key, value, 'PX', ttl_ms)
return 1
`)

// ConsumeAndRotate atomically consumes oldID and creates newID with given TTL.
// Returns true if oldID was consumed, false if already gone (concurrent replay).
func (s *RedisStore) ConsumeAndRotate(ctx context.Context, oldID, newID string, newTTL time.Duration, value string) (bool, error) {
	ttlMs := int64(newTTL / time.Millisecond)
	result, err := consumeAndRotateScript.Run(ctx, s.client,
		[]string{BuildKey(oldID), BuildKey(newID)},
		ttlMs, value,
	).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
