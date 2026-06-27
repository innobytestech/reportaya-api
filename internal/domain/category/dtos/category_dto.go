// Package dtos defines the data transfer objects (DTOs) for the category domain.
// DTOs represent the public API contracts exposed to clients, excluding internal persistence fields.
package dtos

// CategoryResponse is the public contract for a single category in HTTP responses.
// Slug is a URL-safe, stable identifier unique per category; it remains unchanged across updates
// and is used as the canonical key for report creation (A2) and map filtering (A4).
// Name is the human-readable label. Icon and Color encode the visual presentation.
// Internal fields (UUID, IsActive, timestamps) are deliberately omitted (R3 — public contract stability).
type CategoryResponse struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}
