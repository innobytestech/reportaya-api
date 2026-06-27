package http

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"reportaya-api/internal/audit"
	usermodels "reportaya-api/internal/domain/user/models"
	"reportaya-api/internal/security/rbac"
	rbacperm "reportaya-api/internal/security/rbac/permissions"
	"reportaya-api/pkg/response"
)

type AuditFinalize func(status, resourceID string, before, after map[string]interface{})

// RequireStaffPermission validates JWT context and checks a staff RBAC permission.
// It writes the corresponding HTTP response when access is denied and returns false.
func RequireStaffPermission(c *fiber.Ctx, authorizer *rbac.Authorizer, permission rbacperm.Code) bool {
	claims := GetClaims(c)
	if claims == nil {
		_ = response.Unauthorized(c, "unauthorized", nil)
		return false
	}
	if authorizer == nil {
		_ = response.InternalError(c, errors.New("rbac authorizer not configured"))
		return false
	}
	userID, err := claims.UserID()
	if err != nil {
		_ = response.Unauthorized(c, "unauthorized", err)
		return false
	}
	ok, err := authorizer.Can(c.Context(), userID, usermodels.RealmStaff, permission)
	if err != nil {
		_ = response.InternalError(c, err)
		return false
	}
	if !ok {
		_ = response.Fail(c, fiber.StatusForbidden, "MISSING_PERMISSION", fmt.Sprintf("forbidden: missing permission %s", permission), nil)
		return false
	}
	return true
}

// EmitAudit sends an audit event enriched with request metadata.
func EmitAudit(c *fiber.Ctx, emitter *audit.Emitter, log *zerolog.Logger, evt audit.Event) {
	if emitter == nil {
		return
	}
	audit.EnrichFromFiber(c, &evt)
	if evt.Status != "success" {
		code := c.Response().StatusCode()
		if code <= 0 {
			code = 500
		}
		evt.ErrorCode = fmt.Sprintf("HTTP_%d", code)
	}
	if err := emitter.Emit(c.Context(), evt); err != nil && log != nil {
		log.Error().Err(err).Str("action", evt.Action).Msg("audit emit failed")
	}
}

func WithAudit(c *fiber.Ctx, emitter *audit.Emitter, log *zerolog.Logger, action, resourceType string) AuditFinalize {
	return func(status, resourceID string, before, after map[string]interface{}) {
		metadata := map[string]interface{}{}
		if before != nil {
			metadata["before"] = before
		}
		if after != nil {
			metadata["after"] = after
		}
		if len(metadata) == 0 {
			metadata = nil
		}
		EmitAudit(c, emitter, log, audit.Event{
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Status:       status,
			Metadata:     metadata,
		})
	}
}
