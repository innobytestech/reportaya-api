// Package routes contains per-concern HTTP route registrars. Each file exposes
// Register*/Setup* functions that bind a domain's handlers onto a fiber.Router
// (a group that already carries the right middleware chain).
package routes

import (
	"github.com/gofiber/fiber/v2"

	"reportaya-api/internal/persistence/postgres"
)

// RegisterSystem registers liveness and readiness probes.
//
//	GET /health -> 200 always (process is up)
//	GET /ready  -> 200 if dependencies (Postgres) are reachable, else 503
func RegisterSystem(router fiber.Router, db *postgres.DB) {
	router.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	router.Get("/ready", func(c *fiber.Ctx) error {
		if err := db.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unavailable",
				"db":     "down",
			})
		}
		return c.JSON(fiber.Map{"status": "ready", "db": "up"})
	})
}
