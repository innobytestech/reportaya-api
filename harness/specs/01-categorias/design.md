# Design — Feature 1 (A1) · Catálogo de categorías

**Estado:** specReady · **Fuente:** [requirements.md](./requirements.md) · **Fecha:** 2026-06-27

## 1. Estructura de archivos (dominio `category`)

Siguiendo la convención DDD del repo (`AGENTS.md` §3) y el patrón de `internal/domain/user`:

```
internal/domain/category/
  models/category.go            # entidad GORM
  dtos/category_dto.go          # CategoryResponse (contrato público)
  contracts/category.go         # interfaces Repository y Service (inversión de deps)
  repositories/category_repo.go # GORM: ListActive
  services/category_service.go  # mapeo modelo→DTO
  handlers/category_handler.go  # transporte Fiber: GET /categories
internal/app/routes/category_routes.go        # RegisterCategories(router, handler)
internal/persistence/migrations/000002_categories.up.sql / .down.sql
```

Cableado: registrar el módulo en `internal/app/container/container.go` (composition root)
y montar la ruta en `internal/app/container/routes.go` (`RegisterRoutes`).

## 2. Modelo de datos

```go
// internal/domain/category/models/category.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category es una categoría de reporte de infraestructura urbana.
type Category struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Slug      string         `gorm:"type:varchar(50);not null;uniqueIndex;column:slug"`
	Name      string         `gorm:"type:varchar(100);not null"`
	Icon      string         `gorm:"type:varchar(50);not null"`
	Color     string         `gorm:"type:varchar(7);not null"` // hex "#RRGGBB"
	IsActive  bool           `gorm:"default:true;column:is_active"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Category) TableName() string { return "categories" }
```

**DTO público (no filtra UUID ni flags internos — R3):**

```go
// internal/domain/category/dtos/category_dto.go
package dtos

type CategoryResponse struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}
```

## 3. Contratos (interfaces)

```go
// internal/domain/category/contracts/category.go
package contracts

import (
	"context"

	"reportaya-api/internal/domain/category/dtos"
	"reportaya-api/internal/domain/category/models"
)

type CategoryRepository interface {
	// ListActive devuelve categorías con is_active=true, ordenadas por name asc.
	ListActive(ctx context.Context) ([]models.Category, error)
}

type CategoryService interface {
	ListActive(ctx context.Context) ([]dtos.CategoryResponse, error)
}
```

## 4. Repositorio (GORM)

```go
func (r *categoryRepository) ListActive(ctx context.Context) ([]models.Category, error) {
	var out []models.Category
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&out).Error
	return out, err
}
```

- `gorm.DeletedAt` hace que GORM excluya filas con `deleted_at` automáticamente (soft-delete).
- `Where("is_active = ?", true)` implementa R4 (exclusión de inactivas sin borrado físico).
- `Order("name ASC")` implementa R5.

## 5. Servicio (mapeo a DTO)

`ListActive` llama al repo y mapea `[]models.Category → []dtos.CategoryResponse`
copiando solo `Slug/Name/Icon/Color` (R3).

## 6. Handler (Fiber)

```go
func (h *CategoryHandler) List(c *fiber.Ctx) error {
	cats, err := h.svc.ListActive(c.Context())
	if err != nil {
		return response.InternalError(c, err)
	}
	return response.Success(c, cats)
}
```

Usa el envelope unificado `pkg/response` (R2). Ruta pública.

## 7. Rutas y cableado

```go
// internal/app/routes/category_routes.go
func RegisterCategories(router fiber.Router, h *handlers.CategoryHandler) {
	router.Get("/", h.List)
}
```

En `RegisterRoutes` (container/routes.go), bajo el grupo `/api`, sin middleware de auth:

```go
public := api.Group("/categories")
routes.RegisterCategories(public, ctn.CategoryHandler)
```

Se añaden `CategoryHandler` (y service/repo) como campos del `Container` y se construyen
en `New(...)` a partir de `db.DB`.

## 8. Migración + seed

```sql
-- 000002_categories.up.sql
CREATE TABLE IF NOT EXISTS categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       varchar(50)  NOT NULL UNIQUE,
    name       varchar(100) NOT NULL,
    icon       varchar(50)  NOT NULL,
    color      varchar(7)   NOT NULL,
    is_active  boolean      NOT NULL DEFAULT true,
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_categories_deleted_at ON categories (deleted_at);

