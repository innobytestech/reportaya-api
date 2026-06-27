// Package contracts defines the interfaces that decouple domain layers in the category domain.
// Implementations of these contracts are injected at startup via the DI container.
package contracts

import (
	"context"

	"reportaya-api/internal/domain/category/dtos"
	"reportaya-api/internal/domain/category/models"
)

// CategoryRepository abstracts persistence operations for the category domain.
// Implementations interact with the PostgreSQL categories table via an ORM (e.g., GORM).
type CategoryRepository interface {
	// ListActive returns all categories with is_active=true, ordered by name ASC.
	// The result excludes logically deleted rows (soft-delete via deleted_at).
	// Satisfies R4 (exclusion without physical deletion) and R5 (deterministic ordering).
	ListActive(ctx context.Context) ([]models.Category, error)
}

// CategoryService encapsulates the business logic for the category domain.
// It bridges the HTTP transport layer (handlers) and persistence (repository),
// transforming domain entities to public DTOs.
type CategoryService interface {
	// ListActive returns the public DTO list of active categories, ordered by name ASC.
	// The list is mapped to exclude internal fields (R3), ensuring a stable public contract.
	ListActive(ctx context.Context) ([]dtos.CategoryResponse, error)
}
