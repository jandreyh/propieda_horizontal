---
description: Security review completo del backend + multi-tenant compliance
---

Tu tarea: auditar seguridad del codigo en HEAD y reportar findings priorizados.

## Paso 1 — Invocar security-auditor

Invocar el subagent `security-auditor` con prompt:

> Audita seguridad del backend Go en HEAD. Combina la skill `security-review`
> de Anthropic (si esta disponible) con los checks especificos del proyecto
> (CLAUDE.md, ADRs 0002/0003/0005). Reporta findings en castellano con
> severity HIGH/MEDIUM/LOW.

## Paso 2 — Consolidar reporte

Cuando termine, generar `docs/audits/security-<fecha>.md` con el reporte
completo.

## Paso 3 — Decisiones

Si hay findings HIGH:
- Reportar al usuario y pedir prioridad.
- Sugerir crear issues en GitHub: `gh issue create --label security`.

Si hay findings MEDIUM:
- Listar y proponer agruparlos en un PR de hardening.

Si hay solo LOW/INFO:
- Reportar y commitear el audit doc.

## Reglas duras

- NO arregles findings tu mismo. Solo reportar y delegar a un futuro `/fase`
  o issue de GitHub.
- NO publiques credenciales reales encontradas (referenciar archivo:linea solamente).
- Si encuentras un secret REAL en historia de git, marcar CRITICAL y
  detener (NO ejecutar `git filter-repo` automaticamente).
- Reporte en castellano. Severity en ingles.
