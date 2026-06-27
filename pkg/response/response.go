package response

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// APIResponse es la estructura unificada de todas las respuestas HTTP (The Envelope).
type APIResponse struct {
	Ok      bool        `json:"ok"`
	Status  int         `json:"status"`
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Results interface{} `json:"results,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success responde con status 200 y payload.
func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(APIResponse{
		Ok:      true,
		Status:  fiber.StatusOK,
		Results: data,
		Message: "Operation completed successfully",
	})
}

// Created responde con status 201 (recursos creados).
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Ok:      true,
		Status:  fiber.StatusCreated,
		Results: data,
		Message: "Resource created successfully",
	})
}

// Fail respuesta genérica de error.
// code: Código interno para frontend (ej: "INVALID_EMAIL")
// msg: Mensaje legible
// err: Error técnico (opcional, para logs o debug)
func Fail(c *fiber.Ctx, status int, code, msg string, err error) error {
	errStr := ""
	if err != nil && isDebugEnv() {
		errStr = err.Error()
	}
	return c.Status(status).JSON(APIResponse{
		Ok:      false,
		Status:  status,
		Code:    code,
		Message: msg,
		Error:   errStr,
	})
}

func isDebugEnv() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	switch env {
	case "development", "dev", "local", "test":
		return true
	default:
		return false
	}
}

// BadRequest 400.
func BadRequest(c *fiber.Ctx, msg string, err error) error {
	return Fail(c, fiber.StatusBadRequest, "BAD_REQUEST", msg, err)
}

// NotFound 404.
func NotFound(c *fiber.Ctx, msg string, err error) error {
	return Fail(c, fiber.StatusNotFound, "NOT_FOUND", msg, err)
}

// Unauthorized 401.
func Unauthorized(c *fiber.Ctx, msg string, err error) error {
	return Fail(c, fiber.StatusUnauthorized, "UNAUTHORIZED", msg, err)
}

// Forbidden 403.
func Forbidden(c *fiber.Ctx, msg string, err error) error {
	return Fail(c, fiber.StatusForbidden, "FORBIDDEN", msg, err)
}

// Conflict 409.
func Conflict(c *fiber.Ctx, msg string, err error) error {
	return Fail(c, fiber.StatusConflict, "CONFLICT", msg, err)
}

// InternalError 500.
func InternalError(c *fiber.Ctx, err error) error {
	return Fail(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal Server Error", err)
}
