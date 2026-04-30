-- Fase 16 (ADR 0007) — Paso 16.4b.
--
-- La migracion 019 ya droppea la tabla users, crea tenant_user_links y
-- recrea las FK que apuntaban a users(id) hacia tenant_user_links(id),
-- pero las agrega como NOT VALID para no fallar contra filas operativas
-- existentes (cuyos created_by/updated_by/etc. apuntan a UUIDs del
-- usuario antiguo). Esta migracion valida esas constraints despues del
-- re-seed (que ya elimino o reemplazo las filas huerfanas).
--
-- Ejecutarla DESPUES de re-sembrar el tenant (cmd/seed-demo). Si quedan
-- filas huerfanas, la migracion fallara con "violates foreign key
-- constraint"; eso es la senal correcta — el seed olvido limpiar algo.

DO $do$
DECLARE r record;
BEGIN
    FOR r IN
        SELECT tc.table_name AS tbl,
               tc.constraint_name AS cons_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.constraint_column_usage ccu
               ON tc.constraint_name = ccu.constraint_name
              AND tc.table_schema    = ccu.table_schema
        WHERE tc.constraint_type = 'FOREIGN KEY'
          AND ccu.table_schema   = 'public'
          AND ccu.table_name     = 'tenant_user_links'
          AND ccu.column_name    = 'id'
    LOOP
        EXECUTE format(
            'ALTER TABLE public.%I VALIDATE CONSTRAINT %I',
            r.tbl, r.cons_name
        );
    END LOOP;
END
$do$;
