// Package errors provides a unified enterprise error registry for reportaya-api.
//
// All domain sentinel errors are mapped to stable, versioned error codes
// following the format: MS-{DOM}{OP}{SEQ}
//
//	MS    = Muñoz Solutions (fixed prefix)
//	DOM   = Domain code (AUTH, USR, POS, INV, CRD, etc.)
//	OP    = Operation code (GT=Get, CR=Create, UP=Update, DL=Delete, etc.)
//	SEQ   = Sequential number (01-99)
//
// Example: MS-POSCR01 = Muñoz Solutions, POS domain, Create operation, sequence 01
//
// Error codes are NEVER changed once published. New codes are added, old codes
// are deprecated but remain in the registry for backward compatibility.
package errors

import (
	"errors"
	"fmt"
)

// AppError is the enterprise error type returned by handlers.
// It contains a stable error code for the frontend, a public-safe message,
// and the original internal error (never exposed in production responses).
type AppError struct {
	Code       string // Stable code like "MS-POSCR01"
	PublicMsg  string // Safe for frontend (no internal details)
	Internal   error  // Original wrapped error (only visible in dev mode)
	HTTPStatus int    // HTTP status code
	Category   string // validation, conflict, internal, unavailable, auth, not_found
}

// Error implements the error interface.
func (e AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.PublicMsg, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.PublicMsg)
}

// Unwrap returns the internal error for errors.Is/As chaining.
func (e AppError) Unwrap() error {
	return e.Internal
}

// Is allows matching against other AppError by code.
func (e AppError) Is(target error) bool {
	t, ok := target.(AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Category constants.
const (
	CategoryValidation  = "validation"
	CategoryConflict    = "conflict"
	CategoryInternal    = "internal"
	CategoryUnavailable = "unavailable"
	CategoryAuth        = "auth"
	CategoryNotFound    = "not_found"
	CategoryRateLimit   = "rate_limit"
	CategoryIdempotency = "idempotency"
)

// New creates a new AppError from a registered code.
// If the code is not found in the registry, it returns an internal error.
func New(code string, internal error, context ...string) AppError {
	def, ok := registry[code]
	if !ok {
		// Code not registered - create a generic internal error
		return AppError{
			Code:       code,
			PublicMsg:  "Ocurrió un error inesperado",
			Internal:   internal,
			HTTPStatus: 500,
			Category:   CategoryInternal,
		}
	}

	pubMsg := def.PublicMsg
	if len(context) > 0 && context[0] != "" {
		pubMsg = def.PublicMsg
		// Context is appended to internal error, not public message
	}

	intErr := internal
	if len(context) > 0 && context[0] != "" && internal != nil {
		intErr = fmt.Errorf("%s: %w", context[0], internal)
	} else if len(context) > 0 && context[0] != "" {
		intErr = errors.New(context[0])
	}

	return AppError{
		Code:       code,
		PublicMsg:  pubMsg,
		Internal:   intErr,
		HTTPStatus: def.HTTPStatus,
		Category:   def.Category,
	}
}

// Newf creates a new AppError with formatted context.
func Newf(code string, internal error, msgFormat string, args ...interface{}) AppError {
	var ctx string
	if msgFormat != "" {
		ctx = fmt.Sprintf(msgFormat, args...)
	}
	return New(code, internal, ctx)
}

// FromError attempts to match an error against the registry and returns an AppError.
// If no match is found, returns a generic internal error.
func FromError(err error, fallbackCode string) AppError {
	if err == nil {
		return AppError{}
	}

	// Already an AppError
	var appErr AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Try to match against registry
	if def, ok := Match(err); ok {
		return AppError{
			Code:       def.Code,
			PublicMsg:  def.PublicMsg,
			Internal:   err,
			HTTPStatus: def.HTTPStatus,
			Category:   def.Category,
		}
	}

	// Fallback
	return New(fallbackCode, err)
}

// Match attempts to find a registry definition for the given error.
// It checks both sentinel errors (errors.Is) and typed errors (errors.As).
func Match(err error) (ErrorDefinition, bool) {
	if err == nil {
		return ErrorDefinition{}, false
	}

	for _, def := range registry {
		if def.Sentinel != nil && errors.Is(err, def.Sentinel) {
			return def, true
		}
		if def.TypeMatcher != nil && def.TypeMatcher(err) {
			return def, true
		}
	}
	return ErrorDefinition{}, false
}
