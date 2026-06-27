package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"reportaya-api/internal/security/sessionactivity"
)

// HeaderBackgroundRequest signals that the request is a non-interactive call
// (background polling, push subscriptions, etc.) and must NOT count as user
// activity for the purposes of the idle session timeout.
const HeaderBackgroundRequest = "X-Background-Request"

// TrackActivity records that the authenticated user made a request "now" so
// the AuthService can enforce an idle session timeout. Must be mounted AFTER
// RequireAuth so c.Locals(CtxKeyUserID) is populated.
//
// The middleware is best-effort: a Redis failure is logged but does not break
// the request — failing closed here would convert an observability outage
// into an availability one.
func TrackActivity(store sessionactivity.Store, idleTimeout time.Duration, throttle time.Duration, log *zerolog.Logger) fiber.Handler {
	if store == nil || idleTimeout <= 0 {
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	activityTTL := idleTimeout + 5*time.Minute
	return func(c *fiber.Ctx) error {
		if c.Get(HeaderBackgroundRequest) != "" {
			return c.Next()
		}
		userID, ok := c.Locals(CtxKeyUserID).(uuid.UUID)
		if !ok || userID == uuid.Nil {
			return c.Next()
		}
		if err := store.Touch(c.Context(), userID.String(), activityTTL, throttle); err != nil && log != nil {
			log.Warn().Err(err).Str("user_id", userID.String()).Msg("session activity touch failed")
		}
		return c.Next()
	}
}
