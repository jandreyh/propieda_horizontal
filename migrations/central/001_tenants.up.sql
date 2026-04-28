-- Control Plane: registro maestro de tenants.
--
-- Cada fila representa un conjunto residencial provisionado en la
-- plataforma. La columna `database_url` apunta a la base PostgreSQL
-- aislada del tenant (Data Plane).

CREATE TABLE IF NOT EXISTS tenants (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT         NOT NULL UNIQUE,
    display_name  TEXT         NOT NULL,
    database_url  TEXT         NOT NULL,
    status        TEXT         NOT NULL DEFAULT 'active',
    plan          TEXT         NOT NULL DEFAULT 'pilot',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    activated_at  TIMESTAMPTZ,
    suspended_at  TIMESTAMPTZ,
    CONSTRAINT tenants_status_chk CHECK (status IN ('active', 'suspended', 'provisioning', 'archived')),
    CONSTRAINT tenants_slug_chk   CHECK (slug ~ '^[a-z0-9](-?[a-z0-9])*$' AND length(slug) BETWEEN 1 AND 63)
);

CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants (status);
