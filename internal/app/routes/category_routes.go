// Package routes wires HTTP endpoints to their handlers.
package routes

import (
	"github.com/gofiber/fiber/v2"

	"reportaya-api/internal/domain/category/handlers"
)

// RegisterCategories mounts the public category endpoints onto the given router.
// The router is expected to be the /api/categories group.
// Currently unauthenticated (no auth middleware) per R2 (public catalog for MVP);
// authentication and rate-limiting will be added by B3 (API security feature).
// GET / returns the list of active categories in the standard APIResponse envelope.
func RegisterCategories(router fiber.Router, h *handlers.CategoryHandler) {
	router.Get("/", h.List)
}
