package container

import (
	"github.com/gofiber/fiber/v2"

	"reportaya-api/internal/app/routes"
	apphttp "reportaya-api/internal/app/server/http"
)

// RegisterRoutes wires every route group. Add domain route registrars here as
// modules are built, following the pattern below: build the middleware chain for
// the group, then delegate to the per-domain routes.Setup* function.
func (ctn *Container) RegisterRoutes(app *fiber.App) {
	// System endpoints (liveness/readiness), unauthenticated.
	routes.RegisterSystem(app, ctn.DB)

	api := app.Group("/api")

	// Authenticated group example: JWT + session-activity tracking + JSON guard.
	// Replace/extend with real domain groups (admin, portal, ...) as needed.
	trackActivity := apphttp.TrackActivity(
		ctn.SessionActivity,
		ctn.Config.JWT.IdleSessionTimeout,
		ctn.Config.JWT.IdleActivityThrottle,
		ctn.Log,
	)
	authenticated := api.Group("/me",
		apphttp.RequireAuth(ctn.JWT, ctn.TokenBlocklist),
		trackActivity,
	)
	routes.RegisterMe(authenticated)
}
