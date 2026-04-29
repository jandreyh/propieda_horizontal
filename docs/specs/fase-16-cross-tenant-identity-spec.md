# Fase 16 — Spec frozen — Identidad cross-tenant + provisioning + selector

**Estado**: Frozen | Validado por: jandreyh | Fecha: 2026-04-29
**ADR base**: [0007-cross-tenant-identity.md](../adr/0007-cross-tenant-identity.md)

> **Lectura cero (resistente a sesiones)**: este documento es autocontenido.
> Cualquier sesion de Claude Code que lo abra debe poder ejecutar la fase
> entera sin contexto previo. Si algo aqui contradice el codigo actual o ADRs
> previos, **gana este documento** porque consolida una decision posterior.

## 1. Resumen ejecutivo

Re-disenar el subsistema de identidad para que una persona acceda con **un solo
login** a multiples conjuntos, viendo un selector de tenants tras login. La
tabla `users` deja de vivir en cada DB de tenant; se centraliza en
`platform_users` (DB central). Cada tenant DB tiene `tenant_user_links` que
ata la identidad global con su rol y unidad por conjunto. La vinculacion de un
usuario a un nuevo conjunto requiere que el admin pegue el **codigo unico** que
la persona le entrega; nadie puede buscar libremente cross-tenant excepto el
superadmin.

## 2. Decisiones tomadas (referenciadas a memoria persistente)

Las decisiones de negocio estan congeladas en:
- [memory/cross_tenant_identity.md](../../memory/cross_tenant_identity.md)
- [memory/provisioning_decisions.md](../../memory/provisioning_decisions.md)
- [memory/selector_session_decisions.md](../../memory/selector_session_decisions.md)
- [memory/migration_demo_decisions.md](../../memory/migration_demo_decisions.md)

Resumen para esta fase:
- 1 identidad global por persona (`platform_users`).
- `tenant_user_links` por conjunto.
- Login centralizado con email + documento + password.
- Vinculacion a conjunto solo por `public_code`.
- JWT con `memberships[]` y `current_tenant`.
- Superadmin unico crea tenants.
- Existe `platform_administrators` que agrupa N conjuntos.
- Bloqueos por tenant; ban global solo via superadmin.
- Push devices a nivel plataforma.
- `demo` se borra y resiembra.

## 3. Supuestos adoptados (no bloqueantes)

- **S1**: el `public_code` se genera con 12 chars de alfabeto reducido
  `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` (sin O/0/I/l/1) en formato
  `XXXX-XXXX-XXXX`. Colision practicamente nula con UNIQUE constraint.
- **S2**: el JWT incluye hasta 50 `memberships`. Si la persona excede 50
  conjuntos (caso teorico), se trunca al JWT y se ofrece paginacion via
  `GET /me/memberships`.
- **S3**: para esta fase se reusa **un mismo cluster Postgres** (la instancia
  `pg-tenant-template` del compose dev) creando una DB nueva por cada tenant.
  Separacion fisica por plan queda fuera de alcance.
- **S4**: la regenerac​ion del `public_code` (cuando un usuario lo pide) se
  difiere a fase futura. En V1 el codigo se genera al crear el usuario y no
  cambia.
- **S5**: la app Flutter mantiene Material 3 + indigo. La pantalla nueva de
  selector replica el diseno del web.

## 4. Open Questions (resolver antes de programar)

- **Q1**: tenants existentes con datos — confirmar borrado total via
  `migration_demo_decisions.md`. **Resuelto**: borrar `demo`.
- **Q2**: ¿el superadmin tiene MFA obligatorio? Asumido: si.
- **Q3**: ¿push devices se desactivan cuando el usuario cierra sesion en ese
  device? Asumido: si, mediante `DELETE /me/push-devices/{id}`.
- **Q4**: ¿el switcher de tenant deberia mostrar branding/logo? Asumido: si,
  consume `tenant_branding` de cada conjunto en el call inicial al selector.

## 5. Modelo de datos propuesto

### 5.1. Migraciones DB Central (`migrations/central/`)

