---
description: Ejecuta tests E2E con Playwright contra el stack local
argument-hint: [escenario|smoke]
---

Tu tarea: correr tests E2E del SaaS sobre el stack levantado por
`docker-compose.dev.yml`.

## Paso 1 — Pre-flight

1. Verificar stack arriba:
   ```bash
   docker compose -f deployments/docker-compose.dev.yml ps
   ```
   Si algun servicio no esta healthy: PARAR y reportar.

2. Verificar que `apps/web` y la API responden:
   ```bash
   curl -fsS http://localhost:3000
   curl -fsS http://localhost:8080/healthz
   ```

3. Verificar Playwright instalado:
   ```bash
   ls apps/web/node_modules/@playwright/test || \
     (cd apps/web && pnpm add -D @playwright/test && pnpm exec playwright install --with-deps)
   ```

## Paso 2 — Determinar scope

- Si `$ARGUMENTS` esta vacio o es `smoke`: corre el smoke E2E del subagent.
- Si es un nombre: filtrar `--grep "<nombre>"`.
- Si es un archivo: `playwright test <archivo>`.

## Paso 3 — Invocar e2e-runner

Delegar al subagent `e2e-runner`:

> Ejecuta los tests E2E `<scope>` con Playwright. Genera reporte HTML.
> Reporta resumen consolidado.

## Paso 4 — Reportar

Si TODO verde:
- Reportar al usuario, sugerir merge.

Si ROJO:
- Adjuntar screenshots de fallos (paths absolutos).
- Para cada test fallido: probable causa + sugerencia de fix.
- NO arregles tu mismo. Reporta y delega a `/fase` o issue.

## Reglas duras

- NO uses `--update-snapshots` sin pedir aprobacion.
- NO modifiques tests para que pasen.
- NO ejecutes contra DBs no-efimeras o entornos compartidos.
- Reporte en castellano.
