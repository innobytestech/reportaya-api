package http

import (
	"strings"
	"time"

	"reportaya-api/internal/config"
	"reportaya-api/internal/security/ratelimit"
	"reportaya-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// RateLimitLoginMiddleware enforces rate limiting on login attempts by IP.
// Blocks after N failed attempts for a duration.
func RateLimitLoginMiddleware(limiter *ratelimit.RateLimiter, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := getClientIP(c)
		blockKey := ratelimit.BuildBlockKey("login", ip)
		attemptsKey := ratelimit.BuildKey("login_attempts", ip)

		// Capture request context before c.Next() — after Next() it may be closed
		ctx := c.Context()

		// Check if IP is blocked
		blocked, err := limiter.IsBlocked(ctx, blockKey)
		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return response.Fail(c, fiber.StatusInternalServerError, "RATE_LIMIT_ERROR", "rate limit check failed", err)
		}
		if blocked {
			c.Status(fiber.StatusTooManyRequests)
			return response.Fail(c, fiber.StatusTooManyRequests, "IP_BLOCKED", "too many failed attempts, try again later", nil)
		}

		// Continue to handler, then check if login was successful
		err = c.Next()

		// If login failed (4xx or 5xx), increment failed attempts counter
		if c.Response().StatusCode() >= 400 {
			ok, _, _ := limiter.CheckAndIncrement(ctx, attemptsKey, cfg.RateLimit.LoginAttempts, cfg.RateLimit.LoginBlockWindow)
			if !ok {
				// Max attempts exceeded, block IP
				_ = limiter.Block(ctx, blockKey, cfg.RateLimit.LoginBlockWindow)
				_ = limiter.ResetCounter(ctx, attemptsKey)
			}
		} else {
			// Login successful, reset attempts
			_ = limiter.ResetCounter(ctx, attemptsKey)
		}

		return err
	}
}

// getClientIP extracts the real client IP from Fiber context.
// Use Fiber's trusted proxy resolution only.
func getClientIP(c *fiber.Ctx) string {
	return c.IP()
}

// RateLimitGeneralMiddleware enforces global rate limiting.
// For authenticated requests it uses user_id as the rate limit key;
// for unauthenticated requests it falls back to IP.
func RateLimitGeneralMiddleware(limiter *ratelimit.RateLimiter, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if shouldSkipGlobalRateLimit(c) {
			return c.Next()
		}

		keyBuilder := ratelimit.BuildKey("general", getClientIP(c))
		limit := cfg.RateLimit.GeneralRPM

		if claims := GetClaims(c); claims != nil {
			if userID, err := claims.UserID(); err == nil {
				keyBuilder = ratelimit.BuildKey("general_auth", userID.String())
				limit = cfg.RateLimit.AuthRPM
			}
		}

		ctx := c.Context()
		ok, _, err := limiter.CheckAndIncrement(ctx, keyBuilder, limit, time.Minute)
		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return response.Fail(c, fiber.StatusInternalServerError, "RATE_LIMIT_ERROR", "rate limit check failed", err)
		}

		if !ok {
			c.Status(fiber.StatusTooManyRequests)
			return response.Fail(c, fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many requests", nil)
		}

		return c.Next()
	}
}

func shouldSkipGlobalRateLimit(c *fiber.Ctx) bool {
	if c.Method() == fiber.MethodOptions {
		return true
	}

	path := c.Path()
	if path == "/health" || path == "/ready" {
		return true
	}

	return strings.HasSuffix(path, "/notifications/poll") || strings.HasSuffix(path, "/notifications/stream")
}

// RateLimitRefreshMiddleware enforces rate limiting for refresh token endpoint (per IP).
func RateLimitRefreshMiddleware(limiter *ratelimit.RateLimiter, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := getClientIP(c)
		key := ratelimit.BuildKey("refresh", ip)

		ctx := c.Context()
		ok, _, err := limiter.CheckAndIncrement(ctx, key, cfg.RateLimit.RefreshRPM, time.Minute)
		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return response.Fail(c, fiber.StatusInternalServerError, "RATE_LIMIT_ERROR", "rate limit check failed", err)
		}

		if !ok {
			c.Status(fiber.StatusTooManyRequests)
			return response.Fail(c, fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many refresh requests", nil)
		}

		return c.Next()
	}
}
