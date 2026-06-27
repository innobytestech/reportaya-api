package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"reportaya-api/internal/domain/category/dtos"
	"reportaya-api/internal/domain/category/handlers"
	"reportaya-api/pkg/response"
)

// fakeService is an in-memory implementation of contracts.CategoryService.
type fakeService struct {
	results []dtos.CategoryResponse
	err     error
}

func (f *fakeService) ListActive(_ context.Context) ([]dtos.CategoryResponse, error) {
	return f.results, f.err
}

func newTestApp(svc *fakeService) *fiber.App {
	app := fiber.New()
	h := handlers.NewCategoryHandler(svc)
	app.Get("/api/categories", h.List)
	return app
}

// TestList_ReturnsOKWithEnvelope verifies R2 (200 + envelope ok:true) and R3 (public fields).
func TestList_ReturnsOKWithEnvelope(t *testing.T) {
	t.Parallel()

	svc := &fakeService{
		results: []dtos.CategoryResponse{
			{Slug: "baches", Name: "Baches", Icon: "pothole", Color: "#E11D48"},
			{Slug: "luminarias", Name: "Luminarias apagadas", Icon: "lightbulb", Color: "#F59E0B"},
		},
	}

	app := newTestApp(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env response.APIResponse
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("could not parse envelope: %v", err)
	}

	if !env.Ok {
		t.Errorf("expected ok:true, got ok:false")
	}
	if env.Status != http.StatusOK {
		t.Errorf("expected status 200 in envelope, got %d", env.Status)
	}
	if env.Results == nil {
		t.Fatal("expected results to be non-nil")
	}

	// Decode results array to verify public fields.
	raw, _ := json.Marshal(env.Results)
	var cats []dtos.CategoryResponse
	if err := json.Unmarshal(raw, &cats); err != nil {
		t.Fatalf("could not parse results: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
	if cats[0].Slug != "baches" {
		t.Errorf("expected first slug 'baches', got %q", cats[0].Slug)
	}
}

// TestList_ServiceError_Returns500 verifies that service errors propagate to 500.
func TestList_ServiceError_Returns500(t *testing.T) {
	t.Parallel()

	svc := &fakeService{err: errors.New("db unavailable")}
	app := newTestApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// TestList_EmptyResults_Returns200 verifies that an empty catalog still returns 200.
func TestList_EmptyResults_Returns200(t *testing.T) {
	t.Parallel()

	svc := &fakeService{results: []dtos.CategoryResponse{}}
	app := newTestApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
