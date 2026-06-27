package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"reportaya-api/internal/audit"
)

func AuditDeniedMiddleware(emitter *audit.Emitter, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		statusCode := c.Response().StatusCode()
		if statusCode == 401 || statusCode == 403 {
			evt := audit.Event{
				Action:       "auth.access.denied",
				ResourceType: "endpoint",
				ResourceID:   c.Path(),
				Status:       "denied",
				Metadata: map[string]interface{}{
					"method":    c.Method(),
					"endpoint":  c.Path(),
					"http_code": statusCode,
				},
			}
			EmitAudit(c, emitter, log, evt)
		}

		if statusCode >= 500 {
			evt := audit.Event{
				Action:       "system.error",
				ResourceType: "endpoint",
				ResourceID:   c.Path(),
				Status:       "failed",
				ErrorCode:    "HTTP_500",
				Metadata: map[string]interface{}{
					"method":    c.Method(),
					"endpoint":  c.Path(),
					"http_code": statusCode,
				},
			}
			EmitAudit(c, emitter, log, evt)
		}

		return err
	}
}