Nueva migracion `002_platform_identity.up.sql`:

```sql
CREATE TABLE platform_users (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type          TEXT         NOT NULL,
    document_number        TEXT         NOT NULL,
    names                  TEXT         NOT NULL,
    last_names             TEXT         NOT NULL,
    email                  TEXT         NOT NULL,
    phone                  TEXT         NULL,
    photo_url              TEXT         NULL,
    password_hash          TEXT         NOT NULL,
    mfa_secret             TEXT         NULL,
    mfa_enrolled_at        TIMESTAMPTZ  NULL,
    public_code            TEXT         NOT NULL,
    failed_login_attempts  INTEGER      NOT NULL DEFAULT 0,
    locked_until           TIMESTAMPTZ  NULL,
    last_login_at          TIMESTAMPTZ  NULL,
    status                 TEXT         NOT NULL DEFAULT 'active',
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ  NULL,
    CONSTRAINT platform_users_email_unique UNIQUE (email),
    CONSTRAINT platform_users_document_unique UNIQUE (document_type, document_number),
    CONSTRAINT platform_users_public_code_unique UNIQUE (public_code),
    CONSTRAINT platform_users_status_chk CHECK (status IN ('active', 'suspended')),
    CONSTRAINT platform_users_doctype_chk CHECK (document_type IN ('CC','CE','PA','TI','RC','NIT'))
);

CREATE INDEX platform_users_email_idx ON platform_users (lower(email));
CREATE INDEX platform_users_public_code_idx ON platform_users (public_code);
```

Nueva migracion `003_platform_administrators.up.sql`:

```sql
CREATE TABLE platform_administrators (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT         NOT NULL,
    legal_id        TEXT         NULL,
    contact_email   TEXT         NULL,
    contact_phone   TEXT         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT platform_administrators_status_chk CHECK (status IN ('active','inactive'))
);

ALTER TABLE tenants
    ADD COLUMN administrator_id UUID NULL REFERENCES platform_administrators(id),
    ADD COLUMN logo_url TEXT NULL,
    ADD COLUMN primary_color TEXT NULL,
    ADD COLUMN timezone TEXT NOT NULL DEFAULT 'America/Bogota',
    ADD COLUMN country TEXT NOT NULL DEFAULT 'CO',
    ADD COLUMN currency TEXT NOT NULL DEFAULT 'COP',
    ADD COLUMN expected_units INTEGER NULL;

CREATE INDEX tenants_administrator_idx ON tenants (administrator_id);
```

Nueva migracion `004_platform_push_devices.up.sql`:

```sql
CREATE TABLE platform_push_devices (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id    UUID         NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    device_token        TEXT         NOT NULL,
    platform            TEXT         NOT NULL,
    last_seen_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT platform_push_devices_platform_chk CHECK (platform IN ('ios','android','web')),
    CONSTRAINT platform_push_devices_token_unique UNIQUE (platform_user_id, device_token)
);
```

Nueva migracion `005_platform_audit_logs.up.sql`:

```sql
CREATE TABLE platform_audit_logs (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id   UUID         NULL REFERENCES platform_users(id),
    action          TEXT         NOT NULL,
    target_type     TEXT         NULL,
    target_id       UUID         NULL,
    metadata        JSONB        NULL,
    ip              INET         NULL,
    user_agent      TEXT         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX platform_audit_logs_actor_idx ON platform_audit_logs (actor_user_id, created_at DESC);
CREATE INDEX platform_audit_logs_target_idx ON platform_audit_logs (target_type, target_id);
```

### 5.2. Migraciones DB Tenant (`migrations/tenant/`)

Nueva migracion `019_tenant_user_links.up.sql` que reemplaza `users`:

