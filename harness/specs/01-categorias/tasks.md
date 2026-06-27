# Tasks — Feature 1 (A1) · Catálogo de categorías

**Estado:** implemented · **Fuente:** [design.md](./design.md) · **Fecha:** 2026-06-27

> Plan secuencial TDD. Cada paso deja el sistema compilando. `[ ]` pendiente · `[x]` hecho.
> La columna **R** indica qué criterio de aceptación cubre.

- [x] **T1 — Modelo de dominio.** (R3, R4, R6)
  Crear `internal/domain/category/models/category.go` con el struct `Category`
  (UUID PK, `Slug` uniqueIndex, `Name`, `Icon`, `Color`, `IsActive`, timestamps,
  `gorm.DeletedAt`) y `TableName() == "categories"`.

- [x] **T2 — DTO público.** (R3)
  Crear `internal/domain/category/dtos/category_dto.go` con `CategoryResponse`
  (`slug`, `name`, `icon`, `color`). Sin UUID ni flags internos.

- [x] **T3 — Contratos.** (R2, R3)
  Crear `internal/domain/category/contracts/category.go` con las interfaces
  `CategoryRepository.ListActive` y `CategoryService.ListActive`.

- [x] **T4 — Test de servicio (RED).** (R3, R5)
  Crear `services/category_service_test.go` con un repo *fake* que devuelve categorías
  activas; asertar que el service mapea a `CategoryResponse` (solo campos públicos) y
  preserva el orden recibido. Debe fallar por ausencia de implementación.

- [x] **T5 — Servicio (GREEN).** (R3, R5)
  Implementar `services/category_service.go` (`ListActive` → repo → mapeo a DTO).
  T4 pasa.

- [x] **T6 — Test de handler (RED).** (R2, R3)
  Crear `handlers/category_handler_test.go` con un service *fake* y `app.Test(...)` de
  Fiber; asertar `200`, envelope `ok:true` y `results` con los campos públicos.

- [x] **T7 — Handler (GREEN).** (R2, R3)
  Implementar `handlers/category_handler.go` (`List` usando `pkg/response.Success` /
  `InternalError`). T6 pasa.

- [x] **T8 — Test de repositorio (RED→GREEN).** (R4, R5)
  Crear `repositories/category_repo_test.go` con `go-sqlmock`+GORM; asertar que la
  query filtra `is_active = true` y ordena por `name`. Implementar
  `repositories/category_repo.go`. (Si `sqlmock` resulta inviable, aplicar el fallback
  documentado en design.md §9 y anotarlo en impl-notes.)

- [x] **T9 — Migración + seed.** (R1, R6)
  Crear `internal/persistence/migrations/000002_categories.up.sql` (tabla + índice +
  seed de las 12 categorías del catálogo aprobado con `ON CONFLICT DO NOTHING`)
  y `000002_categories.down.sql` (`DROP TABLE`).

- [x] **T10 — Test de seed (R1, R6).**
  Test ligero que lee `000002_categories.up.sql` y aserta presencia de los 12 slugs,
  de `UNIQUE` en `slug` y del default `gen_random_uuid()`.

- [x] **T11 — Rutas.** (R2)
  Crear `internal/app/routes/category_routes.go` con
  `RegisterCategories(router fiber.Router, h *handlers.CategoryHandler)` → `GET /`.

- [x] **T12 — Cableado (DI + montaje).** (R2)
  Añadir `CategoryHandler` (y service/repo) al `Container` y construirlos en `New(...)`.
  En `RegisterRoutes` montar grupo público `api.Group("/categories")` →
  `routes.RegisterCategories`.

- [x] **T13 — Validación de cierre.**
  `go build ./...` · `go vet ./...` · `go test ./...` · `golangci-lint run` en verde.
  Marcar la feature como `implemented` solo si todo pasa.

## Trazabilidad R ↔ Tasks

| R | Criterio | Tasks |
| - | -------- | ----- |
| R1 | Seed mínimo (4 categorías) | T9, T10 |
| R2 | Listado público de activas | T3, T6, T7, T11, T12 |
| R3 | Contrato estable (slug) sin fugas | T1, T2, T4, T5, T6, T7 |
| R4 | Exclusión sin borrado físico | T1, T8 |
| R5 | Orden determinista (name asc) | T4, T5, T8 |
| R6 | Unicidad de slug | T1, T9, T10 |
