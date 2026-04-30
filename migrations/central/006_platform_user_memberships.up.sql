-- Indice central de membresias usuario-tenant.
--
-- En cada tenant DB existe `tenant_user_links` que es la fuente de verdad
-- del rol y la unidad del usuario. Pero consultar N tenants en cada login
-- no escala. Esta tabla mantiene una proyeccion central de la membresia
-- (sin role/unit, solo presencia y status) para responder rapido a
-- /me/memberships y poblar el JWT.
--
-- Se actualiza desde el provisioning (POST /superadmin/tenants crea el
-- vinculo del admin inicial) y desde el endpoint del tenant
-- POST /tenant-members (que crea el link en tenant DB y, en la misma
-- transaccion logica, llama a la API central para indexar aqui).

CREATE TABLE IF NOT EXISTS platform_user_memberships (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_user_id    UUID         NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    tenant_id           UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- role no es la fuente de verdad — solo se duplica para mostrar en el
    -- selector ("entrar como admin / como guarda"). El rol real esta en
    -- tenant_user_links de cada tenant DB.
    role                TEXT         NOT NULL,
    status              TEXT         NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT platform_user_memberships_unique UNIQUE (platform_user_id, tenant_id),
    CONSTRAINT platform_user_memberships_status_chk
        CHECK (status IN ('active','blocked'))
);

CREATE INDEX IF NOT EXISTS platform_user_memberships_user_idx
    ON platform_user_memberships (platform_user_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS platform_user_memberships_tenant_idx
    ON platform_user_memberships (tenant_id)
    WHERE status = 'active';
