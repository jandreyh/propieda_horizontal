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

Hay dos composes equivalentes:

- `deployments/docker-compose.local.yml` — solo los dos Postgres (host pelado).
- `deployments/docker-compose.dev.yml` — Postgres + devcontainer Linux con todo el tooling (Go, Node, sqlc, migrate, lefthook, Claude Code). Usar solo si quieres trabajar dentro del container; en host directo basta levantar los dos servicios pg-*.

```bash
docker compose -f deployments/docker-compose.dev.yml up -d pg-central pg-tenant-template
```

Conexiones por defecto:

- **Control Plane**: `postgres://ph:ph@localhost:5432/ph_central`
- **Tenant template**: `postgres://ph:ph@localhost:5433/ph_tenant_template`

> **Nota**: si los puertos 5432/5433 ya estan ocupados por otros Postgres en tu Docker, Compose reasigna automaticamente (ej. 5434/5435). Revisa con `docker port ph-pg-central` y `docker port ph-pg-tenant-template` y ajusta las URLs abajo.

### 3. Aplicar migraciones

```bash
# Control Plane
migrate -path migrations/central -database "postgres://ph:ph@localhost:5432/ph_central?sslmode=disable" up

# Tenant template (para clonarse al provisionar tenants reales)
migrate -path migrations/tenant -database "postgres://ph:ph@localhost:5433/ph_tenant_template?sslmode=disable" up

# Seed de roles y permisos (no se aplica con `migrate up` porque su nombre
# no es secuencial; se carga manualmente, idempotente):
docker exec -i ph-pg-tenant-template psql -U ph -d ph_tenant_template \
  < migrations/tenant/seed_001_roles_permissions.up.sql
```

### 4. Ejecutar API

```bash
cd apps/api
go build ./...
go test ./... -count=1 -short

# Las variables de entorno reales que lee config.go:
DB_CENTRAL_URL="postgres://ph:ph@localhost:5432/ph_central?sslmode=disable" \
DB_TENANT_TEMPLATE_URL="postgres://ph:ph@localhost:5433/ph_tenant_template?sslmode=disable" \
HTTP_ADDR=":8080" \
LOG_FORMAT=json \
go run ./cmd/api
```

Smoke endpoints (con la API corriendo):

```bash
curl -fsS http://localhost:8080/health   # {"status":"ok",...}
curl -fsS http://localhost:8080/ready    # {"status":"ready"}
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

## Estado del runtime (verificado 2026-04-29)

| Componente | Estado | Comando de verificacion |
|-----------|--------|------------------------|
| Build backend | ✓ limpio | `go build ./...` en `apps/api` |
| Tests unitarios | ✓ 26 paquetes verdes | `go test ./... -count=1 -short` |
| Tests con tag `integration` | ⚠ no hay archivos exclusivos aun | `go test ./... -tags=integration` corre los mismos unitarios |
| Postgres central | ✓ healthy | `docker exec ph-pg-central pg_isready -U ph -d ph_central` |
| Postgres tenant template | ✓ healthy | `docker exec ph-pg-tenant-template pg_isready -U ph -d ph_tenant_template` |
| Migraciones central | ✓ version 1 | `migrate -path migrations/central -database $URL version` |
| Migraciones tenant | ✓ version 18 | `migrate -path migrations/tenant -database $URL version` |
| Seed roles/permissions | ✓ 9 roles, 26 permissions | aplicar `seed_001_roles_permissions.up.sql` con `psql` |
| Reversibilidad tenant (down/up) | ✓ smoke OK | `migrate down 1 && migrate up` |
| API `/health` | ✓ 200 OK | `curl http://localhost:8080/health` |
| API `/ready` | ✓ 200 OK | `curl http://localhost:8080/ready` |
| `apps/web` build | ✓ 17 rutas estaticas | `pnpm --filter web build` |
| `apps/web` lint | ✓ limpio | `pnpm --filter web lint` |
| `apps/mobile` typecheck | ✓ limpio | `pnpm --filter mobile exec tsc --noEmit` |

Caveats abiertos (no bloqueantes):

- No existen tests con `//go:build integration` ni uso de Testcontainers todavia. Suite actual es enteramente unitaria con stubs/mocks.
- `seed_001_roles_permissions.up.sql` no es secuencial -> se aplica manualmente (ver bloque 3 arriba). Diseño intencional para ser idempotente con `ON CONFLICT DO NOTHING`.

## Roadmap de fases

| Bloque | Fases | Comando | Estado |
|--------|-------|---------|--------|
| MVP | 0-7 | `/fase N` | Completas |
| POST-MVP | 8-15 | `/descubrir N` -> spec -> `/fase N` | Completas |
| Frontends | web + mobile | scaffold | Scaffold listo |
| Runtime end-to-end | docker + migraciones + smoke | verificado 2026-04-29 | ✓ |

### Modulos MVP (fases 0-7)
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

### Modulos POST-MVP (fases 8-15)
- `parking` — parqueaderos, asignaciones permanentes, reservas visitante, sorteo determinista.
- `finance` — plan de cuentas, centros de costo, cobros, pagos, asientos contables, cierres de periodo, webhooks idempotentes.
- `reservations` — areas comunes, reservas con blackouts y reglas de antelacion.
- `assemblies` — asambleas, convocatorias, asistencia, poderes, mociones, votaciones, actas, firmas.
- `incidents` — reportes de incidentes con adjuntos, historial de estados, asignaciones.
- `penalties` — catalogo de multas, sanciones, apelaciones, historial.
- `pqrs` — categorias, tickets, respuestas, historial de estados.
- `notifications` — plantillas, preferencias, consentimientos, push tokens, config de proveedores, entregas con outbox.

### Frontends
- `apps/web` — Next.js 16.2.4, App Router, TypeScript, Tailwind CSS v4. Login + dashboard con 13 paginas de modulos.
- `apps/mobile` — Expo SDK 55, React Native 0.83, TypeScript. Login + home con tarjetas de modulos.

## Decisiones arquitectonicas (ADRs)

- [ADR 0001 — Estrategia multi-tenant](docs/adr/0001-architecture-multi-tenant-strategy.md)
- [ADR 0002 — Autenticacion e identidad](docs/adr/0002-authentication-and-identity.md)
- [ADR 0003 — Autorizacion RBAC con scopes](docs/adr/0003-authorization-rbac-scopes.md)
- [ADR 0004 — Auditoria y soft-delete](docs/adr/0004-audit-and-soft-delete-strategy.md)
- [ADR 0005 — Transaccionalidad e idempotencia](docs/adr/0005-transactional-and-idempotency-strategy.md)

## Licencia

Privada. Pendiente de definir en piloto.
