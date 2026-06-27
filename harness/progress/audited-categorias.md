# Security Audit Report: Seguridad, Resiliencia & Performance
## Feature A1 — Catálogo de categorías (`GET /api/categories`)

---

## Resumen Ejecutivo

| Módulo / Capa                                           | Estado        | Criticidad | Tipo de Riesgo                         |
| :------------------------------------------------------ | :------------ | :--------- | :------------------------------------- |
| `internal/domain/category/dtos`                         | SEGURO        | Ninguna    | N/A — DTO filtra correctamente         |
| `internal/domain/category/handlers`                     | SEGURO        | Ninguna    | N/A — errores no filtran detalles      |
| `internal/domain/category/repositories`                 | SEGURO        | Ninguna    | N/A — query parametrizada, sin N+1     |
| `internal/domain/category/services`                     | SEGURO        | Ninguna    | N/A — mapeo O(n), sin goroutines       |
| `internal/app/server/http` (CORS/SecurityHeaders/RL)    | SEGURO        | Ninguna    | N/A — middleware global operativo      |
| `internal/app/container/routes.go` (public by design)   | INFO          | Baja       | Endpoint público sin auth — por diseno |
| `internal/persistence/migrations/000002_categories.sql` | SEGURO        | Ninguna    | N/A — esquema correcto, seed idempotente |

**Bloqueantes:** ninguno.
**Veredicto: APROBADO / [ESTABLE]**

---

## Validacion del Build

```
go build ./...  → OK (sin errores)
go vet ./...    → OK (sin warnings)
go test ./...   → OK
  ok  reportaya-api/internal/domain/category/handlers     0.685s
  ok  reportaya-api/internal/domain/category/repositories 0.560s
  ok  reportaya-api/internal/domain/category/services     1.579s
golangci-lint   → 3 errores PRE-EXISTENTES (no tocados por esta feature):
  internal/domain/rbac/models/permission.go — gofmt
  internal/domain/rbac/models/role.go       — gofmt
  internal/observability/metrics.go         — gofmt
```

Los 3 errores de lint son en archivos sin relacion con A1; no se contabilizan en esta auditoria.

---

## Hallazgos Detallados

### [INFO] Endpoint publico sin auth — por diseno de MVP

- **Ubicacion:** `internal/app/container/routes.go:19-21`, `internal/app/routes/category_routes.go`
- **Tipo:** Decision arquitectonica documentada, no vulnerabilidad.
- **Detalle:** `GET /api/categories` no lleva middleware de autenticacion. Esto es correcto: el
  catalogo es dato publico de solo lectura. El codigo y los comentarios documentan explicitamente
  que B3 (API Key + CORS estricto) lo cubrira. La ausencia de auth en este endpoint no representa
  fuga de informacion dado que los datos expuestos (slug, name, icon, color) son publicos por
  intencion de negocio.
- **Impacto:** Ninguno mientras el catalogo no contenga datos sensibles.
- **Accion requerida:** Ninguna para el MVP. Fast-follow: aplicar el middleware de B3 cuando este
  disponible.

---

### [INFO] CORS configurado desde config, no hardcodeado como wildcard

- **Ubicacion:** `internal/app/server/http/server.go:67-72`
- **Detalle:** `AllowOrigins` se construye a partir de `cfg.CORSOrigins` (inyectado por config),
  no es `*`. `AllowCredentials: true` esta correctamente acompanado de origenes explicitos.
  `SecurityHeaders()` agrega `Cross-Origin-Resource-Policy: same-origin` y
  `Cross-Origin-Opener-Policy: same-origin`. No hay CORS laxo.
- **Riesgo residual:** Si la configuracion de produccion define `CORSOrigins: ["*"]` el riesgo
  reaparece. Fast-follow: validar ese campo en el arranque del servidor.

---

### [INFO] Cabeceras de seguridad aplicadas globalmente

- **Ubicacion:** `internal/app/server/http/middleware_security.go`
- **Detalle:** `X-Content-Type-Options`, `X-Frame-Options: DENY`, `Referrer-Policy`,
  `Content-Security-Policy`, `Permissions-Policy`, HSTS condicional a HTTPS.
  Cobertura correcta para un endpoint de API REST. Sin hallazgos.

---

### [INFO] Rate limiting global activo en el endpoint publico

- **Ubicacion:** `internal/app/server/http/server.go:74-77`, `middleware_ratelimit.go:65-94`
- **Detalle:** `RateLimitGeneralMiddleware` aplica sobre toda la API (incluyendo
  `/api/categories`) usando IP como clave cuando la peticion es anonima. El endpoint no
  tiene rate limiting dedicado (seria innecesario para un catalogo estatico), pero hereda
  la proteccion global. No hay riesgo de DoS sin costo de calculo.

---

### [INFO] Ausencia de paginacion — aceptable y justificada

- **Ubicacion:** `internal/domain/category/repositories/category_repo.go:27-34`
- **Detalle:** El catalogo esta acotado a 12 filas (seed fijo) con crecimiento natural
  minimo. La query `SELECT * ... WHERE is_active = ? AND deleted_at IS NULL ORDER BY name ASC`
  es O(n) sobre una tabla diminuta. No hay riesgo de respuesta masiva. La omision de
  paginacion esta justificada y documentada.