INSERT INTO categories (slug, name, icon, color) VALUES
    ('baches',       'Baches',                        'pothole',       '#E11D48'),
    ('luminarias',   'Luminarias apagadas',           'lightbulb',     '#F59E0B'),
    ('fuga-agua',    'Fugas de agua',                 'water-drop',    '#0EA5E9'),
    ('basura',       'Acumulación de basura',         'trash',         '#65A30D'),
    ('drenaje',      'Drenaje / alcantarilla tapada', 'manhole',       '#7C3AED'),
    ('aguas-negras', 'Aguas negras / fuga de drenaje','sewage',        '#92400E'),
    ('semaforo',     'Semáforo descompuesto',         'traffic-light', '#DC2626'),
    ('senaletica',   'Señalización vial dañada',      'sign',          '#2563EB'),
    ('banquetas',    'Banquetas dañadas',             'sidewalk',      '#475569'),
    ('grafiti',      'Grafiti / vandalismo',          'spray-can',     '#DB2777'),
    ('arboles',      'Árboles caídos / poda',         'tree',          '#16A34A'),
    ('animal-muerto','Animal muerto en vía pública',  'paw',           '#57534E')
ON CONFLICT (slug) DO NOTHING;
```

```sql
-- 000002_categories.down.sql
DROP TABLE IF EXISTS categories;
```

`gen_random_uuid()` proviene de `pgcrypto`, ya habilitada en `000001_init` (R1, R6).

## 9. Estrategia de tests (TDD, sin DB viva)

| Test | Cubre | Seam |
| ---- | ----- | ---- |
| `services/category_service_test.go` | R3, R5 (mapeo a DTO, solo campos públicos) | repo *fake* en memoria que implementa `contracts.CategoryRepository` |
| `handlers/category_handler_test.go` | R2, R3 (200 + envelope + JSON correcto) | service *fake* + `app.Test(httptest.NewRequest(...))` de Fiber |
| `repositories/category_repo_test.go` | R4, R5 (query filtra `is_active=true` y ordena) | `DATA-DOG/go-sqlmock` + GORM con `postgres` driver en modo mock |
| `migrations` (test ligero) | R1, R6 (seed contiene los 4 slugs + UNIQUE) | leer el `.up.sql` y asertar presencia de slugs y de `UNIQUE`/`uuid` |

> El test del repositorio usa `go-sqlmock` para no exigir Postgres en CI; el implementer
> añade la dependencia a `go.mod` si no existe. Si `sqlmock` resulta inviable con el
> dialecto, degradar R4/R5 a verificación vía service+fake y dejar el repo cubierto por
> el smoke-test de integración futuro (documentarlo en impl-notes).

## 10. Decisiones tomadas y descartadas

- **Slug como contrato público (✔) vs. exponer UUID (descartado):** el slug es legible y estable
  para frontend (filtros del mapa, creación de reportes). El UUID queda como PK interna,
  no se expone (R3). Descartado exponer ambos para minimizar superficie.
- **`color varchar(7)` (`#RRGGBB`) (✔) vs. entero/int RGB:** string hex es directo para
  el frontend y legible en BD.
- **Seed en migración (✔) vs. seeder en código Go:** alineado con la decisión de discovery
  (solo lectura, gestión por SQL/migración) y mantiene el catálogo versionado.
- **`is_active` (estado lógico) + `deleted_at` (soft-delete) ambos (✔):** `is_active=false`
  desactiva sin borrar (R4); `deleted_at` se mantiene por consistencia con la convención del
  repo (`user.go`). El MVP solo usa `is_active`.
- **Sin enum Postgres para slug (✔):** se prefirió tabla semilla a un `ENUM` para permitir
  añadir categorías por migración sin `ALTER TYPE`.
- **Endpoint sin paginación (✔):** el catálogo es pequeño y acotado; se devuelve completo.
