# Estado actual del harness — reportaya-api

**Fecha:** 2026-06-26 · **Leader:** Claude (Opus 4.8)

## Estado

Fase de **descomposición de producto** (pre-feature). Se analizó la propuesta
de ReportaYa.app y se fijaron las primeras decisiones de arquitectura. El
esqueleto compila (`go build ./...`, `go vet ./...` en verde). Sin features
activas todavía en `feature-list.json`.

> Alcance de ESTE repo: **solo backend** (Go/Fiber). Angular, Leaflet, cliente
> Turnstile, Docker y Nginx pertenecen a otra capa/repo.

## Decisiones de arquitectura fijadas (humano, 2026-06-26)

1. **Identidad de "apoyo/re-reporte":** token anónimo de navegador. Se persiste
   solo un *hash* del token por interacción para deduplicar votos. Sin datos personales.
2. **Duplicados:** estrategia *sugerir, no bloquear*. `POST /reports` devuelve
   reportes cercanos similares; el usuario decide apoyar o crear. Requiere índice
   geoespacial (PostGIS o `earthdistance`/cubo — a definir en spec).
3. **Almacenamiento de imágenes:** object storage S3-compatible (recomendado
   Cloudflare R2). La BD guarda la *key/URL*, nunca el binario.
4. **Generación de tarjeta FB:** asíncrona vía outbox + worker (reutiliza patrón
   de `internal/audit/`).
5. **SIN gobierno.** Se elimina el lado gubernamental (antigua Épica F). El sistema
   tiene dos actores: ciudadano anónimo y operador interno de ReportaYa.
6. **Resolución = confirmación colectiva + reapertura.** Botón "Ya lo arreglaron"
   con token anónimo; al cruzar un umbral N de confirmaciones, el reporte pasa a
   `resuelto`. Si alguien vuelve a reportar el mismo punto, se reabre. Estado
   percibido por la ciudadanía, auto-corregible (no autoritativo).
7. **Operador con rol moderador mínimo** (JWT/RBAC ya existente): borra spam/
   contenido ofensivo y puede forzar estados. No es gobierno ni ciudadano.
8. **Seguimiento sin cuentas:** panel "mis reportes" atado al token de navegador
   + el permalink público de cada reporte como página de estado. `reports` lleva
   `creator_token_hash`. Web Push queda para fase 2.
9. **Privacidad de la foto:** enfoque pragmático — advertencia en cliente +
   moderación reactiva + botón ciudadano de "reportar abuso". SIN blur automático
   en MVP (queda como fast-follow). La foto se publica tal cual tras sanitizar EXIF.
10. **Geografía:** multi-tenant-ready desde el día 1 — `reports` lleva `city`/zona
    aunque solo se opere Nuevo Laredo. **Geo-cerca** configurable rechaza coordenadas
    fuera de NLD.
11. **Auditoría pública:** estadísticas agregadas en el MVP vía endpoint público de
    solo lectura (conteos por estado/categoría/zona, tasa de resolución, tiempos).

### Máquina de estados del reporte (propuesta, a refinar en spec)

```
abierto → (N confirmaciones) → resuelto
   ↑                              │
   └────── reabierto ◄── re-reporte del mismo punto
oculto  ← (moderador: spam/ofensivo)
```

## Tensiones abiertas a resolver en specs

- Rate-limit real detrás de Cloudflare: leer `CF-Connecting-IP`; store
  persistente (revisar `internal/security/ratelimit/`), no memoria pura.
- Sanitización de imagen: manejo de **HEIC** (la stdlib de Go no decodifica HEIC),
  validación por *magic bytes*, límite de tamaño, anti decompression-bomb.
- Folio único legible + permalink público con Open Graph dinámico.

## Backlog propuesto (épicas → features candidatas)

- **A — Núcleo de reportes:** A1 categorías · A2 crear reporte (incluye `city`/zona +
  geo-cerca + `creator_token_hash`) · A3 sanitización + almacenamiento imagen (S3,
  sin blur) · A4 consulta para mapa (bbox + estado + filtros)
