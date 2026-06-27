package sessionactivity

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store on top of a Redis client.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a Redis-backed activity store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// touchScript wins a per-user throttle lock and, only if it wins, updates the
// activity timestamp. Returns 1 on write, 0 when throttled.
var touchScript = redis.NewScript(`
local activity_key = KEYS[1]
local throttle_key = KEYS[2]
local now_ms       = ARGV[1]
local activity_ttl_ms = tonumber(ARGV[2])
local throttle_ttl_ms = tonumber(ARGV[3])
local locked = redis.call('SET', throttle_key, '1', 'NX', 'PX', throttle_ttl_ms)
if not locked then
  return 0
end
redis.call('SET', activity_key, now_ms, 'PX', activity_ttl_ms)
return 1
`)

// Touch updates the activity timestamp if the per-user throttle window has elapsed.
func (s *RedisStore) Touch(ctx context.Context, userID string, activityTTL time.Duration, throttle time.Duration) error {
	if userID == "" {
		return nil
	}
	nowMs := time.Now().UTC().UnixMilli()
	_, err := touchScript.Run(ctx, s.client,
		[]string{BuildActivityKey(userID), BuildThrottleKey(userID)},
		strconv.FormatInt(nowMs, 10),
		activityTTL.Milliseconds(),
		throttle.Milliseconds(),
	).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}

// LastActivity returns the last recorded activity time for the user.
func (s *RedisStore) LastActivity(ctx context.Context, userID string) (time.Time, bool, error) {
	if userID == "" {
		return time.Time{}, false, nil
	}
	raw, err := s.client.Get(ctx, BuildActivityKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false, err
	}
	return time.UnixMilli(ms).UTC(), true, nil
}
