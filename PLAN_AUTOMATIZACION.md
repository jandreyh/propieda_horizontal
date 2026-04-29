# Plan de automatizacion — Claude Code autonomo en SaaS Propiedad Horizontal

Este documento describe COMO opera Claude Code de forma autonoma sobre este
repo. Es la fuente de verdad sobre subagents, comandos, hooks y MCP. Si lees
esto en frio: empieza por aqui, luego `CLAUDE.md`, luego `PLAN_MAESTRO.md`.

---

## 1. Modelo mental

```
                              Tu PC (Windows)
                                    |
                       +-------- Docker Desktop --------+
                       |                                |
                       |        ph-dev container        |
                       |   +--------------------------+ |
   tu shell  ---->     |   | claude (CLI)             | |
                       |   |    |                     | |
                       |   |    v                     | |       internet
                       |   | api.anthropic.com  -----------> Claude (modelo)
                       |   +--------------------------+ |
                       |                                |
                       |   pg-central, pg-tenant-tpl    |
                       +--------------------------------+
                                    |
                       D:\propieda_horizontal (bind mount)
```

**Claude Code (CLI) corre dentro del container**. El modelo (Claude IA) corre
en Anthropic. Tu Windows solo expone el repo via bind-mount; el container no
puede tocar nada mas de tu disco.

---

## 2. Capas de seguridad (ya configuradas)

1. **Container Docker**: aisla del host.
2. **`--dangerously-skip-permissions`**: sin prompts.
3. **`.claude/settings.json` deny list**: bloquea ops catastroficas.
4. **Hooks `PostToolUse`**: auto-format + auto-lint.
5. **Rama `feat/...` obligatoria**: git como undo.
6. **Lefthook pre-commit**: bloquea commits con lint/format/SQL invalidos.
7. **GitHub branch protection** (a configurar): bloquea push directo a main.

---

## 3. Subagents (`.claude/agents/`)

| Agente | Cuando se invoca | Que hace |
|--------|------------------|----------|
| `design-auditor` | `/auditar-diseno`, `/disenar` | Navega web con Claude Preview, screenshots, axe, scoring 1-10 |
| `accessibility-auditor` | parte de design audit | Valida WCAG 2.2 AA con axe-core + checks manuales |
| `security-auditor` | `/auditar-seguridad` | Corre skill `security-review` + revisa invariantes CLAUDE.md |
| `db-architect` | `/auditar-db`, antes de fase con migraciones | Revisa migraciones, indices, ADR 0001/0004/0005 |
| `product-architect` | `/descubrir <N>` | Conduce entrevista de Discovery POST-MVP |
| `e2e-runner` | `/e2e` | Lanza Playwright contra el stack |

Cada subagent tiene:
- Un `system prompt` en su archivo `.md`
- Lista de tools que puede usar (subset)
- Modelo recomendado (sonnet por defecto)

---

## 4. Slash commands (`.claude/commands/`)

| Comando | Proposito | Manual / Auto |
|---------|-----------|---------------|
| `/onboarding` | Cargar contexto en sesion fresca | Manual |
| `/fase <N>` | Ejecutar fase del plan | Manual o `/avanzar` |
| `/descubrir <N>` | Discovery POST-MVP (entrevista) | **Manual** (requiere usuario) |
| `/verificar-fase <N>` | Auditar DoD | Manual |
| `/avanzar` | Detectar y ejecutar siguiente unidad de trabajo | Manual o `/loop` |
| `/completar-mvp` | Runtime end-to-end + scaffolds | Manual o `/avanzar` |
| `/disenar <pagina>` | Diseño interactivo iterativo | Manual (loop con usuario) |
| `/auditar-diseno [pagina\|all]` | Audit completo UI | Manual o `/avanzar` |
| `/auditar-seguridad` | Security review | Manual o pre-merge |
| `/auditar-db` | DB review | Manual o pre-fase |
| `/e2e [escenario]` | Playwright E2E | Manual |

