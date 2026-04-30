-- Down de 020: VALIDATE CONSTRAINT no se puede revertir directamente
-- (no existe INVALIDATE CONSTRAINT en Postgres). La unica forma de
-- "deshacer" la validacion es droppear y recrear la FK como NOT VALID.
-- Como las FKs fueron creadas en 019 como NOT VALID, este down las
-- vuelve a marcar NOT VALID iterando el mismo set de constraints.

DO $do$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT tc.table_name        AS tbl,
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
          AND ccu.table_name     = 'tenant_user_links'
          AND ccu.column_name    = 'id'
    LOOP
        EXECUTE format('ALTER TABLE public.%I DROP CONSTRAINT %I',
                       r.tbl, r.cons_name);
        EXECUTE format(
            'ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (%I) REFERENCES tenant_user_links(id) ON DELETE %s NOT VALID',
            r.tbl, r.cons_name, r.col,
            CASE r.del_rule
                WHEN 'CASCADE'  THEN 'CASCADE'
                WHEN 'SET NULL' THEN 'SET NULL'
                WHEN 'RESTRICT' THEN 'RESTRICT'
                ELSE 'NO ACTION'
            END
        );
    END LOOP;
END
$do$;