```sql
-- Drop dependencias del modelo anterior. Esto asume tenant nuevo (sin datos).
DROP TABLE IF EXISTS user_mfa_recovery_codes;
DROP TABLE IF EXISTS user_sessions;

-- IMPORTANT: NO borramos `users` directo porque hay FKs hacia ella.
-- En su lugar, renombramos y vamos transicionando.
ALTER TABLE users RENAME TO users_legacy_drop_me;

CREATE TABLE tenant_user_links (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id    UUID         NOT NULL UNIQUE,
    role                TEXT         NOT NULL,
    primary_unit_id     UUID         NULL REFERENCES units(id),
    cartera_status      TEXT         NULL,
    fecha_ingreso       DATE         NULL,
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT tenant_user_links_status_chk CHECK (status IN ('active','blocked'))
);

CREATE INDEX tenant_user_links_platform_user_idx ON tenant_user_links (platform_user_id);
CREATE INDEX tenant_user_links_role_idx ON tenant_user_links (role) WHERE deleted_at IS NULL;
CREATE INDEX tenant_user_links_unit_idx ON tenant_user_links (primary_unit_id) WHERE deleted_at IS NULL;
```

Nueva migracion `020_tenant_fk_realign.up.sql` que ajusta FKs:

```sql
-- Cada tabla que apuntaba a users(id) ahora apunta a tenant_user_links(id).
-- Como esto es destructivo y asumimos tenant fresh, solo aplica a fixtures
-- futuras. Para tenants existentes se reseed.

ALTER TABLE user_role_assignments
    DROP CONSTRAINT user_role_assignments_user_id_fkey,
    ADD CONSTRAINT user_role_assignments_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES tenant_user_links(id) ON DELETE CASCADE;

ALTER TABLE unit_owners
    DROP CONSTRAINT unit_owners_user_id_fkey,
    ADD CONSTRAINT unit_owners_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES tenant_user_links(id) ON DELETE RESTRICT;

ALTER TABLE unit_occupancies
    DROP CONSTRAINT unit_occupancies_user_id_fkey,
    ADD CONSTRAINT unit_occupancies_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES tenant_user_links(id) ON DELETE RESTRICT;

-- ... continuar con cada tabla que tenia FK a users(id):
-- packages.received_by_user_id
-- package_delivery_events.delivered_to_user_id, delivered_by_user_id
-- announcements.published_by_user_id
-- announcement_acknowledgments.user_id
-- visitor_entries.guard_id
-- visitor_pre_registrations.created_by
-- audit_logs.user_id
-- y created_by/updated_by/deleted_by en TODAS las tablas operativas.

-- Finalmente:
DROP TABLE users_legacy_drop_me CASCADE;
```

(El listado completo de FKs se genera en runtime con un script que recorra
`information_schema.key_column_usage`.)

## 6. Endpoints

### 6.1. API Central (no tenant-resolver)

| Verbo | Path | Permiso | Descripcion |
|-------|------|---------|-------------|
| POST | `/auth/login` | publico | email+document+password → JWT con memberships |
| POST | `/auth/mfa/verify` | publico | pre_auth_token + code → JWT con memberships |
| POST | `/auth/refresh` | publico | refresh_token → nuevo JWT |
| POST | `/auth/logout` | bearer | revoca sesion |
| POST | `/auth/switch-tenant` | bearer | tenant_slug → nuevo JWT con current_tenant |
| GET | `/me` | bearer | datos globales de la persona |
| GET | `/me/memberships` | bearer | lista paginada de membresias |
| POST | `/me/push-devices` | bearer | registrar device token |
| DELETE | `/me/push-devices/{id}` | bearer | desactivar device token |

### 6.2. Superadmin (no tenant-resolver, requiere rol `platform_superadmin`)

| Verbo | Path | Descripcion |
|-------|------|-------------|
| POST | `/superadmin/tenants` | crea tenant: DB+migraciones+seed admin |
| GET | `/superadmin/tenants` | lista todos los tenants |
| POST | `/superadmin/tenants/{id}/suspend` | suspende tenant |
| GET | `/superadmin/users?email=` | busca usuario por email/doc cross-tenant |
| POST | `/superadmin/users/{id}/suspend` | ban global |
| POST | `/superadmin/administrators` | crea entidad administradora |
| POST | `/superadmin/administrators/{id}/tenants/{tenant_id}` | asocia tenant a administradora |

