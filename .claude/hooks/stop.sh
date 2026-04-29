#!/usr/bin/env bash
# Hook Stop: se ejecuta cuando Claude termina su respuesta.
# Imprime un resumen de git para que el usuario sepa que cambio.

set +e
cd /workspace 2>/dev/null || exit 0

BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
if [ -z "$BRANCH" ]; then exit 0; fi

# Resumen breve.
CHANGED="$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')"
AHEAD="$(git rev-list --count HEAD ^"origin/$BRANCH" 2>/dev/null || echo 0)"

if [ "$CHANGED" = "0" ] && [ "$AHEAD" = "0" ]; then
  exit 0  # nada que reportar, no contaminar contexto
fi

echo ""
echo "================================================="
echo "[stop hook] Resumen de git"
echo "================================================="
echo "Rama:                $BRANCH"
echo "Archivos modificados: $CHANGED"
echo "Commits ahead:        $AHEAD"
echo ""
git status --short 2>/dev/null | head -20
echo "================================================="
exit 0
