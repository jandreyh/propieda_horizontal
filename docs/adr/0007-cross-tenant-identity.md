# ADR 0007 — Identidad cross-tenant con login centralizado y selector de conjunto

**Estado**: Accepted
**Fecha**: 2026-04-29
**Reemplaza**: parcialmente al ADR 0002 (autenticacion e identidad). Las reglas de
"login por tenant" y "no identidad global para residentes" del ADR 0002
**quedan sin efecto**. El resto del ADR 0002 (MFA obligatorio, hashing argon2id,
JWT como mecanismo, identificador de negocio compuesto doc_type+doc_number) sigue
vigente.

## Contexto

El ADR 0002 establecio "Login por tenant. NO identidad global para residentes.
Una misma persona en dos tenants se trata como dos usuarios distintos. Solo el
superadmin tiene identidad global central". Esto se implemento literal: la
tabla `users` vive dentro de cada DB de tenant, el JWT se firma contra el tenant
resuelto en el request por subdominio o header.

En la entrevista de Discovery 2026-04-29, el usuario reviso la regla a la luz de
varios casos reales del negocio:

1. **Empresas administradoras** que manejan 5–20 conjuntos a la vez. Sus admins
   no pueden tener una contrasena distinta para cada conjunto.
2. **Personal operativo que rota**: un contador o un guarda puede trabajar para
   varios conjuntos. Forzarlos a multiples cuentas y multiples passwords es
   inviable operativamente.
3. **Miembros del consejo** que viven en un conjunto y participan del consejo
   en otro.
4. **Residentes con doble propiedad**: la persona tiene apto en el conjunto A y
   casa en el conjunto B. Caso menos frecuente pero real.

Forzar identidades duplicadas en cada caso genera fricciones serias en UX,
soporte y onboarding, y crea ambiguedad sobre "que persona es la real".

Adicionalmente, durante el Discovery se identifico una **regla de privacidad**
critica: el admin de un conjunto no debe poder buscar libremente cualquier
persona en plataforma para "agregarla" a su conjunto, porque eso permitiria
enumerar en que conjuntos vive cada persona. La vinculacion debe requerir
**accion explicita del usuario duenno de la identidad** que entrega un codigo
unico al admin.

## Decision

### 1. Una sola identidad global por persona

Existe **un unico registro por persona** en la base central de plataforma,
en una tabla nueva `platform_users`. Esta tabla concentra los datos de
identificacion globales:

- nombre, apellidos, foto
- email (UNIQUE en plataforma)
- telefono
- documento (`document_type` + `document_number`, UNIQUE en plataforma)
- password_hash (argon2id, una sola contrasena por persona)
- mfa_secret, mfa_enrolled_at
- public_code (UNIQUE, generado al crear el usuario; ~12 chars alfanumericos
  legibles, sin caracteres confusos como 0/O/1/l/I)
- status, created_at, updated_at, deleted_at
- last_login_at, failed_login_attempts, locked_until

Las tablas `users` que viven dentro de cada DB de tenant **se eliminan**.
Cada tenant DB pasa a tener una tabla `tenant_user_links` que solamente vincula
una identidad global con su rol y unidad dentro del conjunto.

### 2. Tabla `tenant_user_links` por tenant

Esta tabla reemplaza a `users` en cada DB de tenant. Su forma:

```sql
CREATE TABLE tenant_user_links (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id    UUID         NOT NULL UNIQUE,
    -- platform_user_id es FK logica a platform_users.id en la DB central.
    -- NO se declara FK fisica porque la DB central esta en otra logical
    -- instance; la consistencia se valida en application layer.
    role                TEXT         NOT NULL,
    -- role es el rol semilla del tenant (tenant_admin, accountant, guard,
    -- owner, tenant_resident, authorized_resident, board_member, auditor).
    primary_unit_id     UUID         NULL REFERENCES units(id),
    cartera_status      TEXT         NULL,
    fecha_ingreso       DATE         NULL,
    status              TEXT         NOT NULL DEFAULT 'active',
    -- 'active', 'blocked' (bloqueo solo afecta a este tenant).
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    -- Sin created_by/updated_by/deleted_by porque su FK seria circular hacia
    -- esta misma tabla. Se usa platform_user_id en logs en su lugar.
    version             INTEGER      NOT NULL DEFAULT 1
);
```

