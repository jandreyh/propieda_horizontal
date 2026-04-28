# ADR 0003 — Autorizacion RBAC con permisos por namespace y scopes

- **Estado:** Accepted
- **Fecha:** 2026-04-28
- **Autores:** Plataforma / Arquitectura
- **Relacionado:** ADR 0001 (multi-tenant database-per-tenant), ADR 0002 (auth/sesion)

## Contexto

La plataforma SaaS de Propiedad Horizontal opera en modelo multi-tenant **con base de datos por tenant** (ADR 0001) sobre Go (chi, pgx, sqlc) y PostgreSQL 18. Cada tenant tiene perfiles muy distintos (administrador, contador, vigilante, propietario, residente, miembro de consejo, revisor fiscal, residente autorizado) con necesidades de granularidad fina: un guard de la torre 1 no debe operar paquetes de la torre 2; un revisor fiscal solo lee modulos contables; un residente autorizado actua en nombre de una unidad especifica.

Necesitamos un modelo de autorizacion que: (i) sea predecible y auditable, (ii) permita roles custom por tenant sin abrir un agujero de privilegios cruzados, (iii) soporte filtrado por recurso (torre/unidad) en consultas SQL, (iv) sea barato de evaluar en hot path HTTP, y (v) no dependa de un servicio externo (Keto, OPA) en el MVP.

**Regla dura del modelo (CLAUDE.md y ADR 0001):** las tablas de RBAC operativas (`roles`, `permissions`, `role_permissions`, `user_role_assignments`) viven **dentro de la base del tenant**. El tenant es implicito por la base; NO se incluye columna `tenant_id` en estas tablas ni en sus claves UNIQUE. Las definiciones comunes (catalogo de permisos, roles semilla del producto) se inyectan en cada Tenant DB durante el provisioning via migracion seed; **no existe una tabla "global de roles" en el Control Plane**.

## Decision

- Adoptar **RBAC con permisos por namespace + scopes opcionales**, deny-by-default, union de capacidades.
- **Permisos** son strings con namespace `recurso.accion` (`package.deliver`, `visit.approve`, `role.create`). Catalogo **estatico**, versionado en migracion seed inyectada en cada Tenant DB durante el provisioning; tenants **no** crean permisos nuevos.
- **Roles** son nombrados; existen roles semilla del producto (insertados via seed en cada Tenant DB, marcados `is_system = true` e inmutables a nivel de definicion) y roles **custom por tenant** que se construyen como combinacion del catalogo de permisos. Todos viven en la misma tabla `roles` del Tenant DB.
- **Roles semilla (en cada Tenant DB):** `tenant_admin`, `accountant`, `guard`, `owner`, `tenant_resident`, `authorized_resident`, `board_member`, `auditor_or_revisor`. El rol `platform_superadmin` **no se materializa** en las Tenant DB: vive en el Control Plane (ver ADR 0002, tabla `platform_users`) y sus chequeos viven en un middleware separado.
- **Scopes** en MVP: `tenant` (default implicito; representa "toda la base del tenant"), `tower`, `unit`. **POST-MVP** explicitamente: `stage` (etapa), `module`, `schedule` (horario p.ej. guard 06:00–18:00). El campo se modela desde ya pero los tipos no soportados se rechazan en escritura.
- Asignacion = `(user, role, scope_type?, scope_id?)`. Un usuario puede tener N asignaciones; el conjunto efectivo de permisos es la **union** (sin denies explicitos en MVP).
- **Prevalencia:** si dos asignaciones otorgan el mismo permiso con scopes distintos, gana la que **cubra** el recurso solicitado (`tenant` cubre todo > `tower` cubre sus units > `unit` cubre solo esa unit). No hay conflicto: la pregunta siempre es "¿alguna asignacion concede `ns` sobre `recurso`?".
- Middleware Go `RequirePermission(ns string)` (chi) lee `user_id` del contexto (el tenant lo da el connection pool ya inyectado por `TenantResolver`), resuelve permisos, y opcionalmente recibe un extractor de scope (`WithScope(func(r) (scopeType, scopeID))`) para chequeo a nivel de recurso.
- **Cache en proceso por sesion:** TTL corto (60s) de `(permisos, scopes)` por `(user_id, session_id)` con invalidacion explicita ante cambios de asignaciones. Sin Redis en MVP. Como el cache es local al pool del tenant, la clave no necesita codificar el tenant.
- `platform_superadmin` vive en el Control Plane y **nunca** se asigna como rol de tenant; sus chequeos viven en un middleware separado que consulta `platform_users` en la base central.
- Auditoria: toda asignacion/revocacion de rol genera evento en `audit_log` del Tenant DB (ADR pendiente).

