-- Reversa de 003_tenant_config.up.sql.
-- Drop en orden inverso (no hay FK entre las dos, pero por consistencia).

DROP TABLE IF EXISTS tenant_branding;
DROP TABLE IF EXISTS tenant_settings;