- **B — Anti-abuso/anonimato:** B1 rate-limit real · B2 verificación Turnstile ·
  B3 API Key + CORS estricto
- **C — Evidencia/difusión:** C1 tarjeta FB (folio + render async) · C2 permalink
  + Open Graph (= página de estado pública)
- **D — Interacción ciudadana (token anónimo):** D1 apoyar/re-reportar · D2 detección
  de duplicados · D3 confirmar resolución + reapertura · D4 panel "mis reportes" por
  token · D5 reportar abuso (auto-oculta al cruzar umbral)
- **E — Historias:** E1 feed de reportes últimas 24 h (vista, no almacenamiento)
- **G — Operador (moderación interna, JWT/RBAC):** G1 moderar (borrar spam/ofensivo +
  cola de abuso + forzar estado) · G2 exportación consolidada (Excel/PDF)
- **H — Auditoría pública:** H1 estadísticas agregadas (endpoint público: conteos por
  estado/categoría/zona, tasa de resolución, tiempos)

## Backlog registrado (2026-06-27)

Las 18 features ya están en `feature-list.json` (estado `pending`, `sdd:true`).
El `id` numérico refleja el orden de construcción (1 = primero); el arreglo está
ordenado de forma DESCENDENTE por id (18 → 1). `dependsOn` referencia ids.

**Orden de construcción (id):** 1 A1 categorías · 2 A2 crear reporte · 3 A3 imagen
S3 · 4 A4 consulta mapa · 5 B3 API Key+CORS · 6 B1 rate-limit · 7 B2 Turnstile ·
8 C2 permalink+OG · 9 C1 tarjeta FB · 10 D2 dedupe · 11 D1 apoyar · 12 D3 confirmar/
reabrir · 13 D4 mis reportes · 14 D5 reportar abuso · 15 E1 historias · 16 H1
estadísticas · 17 G1 moderación · 18 G2 exportación.

## Próximo paso

**Feature id 1 (A1 — Catálogo de categorías) en `audited` ⏸ (2026-06-27).**
Flujo SDD completo recorrido hoy:
`pending → discovery → readyForSpec → specReady (aprob. humano) → inProgress → implemented
→ documented → reviewed → audited`.

- Discovery (3 decisiones): seed estático solo-lectura · slug textual + UUID interno ·
  metadatos slug+nombre+icono+color. Catálogo **ampliado a 12 categorías** por petición humana.
- Specs: `harness/specs/01-categorias/{discovery,requirements (R1–R6),design,tasks (T1–T13)}.md`.
- Implementación: dominio `internal/domain/category/**`, ruta `GET /api/categories` (pública,
  solo lectura), migración `000002_categories` + seed 12, tests TDD. `go build/vet/test` VERDE.
- Documenter: GoDoc + `docs/api/categories.md` + `harness/frontend/01-categorias-changelog.md`.
- Reviewer: **APROBADO** (`harness/progress/review-categorias.md`), trazabilidad R↔tests OK.
- Security auditor: **APROBADO**, sin bloqueantes (`harness/progress/audited-categorias.md`).
  5 recomendaciones fast-follow (índice parcial, Cache-Control/ETag, validación CORS, bucket
  rate-limit, test integración Postgres) — para B3/CI, no bloquean MVP.

**⏸ PUERTA HUMANA FINAL:** se requiere aprobación del humano para cerrar a `done`.

### Deuda separada (fuera del flujo SDD)
`golangci-lint` reporta 3 errores PRE-EXISTENTES por CRLF en archivos del esqueleto NO tocados
por esta feature (jwt / filevalidation / pgutil). Pendiente tarea de mantenimiento (normalizar
finales de línea). Los archivos nuevos de A1 pasan el linter limpio.

---

> Bitácora histórica: usa `harness/progress/history.md` (créalo al cerrar la primera feature).
