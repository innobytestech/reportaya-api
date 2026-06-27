package errors

import (
	"github.com/gofiber/fiber/v2"

	"reportaya-api/pkg/response"
)

// Respond sends an HTTP response using the AppError's code and status.
// The internal error is only exposed in debug environments.
func Respond(c *fiber.Ctx, err AppError) error {
	return response.Fail(c, err.HTTPStatus, err.Code, err.PublicMsg, err.Internal)
}

// RespondWithCode looks up the registered error code and responds with it.
// If the code is not found, falls back to a 500 Internal Server Error.
func RespondWithCode(c *fiber.Ctx, code string, internal error) error {
	def, ok := Get(code)
	if !ok {
		// Unknown code - use generic internal error
		return response.Fail(c, 500, code, "Ocurrió un error inesperado", internal)
	}
	return response.Fail(c, def.HTTPStatus, def.Code, def.PublicMsg, internal)
}

// Handle is a convenience wrapper: if err is nil, returns nil.
// Otherwise, matches against the registry and responds with the matched code.
// If no match is found, uses the fallbackCode.
func Handle(c *fiber.Ctx, err error, fallbackCode string) error {
	if err == nil {
		return nil
	}
	appErr := FromError(err, fallbackCode)
	return Respond(c, appErr)
}

// HandleWithDefault is like Handle but uses MS-GENRL01 as fallback.
func HandleWithDefault(c *fiber.Ctx, err error) error {
	return Handle(c, err, "MS-GENRL01")
}
