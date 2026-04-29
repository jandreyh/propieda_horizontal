---
description: Identifica el siguiente trabajo pendiente del proyecto y lo ejecuta autonomamente
---

Tu tarea: avanzar el SaaS Propiedad Horizontal una unidad de trabajo a la vez,
autonomamente, hasta que llegues a un punto que SI requiera intervencion humana
(por ejemplo, una entrevista de Discovery POST-MVP).

## Paso 1 — Cargar contexto

1. Lee `CLAUDE.md` (invariantes, prohibiciones, stack obligatorio).
2. Lee el indice de `PLAN_MAESTRO.md`.
3. Lee `README.md` raiz para conocer estado actual de modulos entregados.
4. Verifica que estas dentro del devcontainer: `hostname` debe contener `ph-dev`.

## Paso 2 — Determinar la siguiente unidad de trabajo

En este orden, escoge la primera condicion que aplique:

**A. MVP runtime no verificado**
Si alguno de los siguientes es cierto, ejecuta `/completar-mvp` y termina:
- `apps/web` no existe.
- `apps/mobile` no existe.
- `docker compose -f deployments/docker-compose.dev.yml ps` no muestra
  `pg-central` y `pg-tenant-template` healthy.
- Las migraciones `migrations/central` o `migrations/tenant` no se han
  aplicado contra esas DBs (ver tabla `schema_migrations`).
- `go test ./apps/api/...` no pasa con DB real.

**B. POST-MVP con spec frozen pendiente de codigo**
Si existe `/docs/specs/fase-<N>-spec.md` para alguna N en {8..15} pero el
codigo de esa fase no esta entregado: ejecuta `/fase <N>`.

**C. POST-MVP sin spec**
Si todas las anteriores estan listas, identifica la siguiente fase
POST-MVP sin spec. **PARA y reporta al usuario** que hay que ejecutar
`/descubrir <N>` (porque Discovery requiere entrevista — no se puede
automatizar).

**D. Todo completo**
Reportar al usuario que el plan esta cerrado.

## Paso 3 — Ejecutar la unidad escogida

- Antes de empezar: crea (o cambia a) una rama `feat/auto-<descripcion-corta>`
  desde main. Nunca trabajes directamente en main.
- Usa `TodoWrite` para listar los pasos del trabajo escogido.
- Ejecuta el comando correspondiente (`/completar-mvp` o `/fase <N>`).

## Paso 4 — Verificar

Despues de la ejecucion:
- Si era una fase numerada: ejecuta `/verificar-fase <N>`.
- Si era `/completar-mvp`: ejecuta los smokes definidos al final de ese comando.

Si la verificacion falla: **NO commitees**. Reporta al usuario el fallo
concreto y los pasos para reproducirlo.

## Paso 5 — Cierre

Si verificacion paso:
1. Asegura que `gofmt`, `goimports`, `golangci-lint run ./apps/api/...`
   estan limpios.
2. Asegura que tests pasan: `go test ./apps/api/... -count=1`.
3. Si hay cambios en TS/JS: `pnpm -r lint` y `pnpm -r build` limpios.
4. `git add -A && git commit -m "<prefijo>: <resumen>"`. Lefthook valida.
5. `git push -u origin <rama>` (la deny list bloquea push a main).
6. `gh pr create` con titulo y body resumiendo cambios.
7. Reporta al usuario: rama, PR URL, que avanzo, que sigue.

## Reglas duras (NO violar)

- NO saltar fases.
- NO trabajar en main.
- NO violar `CLAUDE.md` (stack, prohibiciones multi-tenant, soft-delete, etc.).
- NO usar `git push --force` ni `--force-with-lease`.
- NO hacer Discovery autonomo (requiere usuario).
- Si un paso falla, PARAR. No enmascarar el fallo.
- Reportar siempre en castellano, conciso.
