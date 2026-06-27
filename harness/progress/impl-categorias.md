# impl-categorias — Feature 1 (A1) · Catálogo de categorías

**Estado final:** implemented
**Fecha:** 2026-06-27

---

## Trazabilidad R → Tests

| R | Criterio | Test(s) |
|---|----------|---------|
| R1 | Seed 12 categorías | `TestSeed_ContainsAll12Slugs` |
| R2 | GET /api/categories 200 + envelope | `TestList_ReturnsOKWithEnvelope`, `TestList_EmptyResults_Returns200` |
| R3 | Contrato público (solo slug/name/icon/color) | `TestListActive_MapsToDTO`, `TestListActive_NoInternalFields`, `TestList_ReturnsOKWithEnvelope` |
| R4 | Filtro is_active=true sin borrado físico | `TestListActive_FiltersIsActiveAndOrdersByName` |
| R5 | Orden name ASC | `TestListActive_MapsToDTO` (orden preservado), `TestListActive_FiltersIsActiveAndOrdersByName` (ORDER BY name ASC) |
| R6 | UNIQUE en slug | `TestSeed_HasUniqueConstraintOnSlug`, `TestSeed_OnConflictDoNothing` |

---

## Tasks completadas

T1 [x] T2 [x] T3 [x] T4 [x] T5 [x] T6 [x] T7 [x] T8 [x] T9 [x] T10 [x] T11 [x] T12 [x] T13 [x]

---

## Archivos creados

- `internal/domain/category/models/category.go`
- `internal/domain/category/dtos/category_dto.go`
- `internal/domain/category/contracts/category.go`
- `internal/domain/category/services/category_service.go`
- `internal/domain/category/services/category_service_test.go`
- `internal/domain/category/handlers/category_handler.go`
- `internal/domain/category/handlers/category_handler_test.go`
- `internal/domain/category/repositories/category_repo.go`
- `internal/domain/category/repositories/category_repo_test.go`
- `internal/app/routes/category_routes.go`
- `internal/persistence/migrations/000002_categories.up.sql`
- `internal/persistence/migrations/000002_categories.down.sql`
- `internal/persistence/migrations/migrations_test.go`

## Archivos modificados

- `internal/app/container/container.go` — añadidos imports y campo `CategoryHandler`, construcción de repo/svc/handler en `New()`
- `internal/app/container/routes.go` — montado grupo público `/api/categories`
- `go.mod` / `go.sum` — añadida dependencia `github.com/DATA-DOG/go-sqlmock v1.5.2`

---

## Resultado de validación (T13)

```
go build ./...    VERDE (sin salida)
go vet ./...      VERDE (sin salida)
go test ./...     VERDE
  ok  reportaya-api/internal/domain/category/handlers
  ok  reportaya-api/internal/domain/category/repositories
  ok  reportaya-api/internal/domain/category/services
  ok  reportaya-api/internal/persistence/migrations
golangci-lint run PARCIAL — 3 errores en archivos PRE-EXISTENTES no tocados por esta feature:
  internal/security/jwt/claims.go, jwt.go, public_resource.go (gofmt CRLF→LF)
  pkg/filevalidation/filevalidation.go (gofmt CRLF→LF)
  pkg/pgutil/errors.go (gofmt CRLF→LF)
  Todos los archivos nuevos de esta feature pasan golangci-lint sin errores.
```

---

## Decisiones y notas de implementación

- **go-sqlmock v1.5.2** se usó para el test de repositorio. La v2 no existe como path de módulo separado; la v1 es suficiente para el caso de uso (regexp matcher sobre query GORM + postgres dialect en modo `PreferSimpleProtocol`).
- **Regexp del mock**: la query generada por GORM/postgres dialecto es `SELECT * FROM "categories" WHERE is_active = $1 AND "categories"."deleted_at" IS NULL ORDER BY name ASC`. El test usa un regexp literal exacto con `\$1` escapado.
- **gofmt en archivos modificados**: se aplicó `gofmt -w` a `container.go` y `routes.go` porque GORM/Windows los tenía con CRLF. Los archivos nuevos se escribieron con LF desde el inicio.
- **Race detector**: `CGO_ENABLED=0` en el entorno Windows impide `-race`. Los tests corren sin él; el flag se habilitará en CI (Linux con CGO).
- **Ruta pública sin auth**: confirmado por spec (B3 aún no existe). El comentario en `routes.go` lo documenta para el implementer de B3.
