// Package handlers implements the HTTP transport layer for the category domain.
package handlers

import (
	"github.com/gofiber/fiber/v2"

	"reportaya-api/internal/domain/category/contracts"
	"reportaya-api/pkg/response"
)

// CategoryHandler handles HTTP requests for the category domain.
// It translates HTTP requests/responses and delegates business logic to the service layer.
type CategoryHandler struct {
	svc contracts.CategoryService
}

// NewCategoryHandler constructs a CategoryHandler with the given service.
// The service is injected to decouple the HTTP layer from domain logic.
func NewCategoryHandler(svc contracts.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// List handles GET /api/categories — returns active categories (R2, R3).
func (h *CategoryHandler) List(c *fiber.Ctx) error {
	cats, err := h.svc.ListActive(c.Context())
	if err != nil {
		return response.InternalError(c, err)
	}
	return response.Success(c, cats)
}
