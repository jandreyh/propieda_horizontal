---
description: Ejecuta una fase del PLAN_MAESTRO.md con multi-agentes
argument-hint: <numero-de-fase>
---

Ejecuta la **Fase $ARGUMENTS** del PLAN_MAESTRO.md siguiendo este protocolo
estricto:

## Paso 1 — Cargar contexto
1. Lee `CLAUDE.md` completo (invariantes y prohibiciones).
2. Lee la seccion `## FASE $ARGUMENTS` del `PLAN_MAESTRO.md`.
3. Verifica las **Pre-condiciones** de la fase. Si la fase anterior no
   tiene su DoD cumplido, REPORTA al usuario y DETENTE.

## Paso 2 — Plan de ejecucion
1. Crea un `TodoWrite` con TODAS las tareas de la seccion "Tareas" de la fase.
2. Si la fase tiene seccion "Estrategia multi-agente", identifica que agentes
   se lanzan en paralelo y con que briefs.
3. Si la fase requiere decisiones arquitectonicas no triviales, lanza
   primero un agente `Plan` para validar el enfoque.

## Paso 3 — Ejecucion
1. Marca cada todo como `in_progress` cuando empieces.
2. Si hay agentes paralelos: lanzalos en UNA sola tool call con multiples
   invocaciones a `Agent`. NO los lances secuencialmente.
3. Cada agente recibe un brief autocontenido (ver "Brief base por agente"
   en la seccion de la fase).
4. Espera todos los resultados, sintetiza, integra.
5. Marca cada todo como `completed` apenas termine — no batch.

## Paso 4 — Verificacion
Ejecuta TODOS los comandos del bloque "Verificacion" de la fase. Si alguno
falla:
1. Reporta el fallo concreto al usuario.
2. NO marques DoD como completo.
3. Sugiere arreglo o pide instrucciones.

## Paso 5 — Cierre
1. Recorre el checklist DoD de la fase. Marca cada item con [x] o [ ].
2. Si todos los DoD pasan: actualiza el `README.md` raiz marcando la fase
   como completada.
3. Reporta al usuario: que se hizo, que tests pasaron, que falta (si algo).

## Reglas
- NO hagas trabajo fuera del scope de la fase $ARGUMENTS.
- NO modifiques `CLAUDE.md` ni `PLAN_MAESTRO.md` salvo que el usuario lo pida.
- NO commitees (a menos que el usuario lo pida explicitamente).
- Si un agente paralelo escribe en un archivo que otro tambien edita, eso
  es un bug en la planificacion — REPORTA y replanifica.
- Reportes finales en castellano, concisos. Lo que cambio, que falta, blockers.
