// Package repositories implements the persistence layer for the category domain.
package repositories

import (
	"context"

	"gorm.io/gorm"

	"reportaya-api/internal/domain/category/contracts"
	"reportaya-api/internal/domain/category/models"
)

// categoryRepository is the concrete GORM-backed implementation of CategoryRepository.
// It interacts with the PostgreSQL categories table and handles soft-delete logic.
type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository constructs a CategoryRepository backed by GORM.
// The GORM instance is injected to enable testing with mocks and to centralize DB configuration.
func NewCategoryRepository(db *gorm.DB) contracts.CategoryRepository {
	return &categoryRepository{db: db}
}

// ListActive returns all categories with is_active=true, ordered by name ASC (R4, R5).
// GORM's soft-delete guard on DeletedAt automatically excludes logically deleted rows.
func (r *categoryRepository) ListActive(ctx context.Context) ([]models.Category, error) {
	var out []models.Category
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&out).Error
	return out, err
}