### 6.3. Tenant (con tenant-resolver via JWT current_tenant)

| Verbo | Path | Permiso | Descripcion |
|-------|------|---------|-------------|
| POST | `/tenant-members` | `member.create` | agrega usuario por public_code |
| GET | `/tenant-members` | `member.read` | lista links del tenant |
| PUT | `/tenant-members/{id}` | `member.update` | cambia rol o unidad |
| POST | `/tenant-members/{id}/block` | `member.block` | bloquea solo en este tenant |
| GET | `/tenant-branding` | `branding.read` | lee logo + color para selector |

## 7. Permisos nuevos a registrar

- `platform.tenant.create`, `platform.tenant.read`, `platform.tenant.suspend`
- `platform.user.search`, `platform.user.suspend`
- `platform.administrator.manage`
- `member.create`, `member.read`, `member.update`, `member.block`

## 8. Casos extremos (edge cases)

- **C1**: usuario hace login con membresias, luego un admin lo bloquea en uno
  de los tenants. Cuando intenta `switch-tenant` a ese, falla con 403.
- **C2**: usuario regenera codigo (post-V1) — el codigo viejo deja de
  funcionar; los links existentes NO se rompen (ya estan creados).
- **C3**: superadmin suspende `platform_users.status='suspended'` — el JWT
  vigente sigue funcionando hasta `exp`; al refresh falla.
- **C4**: dos admins de diferentes conjuntos intentan agregar a la misma
  persona simultaneamente con su public_code → ambos exitosos, son tenants
  distintos.
- **C5**: persona pierde acceso a su email (corporativo) → debe contactar al
  superadmin manualmente para recuperar contrasena. (Self-recovery por
  documento queda fuera de V1.)
- **C6**: el JWT de un usuario con 50 conjuntos pesa demasiado → trunca a 50
  y avisa al cliente "tienes mas conjuntos, usa /me/memberships paginado".
- **C7**: tenant en estado `provisioning` mientras se corren migraciones →
  no aparece en `memberships[]` aunque exista el link.

## 9. Operaciones transaccionales / idempotentes

- `POST /superadmin/tenants` — transaccional con compensacion: si falla en
  paso N, deshace los pasos 1..N-1 (drop DB, delete tenant row).
- `POST /tenant-members` — idempotente con `Idempotency-Key`. Si el mismo
  public_code se agrega dos veces, devuelve el link ya existente, no crea
  duplicado.
- `POST /auth/switch-tenant` — idempotente: aunque se llame N veces con el
  mismo slug, devuelve el mismo JWT (mismo `iat`/`exp`).

## 10. Configuracion por tenant

Sin keys nuevas en `tenant_settings` de momento. La configuracion del tenant
en V2 podria incluir:
- `member.allow_admin_self_assign_role` (bool) — si el admin puede asignarse
  roles distintos a tenant_admin en su mismo conjunto.
- `member.require_email_verification` (bool) — si nuevos miembros deben
  confirmar email antes de tener acceso.

## 11. Notificaciones / eventos

- Evento `member.added` (tenant DB) → notif al usuario "te agregaron al
  conjunto X" via push.
- Evento `member.blocked` (tenant DB) → notif "te bloquearon en X".
- Evento `tenant.created` (central) → notif al admin inicial "tu conjunto X
  esta listo".

## 12. Reportes / metricas

- Dashboard del superadmin: conteo de tenants activos, usuarios
  globales, push devices registrados.
- Dashboard de administradora: conteo de conjuntos bajo su mano, total de
  unidades agregadas.

## 13. Riesgos y mitigaciones

(Ver seccion "Riesgos" del ADR 0007.)

## 14. Multi-agente sugerido

5 agentes paralelos para esta fase:

