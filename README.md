# reportaya-api

Backend en Go (Fiber) con arquitectura **Clean / Modular Monolith** y **Domain-Driven Design (DDD)**.
Este repositorio es el **esqueleto arquitectónico** (infraestructura transversal lista para
construir dominios encima), derivado de la arquitectura de `ms-sys-backend`.

## Arquitectura

### Patrones
- **DDD**: el código se organiza por dominios de negocio bajo `internal/domain/<dominio>`.
- **Clean Architecture** por dominio:
  - `handlers`  — capa de transporte HTTP (Fiber).
  - `services`  — lógica de negocio / casos de uso.
  - `repositories` — acceso a datos.
  - `models` y `dtos` — entidades de dominio y objetos de transferencia.
  - `contracts` — interfaces para inversión de dependencias.
- **Inyección de dependencias**: todo se compone en `internal/app/container`.

### Estructura
```
cmd/api/                      # entrypoint (main)
internal/
  app/
    container/                # composition root (DI) + RegisterRoutes
    routes/                   # registradores de rutas por concern/dominio
    server/http/              # servidor Fiber + middlewares (auth, rbac, rate limit, audit, tracing...)
  config/                     # carga y validación de configuración (fail-fast)
  observability/              # tracing (OTel) + métricas (Prometheus)
  persistence/
    postgres/                 # conexión GORM
    migrate/                  # runner de golang-migrate
    migrations/               # SQL versionado (000001_init ...)
  security/
    jwt/                      # firma/verificación de JWT
    rbac/                     # authorizer + catálogo de permisos
    ratelimit/ refresh/ sessionactivity/ tokenblocklist/   # infra de sesión (Redis)
  audit/                      # outbox de auditoría + worker + sink (Mongo)
  domain/                     # dominios (aquí solo los modelos base user/rbac)
pkg/                          # utilidades reutilizables (errors, response, validate, pagination, ...)
```

## Stack
- **Go** 1.26
- **Web**: Fiber v2
- **DB**: PostgreSQL (GORM) + `golang-migrate`
- **Cache/sesión**: Redis
- **Auth**: JWT (access + refresh) con blocklist e idle/absolute session timeouts
- **Authz**: RBAC con caché en Redis
- **Observabilidad**: OpenTelemetry + Prometheus + `zerolog`
- **Auditoría**: patrón outbox con sink opcional a MongoDB

## Cómo correr
1. `cp .env.example .env` y completa `DATABASE_URL`, `JWT_SECRET`, `API_KEY` (mínimo).
2. Levanta Postgres y Redis (local o `docker compose up`).
3. `go run ./cmd/api`  — corre migraciones y arranca el servidor en `:8080`.

Endpoints de arranque: `GET /health`, `GET /ready`, y un ejemplo autenticado en `GET /api/me`.

## Próximos pasos
Crea cada dominio bajo `internal/domain/<dominio>` con sus capas, registra sus rutas en
`internal/app/routes/` y cablea sus dependencias en `internal/app/container/`. Añade el
esquema real (usuarios, RBAC, audit_outbox, ...) como nuevas migraciones en
`internal/persistence/migrations/`.
