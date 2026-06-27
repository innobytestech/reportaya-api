# AGENTS.md — Mapa de navegación para agentes de IA (reportaya-api)

> Punto de entrada para agentes. Lee solo lo necesario (divulgación progresiva).

---

## 1. Antes de empezar (obligatorio)

1. Ejecuta la suite de validación de Go y confírmala en verde:
   ```bash
   go build ./...
   go vet ./...
   go test ./...
   golangci-lint run
   ```
   Si algo falla, **para** y resuelve el entorno antes de continuar.
2. Lee `harness/progress/current.md` para entender el estado de la sesión (créalo si no existe).
3. Lee `harness/feature-list.json`. Toda feature `"sdd": true` sigue el flujo de §4.
4. Lee `CLAUDE.md` para el contrato de orquestación (roles, reglas duras, conmutación).

## 2. Mapa del repositorio

| Archivo / carpeta             | Qué contiene                                                                                                                  | Cuándo leerlo            |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ------------------------ |
| `harness/feature-list.json`   | Tareas con estados: `pending`, `discovery`, `readyForSpec`, `specReady`, `inProgress`, `implemented`, `documented`, `reviewed`, `audited`, `done`, `blocked` | Siempre, al empezar      |
| `harness/progress/`           | `current.md` (sesión activa) e `history.md` (bitácora)                                                                        | Al inicio y cierre       |
| `harness/specs/<feature>/`    | `discovery.md`, `requirements.md`, `design.md`, `tasks.md`                                                                    | Durante el SDD           |
| `.claude/agents/`             | Perfiles de `leader`, `interviewer`, `spec_author`, `implementer`, `documenter`, `reviewer`, `security_auditor`              | Para orquestación        |
| `cmd/`, `internal/`, `pkg/`   | Código fuente y tests en Go                                                                                                   | Implementación           |
| `docs/`                       | Documentación de arquitectura/API (créala conforme crezca el proyecto)                                                        | Antes de diseñar/revisar |

## 3. Arquitectura — convención por capas

`reportaya-api` es un **Modular Monolith** con **Clean Architecture / DDD**.

```
cmd/api/                       # entrypoint (main): config → migraciones → tracing → container → servidor
internal/
  app/
    container/                 # composition root (DI) + RegisterRoutes
    routes/                    # registradores de rutas por concern/dominio
    server/http/               # servidor Fiber + middlewares (auth JWT, RBAC, rate-limit, audit, tracing, ...)
  config/                      # carga y validación de configuración (fail-fast)
  observability/               # tracing (OTel) + métricas (Prometheus)
  persistence/
    postgres/                  # conexión GORM
    migrate/                   # runner de golang-migrate
    migrations/                # SQL versionado (000001_init ...)
  security/                    # jwt, rbac, ratelimit, refresh, sessionactivity, tokenblocklist
  audit/                       # outbox de auditoría + worker + sink (Mongo)
  domain/<dominio>/            # cada dominio con sus capas:
    contracts/                 #   interfaces (inversión de dependencias)
    dtos/                      #   objetos de transferencia
    handlers/                  #   transporte HTTP (Fiber)
    models/                    #   entidades de dominio
    repositories/              #   acceso a datos
    services/                  #   lógica de negocio / casos de uso
pkg/                           # utilidades reutilizables (errors, response, validate, pagination, ...)
```

**Para crear un dominio nuevo:** crea `internal/domain/<dominio>/` con sus capas, registra
sus rutas en `internal/app/routes/`, cablea sus dependencias en `internal/app/container/`
y agrega el esquema en `internal/persistence/migrations/`.

## 4. Flujo de trabajo (SDD)

```text
pending → [interviewer] → discovery → ⏸ HUMANO → readyForSpec → [spec_author] → specReady
       → ⏸ HUMANO → inProgress → [implementer] → implemented → [documenter] → documented
       → [reviewer] → reviewed → [security_auditor] → audited → ⏸ HUMANO → done
```

## 5. Reglas duras (no negociables)

- **Una sola feature a la vez.** No mezcles tareas.
- **Validación total.** No cierres una tarea sin la suite de §1 en verde.
- **PascalCase / camelCase** consistentes en los estados del JSON.
- **No asumas reglas de arquitectura: búscalas o documéntalas** en `docs/` y respeta las capas de §3.