---

## Analisis de Seguridad por Area

### AppSec Defensivo

| Aspecto                     | Resultado                                                                                      |
| :-------------------------- | :--------------------------------------------------------------------------------------------- |
| Filtracion de campos internos | CORRECTO — `CategoryResponse` expone solo `slug`, `name`, `icon`, `color`. UUID, `is_active`, timestamps y `deleted_at` estan ausentes del DTO por construccion (compile-time). |
| Error handling sin filtracion | CORRECTO — `response.InternalError` llama a `Fail(..., "INTERNAL_ERROR", "Internal Server Error", err)`. El error tecnico solo se serializa si `isDebugEnv()` es true (basado en `APP_ENV`). En produccion, `err.Error()` nunca llega al cliente. |
| SQL Injection                | CORRECTO — GORM usa `Where("is_active = ?", true)` con binding parametrizado. No hay interpolacion de strings ni input del cliente en la query. |
| Input del cliente            | NO HAY — el endpoint no recibe body, query params ni path params. Superficie de ataque: cero. |
| IDOR                         | NO APLICA — no hay recursos por propietario ni identificadores de usuario en esta feature.  |

### Resiliencia Operativa

| Aspecto                  | Resultado                                                                             |
| :----------------------- | :------------------------------------------------------------------------------------ |
| Context propagacion      | CORRECTO — `h.svc.ListActive(c.Context())` propaga el contexto de Fiber (con timeout del servidor) hasta GORM (`r.db.WithContext(ctx)`). |
| Timeouts en servidor     | CORRECTO — `fiber.Config.ReadTimeout` y `WriteTimeout` configurados en `server.go:49-51`. |
| Loops / recursion        | NINGUNO — no hay recursion ni bucles de reintentos introducidos por esta feature.      |
| Goroutines sin cierre    | NINGUNA — la feature no lanza goroutines propias. El flujo es sincrono request-response. |
| Fugas de conexion        | NINGUNA — el pool de conexiones es gestionado por GORM/pgxpool, configurado en el container. |

### Complejidad Algoritmica

| Aspecto           | Resultado                                                                            |
| :---------------- | :----------------------------------------------------------------------------------- |
| Big-O query       | O(n) sobre ~12 filas con filtro sobre columna booleana. Aceptable.                   |
| N+1               | NINGUNO — es una sola query sin relaciones asociadas.                                |
| Mapeo DTO         | O(n) con `make([]CategoryResponse, 0, len(cats))` pre-alocado. Correcto.             |
| Indice UNIQUE slug | Presente en DDL: `slug varchar(50) NOT NULL UNIQUE`. Busquedas por slug son O(log n). |
| Indice deleted_at  | Presente: `idx_categories_deleted_at`. Correcto para soft-delete.                    |
| Indice parcial is_active | AUSENTE en el DDL actual. Fast-follow no bloqueante (ver recomendaciones).      |

### Pentest / IDOR

- No hay parametros de entrada del cliente: el endpoint no acepta path params, query strings
  ni body.
- No hay recursos por propietario: todos los datos son del catalogo global.
- No hay IDOR posible en esta superficie.
- Enumeracion de informacion: los datos expuestos (icono, color, slug semantico como "baches")
  son publicos por definicion. Riesgo de fuga: nulo.

---

## Recomendaciones Fast-Follow (no bloquean el MVP)

1. **Indice parcial is_active:** agregar `CREATE INDEX IF NOT EXISTS idx_categories_is_active ON categories (is_active) WHERE is_active = true;` en una migracion futura. Irrelevante hoy (12 filas), util si el catalogo crece.

2. **Cache / ETag:** el catalogo es inmutable en operacion normal. Agregar `Cache-Control: public, max-age=300` y soporte de ETag en B3 eliminaria el viaje a Postgres en la gran mayoria de peticiones.

3. **Validacion de CORSOrigins en arranque:** rechazar en startup si `cfg.CORSOrigins` contiene `*` o esta vacio, para prevenir CORS laxo por mala configuracion de ambiente.

4. **Rate limiting dedicado por endpoint en B3:** aunque el RL global cubre esta ruta, un bucket separado con limite mas alto para el catalogo (lectura pura, cacheable) evitaria que el RL general la penalice en picos de trafico legitimo.

5. **Test de integracion con Postgres real:** los tests actuales usan sqlmock y fakes en memoria. Un test de integracion contra Postgres (en CI con Docker) validaria el comportamiento real del soft-delete y el orden `ASC` directamente sobre el motor.

---

## Veredicto Final

**APROBADO / [ESTABLE]**

- Bloqueantes: **0**
- Advertencias: **0**
- Informativos: **2** (endpoint publico por diseno documentado; CORS desde config correcto)
- Fast-follow: **5** (no bloquean el MVP)

La feature A1 queda en estado `audited`, lista para el cierre humano a `done`.
