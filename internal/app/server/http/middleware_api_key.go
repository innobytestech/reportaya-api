package http

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"reportaya-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

func ApiKeyMiddleware(apiKey string) fiber.Handler {
	expected := sha256.Sum256([]byte(apiKey))
	return func(c *fiber.Ctx) error {
		providedKey := strings.TrimSpace(c.Get("X-API-KEY"))
		if providedKey == "" {
			return response.Unauthorized(c, "missing API key", nil)
		}
		// Hash both before comparing so the constant-time compare runs over
		// fixed-length digests; comparing raw keys leaks the key length via timing.
		provided := sha256.Sum256([]byte(providedKey))
		if subtle.ConstantTimeCompare(provided[:], expected[:]) != 1 {
			return response.Unauthorized(c, "invalid API key", nil)
		}

		return c.Next()
	}
}
