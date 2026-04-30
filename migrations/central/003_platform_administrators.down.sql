ALTER TABLE tenants
    DROP COLUMN IF EXISTS expected_units,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS country,
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS primary_color,
    DROP COLUMN IF EXISTS logo_url,
    DROP COLUMN IF EXISTS administrator_id;

DROP TABLE IF EXISTS platform_administrators;
