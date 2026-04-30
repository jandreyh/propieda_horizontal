-- Empresas administradoras que gestionan N conjuntos.
-- ADR 0007 seccion B5: la entidad existe explicitamente para cobranza
-- consolidada futura y dashboard cross-tenant.

CREATE TABLE IF NOT EXISTS platform_administrators (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT         NOT NULL,
    legal_id        TEXT         NULL,
    contact_email   TEXT         NULL,
    contact_phone   TEXT         NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT platform_administrators_status_chk
        CHECK (status IN ('active', 'inactive'))
);

-- Indice para busqueda por correo de contacto en superadmin dashboard.
CREATE INDEX IF NOT EXISTS platform_administrators_contact_idx
    ON platform_administrators (lower(contact_email))
    WHERE contact_email IS NOT NULL;

-- Ampliar la tabla tenants con datos faltantes y la asociacion a administradora.
-- Si el conjunto se autogestiona, administrator_id queda NULL.
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS administrator_id UUID NULL REFERENCES platform_administrators(id),
    ADD COLUMN IF NOT EXISTS logo_url        TEXT NULL,
    ADD COLUMN IF NOT EXISTS primary_color   TEXT NULL,
    ADD COLUMN IF NOT EXISTS timezone        TEXT NOT NULL DEFAULT 'America/Bogota',
    ADD COLUMN IF NOT EXISTS country         TEXT NOT NULL DEFAULT 'CO',
    ADD COLUMN IF NOT EXISTS currency        TEXT NOT NULL DEFAULT 'COP',
    ADD COLUMN IF NOT EXISTS expected_units  INTEGER NULL;

CREATE INDEX IF NOT EXISTS tenants_administrator_idx
    ON tenants (administrator_id)
    WHERE administrator_id IS NOT NULL;
