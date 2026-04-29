---
description: Completa runtime end-to-end del MVP (DB up + migraciones + API + scaffolding web/mobile)
---

Tu tarea: dejar el MVP del SaaS Propiedad Horizontal corriendo end-to-end y
con frontends scaffolded.

El codigo del backend Go ya existe (fases 0-7 entregadas). Faltan:

1. Verificacion runtime contra Postgres 18 real.
2. Migraciones aplicadas (central + tenant template).
3. Smoke-test endpoints clave del API.
4. `apps/web` (Next.js 16.2.3) scaffolded.
5. `apps/mobile` (Expo SDK 55) scaffolded.
6. Documentacion del estado actualizada.

## Paso 0 — Pre-condiciones

1. Lee `CLAUDE.md`.
2. `hostname` debe contener `ph-dev` (estas en el devcontainer).
3. Crea (o cambia a) la rama `feat/mvp-runtime-verification` desde main:
   ```bash
   git fetch origin
   git checkout -B feat/mvp-runtime-verification origin/main
   ```
4. Crea `TodoWrite` con los pasos 1-8 de abajo.

## Paso 1 — Compilacion + tests unitarios

```bash
cd /workspace/apps/api
go build ./...
go test ./... -count=1 -short
```

Si falla: parar, reportar el error concreto. NO continuar.

## Paso 2 — Postgres up

Verifica que los servicios postgres estan accesibles desde dentro del
devcontainer (resueltos por nombre de servicio en la red de compose):

```bash
pg_isready -h pg-central -p 5432 -U ph -d ph_central
pg_isready -h pg-tenant-template -p 5432 -U ph -d ph_tenant_template
```

Si alguno falla: reportar al usuario que `docker compose up -d pg-central pg-tenant-template`
no esta listo. Parar.

## Paso 3 — Migraciones central + tenant

```bash
cd /workspace
migrate -path migrations/central \
        -database "$DATABASE_URL_CENTRAL" up

migrate -path migrations/tenant \
        -database "$DATABASE_URL_TENANT_TEMPLATE" up
```

Verificar que cada `up` retorno exit 0. Probar reversibilidad de la ultima
migracion de tenant (down + up) como smoke. Volver a estado UP.

## Paso 4 — API runtime + smoke

Arrancar API en background:

```bash
cd /workspace/apps/api
PORT=8080 \
DATABASE_URL_CENTRAL="$DATABASE_URL_CENTRAL" \
DATABASE_URL_TENANT_TEMPLATE="$DATABASE_URL_TENANT_TEMPLATE" \
go run ./cmd/api &
API_PID=$!
sleep 3
```

Smoke:

```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/readyz
```

Esperado: `200 OK`. Matar el proceso al terminar:

```bash
kill $API_PID 2>/dev/null || true
```

Si falla: reportar logs y parar.

## Paso 5 — Tests de integracion

```bash
cd /workspace/apps/api
go test ./... -count=1 -tags=integration
```

(Estos tests usan los Postgres reales via env vars.) Si fallan: parar y
reportar.

## Paso 6 — Frontend Web

Si `apps/web` no existe:

```bash
cd /workspace/apps
pnpm create next-app@16.2.3 web \
  --ts --app --tailwind --eslint \
  --src-dir --import-alias "@/*" \
  --use-pnpm --no-turbopack
cd web
pnpm install
pnpm build
pnpm lint
```

Crear placeholder de pagina `/login` y `/dashboard` que consuman
`http://localhost:8080/v1/auth/login` y `/v1/me` respectivamente. Tipos
TS minimos.

## Paso 7 — Mobile

Si `apps/mobile` no existe:

```bash
cd /workspace/apps
npx --yes create-expo-app@latest mobile -t blank-typescript --no-install
cd mobile
pnpm install
```

Pin Expo SDK a 55 si el template usa otro. Verificar que `npx expo
prebuild --no-install` no falla en seco.

## Paso 8 — Documentar y commit

Actualizar `README.md` raiz:
- Sustituir el bloque "Runtime contra Postgres 18 esta en construccion" por
  el estado real (verificado, fecha, comandos usados).
- Listar `apps/web` y `apps/mobile` con su estado scaffold.

Commits incrementales:

```bash
git add -A
git commit -m "fase-mvp: verificacion runtime end-to-end + scaffolding web/mobile"
git push -u origin feat/mvp-runtime-verification
```

PR:

```bash
gh pr create \
  --title "MVP runtime verificado + frontends scaffold" \
  --body "$(cat <<EOF
## Resumen
- Postgres 18 (central + tenant template) up via docker-compose.dev.yml
- Migraciones central + tenant aplicadas (UP/DOWN reversibles)
- API Go corriendo, /healthz y /readyz responden 200
- Tests unitarios + integracion en verde
- apps/web (Next.js 16.2.3, App Router, TS, Tailwind) scaffolded
- apps/mobile (Expo SDK 55, TS) scaffolded

## Test plan
- [ ] docker compose -f deployments/docker-compose.dev.yml up -d
- [ ] go test ./apps/api/... -count=1
- [ ] curl http://localhost:8080/healthz devuelve 200
- [ ] pnpm --filter web build sin warnings
EOF
)"
```

## Paso 9 — Reporte final

En castellano, conciso:
- Que se verifico runtime.
- Que tests pasaron y cuantos.
- URLs de la rama y el PR.
- Bloqueadores (si los hay) o "todo verde".

## Reglas duras (NO violar)

- Trabajar SIEMPRE en `feat/mvp-runtime-verification`. Nunca en main.
- Si un paso falla: parar, reportar, NO continuar al siguiente.
- NO violar `CLAUDE.md` (sin ORM pesado, sin `database/sql` directo, sin
  SQL inline en Go, sin `tenant_id` en Tenant DB, etc.).
- NO usar `git push --force` ni `--force-with-lease`.
- Codigo Go: `gofmt -w`, `goimports -w -local github.com/saas-ph/api` y
  `golangci-lint run` limpios antes de cada commit.
- Codigo TS/JS: `pnpm lint` limpio antes de cada commit.
- Lefthook bloqueara commits malos. Si bloquea, arreglar y reintentar — no
  bypassear con `--no-verify`.
- Reporte final en castellano.
