#!/usr/bin/env bash
# Hook PostToolUse: se ejecuta despues de Edit/Write/MultiEdit.
# Lee del stdin un JSON con la entrada del tool y aplica formateo o lint
# segun el tipo de archivo.
#
# Mantener IDEMPOTENTE y RAPIDO. Si falla, NO bloquear (exit 0 siempre).

set +e  # nunca bloquear el flujo

# Solo correr dentro del workspace del proyecto.
cd /workspace 2>/dev/null || { exit 0; }

# Capturar stdin JSON.
INPUT="$(cat 2>/dev/null)"
if [ -z "$INPUT" ]; then exit 0; fi

# Extraer file_path con jq (instalado en el devcontainer).
FILE_PATH="$(echo "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null)"
if [ -z "$FILE_PATH" ]; then exit 0; fi

# Si la ruta no es absoluta, hacerla relativa al workspace.
case "$FILE_PATH" in
  /*) ;;
  *) FILE_PATH="/workspace/$FILE_PATH" ;;
esac

# Si el archivo no existe (ej: Write fallido), salir.
[ -f "$FILE_PATH" ] || exit 0

# Decidir accion por extension / ruta.
case "$FILE_PATH" in
  *.go)
    # Format Go con gofmt + goimports
    gofmt -w "$FILE_PATH" 2>/dev/null
    goimports -w -local github.com/saas-ph/api "$FILE_PATH" 2>/dev/null
    ;;

  /workspace/apps/web/*.ts|/workspace/apps/web/*.tsx|/workspace/apps/web/*.js|/workspace/apps/web/*.jsx)
    # Solo formatear con prettier si esta instalado y el archivo no esta en node_modules.
    case "$FILE_PATH" in
      */node_modules/*) exit 0 ;;
    esac
    if [ -x /workspace/apps/web/node_modules/.bin/prettier ]; then
      /workspace/apps/web/node_modules/.bin/prettier --write --log-level=warn "$FILE_PATH" 2>/dev/null
    fi
    ;;

  /workspace/apps/mobile/*.ts|/workspace/apps/mobile/*.tsx)
    case "$FILE_PATH" in
      */node_modules/*) exit 0 ;;
    esac
    if [ -x /workspace/apps/mobile/node_modules/.bin/prettier ]; then
      /workspace/apps/mobile/node_modules/.bin/prettier --write --log-level=warn "$FILE_PATH" 2>/dev/null
    fi
    ;;

  /workspace/migrations/tenant/*.sql)
    # CRITICO: prohibido usar tenant_id en Tenant DB (CLAUDE.md).
    if grep -nE '\btenant_id\b' "$FILE_PATH" >/dev/null 2>&1; then
      echo "ERROR (post-edit hook): el archivo $FILE_PATH contiene 'tenant_id'." >&2
      echo "PROHIBIDO en migrations/tenant/ — ver CLAUDE.md seccion 2." >&2
      grep -nE '\btenant_id\b' "$FILE_PATH" >&2
      exit 2  # exit 2 envia mensaje de stderr de vuelta al modelo
    fi
    ;;

  *.sql)
    # SQL en general: no intentar formatear (sqlc maneja su parte).
    :
    ;;

  *.md)
    # Markdown: no formatear (preservar estilo del autor).
    :
    ;;
esac

exit 0