| Agente | Foco | Archivos disjuntos |
|--------|------|--------------------|
| A | DB central + migraciones | `migrations/central/00{2,3,4,5}_*.sql` + sqlc queries |
| B | Modulo `platform_identity` (login, switch-tenant, me) | `apps/api/internal/modules/platform_identity/` |
| C | Modulo `superadmin` (tenants, users, administrators) | `apps/api/internal/modules/superadmin/` + `provisioning/` |
| D | Reescritura `tenant_resolver` middleware + migraciones tenant FK realign | `apps/api/internal/platform/middleware/tenant_resolver.go` + `migrations/tenant/019,020_*.sql` |
| E | Frontend web: `/select-tenant`, switcher, login form 3-campos | `apps/web/src/app/select-tenant/` + `<TenantSwitcher>` + `login/page.tsx` reescrito |

Mobile Flutter (login form 3-campos + selector) puede hacerse despues como
agente F secuencial, dependiendo de E.

## 15. DoD adicional especifico de la fase

Adicional al DoD universal de CLAUDE.md:

- [ ] ADR 0002 marcado como **Superseded by 0007** en su seccion Estado.
- [ ] Migraciones central 002-005 aplicadas y reversibles.
- [ ] Migraciones tenant 019-020 aplicadas y reversibles.
- [ ] `cmd/seed-demo` reescrito; al correrlo, crea `platform_users` + tenant
  `demo` + `tenant_user_links` con admin.
- [ ] Test E2E Playwright: login con admin → ver selector de 1 conjunto →
  entra directo al dashboard. Si se siembra un segundo conjunto demo2,
  muestra ambos.
- [ ] El flujo del PR #4 (web + Flutter) sigue funcionando con el nuevo
  modelo, o se reescribe.
- [ ] OpenAPI 3.0 actualizado en `docs/openapi/platform_identity.yaml` y
  `docs/openapi/superadmin.yaml`.
- [ ] Memoria persistente en `~/.claude/projects/.../memory/` referenciada
  desde MEMORY.md.

## 16. Plan paso-a-paso resistente

> Cada paso es **autocontenido**: una sesion futura puede ejecutar solo este
> paso, marcarlo done, y la siguiente sesion arrancar el siguiente.

### Paso 16.0 — Pre-flight

- Confirmar que estamos en una rama nueva `feat/fase-16-cross-tenant-identity`
  desde `main` actualizado.
- Confirmar que el ADR 0007 esta mergeado a main (este mismo PR debe
  mergearse antes de que arranque codigo de la fase).
- Leer las 4 memorias project: `cross_tenant_identity.md`,
  `provisioning_decisions.md`, `selector_session_decisions.md`,
  `migration_demo_decisions.md`.

### Paso 16.1 — Migraciones DB Central (agente A)

- Crear `migrations/central/002_platform_identity.up.sql` y `.down.sql`.
- Crear `migrations/central/003_platform_administrators.up.sql` y `.down.sql`.
- Crear `migrations/central/004_platform_push_devices.up.sql` y `.down.sql`.
- Crear `migrations/central/005_platform_audit_logs.up.sql` y `.down.sql`.
- Probar `migrate up` y `migrate down` reversibles.
- DoD: `migrate version` reporta 5; `pg_dump --schema-only ph_central` muestra las 5 nuevas tablas.

### Paso 16.2 — Modulo `platform_identity` (agente B)

- Estructura Clean Architecture en `apps/api/internal/modules/platform_identity/`.
- DTOs: `LoginRequest{email, document_type, document_number, password}`,
  `LoginResponse{access_token, refresh_token, expires_in, token_type, mfa_required, pre_auth_token}`,
  `MeResponse`, `MembershipsResponse{items: [{tenant_id, slug, name}]}`,
  `SwitchTenantRequest{tenant_slug}`, `SwitchTenantResponse{access_token, expires_in}`.
- Usecases: `Login`, `MFAVerify`, `Refresh`, `Logout`, `Me`, `ListMemberships`,
  `SwitchTenant`, `RegisterPushDevice`, `RemovePushDevice`.
- Repositorio `PlatformUserRepository` con sqlc.
- Handlers HTTP montados en routes globales (no tras tenant_resolver).
- Tests unitarios para cada usecase + tests de integracion con Testcontainers.
- DoD: `POST /auth/login` con credenciales validas devuelve JWT con `memberships[]`.