### 3. Vinculacion de un usuario a un conjunto: por codigo unico

Para que un admin de tenant agregue una persona ya existente en plataforma a
su conjunto, la persona **debe entregar verbalmente o por chat el `public_code`**
que aparece en su perfil. El admin pega ese codigo en un endpoint
`POST /tenant-members` y la plataforma resuelve `platform_user_id` por
codigo, no por email ni documento.

**El admin de un tenant NO puede buscar usuarios por email, documento, telefono
ni nombre.** Solo el `platform_superadmin` tiene acceso a busquedas libres
cross-tenant via endpoints `/superadmin/users`.

Esta regla **previene enumeration**: un admin malicioso no puede descubrir
en que otros conjuntos vive un residente.

### 4. Login centralizado

El login se hace contra **una API central**. La forma del request:

```http
POST /auth/login
Content-Type: application/json

{
  "email": "...",
  "document_type": "CC",
  "document_number": "...",
  "password": "..."
}
```

El servidor valida los tres factores (email + documento + contrasena) contra
`platform_users`. La idea de pedir documento + email es una capa adicional
de verificacion que el usuario quiere mantener.

Tras login exitoso, el JWT se emite con esta forma:

```json
{
  "sub": "<platform_user_id>",
  "iss": "ph-saas",
  "aud": "ph-platform",
  "iat": ...,
  "exp": ...,
  "memberships": [
    {"tenant_id": "uuid", "tenant_slug": "demo", "tenant_name": "Conjunto Demo"},
    {"tenant_id": "uuid", "tenant_slug": "acacias", "tenant_name": "Acacias 21"}
  ],
  "current_tenant": null,
  "amr": ["pwd"]
}
```

`current_tenant` arranca en null. El frontend muestra la pantalla de seleccion.

### 5. Comportamiento del selector

- Si `memberships.length === 0` → pantalla "sin acceso, espera invitacion con
  tu codigo unico".
- Si `memberships.length === 1` → llama `POST /auth/switch-tenant` con el unico
  slug, recibe nuevo JWT con `current_tenant` ya asignado, y entra al
  dashboard.
- Si `memberships.length > 1` → muestra tarjetas, el usuario hace click,
  llama `/auth/switch-tenant`, recibe nuevo JWT, entra al dashboard.

### 6. Switcher post-login

En el header / sidebar del dashboard hay un componente `<TenantSwitcher>`
con la lista de membresias del JWT actual. Al cambiar de conjunto:

```http
POST /auth/switch-tenant
Authorization: Bearer <jwt-actual>
Content-Type: application/json

{"tenant_slug": "acacias"}
```

El servidor valida que `acacias` este en `memberships[]` del JWT, re-firma un
JWT nuevo con `current_tenant="acacias"` y mismo `exp`, y lo devuelve.

El refresh_token NO cambia (la membresia no cambio, solo el contexto activo).

### 7. Tenant resolver redisenado

El middleware `tenant_resolver` deja de usar subdominio o header como fuente
primaria. Pasa a leer `current_tenant` del JWT como fuente unica de verdad.

- Si la ruta es `/auth/*` → no se aplica resolver (auth es global).
- Si la ruta es `/superadmin/*` → tampoco se aplica resolver.
- Para todo lo demas → se exige `current_tenant` en el JWT. Si `current_tenant`
  es null → 412 Precondition Failed con mensaje "select a tenant".

### 8. Bloqueos por tenant, no por plataforma

Si el admin del conjunto A bloquea a un usuario, **se marca su
`tenant_user_links.status = 'blocked'` solo en la DB de A**. El
`platform_users.status` sigue en `active` y el usuario puede seguir accediendo
a sus otros conjuntos.

Un "ban global" requiere intervencion del `platform_superadmin` cambiando
manualmente `platform_users.status = 'suspended'`.

### 9. Auditoria