## Consecuencias

**Positivas**

- Modelo entendible por humanos (roles tipicos del dominio) con extensibilidad real (permisos namespaced + scopes).
- Esquema mas limpio: sin columna `tenant_id` redundante en tablas operativas; los joins entre `users`, `user_role_assignments`, `roles` y `role_permissions` no requieren filtro de tenant porque la base entera es de ese tenant.
- Evaluacion O(n) sobre un set pequeno (permisos por usuario suelen ser <100); cacheable trivialmente.
- Filtrado en SQL barato: `WHERE tower_id = ANY($scopes_tower)` se construye desde el set efectivo.
- Catalogo estatico evita "permission sprawl" entre tenants y permite reviews de seguridad centralizadas, aun cuando el catalogo este replicado en cada Tenant DB (la migracion seed es la unica fuente de verdad).

**Negativas / costos**

- Cambiar el catalogo de permisos exige migracion de plataforma corrida en **todas** las Tenant DB (no self-service). Mitigacion: namespaces amplios, review trimestral del catalogo, herramienta de fan-out de migraciones por tenant.
- La logica de prevalencia de scopes tiene casos borde (ej. permiso con scope `unit:U-204` debe permitir leer la unidad pero no listar todas; cada handler debe decidir si usa el scope como filtro o como guard). Mitigacion: helpers de scope filtering en `internal/authz`.
- Cache en proceso obliga a invalidacion explicita; en cluster con N replicas esto es eventualmente consistente (max 60s de stale). Aceptable para MVP.
- Sin denies explicitos: si en el futuro se requiere "todo menos X", habra que rediseñar (probable pase a politicas).

## Alternativas consideradas

- **Tabla `roles` global en el Control Plane con `tenant_id`.** Descartada: contradice el modelo database-per-tenant del ADR 0001 y la regla dura de CLAUDE.md. Ademas, evaluar permisos en hot path requeriria saltar entre la base central y la del tenant, encareciendo el RBAC.
- **ABAC puro (atributos + politicas).** Mas expresivo pero mas dificil de auditar y de cachear; requiere DSL o motor de politicas. Sobreingenieria para el dominio (los casos reales son enumerables).
- **ACL por recurso (lista de usuarios por objeto).** Explota en escritura (cada paquete, cada visita, cada anuncio tendria su ACL). Inviable a escala de un conjunto con miles de paquetes/mes.
- **Casbin / Ory Keto.** Casbin agrega DSL y dependencia; Keto agrega un servicio externo y latencia de red en hot path. Para el MVP el costo operativo no se justifica; reconsiderar si aparece necesidad de relaciones (Zanzibar-style: "delegado del propietario de la unidad").
- **Roles planos sin scope.** Insuficiente: el caso del guard por torre es requisito explicito de negocio.

## Implicaciones tecnicas

### Esquema SQL minimo (Tenant DB)

Estas tablas viven **dentro de cada Tenant DB**. Ninguna lleva columna `tenant_id`: la pertenencia al tenant es implicita por la base y la unicidad de `name` en `roles` es global dentro de la Tenant DB.

```sql
CREATE TABLE permissions (
    namespace   TEXT PRIMARY KEY,           -- 'package.deliver'
    description TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name         TEXT NOT NULL UNIQUE,      -- unico dentro de la Tenant DB
    is_system    BOOLEAN NOT NULL DEFAULT false,  -- true = rol semilla del producto
    description  TEXT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
    role_id        UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_ns  TEXT NOT NULL REFERENCES permissions(namespace),
    PRIMARY KEY (role_id, permission_ns)
);

CREATE TYPE scope_type AS ENUM ('tenant', 'tower', 'unit'); -- post-MVP: 'stage','module','schedule'

CREATE TABLE user_role_assignments (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id),
    scope_type  scope_type NOT NULL DEFAULT 'tenant',
    scope_id    UUID NULL,                  -- NULL si scope_type='tenant'
    granted_by  UUID NOT NULL REFERENCES users(id),
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at  TIMESTAMPTZ NULL,
    CHECK ((scope_type = 'tenant' AND scope_id IS NULL)
        OR (scope_type <> 'tenant' AND scope_id IS NOT NULL))
);

CREATE INDEX idx_assign_user_active
    ON user_role_assignments (user_id)
    WHERE revoked_at IS NULL;
```