### Paso 16.3 — Reescribir `tenant_resolver` middleware (agente D)

- Reemplazar logica de subdominio/header por lectura de `current_tenant` del JWT.
- Si `current_tenant` ausente: 412 Precondition Failed.
- Si tenant suspended: 403.
- Si membership ausente: 403.
- Mantener cache de pools por tenant (igual que hoy).
- DoD: tests del middleware verdes; tests E2E del flujo completo verdes.

### Paso 16.4 — Migraciones DB Tenant (agente D, despues de 16.3)

- Crear `migrations/tenant/019_tenant_user_links.up.sql` y `.down.sql`.
- Crear `migrations/tenant/020_tenant_fk_realign.up.sql` y `.down.sql` con
  todos los `ALTER TABLE ... FOREIGN KEY` para apuntar a `tenant_user_links`.
- Validar `migrate up` reversible.
- DoD: tenant fresh tiene `tenant_user_links` y todas las FKs ajustadas.

### Paso 16.5 — Modulo `superadmin` + `provisioning` (agente C)

- `apps/api/internal/modules/superadmin/` con endpoints listados en seccion 6.2.
- `apps/api/internal/modules/provisioning/` con `CreateTenant` que:
  1. crea fila en `tenants` (status=`provisioning`)
  2. ejecuta `CREATE DATABASE` via SQL inline (excepcion permitida porque es
     metaoperacion, no logica de negocio)
  3. corre `golang-migrate` programaticamente sobre la DB nueva
  4. corre seed roles+permisos
  5. crea `platform_users` admin si no existe (o reusa por email/doc)
  6. crea `tenant_user_links` con role=`tenant_admin`
  7. update `tenants.status='active'`
- Compensaciones en falla.
- DoD: endpoint `POST /superadmin/tenants` crea conjunto en <60s y admin puede
  hacer login + ver memberships.

### Paso 16.6 — Reescribir `cmd/seed-demo`

- Crear `platform_users` admin con email `admin@demo.ph.localhost`,
  documento `CC:1000000001`, password `admin123` (argon2id).
- Generar `public_code`.
- Crear `platform_administrators` "Demo Administradora" (opcional).
- Llamar al usecase `CreateTenant` con slug=`demo`.
- Idempotente.
- DoD: `go run ./cmd/seed-demo` resetea y resiembra correctamente.

### Paso 16.7 — Frontend web (agente E)

- Reescribir `apps/web/src/app/login/page.tsx`: form con email + documento + password.
- Crear `apps/web/src/app/select-tenant/page.tsx` server component.
- Crear `<TenantSwitcher>` client component en sidebar.
- Adaptar `lib/api/server.ts` para leer `current_tenant` del JWT y NO usar
  `X-Tenant-Slug` por env var.
- Adaptar middleware Next.js para no exigir cookie en `/select-tenant`.
- Tests Playwright E2E del nuevo flujo.
- DoD: login con admin demo → muestra selector con 1 conjunto → click → dashboard.

### Paso 16.8 — Frontend Flutter (agente F)

- Reescribir `LoginScreen` con 3 campos.
- Crear `SelectTenantScreen` con tarjetas.
- Adaptar `ApiClient` para guardar JWT con `current_tenant`.
- Endpoint `POST /me/push-devices` en cliente.
- DoD: `flutter analyze` limpio + `flutter build web --release` exitoso.

### Paso 16.9 — Verificacion final

- Ejecutar `/verificar-fase 16`.
- Demo: levantar stack, registrar 2 conjuntos via superadmin, vincular el
  mismo usuario a ambos, login → ver selector con 2 → switch entre ellos.
- Screenshots en `docs/demo-screenshots/fase-16/`.
- DoD: video o GIF demo del flujo completo.

### Paso 16.10 — Cleanup

- Marcar ADR 0002 como Superseded en su seccion Estado.
- Actualizar README con el nuevo flujo de login.
- Cerrar PR #4 si es duplicado o rebase sobre la nueva auth.
