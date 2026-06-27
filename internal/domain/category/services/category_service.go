// Package services implements the category domain use cases.
package services

import (
	"context"

	"reportaya-api/internal/domain/category/contracts"
	"reportaya-api/internal/domain/category/dtos"
)

// categoryService is the concrete implementation of CategoryService.
// It depends on a CategoryRepository for data access and handles the mapping
// from persistence entities to public DTOs.
type categoryService struct {
	repo contracts.CategoryRepository
}

// NewCategoryService constructs a CategoryService backed by the given repository.
// The repository is injected to decouple this service from persistence details.
func NewCategoryService(repo contracts.CategoryRepository) contracts.CategoryService {
	return &categoryService{repo: repo}
}

// ListActive returns the public DTO list of active categories, ordered by name ASC (R3, R5).
func (s *categoryService) ListActive(ctx context.Context) ([]dtos.CategoryResponse, error) {
	cats, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dtos.CategoryResponse, 0, len(cats))
	for _, c := range cats {
		result = append(result, dtos.CategoryResponse{
			Slug:  c.Slug,
			Name:  c.Name,
			Icon:  c.Icon,
			Color: c.Color,
		})
	}
	return result, nil
}
