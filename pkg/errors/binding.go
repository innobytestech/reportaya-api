package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"reportaya-api/pkg/validate"
)

// BindAndValidate parses the Fiber request body into payload and runs the
// struct-tag validation chain. It returns an AppError suitable for passing to
// Respond; internal details are only surfaced in debug environments via
// AppError.Internal (handled by pkg/response).
//
// Usage in handlers:
//
//	var req dtos.XxxCreateRequest
//	if err := errors.BindAndValidate(c, &req); err != nil {
//	    return errors.Respond(c, err.(AppError))
//	}
func BindAndValidate(c *fiber.Ctx, payload any) error {
	if err := c.BodyParser(payload); err != nil {
		return New("MS-GENVA02", err)
	}
	return ValidateStruct(payload)
}

// ValidateStruct runs the struct-tag validation chain on an already-populated
// payload (e.g. one filled from multipart form fields rather than JSON body).
// Returns the same MS-GENVA01 / MS-GENVA02 contract as BindAndValidate.
func ValidateStruct(payload any) error {
	if err := validate.V().Struct(payload); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			parts := make([]string, 0, len(ve))
			for _, fe := range ve {
				parts = append(parts, validate.TranslateFieldError(fe))
			}
			return New("MS-GENVA01", fmt.Errorf("%s", strings.Join(parts, "; ")))
		}
		return New("MS-GENVA01", err)
	}
	return nil
}
