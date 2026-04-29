# SaaS Propiedad Horizontal — Reglas Maestras

Este archivo se carga automaticamente en cada sesion de Claude Code. Contiene
INVARIANTES que NUNCA se violan. Si una instruccion contradice este documento,
gana este documento — pide aclaracion al usuario antes de proceder.

---

## 1. Stack obligatorio (no negociable)

| Capa | Tecnologia | Version | Notas |
|------|-----------|---------|-------|
| Backend lenguaje | Go (Golang) | 1.26+ | |
| Router HTTP | `chi` | latest | ligero, composable |
| DB driver | `pgx` (jackc/pgx/v5) | latest | NO usar `database/sql` directamente |
| Type-safe SQL | `sqlc` | latest | queries en `.sql`, codigo generado |
| Migraciones | `golang-migrate` | latest | carpetas `central/` y `tenant/` |
| Base de datos | PostgreSQL | 18 | |
| Frontend Web | Next.js | 16.2.3 | App Router, TypeScript |
| Movil | Expo | SDK 55 | RN 0.83, TypeScript, app dinamica unica |
| Observabilidad | OpenTelemetry para Go | latest | logs estructurados, traces, metrics |
| Testing backend | `go test` + Testcontainers for Go | latest | integration tests reales contra Postgres |
| Testing E2E web | Playwright | latest | |

### Prohibiciones explicitas
- NO usar ORMs pesados (GORM, ent, beego). Solo `pgx` + `sqlc`.
- NO usar `database/sql` directamente — siempre via `pgxpool`.
- NO escribir SQL inline en Go — siempre en archivos `.sql` para `sqlc`.
- NO mezclar logica de negocio con handlers HTTP.

---

## 2. Arquitectura

- **Modular Monolith** (no microservicios prematuros).
- **Clean Architecture** estricta por modulo.
- **Multi-tenant**: dos planos de datos.
  - **Control Plane** = base central. Tabla `tenants`, dominios, branding,
    plan SaaS, superadmin, audit logs de plataforma, impersonation.
  - **Data Plane** = una base PostgreSQL POR TENANT. Aislamiento fuerte.

### Reglas multi-tenant criticas
- **PROHIBIDO**: columna `tenant_id` en tablas operativas del Tenant DB.
  El tenant ya esta implicito porque la base entera es de ese tenant.
- **Resolucion del tenant**: post-Fase 16 → por `current_tenant` del JWT
  (NO por subdominio). El middleware lee el JWT y enruta al pool del tenant
  activo. Pre-Fase 16 → por subdominio o header `X-Tenant-Slug`.
- **Cache**: metadata del tenant cacheada para no consultar la base central
  en cada request.
- **Login centralizado** (ADR 0007 supersede ADR 0002): UN solo `platform_users`
  por persona en la DB central; cada tenant DB tiene `tenant_user_links`. El
  JWT lleva `memberships[]` y `current_tenant`. Vinculacion de usuario a un
  conjunto solo por `public_code` que la persona entrega al admin (NO busqueda
  libre cross-tenant excepto por `platform_superadmin`).

### Decisiones congeladas adicionales (post-Discovery 2026-04-29)
- **Provisioning**: solo el `platform_superadmin` crea tenants. Sincrono:
  crear DB + migrar + sembrar admin + tenant funcional en una sola llamada.
- **Entidad `platform_administrators`**: agrupa N tenants para cobranza
  consolidada y dashboard cross-conjunto.
- **Bloqueos**: por tenant, no globales. Ban global solo via superadmin.
- **Migracion del `demo` actual**: borrar y resembrar con el nuevo modelo.
- **Push notifications**: a nivel plataforma (servicio centralizado enruta
  notifs de N conjuntos al device del usuario).
- Detalles completos en [ADR 0007](docs/adr/0007-cross-tenant-identity.md) y
  [spec Fase 16](docs/specs/fase-16-cross-tenant-identity-spec.md).

---

## 3. Estructura obligatoria por modulo Go

`/apps/api/internal/modules/<nombre-modulo>/`

