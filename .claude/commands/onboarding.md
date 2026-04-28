---
description: Carga el contexto del proyecto en una sesion fresca
---

Estamos trabajando en un SaaS multi-tenant para Propiedad Horizontal en Go.
Antes de hacer nada:

1. Lee completo `CLAUDE.md` — son los invariantes (stack, prohibiciones,
   estructura de modulos, reglas multi-tenant).
2. Lee el indice y el glosario de `PLAN_MAESTRO.md`.
3. Lista los slash commands disponibles en `.claude/commands/`.
4. Lista las specs frozen ya consolidadas en `/docs/specs/` (si existe).
5. Reporta al usuario:
   - Stack confirmado.
   - Cuantas fases hay (8 MVP + 8 POST-MVP = 16 totales).
   - En cual estamos (verificar via README, git log, o specs/ existentes).
   - Comandos disponibles:
     - `/fase <N>` — ejecuta una fase
     - `/descubrir <N>` — Discovery con entrevista (solo POST-MVP, fases 8-15)
     - `/verificar-fase <N>` — audita el DoD de una fase
     - `/onboarding` — este comando

NO escribas codigo en este momento. Es solo onboarding.
