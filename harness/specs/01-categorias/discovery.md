# Discovery — Feature 1 (A1) · Catálogo de categorías

**Fecha:** 2026-06-27 · **Rol:** interviewer · **Estado resultante:** `readyForSpec`

Feature: `id 1` / `code A1` / Epic A — Núcleo de reportes.
Base de la que dependen la creación de reportes (A2) y los filtros del mapa (A4).

## Descripción

Catálogo de categorías de reporte de infraestructura urbana (baches, luminarias
apagadas, fugas de agua, acumulación de basura, …). Solo backend (Go/Fiber).

## Decisiones resueltas con el humano (2026-06-27)

### 1. Gestión del catálogo — **Seed estático, solo lectura**
- Las categorías se cargan como datos semilla (seed) en una migración SQL.
- El backend expone **únicamente lectura** (GET). No hay CRUD de administración en el MVP.
- Activar/desactivar una categoría se hace por migración o SQL directo (no por API).
- **Soft-disable, no delete físico:** una categoría inactiva se marca como inactiva
  (`is_active = false`); no se borra de la tabla. La consulta pública la excluye.

### 2. Identificador estable — **Slug textual + UUID interno**
- Cada categoría tiene un **slug** legible y estable como contrato público:
  `baches`, `luminarias`, `fuga-agua`, `basura` (kebab-case, único).
- Internamente conserva un **UUID** como PK (convención del repo: GORM + `uuid`).
- El frontend filtra el mapa (A4) y crea reportes (A2) **por slug**.

### 3. Metadatos visuales — **id (slug) + nombre + icono + color**
- `name`: nombre legible en español (ej. "Baches").
- `icon`: identificador de icono como string (ej. `pothole`) — el frontend mapea el asset.
- `color`: color hex para el marcador del mapa (ej. `#E11D48`).
- Objetivo: el frontend NO hardcodea estilos; el catálogo es la fuente de verdad visual.

## Alcance

**Entra:**
- Modelo de dominio `category` (UUID, slug, name, icon, color, is_active, timestamps).
- Migración con la tabla + seed del catálogo aprobado de **12 categorías** (las 4 mínimas
  —baches, luminarias, fuga-agua, basura— + 8 incidencias municipales comunes: drenaje,
  aguas-negras, semaforo, senaletica, banquetas, grafiti, arboles, animal-muerto).
  *(Ampliado por el humano el 2026-06-27: "al menos" en las acceptances ⇒ catálogo completo.)*
- Endpoint público de solo lectura que devuelve las categorías **activas**.
- Tests TDD (servicio/handler/repositorio según capas DDD del repo).

**No entra (out of scope MVP):**
- CRUD de administración de categorías (crear/editar/desactivar vía API).
- Scoping por ciudad/zona del catálogo (las categorías son globales en el MVP).
- Localización multi-idioma (solo español).
- Iconos/colores como assets binarios (solo se guarda el identificador/string).

## Criterios de aceptación preliminares (del feature-list, a refinar por spec_author)

- AC1: El catálogo incluye al menos baches, luminarias apagadas, fugas de agua y
  acumulación de basura (vía seed).
- AC2: Un endpoint de solo lectura devuelve la lista de categorías **activas**.
- AC3: Cada categoría tiene un identificador estable (slug + UUID) y un nombre legible.
- AC4: Una categoría inactiva no aparece en la consulta pública pero **no** se elimina
  físicamente (`is_active = false`).

## Notas para spec_author

- Seguir convención de capas DDD: `internal/domain/category/{models,dtos,handlers,
  repositories,services,contracts}`.
- Convención de modelos del repo: GORM, PK `uuid` con `default:gen_random_uuid()`,
  `CreatedAt/UpdatedAt`, soft-delete `gorm.DeletedAt` (ver `internal/domain/user/models/user.go`).
- Registrar rutas en `internal/app/routes/`, cablear DI en `internal/app/container/`,
  añadir migración en `internal/persistence/migrations/` (siguiente versión tras `000001_init`).
- Endpoint público: pendiente decidir si pasa por API Key/CORS (feature 5 / B3, aún no construida).
  En el MVP la ruta es pública de lectura; el middleware transversal se aplicará cuando exista B3.
- Definir el slug exacto de cada categoría semilla en `requirements.md`.