```
domain/
  entities/             # structs puros, SIN tags JSON ni DB
  repository_interfaces/ # interfaces que el dominio define
  policies/             # funciones puras de logica de negocio
application/
  usecases/             # orquestacion, recibe interfaces por DI
  dto/                  # request/response, AQUI van tags JSON
interfaces/http/
  handlers/             # adaptan HTTP al usecase
infrastructure/persistence/
  queries/              # archivos .sql para sqlc
  <modulo>_repository.go # implementacion concreta usando codigo generado
```

Las entidades del dominio NO conocen JSON ni DB. Los DTOs NO conocen DB. La
inversion de dependencias es estricta: `domain` no importa nada hacia
afuera; `application` solo importa `domain`; `infrastructure` e `interfaces`
implementan/llaman a `application`.

---

## 4. Campos estandar (todas las tablas operativas del tenant)

```sql
id          UUID        PRIMARY KEY DEFAULT gen_random_uuid()
status      TEXT        NOT NULL
created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
deleted_at  TIMESTAMPTZ NULL
created_by  UUID        NULL REFERENCES users(id)
updated_by  UUID        NULL REFERENCES users(id)
deleted_by  UUID        NULL REFERENCES users(id)
version     INTEGER     NOT NULL DEFAULT 1
```

- **Soft delete por defecto**. Hard delete solo con justificacion explicita en ADR.
- **Concurrencia optimista** via columna `version` para entidades criticas.

---

## 5. Identidad y seguridad

- **PK interna**: `UUID` (preferir `UUIDv7` cuando sea posible para orden temporal).
- **Identificador de negocio**: `document_type` + `document_number` (compuesto).
- **Email**: `nullable`. Hay residentes mayores y personal sin correo.
- **MFA**: obligatorio para todos los roles operativos y administrativos
  (`platform_superadmin`, `tenant_admin`, `accountant`, `guard`).
- **Contrasenas**: `bcrypt` o `argon2id`, nunca SHA simple.
- **JWT/sesiones**: cookie HttpOnly + Secure + SameSite=Strict; expiracion corta + refresh.

---

## 6. Errores y observabilidad

- Errores HTTP siguen **RFC 7807** (Problem Details), `application/problem+json`.
- Logs estructurados (JSON) con `request_id`, `tenant_id` (resuelto en middleware),
  `user_id`, `trace_id`.
- Cada handler emite span de OpenTelemetry.

---

## 7. Reglas de codigo

- `gofmt` + `goimports` siempre.
- `golangci-lint` sin warnings antes de merge.
- Funciones de dominio puras y testeables sin DB.
- DTO != entidad. Mapeo explicito.
- Migraciones siempre con par Up/Down probado.
- OpenAPI 3.0 obligatorio por endpoint nuevo (en `/docs/openapi/`).

---

## 8. Como ejecutar fases del plan

El plan completo vive en `PLAN_MAESTRO.md`. Para ejecutar una fase:

```
/fase 0
/fase 1
...
```

(Comando custom definido en `.claude/commands/fase.md`.)

Antes de iniciar cualquier fase: verificar que la fase anterior cumplio su
**Definition of Done** (DoD). NO saltar fases.

---

## 9. Multi-agente: cuando paralelizar

- ADRs independientes -> un agente por ADR en paralelo.
- Modulos sin dependencias mutuas -> un agente por modulo en paralelo.
- Investigacion abierta -> agente `Explore` con thoroughness apropiada.
- Diseno arquitectonico complejo -> agente `Plan`.
- Codigo paralelo en archivos disjuntos -> agentes `general-purpose` en paralelo.

NO paralelizar cuando hay dependencias secuenciales o cuando dos agentes
podrian editar el mismo archivo.

---

## 10. Definition of Done por fase (plantilla universal)

Una fase NO se da por terminada hasta cumplir esta lista:

- [ ] Codigo compila: `go build ./...` sin errores.
- [ ] Linter limpio: `golangci-lint run` sin warnings.
- [ ] Tests unitarios pasan: `go test ./...`.
- [ ] Tests de integracion pasan (Testcontainers).
- [ ] Migraciones Up/Down ejecutadas y reversibles sin bloqueo.
- [ ] OpenAPI actualizado en `/docs/openapi/`.
- [ ] ADRs relevantes escritos en `/docs/adr/`.
- [ ] CHANGELOG / README actualizado marcando fase completa.
- [ ] PR de la fase con descripcion vinculando a este plan.
