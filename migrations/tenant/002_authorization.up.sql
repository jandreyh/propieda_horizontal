-- Tenant DB: modulo authorization (RBAC con permisos por namespace + scopes).
--
-- Crea las cuatro tablas operativas del modulo de autorizacion segun
-- ADR 0003:
--   * roles                  : roles del producto (semilla, is_system=true)
--                              y custom por tenant.
--   * permissions            : catalogo estatico de permisos (namespaces).
--   * role_permissions       : relacion N:N rol -> permiso.
--   * user_role_assignments  : asignaciones de roles a usuarios con scope
--                              opcional (tenant/tower/unit).
--
-- Reglas obligatorias (CLAUDE.md / ADR 0001 / ADR 0003):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar (id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version) en tablas operativas.
--   * permissions NO lleva version (catalogo estatico, sin concurrencia
--     optimista).
--   * user_role_assignments NO lleva soft delete (deleted_*) porque
--     materializa sus revocaciones via revoked_at + revocation_reason.

CREATE TABLE IF NOT EXISTS roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT         NOT NULL UNIQUE,
    description TEXT         NULL,
    is_system   BOOLEAN      NOT NULL DEFAULT false,
    status      TEXT         NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ  NULL,
    created_by  UUID         NULL REFERENCES users(id),
    updated_by  UUID         NULL REFERENCES users(id),
    deleted_by  UUID         NULL REFERENCES users(id),
    version     INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT roles_status_chk
        CHECK (status IN ('active', 'archived'))
);

CREATE INDEX IF NOT EXISTS roles_status_idx
    ON roles (status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace   TEXT         NOT NULL UNIQUE,
    description TEXT         NULL,
    status      TEXT         NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ  NULL,
    created_by  UUID         NULL REFERENCES users(id),
    updated_by  UUID         NULL REFERENCES users(id),
    deleted_by  UUID         NULL REFERENCES users(id),
    CONSTRAINT permissions_status_chk
        CHECK (status IN ('active', 'archived'))
);

CREATE INDEX IF NOT EXISTS permissions_namespace_idx
    ON permissions (namespace)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID         NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID         NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS role_permissions_permission_idx
    ON role_permissions (permission_id);

CREATE TABLE IF NOT EXISTS user_role_assignments (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id            UUID         NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_type         TEXT         NULL,
    scope_id           UUID         NULL,
    granted_by         UUID         NULL REFERENCES users(id),
    granted_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    revoked_at         TIMESTAMPTZ  NULL,
    revocation_reason  TEXT         NULL,
    status             TEXT         NOT NULL DEFAULT 'active',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by         UUID         NULL REFERENCES users(id),
    updated_by         UUID         NULL REFERENCES users(id),
    version            INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT user_role_assignments_status_chk
        CHECK (status IN ('active', 'revoked')),
    CONSTRAINT user_role_assignments_scope_type_chk
        CHECK (scope_type IS NULL OR scope_type IN ('tenant', 'tower', 'unit')),
    CONSTRAINT user_role_assignments_scope_pair_chk
        CHECK (
            (scope_type IS NULL AND scope_id IS NULL)
            OR (scope_type = 'tenant' AND scope_id IS NULL)
            OR (scope_type IN ('tower', 'unit') AND scope_id IS NOT NULL)
        )
);

CREATE INDEX IF NOT EXISTS user_role_assignments_user_active_idx
    ON user_role_assignments (user_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS user_role_assignments_role_idx
    ON user_role_assignments (role_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS user_role_assignments_scope_idx
    ON user_role_assignments (scope_type, scope_id)
    WHERE revoked_at IS NULL;
