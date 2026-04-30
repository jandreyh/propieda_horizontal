-- Fase 16 (ADR 0007): la identidad pasa a vivir en la DB central
-- (`platform_users`). Cada tenant DB conserva solo un vinculo
-- `tenant_user_links` que ata la identidad global con su rol y unidad
-- en este conjunto.
--
-- ESTA MIGRACION ES DESTRUCTIVA. Asume que el tenant esta fresh o que
-- los datos van a re-sembrarse desde central.

-- ---------------------------------------------------------------------------
-- 1) Eliminar tablas operativas que no aplican post-Fase 16.
--    Las sesiones y los codigos MFA viven en central, no en cada tenant.
-- ---------------------------------------------------------------------------
DROP TABLE IF EXISTS user_mfa_recovery_codes CASCADE;
DROP TABLE IF EXISTS user_sessions CASCADE;

-- ---------------------------------------------------------------------------
-- 2) Capturar todas las FK que apuntan a users(id) ANTES de droppearlas,
--    para poder recrearlas apuntando a tenant_user_links(id) preservando
--    la regla ON DELETE original.
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE _users_fk_backup ON COMMIT PRESERVE ROWS AS
SELECT
    tc.table_name        AS tbl,
    kcu.column_name      AS col,
    tc.constraint_name   AS cons_name,
    rc.delete_rule       AS del_rule
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
       ON tc.constraint_name = kcu.constraint_name
      AND tc.table_schema    = kcu.table_schema
JOIN information_schema.constraint_column_usage ccu
       ON tc.constraint_name = ccu.constraint_name
      AND tc.table_schema    = ccu.table_schema
JOIN information_schema.referential_constraints rc
       ON tc.constraint_name = rc.constraint_name
      AND tc.table_schema    = rc.constraint_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND ccu.table_schema   = 'public'
  AND ccu.table_name     = 'users'
  AND ccu.column_name    = 'id'
  -- Excluir self-references: la tabla users tiene FKs a si misma
  -- (created_by, updated_by, deleted_by) que despues del DROP TABLE users
  -- ya no existen y no podemos recrearlas.
  AND tc.table_name      <> 'users';

-- ---------------------------------------------------------------------------
-- 3) Drop FKs.
-- ---------------------------------------------------------------------------
DO $do$
DECLARE r record;
BEGIN
    FOR r IN SELECT tbl, cons_name FROM _users_fk_backup LOOP
        EXECUTE format('ALTER TABLE public.%I DROP CONSTRAINT %I',
                       r.tbl, r.cons_name);
    END LOOP;
END
$do$;

-- ---------------------------------------------------------------------------
-- 4) Drop la tabla users (sin FKs apuntando, ya es seguro).
-- ---------------------------------------------------------------------------
DROP TABLE IF EXISTS users CASCADE;

-- ---------------------------------------------------------------------------
-- 5) Crear tenant_user_links.
-- ---------------------------------------------------------------------------
CREATE TABLE tenant_user_links (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    -- platform_user_id es FK logica a platform_users(id) en la DB central.
    -- NO se declara FK fisica porque la DB central esta en otra logical instance.
    -- La consistencia se valida en application layer.
    platform_user_id    UUID         NOT NULL,
    role                TEXT         NOT NULL,
    primary_unit_id     UUID         NULL REFERENCES units(id),
    cartera_status      TEXT         NULL,
    fecha_ingreso       DATE         NULL,
    -- status local del vinculo en este tenant: 'active' o 'blocked'.
    -- Bloqueo aqui NO suspende al usuario en otros conjuntos.
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ  NULL,
    version             INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT tenant_user_links_status_chk
        CHECK (status IN ('active','blocked')),
    CONSTRAINT tenant_user_links_platform_user_unique
        UNIQUE (platform_user_id)
);

CREATE INDEX tenant_user_links_role_idx
    ON tenant_user_links (role)
    WHERE deleted_at IS NULL;

CREATE INDEX tenant_user_links_unit_idx
    ON tenant_user_links (primary_unit_id)
    WHERE deleted_at IS NULL AND primary_unit_id IS NOT NULL;

CREATE INDEX tenant_user_links_status_idx
    ON tenant_user_links (status)
    WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- 6) Recrear las FKs capturadas en (2), apuntando a tenant_user_links(id).
--    Mantiene mismo nombre de constraint, misma regla ON DELETE.
-- ---------------------------------------------------------------------------
DO $do$
DECLARE r record;
        del_clause text;
BEGIN
    FOR r IN SELECT tbl, col, cons_name, del_rule FROM _users_fk_backup LOOP
        del_clause := CASE r.del_rule
            WHEN 'CASCADE'   THEN 'CASCADE'
            WHEN 'SET NULL'  THEN 'SET NULL'
            WHEN 'RESTRICT'  THEN 'RESTRICT'
            ELSE 'NO ACTION'
        END;
        -- NOT VALID: permite agregar el FK sin validar filas existentes
        -- (que podrian apuntar a UUIDs del usuario antiguo). Los INSERT
        -- futuros se validan normalmente. El seeder se encarga de
        -- limpiar / re-sembrar datos despues de aplicar la migracion.
        EXECUTE format(
            'ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (%I) REFERENCES tenant_user_links(id) ON DELETE %s NOT VALID',
            r.tbl, r.cons_name, r.col, del_clause
        );
    END LOOP;
END
$do$;

-- 7) Cleanup temp table (no usamos ON COMMIT DROP porque migrate lo aplica
--    fuera de una transaccion por sentencia y el temp moriria a mitad).
DROP TABLE IF EXISTS _users_fk_backup;
