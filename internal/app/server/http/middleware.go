package http

import (
	"strings"

	jwtpkg "reportaya-api/internal/security/jwt"
	"reportaya-api/internal/security/tokenblocklist"
	"reportaya-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// Context keys for auth data (avoid collisions).
const (
	CtxKeyClaims = "jwt_claims"
	CtxKeyUserID = "user_id"
)

// RequireAuth extracts Bearer token, verifies JWT, sets claims in context, checks blacklist.
func RequireAuth(jwt *jwtpkg.SignerVerifier, blocklist tokenblocklist.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return response.Unauthorized(c, "missing authorization header", nil)
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			return response.Unauthorized(c, "invalid authorization format", nil)
		}
		token := strings.TrimPrefix(auth, prefix)
		claims, err := jwt.Verify(token, jwtpkg.TokenTypeAccess)
		if err != nil {
			return response.Unauthorized(c, "invalid or expired token", err)
		}
		if blocklist != nil && claims.ID != "" {
			blocked, err := blocklist.Exists(c.Context(), claims.ID)
			if err != nil {
				return response.InternalError(c, err)
			}
			if blocked {
				return response.Unauthorized(c, "token revoked", nil)
			}
		}
		c.Locals(CtxKeyClaims, claims)
		userID, _ := claims.UserID()
		c.Locals(CtxKeyUserID, userID)
		return c.Next()
	}
}

// RequireRealm ensures the authenticated user has the given realm (staff or customer).
func RequireRealm(realm jwtpkg.Realm) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cl, ok := c.Locals(CtxKeyClaims).(*jwtpkg.Claims)
		if !ok || cl == nil {
			return response.Unauthorized(c, "unauthorized", nil)
		}
		if cl.Realm != realm {
			return response.Forbidden(c, "forbidden: wrong realm", nil)
		}
		return c.Next()
	}
}

// GetClaims returns JWT claims from context (nil if not set).
func GetClaims(c *fiber.Ctx) *jwtpkg.Claims {
	cl, _ := c.Locals(CtxKeyClaims).(*jwtpkg.Claims)
	return cl
}
