package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter provides Redis-backed rate limiting for multiple strategies.
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a rate limiter connected to Redis.
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// CheckAndIncrement checks if a key has exceeded the limit, increments counter, returns (ok, remaining).
// If limit is exceeded, returns (false, 0).
func (rl *RateLimiter) CheckAndIncrement(ctx context.Context, key string, limit int, window time.Duration) (ok bool, remaining int, err error) {
	windowSeconds := int(window / time.Second)
	if windowSeconds < 1 {
		windowSeconds = 1
	}

	const script = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`

	res, err := rl.client.Eval(ctx, script, []string{key}, strconv.Itoa(windowSeconds)).Int64()
	if err != nil {
		return false, 0, err
	}

	if res > int64(limit) {
		return false, 0, nil
	}
	remaining = limit - int(res)
	return true, remaining, nil
}

// GetCounter retrieves the current counter value for a key.
func (rl *RateLimiter) GetCounter(ctx context.Context, key string) (int, error) {
	val, err := rl.client.Get(ctx, key).Int()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, err
}

// ResetCounter resets (deletes) a counter for a key.
func (rl *RateLimiter) ResetCounter(ctx context.Context, key string) error {
	return rl.client.Del(ctx, key).Err()
}

// IsBlocked checks if an IP/user is temporarily blocked.
func (rl *RateLimiter) IsBlocked(ctx context.Context, blockKey string) (bool, error) {
	exists, err := rl.client.Exists(ctx, blockKey).Result()
	return exists > 0, err
}

// Block temporarily blocks a key (IP/user).
func (rl *RateLimiter) Block(ctx context.Context, blockKey string, duration time.Duration) error {
	return rl.client.Set(ctx, blockKey, "1", duration).Err()
}

// BuildKey constructs a standardized Redis key for rate limiting.
func BuildKey(strategy, identifier string) string {
	return fmt.Sprintf("ratelimit:%s:%s", strategy, identifier)
}

// BuildBlockKey constructs a block key for rate limiting.
func BuildBlockKey(strategy, identifier string) string {
	return fmt.Sprintf("ratelimit:blocked:%s:%s", strategy, identifier)
}
