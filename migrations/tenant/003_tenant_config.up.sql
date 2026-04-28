-- Tenant DB: modulo tenant_config.
--
-- Crea las dos tablas operativas del modulo de configuracion del tenant:
--   * tenant_settings : pares (key, value JSONB) con descripcion y categoria.
--   * tenant_branding : fila singleton (forzada via UNIQUE) con la imagen
--                       del conjunto (display_name, logo, colores, tz, locale).
--
-- Reglas obligatorias (CLAUDE.md):
--   * NO existe columna tenant_id (la base entera ya es del tenant).
--   * Campos estandar: id, status, created_at, updated_at, deleted_at,
--     created_by, updated_by, deleted_by, version.
--   * El branding del tenant es uno y solo uno: forzamos la singularidad
--     con la columna `singleton BOOL UNIQUE` con default `true`.

CREATE TABLE IF NOT EXISTS tenant_settings (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    key          TEXT         NOT NULL UNIQUE,
    value        JSONB        NOT NULL,
    description  TEXT         NULL,
    category     TEXT         NULL,
    status       TEXT         NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ  NULL,
    created_by   UUID         NULL REFERENCES users(id),
    updated_by   UUID         NULL REFERENCES users(id),
    deleted_by   UUID         NULL REFERENCES users(id),
    version      INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT tenant_settings_status_chk
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT tenant_settings_key_chk
        CHECK (key ~ '^[a-z][a-z0-9_.]*$')
);

CREATE INDEX IF NOT EXISTS tenant_settings_category_idx
    ON tenant_settings (category)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tenant_branding (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    -- singleton fuerza una sola fila por tenant (la base entera ES el tenant).
    singleton        BOOLEAN      NOT NULL DEFAULT TRUE UNIQUE,
    display_name     TEXT         NOT NULL,
    logo_url         TEXT         NULL,
    primary_color    TEXT         NULL,
    secondary_color  TEXT         NULL,
    timezone         TEXT         NOT NULL DEFAULT 'America/Bogota',
    locale           TEXT         NOT NULL DEFAULT 'es-CO',
    status           TEXT         NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ  NULL,
    created_by       UUID         NULL REFERENCES users(id),
    updated_by       UUID         NULL REFERENCES users(id),
    deleted_by       UUID         NULL REFERENCES users(id),
    version          INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT tenant_branding_status_chk
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT tenant_branding_singleton_chk
        CHECK (singleton = TRUE)
);

-- Seed: settings minimos por defecto.
INSERT INTO tenant_settings (key, value, description, category, status) VALUES
    ('contact.email',                     '"admin@conjunto.test"'::jsonb, 'Email de contacto del conjunto',                'general',       'active'),
    ('contact.phone',                     '"+57 000 000 0000"'::jsonb,    'Telefono',                                      'general',       'active'),
    ('visits.require_resident_approval',  'true'::jsonb,                  'Visitas requieren aprobacion del residente',    'visits',        'active'),
    ('packages.notify_on_arrival',        'true'::jsonb,                  'Notificar al residente al recibir paquete',     'packages',      'active'),
    ('announcements.allow_attachments',   'true'::jsonb,                  'Permitir adjuntos en anuncios',                 'announcements', 'active')
ON CONFLICT (key) DO NOTHING;

-- Seed: fila singleton de branding con valores default.
INSERT INTO tenant_branding (display_name, timezone, locale, status)
VALUES ('Conjunto', 'America/Bogota', 'es-CO', 'active')
ON CONFLICT (singleton) DO NOTHING;
