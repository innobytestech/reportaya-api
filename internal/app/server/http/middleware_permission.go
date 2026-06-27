package http

import (
	"errors"
	"fmt"
	"strings"

	usermodels "reportaya-api/internal/domain/user/models"
	"reportaya-api/internal/security/rbac"
	rbacperm "reportaya-api/internal/security/rbac/permissions"
	"reportaya-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

func RequirePermission(authorizer *rbac.Authorizer, permission rbacperm.Code) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if authorizer == nil {
			return response.InternalError(c, errors.New("rbac authorizer not configured"))
		}
		if !RequireStaffPermission(c, authorizer, permission) {
			return nil
		}
		return c.Next()
	}
}

// RequireAnyPermission grants access when the authenticated staff user holds
// at least ONE of the supplied permissions. It is the OR-equivalent of
// RequirePermission and is used by endpoints that intentionally bridge two
// RBAC scopes (e.g. /admin/products serves both the admin catalog UI —
// `products.read` — and the POS cashier UI — `pos.sales.create` — see
// Feature #9 Q6).
//
// On deny it returns 403 MISSING_PERMISSION listing the alternatives so the
// frontend can render an actionable error. On an empty permission list it
// returns 500 because that is a wiring bug.
func RequireAnyPermission(authorizer *rbac.Authorizer, permissions ...rbacperm.Code) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if authorizer == nil {
			return response.InternalError(c, errors.New("rbac authorizer not configured"))
		}
		if len(permissions) == 0 {
			return response.InternalError(c, errors.New("RequireAnyPermission called with empty permission list"))
		}
		claims := GetClaims(c)
		if claims == nil {
			return response.Unauthorized(c, "unauthorized", nil)
		}
		userID, err := claims.UserID()
		if err != nil {
			return response.Unauthorized(c, "unauthorized", err)
		}
		for _, perm := range permissions {
			ok, err := authorizer.Can(c.Context(), userID, usermodels.RealmStaff, perm)
			if err != nil {
				return response.InternalError(c, err)
			}
			if ok {
				return c.Next()
			}
		}
		names := make([]string, 0, len(permissions))
		for _, p := range permissions {
			names = append(names, string(p))
		}
		return response.Fail(c, fiber.StatusForbidden, "MISSING_PERMISSION", fmt.Sprintf("forbidden: missing any of [%s]", strings.Join(names, ", ")), nil)
	}
}
