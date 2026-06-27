package errors

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// FiberErrorHandler returns a fiber.Config-compatible error handler that
// translates any error returned from a handler into a stable AppError JSON
// response using the registry.
//
// The optional log callback receives a (path, err) pair for each 5xx error
// so the caller can route it through zerolog/otel as appropriate.
func FiberErrorHandler(log func(path string, err error)) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// 1. Already an AppError.
		var appErr AppError
		if errors.As(err, &appErr) {
			if appErr.HTTPStatus >= 500 && log != nil {
				log(c.Path(), err)
			}
			return Respond(c, appErr)
		}
		// 2. Fiber-native error.
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			if fiberErr.Code >= 500 && log != nil {
				log(c.Path(), err)
			}
			switch fiberErr.Code {
			case fiber.StatusNotFound, fiber.StatusMethodNotAllowed:
				return RespondWithCode(c, "MS-GENGT01", err)
			case fiber.StatusRequestEntityTooLarge, fiber.StatusBadRequest:
				return RespondWithCode(c, "MS-GENVA02", err)
			case fiber.StatusUnauthorized:
				return RespondWithCode(c, "MS-GENAU01", err)
			case fiber.StatusForbidden:
				return RespondWithCode(c, "MS-GENAU02", err)
			}
			return RespondWithCode(c, "MS-GENRL01", err)
		}
		// 3. Match against registry.
		if def, ok := Match(err); ok {
			return RespondWithCode(c, def.Code, err)
		}
		// 4. Generic fallback.
		if log != nil {
			log(c.Path(), err)
		}
		return RespondWithCode(c, "MS-GENRL01", err)
	}
}
