---
description: Loop interactivo de diseno UI sobre apps/web — el agente design-auditor itera contigo hasta aprobacion
argument-hint: <ruta-de-pagina-o-componente>
---

Tu tarea: orquestar un ciclo iterativo de diseno sobre `$ARGUMENTS` (puede ser
una ruta `/dashboard`, un componente `apps/web/components/Header.tsx`, o
"todo" para el tour completo).

## Paso 1 — Pre-flight

1. Verifica que `apps/web` esta corriendo en `:3000`. Si no, lanzalo en
   background: `pnpm --filter web dev`.
2. Verifica que `docs/DESIGN_SYSTEM.md` y `docs/UX_PRINCIPIOS.md` existen.
3. Crea (o cambia a) la rama `feat/design-<slug-de-pagina>` desde main.

## Paso 2 — Audit base

Invoca el subagent `design-auditor` con prompt:

> Audita la pagina/componente `$ARGUMENTS` de `apps/web`. Reporta scoring,
> capturas en 360/768/1280 y violaciones a11y. NO modifiques codigo aun.

Recibe el reporte y muestralo al usuario.

## Paso 3 — Loop iterativo

Mientras el usuario no diga "aprobado" o equivalente:

1. Pregunta al usuario: "Que cambios quieres? (o 'aplicar sugerencias del auditor', o 'aprobado')".
2. Si el usuario dice "aprobado": salta al Paso 4.
3. Si pide cambios o "aplicar sugerencias":
   - Aplica los cambios (`Edit` sobre los archivos `apps/web/**/*.{tsx,css}`).
   - Hooks `PostToolUse` formateara y lintera automaticamente.
   - Re-invoca `design-auditor` con el mismo argumento.
   - Muestra capturas nuevas y diff de scoring.
4. Repite.

## Paso 4 — Cierre

1. `pnpm --filter web build` (debe pasar).
2. `pnpm --filter web lint` (debe pasar).
3. Si hay tests E2E afectados: `pnpm --filter web exec playwright test`.
4. `git add` solo los archivos modificados de `apps/web/`.
5. `git commit -m "design: <pagina> — <resumen-de-cambios>"`.
6. `git push -u origin feat/design-<slug>`.
7. `gh pr create` con titulo y body resumiendo cambios + screenshots
   embebidos del antes/despues.
8. Reportar URL del PR.

## Reglas duras

- NO toques `apps/api/`, migraciones, ni docs/specs/.
- NO modifiques tokens del design system sin aprobacion explicita del usuario.
- NO uses CSS arbitrario; siempre Tailwind + shadcn.
- NO hagas commits sin que el usuario diga "aprobado".
- Reporta en castellano. Capturas con paths absolutos.
- Si un cambio rompe el build: REVERT inmediato y reportar.
