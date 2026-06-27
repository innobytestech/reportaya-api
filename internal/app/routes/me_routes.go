package routes

import (
	"github.com/gofiber/fiber/v2"

	apphttp "reportaya-api/internal/app/server/http"
)

// RegisterMe is an example authenticated route group: it echoes the JWT claims
// of the current session. Use it as a template for real domain groups.
func RegisterMe(router fiber.Router) {
	router.Get("/", func(c *fiber.Ctx) error {
		claims := apphttp.GetClaims(c)
		if claims == nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.JSON(fiber.Map{"claims": claims})
	})
}