Los logs de operacion siguen viviendo en cada tenant DB (`audit_logs`
existente). No se introduce una tabla central global de auditoria de
operaciones — solo `platform_audit_logs` para eventos de plataforma
(provisioning, impersonation, suspension de tenants/usuarios).

### 10. Push notifications cross-tenant

Las notificaciones se enrutan a la persona, no al tenant.

- `platform_push_devices(platform_user_id, device_token, platform, last_seen_at)`
  vive en la DB central.
- Cada tenant DB tiene su outbox local de notificaciones, pero los eventos
  se emiten con `recipient_platform_user_id` (UUID, FK logica al global).
- Un worker central consume todos los outboxes y dispara FCM/APNs al device
  correcto.
- En el body de la notif se incluye `tenant_id` y `tenant_name` para que el
  cliente sepa de cual conjunto vino y abra el conjunto correcto al tap.

## Consecuencias

### Positivas

- UX simple: una sola contrasena por persona, un solo login.
- Soporte mas barato: una identidad por persona.
- Privacidad: el admin no puede enumerar otros conjuntos de un residente.
- Operacion realista para administradoras y personal operativo que rota.
- Mobile sigue funcionando con un solo bundle (no apps por tenant).

### Negativas

- Migracion no trivial. Todas las tablas del tenant que tienen FK a `users(id)`
  pasan a referenciar `tenant_user_links(id)`. Implica nuevas migraciones
  `019_*` en adelante y reescritura de los repositorios de identity y
  authorization.
- El backend ya tiene `users`, `user_sessions`, `user_role_assignments` y
  varias tablas con FKs hacia `users(id)` en la DB del tenant. Hay que
  reescribir mas de 10 migraciones SQL.
- El JWT crece (lleva `memberships[]`). Si el usuario tiene 50 conjuntos, el
  token crece. Mitigacion: limitar a 50 entradas y truncar en el JWT, ofrecer
  un endpoint paginado `GET /me/memberships?page=...` para listas largas.
- El cliente web pasa de "tenant resolver por subdominio" a "tenant resolver
  por JWT". Esto invalida la URL convencion `conjunto.dominio.com`. La nueva
  URL es `app.dominio.com/{path}` con el conjunto activo en el contexto
  client-side.
- Se rompe la regla del ADR 0001 sobre subdominios. Mantenemos el ADR 0001
  para "DB por tenant", pero ya no para "resolucion por subdominio".

### Riesgos

- **Filtracion del codigo unico**: si una persona comparte su codigo en un
  chat publico, alguien podria intentar agregarla a un conjunto. Mitigacion:
  el codigo se rota cuando el usuario lo solicite ("regenerar codigo"); si
  alguien intenta agregar un codigo ya invalidado, falla.
- **JWT pesado en serializacion**: con muchas membresias, cada request lleva
  mas bytes. Mitigacion: el campo `memberships[]` solo lleva
  `(tenant_id, slug, name)`, no permisos. Permisos se resuelven server-side
  por `current_tenant` consultando `tenant_user_links`.
- **Inconsistencia entre central y tenant**: si el `platform_user_id` cambia
  o se borra, los `tenant_user_links` quedan colgados. Mitigacion: soft
  delete en `platform_users`, los links no se eliminan; cuando el usuario
  intenta entrar, falla en el login porque el global esta suspended.

## Alternativas consideradas

| Alternativa | Por que se descarto |
|-------------|---------------------|
| Mantener ADR 0002 literal | El usuario lo descarto por las razones del Contexto. |
| Identidad replicada por tenant + tabla central de "links" | Mas complejo: dos fuentes de verdad para nombre/email. Genera ambiguedad. |
| Federation por documento sin tabla central | Imposible mantener password unica sin centralizar. |
| Subdominio + login por tenant pero "passwords sincronizadas" via servicio | Magico, fragil, no es industria estandar. |
| Usar Auth0 / Cognito / Keycloak | Costo + dependencia externa + perdemos control sobre el flow del codigo unico. |

## Implicaciones tecnicas concretas

### Esquema de la base central nueva (resumen)