---

## 5. Hooks en `.claude/settings.json`

| Hook | Trigger | Accion |
|------|---------|--------|
| `PostToolUse` | `Edit\|Write` sobre `*.go` | `gofmt -w` + `goimports -w -local github.com/saas-ph/api` |
| `PostToolUse` | `Edit\|Write` sobre `apps/web/**/*.{ts,tsx}` | `pnpm --filter web lint --fix` |
| `PostToolUse` | `Edit\|Write` sobre `migrations/tenant/*.sql` | abortar si aparece `tenant_id` |
| `Stop` | fin de respuesta | imprimir `git status` |

---

## 6. MCP servers (configuracion en `.claude/mcp.json`)

| Server | Proposito | Auth |
|--------|-----------|------|
| `claude-preview` | Preview server + screenshots para design audit | local |
| `claude-in-chrome` | Browser automation real para E2E | local |
| `postgres-mcp` (opcional) | Queries directos a Postgres durante debug | DATABASE_URL |

Comando para activar dentro del container:

```bash
claude mcp install claude-preview
claude mcp install claude-in-chrome
```

---

## 7. Skills de diseño instaladas en `apps/web`

```bash
cd apps/web
pnpm dlx shadcn@latest init           # design system base
pnpm add @axe-core/react              # a11y runtime
pnpm add lucide-react                 # iconos
pnpm add framer-motion                # motion
pnpm add next-intl                    # i18n
pnpm add -D @playwright/test          # E2E
pnpm add -D @storybook/nextjs         # catalogo (opcional)
```

---

## 8. Workflow autonomo recomendado

```bash
# Una vez al dia (o cuando quieras):
docker compose -f deployments/docker-compose.dev.yml up -d
docker compose -f deployments/docker-compose.dev.yml exec dev bash

# Dentro del container:
claude --dangerously-skip-permissions

# En Claude:
/onboarding              # primero, carga contexto
/avanzar                 # arranca trabajo
/loop /avanzar           # opcional: ciclo continuo
```

Si quieres autonomia overnight (sin tu PC):

```bash
# Dentro de Claude Code (cualquier shell con auth):
/schedule
# Configura: cada noche 2 AM correr /avanzar contra el repo en GitHub
```

---

## 9. Definition of Done de la automatizacion

Esto se considera completo cuando:

- [ ] `docker compose up -d` levanta `pg-central`, `pg-tenant-template`, `dev` saludables
- [ ] `docker compose exec dev claude --version` devuelve `2.1.x`
- [ ] `claude` dentro del container reconoce: `/onboarding`, `/fase`, `/avanzar`, `/completar-mvp`, `/disenar`, `/auditar-diseno`, `/auditar-seguridad`, `/auditar-db`, `/e2e`
- [ ] Subagents disponibles: `design-auditor`, `security-auditor`, `db-architect`, `accessibility-auditor`, `product-architect`, `e2e-runner`
- [ ] Hooks `PostToolUse` ejecutan `gofmt`+`goimports` en `*.go`
- [ ] `claude mcp list` muestra `claude-preview` y `claude-in-chrome`
- [ ] `apps/web` tiene shadcn, axe-core, lucide, framer-motion, playwright instalados
- [ ] `pnpm --filter web build` pasa limpio
- [ ] `go test ./apps/api/...` pasa
- [ ] PR autoabierto por `/avanzar` en una rama `feat/auto-...`

---

## 10. Que sigue (aplicar este plan)

El orden recomendado para terminar la implementacion:

1. **Verificar stack** (compose ps healthy)
2. **Smoke del container** (claude, go, node, migrate corren)
3. **Migrar DBs** (`/completar-mvp` paso 3)
4. **Instalar skills web** (paso 7)
5. **Configurar MCP** (paso 6)
6. **Lanzar `/auditar-diseno all`** para baseline
7. **`/loop /avanzar`** para autonomia continua