Notas sobre el esquema:

- `roles.name` es unico globalmente dentro de la Tenant DB. Roles semilla del producto (`tenant_admin`, `accountant`, etc.) y roles custom comparten namespace; la migracion seed reserva los nombres de los roles del producto.
- `is_system = true` marca los roles que la migracion seed inyecto y que la API de gestion no permite editar/borrar.
- `permissions` se popula en el provisioning de cada Tenant DB con el catalogo estatico del producto.
- No hace falta scope `tenant`-explicito ni FKs a tablas centrales: el "tenant" es la base entera.

### Roles semilla y `platform_superadmin`

- En cada Tenant DB, la migracion seed ejecuta inserts equivalentes a:

```sql
INSERT INTO roles (name, is_system, description) VALUES
    ('tenant_admin',        true, 'Administrador del conjunto'),
    ('accountant',          true, 'Contador'),
    ('guard',               true, 'Vigilante / porteria'),
    ('owner',               true, 'Propietario'),
    ('tenant_resident',     true, 'Residente arrendatario'),
    ('authorized_resident', true, 'Residente autorizado por propietario'),
    ('board_member',        true, 'Miembro del consejo'),
    ('auditor_or_revisor',  true, 'Revisor fiscal / auditor');
```

- El rol `platform_superadmin` **no se inserta** en ninguna Tenant DB. Vive como identidad en `platform_users` (Control Plane, ADR 0002) y se evalua en un middleware separado que no toca el `users` ni el `user_role_assignments` del tenant.

### Chequeo en Go (esquema)

```go
// internal/authz/middleware.go
func RequirePermission(ns string, opts ...Opt) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            actor := authn.FromCtx(r.Context())  // user_id, session_id (tid implicito en el pool)
            tdb   := dbctx.From(r.Context())     // pool del Tenant DB ya inyectado por TenantResolver

            eff, err := cache.Effective(r.Context(), tdb, actor) // permisos + scopes, TTL 60s
            if err != nil { httpx.Fail(w, err); return }

            scopeT, scopeID := resolveScope(r, opts) // tenant si no hay extractor
            if !eff.Allows(ns, scopeT, scopeID) {
                httpx.Forbidden(w, ns); return
            }
            next.ServeHTTP(w, r.WithContext(authz.With(r.Context(), eff)))
        })
    }
}

// Uso:
r.With(authz.RequirePermission("package.deliver",
    authz.WithScope(scope.FromTowerParam("towerID")))).
    Post("/towers/{towerID}/packages", h.Deliver)
```

`eff.Allows` itera asignaciones del usuario y devuelve `true` si alguna concede `ns` con scope que **cubre** `(scopeT, scopeID)`: `tenant` cubre todo, `tower:T1` cubre `tower:T1` y `unit:U` si U pertenece a T1 (resuelto en build del cache, no en hot path).

### Namespaces semilla MVP

`package.read`, `package.deliver`, `package.handover`, `visit.create`, `visit.approve`, `visit.read`, `announcement.read`, `announcement.publish`, `unit.read`, `unit.update`, `resident.read`, `resident.invite`, `role.read`, `role.create`, `role.assign`, `settings.read`, `settings.update`, `branding.read`, `branding.write`, `accounting.read`, `accounting.write`, `audit.read`.

### Mapeo inicial rol → permisos (resumen)

- `tenant_admin`: todos excepto `audit.read` (que se asigna explicitamente al revisor).
- `accountant`: `accounting.*`, `unit.read`, `resident.read`, `settings.read`.
- `guard`: `package.read`, `package.deliver`, `package.handover`, `visit.create`, `visit.approve`, `visit.read` (scope tipico `tower`).
- `owner`: `unit.read`, `package.read`, `visit.create`, `announcement.read` (scope tipico `unit`).
- `tenant_resident`: como `owner` sin `unit.read` global.
- `authorized_resident`: `package.read`, `visit.create` (scope `unit`).
- `board_member`: lecturas amplias + `announcement.publish`.
- `auditor_or_revisor`: `audit.read`, `accounting.read`, `settings.read`.
- `platform_superadmin`: control plane (`platform_users`), fuera de este modelo, evaluado por middleware separado.
