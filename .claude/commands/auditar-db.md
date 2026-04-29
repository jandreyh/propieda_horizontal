---
description: DBA review de migraciones, modelo de datos, indices y compliance multi-tenant
argument-hint: [migracion-glob|all]
---

Tu tarea: auditar el modelo de datos y migraciones en `$ARGUMENTS` (default: las
nuevas en HEAD vs main; `all` para auditoria global).

## Paso 1 — Determinar scope

Si `$ARGUMENTS` esta vacio:
- `git diff main...HEAD --name-only -- migrations/` → lista de migraciones nuevas/modificadas.

Si `$ARGUMENTS` es `all`:
- Todas las migraciones en `migrations/central/` y `migrations/tenant/`.

Si es un glob (ej: `migrations/tenant/01*`):
- Resolverlo.

## Paso 2 — Invocar db-architect

Invocar el subagent `db-architect`:

> Audita las migraciones: <lista>. Verifica compliance con CLAUDE.md secciones
> 2 y 4, ADR 0001/0004/0005. Reporta findings con severity BLOCKER/WARN/INFO.

## Paso 3 — Validar runtime (opcional)

Si el stack esta arriba:
1. Smoke aplicar las migraciones sobre una DB efimera:
   ```bash
   docker exec ph-pg-tenant-template psql -U ph -d ph_tenant_template \
     -c "DROP DATABASE IF EXISTS ph_smoke; CREATE DATABASE ph_smoke;"
   migrate -path migrations/tenant \
     -database "postgres://ph:ph@localhost:5433/ph_smoke?sslmode=disable" up
   ```
2. Reversibilidad: `migrate ... down 1` y `migrate ... up` debe pasar.
3. `pg_dump` de schema y revisar `\d+` de tablas nuevas.

## Paso 4 — Reportar

Generar `docs/audits/db-<fecha>.md` con findings consolidados.

Si BLOCKERS:
- Detener el merge / fase. Reportar.
Si solo WARN/INFO:
- Permitir avanzar pero sugerir fixes en proximo PR.

## Reglas duras

- NO ejecutes migraciones sobre `pg_central` o `pg_tenant_template` reales
  para tests; usa siempre DB efimera (ej: `ph_smoke`).
- NO sugieras `tenant_id` en Tenant DB (PROHIBIDO).
- NO uses `DROP COLUMN` sin DOWN reversible.
- Reporte en castellano. Severity en ingles.