```sql
-- platform_users (NUEVA)
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
    -- 'active', 'suspended' (suspended = ban global por superadmin).
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ  NULL,
    CONSTRAINT platform_users_email_unique UNIQUE (email),
    CONSTRAINT platform_users_document_unique UNIQUE (document_type, document_number),
    CONSTRAINT platform_users_public_code_unique UNIQUE (public_code),
    CONSTRAINT platform_users_status_chk CHECK (status IN ('active', 'suspended'))
);

-- platform_administrators (NUEVA, opcional para agrupar conjuntos)
CREATE TABLE platform_administrators (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT         NOT NULL,
    legal_id        TEXT         NULL,
    contact_email   TEXT         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- tenants (existe; se le agrega administrator_id)
ALTER TABLE tenants ADD COLUMN administrator_id UUID NULL REFERENCES platform_administrators(id);

-- platform_push_devices (NUEVA, para notificaciones cross-tenant)
CREATE TABLE platform_push_devices (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id    UUID         NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    device_token        TEXT         NOT NULL,
    platform            TEXT         NOT NULL,  -- 'ios','android','web'
    last_seen_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- platform_audit_logs (existe en concepto; se materializa)
CREATE TABLE platform_audit_logs (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id     UUID         NULL REFERENCES platform_users(id),
    action            TEXT         NOT NULL,
    target_type       TEXT         NULL,
    target_id         UUID         NULL,
    metadata          JSONB        NULL,
    ip                INET         NULL,
    user_agent        TEXT         NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

### Esquema en cada DB de tenant

- DROP TABLE `users` (y dependencias `user_sessions`, `user_mfa_recovery_codes`).
- CREATE TABLE `tenant_user_links` (ver seccion 2).
- ALTER cada tabla con FK a `users(id)` para apuntar a `tenant_user_links(id)`.

### Backend Go

- Nuevo modulo `internal/modules/platform_identity/` (vive en la DB central).
  - `domain/entities/PlatformUser.go`
  - `application/usecases/Login.go`, `SwitchTenant.go`, `Refresh.go`, `Me.go`,
    `RegisterPushDevice.go`
  - `interfaces/http/routes.go` con `POST /auth/login`, `POST /auth/switch-tenant`,
    `POST /auth/refresh`, `POST /auth/logout`, `GET /me`, `GET /me/memberships`.
- Modulo `internal/modules/identity/` actual (que vive en tenant DB) **se
  reduce** a manejar `tenant_user_links`. Pierde login, sesiones, MFA — eso
  es responsabilidad del modulo central.
- Middleware `tenant_resolver` se reescribe para tomar `current_tenant` del JWT.
- Modulo `internal/modules/superadmin/` para `POST /superadmin/tenants`,
  `GET /superadmin/users?email=`, `POST /superadmin/users/{id}/suspend`.
- Modulo `internal/modules/provisioning/` para `application/usecases/CreateTenant.go`
  que crea DB, aplica migraciones, corre seed.

### Frontend web

- Nueva pagina `/select-tenant` (server component).
- `<TenantSwitcher>` en sidebar.
- API client deja de fijar `X-Tenant-Slug` por env var; lo lee del JWT
  decodificado en server-side.
- Login form pide email + documento + password (3 campos).

### Frontend mobile (Flutter)

- LoginScreen pide los 3 campos.
- SelectTenantScreen muestra tarjetas si `memberships.length > 1`.
- HomeScreen lee `current_tenant` y muestra el switcher en el AppBar.
- Push tokens se registran via `POST /me/push-devices` (no por tenant).

## Tareas de seguimiento

- [ ] Implementar `Fase 16` segun `docs/specs/fase-16-cross-tenant-identity-spec.md`.
- [ ] Mergear PR #4 (frontend MVP + Flutter) ANTES de Fase 16, o despues como
      base para reescritura de auth web. Decidir en su momento.
- [ ] Revocar el ADR 0002 explicitamente en su seccion "Estado".
- [ ] Reescribir `cmd/seed-demo` para crear `platform_users` + `tenant_user_links`.
- [ ] Documentar en README el nuevo flujo de login con los 3 campos.
- [ ] Revisar el ADR 0006 de mobile Flutter — el flujo de auth cambia.
