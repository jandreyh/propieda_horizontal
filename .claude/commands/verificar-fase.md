---
description: Verifica el DoD de una fase sin ejecutarla
argument-hint: <numero-de-fase>
---

Verifica si la **Fase $ARGUMENTS** del PLAN_MAESTRO.md cumple su Definition
of Done.

## Pasos
1. Lee la seccion `## FASE $ARGUMENTS` del PLAN_MAESTRO.md.
2. Para cada item del checklist DoD:
   - Si es un test/comando: ejecutalo y reporta resultado.
   - Si es un archivo: verifica que existe y tiene contenido sustantivo.
   - Si es una propiedad de runtime: ejecuta el escenario.
3. Reporta tabla:

| DoD item | Estado | Evidencia |
|----------|--------|-----------|
| ... | OK / FAIL | comando o archivo |

4. Si todo OK: confirma fase lista para avanzar.
5. Si algo FAIL: lista pendientes y sugiere proximos pasos.

NO modifiques codigo. Solo verificas.
