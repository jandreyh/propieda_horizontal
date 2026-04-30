-- Reversion best-effort. La estructura original de `users` era amplia
-- (con sesiones, MFA, intentos fallidos, etc.). Aqui solo restauramos
-- la forma minima para que un `migrate up` posterior funcione. Los
-- datos NO se restauran.

-- 1) Drop FKs hacia tenant_user_links capturadas.
DO $do$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT tc.table_name AS tbl, tc.constraint_name AS cons
        FROM information_schema.table_constraints tc
        JOIN information_schema.constraint_column_usage ccu
          ON tc.constraint_name = ccu.constraint_name
         AND tc.table_schema    = ccu.table_schema
        WHERE tc.constraint_type = 'FOREIGN KEY'
          AND ccu.table_schema   = 'public'
          AND ccu.table_name     = 'tenant_user_links'
          AND ccu.column_name    = 'id'
    LOOP
        EXECUTE format('ALTER TABLE public.%I DROP CONSTRAINT %I',
                       r.tbl, r.cons);
    END LOOP;
END
$do$;

-- 2) Drop tenant_user_links.
DROP TABLE IF EXISTS tenant_user_links CASCADE;

-- 3) Recrear users con su forma minima original.
CREATE TABLE IF NOT EXISTS users (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type          TEXT         NOT NULL,
    document_number        TEXT         NOT NULL,
    names                  TEXT         NOT NULL,
    last_names             TEXT         NOT NULL,
    email                  TEXT         NULL,
    phone                  TEXT         NULL,
    password_hash          TEXT         NOT NULL,
    mfa_secret             TEXT         NULL,
    mfa_enrolled_at        TIMESTAMPTZ  NULL,
    failed_login_attempts  INTEGER      NOT NULL DEFAULT 0,
    locked_until           TIMESTAMPTZ  NULL,
    last_login_at          TIMESTAMPTZ  NULL,
    status                 TEXT         NOT NULL DEFAULT 'active',
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ  NULL,
    created_by             UUID         NULL REFERENCES users(id),
    updated_by             UUID         NULL REFERENCES users(id),
    deleted_by             UUID         NULL REFERENCES users(id),
    version                INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT users_status_chk_down       CHECK (status IN ('active', 'inactive', 'suspended')),
    CONSTRAINT users_doctype_chk_down      CHECK (document_type IN ('CC','CE','PA','TI','RC','NIT')),
    CONSTRAINT users_document_unique_down  UNIQUE (document_type, document_number)
);
