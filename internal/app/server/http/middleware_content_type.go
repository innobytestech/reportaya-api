package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"reportaya-api/pkg/response"
)

// RequireJSON enforces Content-Type: application/json on mutating HTTP methods
// (POST, PUT, PATCH) only when a request body is present. GET, DELETE, OPTIONS
// and HEAD are skipped because they typically carry no body. Multipart/form-data
// requests (file uploads) are also allowed through.
func RequireJSON() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodDelete ||
			method == fiber.MethodOptions || method == fiber.MethodHead {
			return c.Next()
		}

		ct := strings.ToLower(c.Get("Content-Type"))

		// If no body is present, Content-Type is not required.
		if len(c.Body()) == 0 {
			return c.Next()
		}

		// Allow multipart uploads through (file upload endpoints).
		if strings.Contains(ct, "multipart/form-data") {
			return c.Next()
		}

		if !strings.Contains(ct, "application/json") {
			return response.BadRequest(c, "Content-Type must be application/json", nil)
		}

		return c.Next()
	}
}
