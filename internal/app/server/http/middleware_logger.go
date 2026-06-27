package http

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// LoggerMiddleware logs requests in structured JSON (request_id, ip, latency, user_id if present).
func LoggerMiddleware(log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		chainErr := c.Next()
		latency := time.Since(start)
		ev := log.Info()
		if requestID := c.Locals("requestid"); requestID != nil {
			ev = ev.Str("request_id", requestID.(string))
		}
		if traceID := c.Locals("trace_id"); traceID != nil {
			ev = ev.Str("trace_id", traceID.(string))
		}
		ev = ev.Str("ip", c.IP()).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("latency_ms", latency)
		if userID := c.Locals(CtxKeyUserID); userID != nil {
			ev = ev.Str("user_id", fmt.Sprint(userID))
		}
		if chainErr != nil {
			ev = ev.Err(chainErr)
		}
		ev.Msg("request")
		return chainErr
	}
}
