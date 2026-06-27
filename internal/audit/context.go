package audit

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	jwtpkg "reportaya-api/internal/security/jwt"
)

// EnrichFromFiber injects request and actor metadata from HTTP context.
// It populates fields like RequestID, IP, UserAgent, ActorUserID, ActorRealm, and TenantID
func EnrichFromFiber(c *fiber.Ctx, evt *Event) {
	if evt == nil {
		return
	}
	if evt.RequestID == "" {
		if requestID := c.Locals("requestid"); requestID != nil {
			evt.RequestID = fmt.Sprint(requestID)
		}
	}
	if evt.IP == "" {
		evt.IP = c.IP()
	}
	if evt.UserAgent == "" {
		evt.UserAgent = c.Get("User-Agent")
	}
	if evt.ActorUserID == "" {
		if userID := c.Locals("user_id"); userID != nil {
			evt.ActorUserID = fmt.Sprint(userID)
		}
	}
	claims, _ := c.Locals("jwt_claims").(*jwtpkg.Claims)
	if claims != nil {
		if evt.ActorRealm == "" {
			evt.ActorRealm = string(claims.Realm)
		}
		if evt.TenantID == "" && claims.CustomerID != nil {
			evt.TenantID = claims.CustomerID.String()
		}
	}
}
