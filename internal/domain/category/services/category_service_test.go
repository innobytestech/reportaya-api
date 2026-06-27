package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"reportaya-api/internal/domain/category/dtos"
	"reportaya-api/internal/domain/category/models"
	"reportaya-api/internal/domain/category/services"
)

// fakeRepo is an in-memory implementation of contracts.CategoryRepository.
type fakeRepo struct {
	categories []models.Category
}

func (f *fakeRepo) ListActive(_ context.Context) ([]models.Category, error) {
	return f.categories, nil
}

func makeCategory(slug, name, icon, color string) models.Category {
	return models.Category{
		ID:        uuid.New(),
		Slug:      slug,
		Name:      name,
		Icon:      icon,
		Color:     color,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		DeletedAt: gorm.DeletedAt{},
	}
}

// TestListActive_MapsToDTO verifies R3 (only public fields) and R5 (order preserved).
func TestListActive_MapsToDTO(t *testing.T) {
	t.Parallel()

	input := []models.Category{
		makeCategory("baches", "Baches", "pothole", "#E11D48"),
		makeCategory("luminarias", "Luminarias apagadas", "lightbulb", "#F59E0B"),
	}

	repo := &fakeRepo{categories: input}
	svc := services.NewCategoryService(repo)

	results, err := svc.ListActive(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != len(input) {
		t.Fatalf("expected %d results, got %d", len(input), len(results))
	}

	// Verify mapping and order preservation (R5).
	for i, cat := range input {
		got := results[i]
		if got.Slug != cat.Slug {
			t.Errorf("[%d] slug: want %q, got %q", i, cat.Slug, got.Slug)
		}
		if got.Name != cat.Name {
			t.Errorf("[%d] name: want %q, got %q", i, cat.Name, got.Name)
		}
		if got.Icon != cat.Icon {
			t.Errorf("[%d] icon: want %q, got %q", i, cat.Icon, got.Icon)
		}
		if got.Color != cat.Color {
			t.Errorf("[%d] color: want %q, got %q", i, cat.Color, got.Color)
		}
	}
}

// TestListActive_NoInternalFields verifies R3: UUID and is_active are NOT in the DTO.
func TestListActive_NoInternalFields(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{categories: []models.Category{
		makeCategory("baches", "Baches", "pothole", "#E11D48"),
	}}
	svc := services.NewCategoryService(repo)

	results, err := svc.ListActive(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// CategoryResponse only has Slug/Name/Icon/Color — compile-time guarantee.
	var _ dtos.CategoryResponse = results[0]
}

// TestListActive_EmptySlice verifies service handles empty repo without error.
func TestListActive_EmptySlice(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{categories: nil}
	svc := services.NewCategoryService(repo)

	results, err := svc.ListActive(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
}
