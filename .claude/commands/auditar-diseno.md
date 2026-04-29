---
description: Audit de diseno completo (visual + a11y) sobre apps/web — invoca design-auditor + accessibility-auditor en paralelo
argument-hint: [pagina|all]
---

Tu tarea: ejecutar un audit completo de diseno y accesibilidad sobre
`$ARGUMENTS` (default: `all`).

## Paso 1 — Pre-flight

1. Verifica que `apps/web` corre. Si no, lanzalo: `pnpm --filter web dev`.
2. Verifica que `@axe-core/react` esta instalado en `apps/web/package.json`.
   - Si no: reportar y proponer instalacion (no instalar sin aprobacion).
3. Determinar set de paginas:
   - Si `$ARGUMENTS` es una ruta especifica: solo esa.
   - Si es `all` o vacio: barrer `apps/web/app/**/page.tsx` y armar lista.

## Paso 2 — Lanzar agentes en paralelo

Invocar UNA sola tool call con dos `Agent`:

```
[
  Agent(design-auditor, "Audita las paginas: <lista>. Reporta scoring + capturas + violaciones."),
  Agent(accessibility-auditor, "Audita las paginas: <lista>. Reporta WCAG 2.2 AA con axe + manual.")
]
```

## Paso 3 — Sintesis

Cuando ambos terminen:

1. Consolidar findings por pagina en un solo reporte.
2. Priorizar issues:
   - **P0 BLOCKER**: a11y critical, build roto, ruta inaccesible.
   - **P1 HIGH**: a11y serious, design system inconsistencia mayor, score < 6.
   - **P2 MEDIUM**: a11y moderate, microinteraccion ausente, score 6-7.
   - **P3 LOW**: polish, score 7-8.
3. Generar `docs/audits/design-<fecha>.md` con el reporte.

## Paso 4 — Decision

Pregunta al usuario:
- "Hay X issues P0 y Y issues P1. Aplico fixes via `/disenar` por pagina?
  o solo dejo el reporte?"

Si el usuario dice "aplica":
- Para cada pagina con score < 7 o issues P0/P1: invocar `/disenar <pagina>`
  secuencialmente.
- Crear PR consolidado al final.

Si dice "solo reporte":
- Hacer commit del reporte: `git add docs/audits/ && git commit -m "audit: design <fecha>"`.
- Reportar al usuario el path del reporte.

## Reglas duras

- NO ejecutes paralelo dos agents que escriban al mismo archivo.
- NO modifiques codigo en este comando (solo audit + delegate). Los fixes
  los hace `/disenar`.
- Reporte en castellano.
