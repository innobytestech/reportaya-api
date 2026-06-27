# Requirements — Feature 1 (A1) · Catálogo de categorías

**Estado:** specReady · **Fuente:** [discovery.md](./discovery.md) · **Fecha:** 2026-06-27

## Objetivo (una frase)

Exponer un endpoint público de **solo lectura** que devuelva el catálogo de
categorías **activas** de reportes (slug, nombre, icono y color), sembradas por
migración, como base para la creación de reportes (A2) y los filtros del mapa (A4).

## Alcance

### Lo que ENTRA
- Dominio `category` con sus capas (`models`, `dtos`, `contracts`, `repositories`,
  `services`, `handlers`) según la convención DDD del repo.
- Migración `000002_categories` que crea la tabla `categories` y **siembra** las 12
  categorías del catálogo aprobado (las 4 mínimas + 8 incidencias municipales comunes).
- Endpoint `GET /api/categories` que devuelve solo las categorías con `is_active = true`.
- Tests TDD ejecutables **sin base de datos viva** (suite `go test ./...`).

### Lo que NO ENTRA
- CRUD de administración (crear/editar/activar/desactivar vía API). Se gestiona por SQL/migración.
- Scoping por ciudad/zona (catálogo global en el MVP).
- Localización multi-idioma (solo español).
- Middleware de API Key / CORS estricto (feature 5 / B3, aún no construida). La ruta
  queda pública; el middleware transversal se aplicará cuando exista B3.

## Criterios de Aceptación (EARS)

- **R1 — Seed del catálogo.** CUANDO se aplica la migración `000002_categories`, el sistema
  DEBE crear las 12 categorías del catálogo aprobado, todas activas, cada una con `slug`,
  `name`, `icon` y `color`: `baches`, `luminarias`, `fuga-agua`, `basura`, `drenaje`,
  `aguas-negras`, `semaforo`, `senaletica`, `banquetas`, `grafiti`, `arboles`,
  `animal-muerto` (las 4 primeras son el mínimo exigido por el feature-list).

- **R2 — Listado público de activas.** CUANDO se recibe `GET /api/categories`, el
  sistema DEBE responder `200` con el envelope estándar (`response.APIResponse`) cuyo
  `results` es la lista de categorías con `is_active = true`.

- **R3 — Contrato estable y legible.** Para cada categoría devuelta, el sistema DEBE
  exponer un identificador estable y legible (`slug`) y un `name`, `icon` y `color`,
  y NO DEBE exponer el UUID interno ni `is_active`/timestamps en la respuesta pública.

- **R4 — Exclusión sin borrado físico.** CUANDO una categoría tiene `is_active = false`,
  el sistema NO DEBE incluirla en `GET /api/categories`, y la fila DEBE permanecer en la
  tabla (sin DELETE físico; soft-state vía `is_active`, soft-delete vía `deleted_at`).

- **R5 — Orden determinista.** CUANDO se devuelve el listado, el sistema DEBE ordenarlo
  de forma determinista (por `name` ascendente) para una salida estable.

- **R6 — Unicidad de slug.** El sistema DEBE garantizar unicidad del `slug` a nivel de
  esquema (índice único), de modo que no existan dos categorías con el mismo slug.

## Validación de cierre (no negociable)

`go build ./...` · `go vet ./...` · `go test ./...` · `golangci-lint run` en verde,
con trazabilidad R1–R6 ↔ tests reales.
