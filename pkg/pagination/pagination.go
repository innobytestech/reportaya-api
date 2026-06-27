package pagination

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// DefaultPage y DefaultLimit para listados.
const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// PaginatedResult DTO de respuesta paginada estándar.
type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"totalPages"`
}

// Params contiene page y limit parseados desde query.
type Params struct {
	Page  int
	Limit int
}

// Offset devuelve el offset para la consulta: (page-1)*limit.
func (p Params) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	return (p.Page - 1) * p.Limit
}

// FromQuery parsea page y limit desde fiber context (query params).
// Query params estándar: page (default 1), limit (default DefaultLimit, max MaxLimit).
func FromQuery(c *fiber.Ctx) Params {
	page, _ := strconv.Atoi(c.Query("page", strconv.Itoa(DefaultPage)))
	limit, _ := strconv.Atoi(c.Query("limit", strconv.Itoa(DefaultLimit)))
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return Params{Page: page, Limit: limit}
}

// TotalPages calcula el número de páginas dado total y limit.
func TotalPages(total int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	p := int(total) / limit
	if int(total)%limit > 0 {
		p++
	}
	return p
}

// NewResult construye un PaginatedResult.
func NewResult(data interface{}, page, limit int, total int64) *PaginatedResult {
	return &PaginatedResult{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: TotalPages(total, limit),
	}
}
