# SaaS Propiedad Horizontal

Backend Go + Next.js + Expo para administracion de conjuntos residenciales
(MVP enfocado en Colombia). Multi-tenant **DB-por-tenant** (ver
[ADR 0001](docs/adr/0001-architecture-multi-tenant-strategy.md)).

## Stack

| Capa | Tecnologia |
|------|-----------|
| Backend | Go 1.26 + chi v5 + pgx/v5 + sqlc + golang-migrate |
| Base de datos | PostgreSQL 18 (Control Plane + 1 DB por tenant) |
| Web | Next.js 16.2.3 (App Router, TypeScript) |
| Movil | Expo SDK 55 (RN 0.83, TypeScript) |
| Observabilidad | OpenTelemetry para Go |
| Testing | go test + Testcontainers + Playwright |

Reglas duras: ver [`CLAUDE.md`](CLAUDE.md). Plan de fases: ver
[`PLAN_MAESTRO.md`](PLAN_MAESTRO.md).

## Estructura del repositorio

```
apps/
  api/              # Backend Go (modular monolith)
  web/              # Frontend Next.js
  mobile/           # App Expo
deployments/
  docker-compose.local.yml   # Postgres central + tenant template
docs/
  adr/              # Architectural Decision Records
  openapi/          # Specs OpenAPI 3.0 por modulo
  specs/            # Specs frozen de fases POST-MVP (8-15)
migrations/
  central/          # Migraciones del Control Plane
  tenant/           # Migraciones de cada Tenant DB (se aplican por tenant)
.claude/
  commands/         # Slash commands custom (fase, descubrir, verificar-fase, onboarding)
.golangci.yml       # Lint estricto
lefthook.yml        # Git hooks (gofmt, goimports, golangci-lint)
```

## Levantar entorno local

### 1. Pre-requisitos

- Go 1.26+
- Docker (con Compose v2)
- `golangci-lint`, `goimports`, `sqlc`, `migrate` (golang-migrate), `lefthook`
- Node 22+ y `pnpm` o `npm` (para `apps/web`)

Instalacion rapida de tooling Go:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/evilmartians/lefthook@latest
```

### 2. Levantar Postgres (Control Plane + tenant template)

```bash
docker compose -f deployments/docker-compose.local.yml up -d
docker compose -f deployments/docker-compose.local.yml ps   # ambos UP
```

Conexiones por defecto:

- **Control Plane**: `postgres://ph:ph@localhost:5432/ph_central`
- **Tenant template**: `postgres://ph:ph@localhost:5433/ph_tenant_template`

### 3. Aplicar migraciones

```bash
# Control Plane
migrate -path migrations/central -database "postgres://ph:ph@localhost:5432/ph_central?sslmode=disable" up

# Tenant template (para clonarse al provisionar tenants reales)
migrate -path migrations/tenant -database "postgres://ph:ph@localhost:5433/ph_tenant_template?sslmode=disable" up
```

### 4. Ejecutar API

```bash
cd apps/api
go build ./...
go test ./...
go run ./cmd/api    # Disponible despues de Fase 1
```

### 5. Hooks de pre-commit

```bash
lefthook install
```

## Convenciones del repositorio

- **Modular Monolith** con Clean Architecture estricta por modulo
  (`internal/modules/<nombre>/{domain,application,interfaces,infrastructure}`).
- **Soft delete** generalizado y campos estandar (ver
  [ADR 0004](docs/adr/0004-audit-and-soft-delete-strategy.md)).
- **Concurrencia optimista** con columna `version` (ver
  [ADR 0005](docs/adr/0005-transactional-and-idempotency-strategy.md)).
- Errores HTTP en formato **RFC 7807** (`application/problem+json`).
- Logs estructurados JSON con `request_id`, `trace_id`, `user_id` (sin
  `tenant_id` como columna en Tenant DB; se incluye en logs por contexto).
- Commits con prefijo `fase-N: <resumen>` (validado por lefthook).

## Roadmap de fases

| Bloque | Fases | Comando | Estado |
|--------|-------|---------|--------|
| MVP | 0-7 | `/fase N` | Completas (0,1,2,3,4,5,6,7) |
| POST-MVP | 8-15 | `/descubrir N` -> spec frozen -> `/fase N` | Pendientes |

Modulos MVP entregados:
- Plataforma: chi server + middlewares (request_id, logging, recovery,
  rate_limit, tenant_resolver), pgxpool central + Registry por tenant
  con single-flight, RFC 7807, golang-migrate.
- `identity` — login + MFA TOTP + refresh con rotacion + /me.
- `authorization` — RBAC con namespaces + scopes + RequirePermission.
- `tenant_config` — settings (key/JSONB) y branding singleton.
- `residential_structure` — torres/bloques/etapas en jerarquia.
- `units` — unidades + owners + ocupantes + endpoint critico
  `GET /units/{id}/people`.
- `people` — vehiculos + asignaciones a unidades.
- `access_control` — porteria con QR pre-registro, blacklist, manual.
- `packages` — paqueteria con bloqueo optimista + idempotency + outbox.
- `announcements` — tablero con audiencias y feed filtrado.
- Hardening: `audit_logs` append-only con trigger, indices criticos,
  runbook operativo en `docs/runbook.md`.

Runtime contra Postgres 18 esta en construccion: el codigo y las
migraciones estan listos. La verificacion end-to-end con Docker se hace
en piloto (ver runbook).

## Decisiones arquitectonicas (ADRs)

- [ADR 0001 — Estrategia multi-tenant](docs/adr/0001-architecture-multi-tenant-strategy.md)
- [ADR 0002 — Autenticacion e identidad](docs/adr/0002-authentication-and-identity.md)
- [ADR 0003 — Autorizacion RBAC con scopes](docs/adr/0003-authorization-rbac-scopes.md)
- [ADR 0004 — Auditoria y soft-delete](docs/adr/0004-audit-and-soft-delete-strategy.md)
- [ADR 0005 — Transaccionalidad e idempotencia](docs/adr/0005-transactional-and-idempotency-strategy.md)

## Licencia

Privada. Pendiente de definir en piloto.
