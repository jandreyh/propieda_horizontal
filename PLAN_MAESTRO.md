# PLAN MAESTRO DE EJECUCION — SaaS Propiedad Horizontal (MVP)

**Audiencia**: este documento esta optimizado para ejecucion por Claude Code
con agentes en paralelo. Cada fase tiene un brief autocontenido (puede leerse
en frio sin contexto previo), una estrategia multi-agente explicita, salidas
esperadas concretas, y verificaciones automatizables.

**Como usar**:
1. Lee [CLAUDE.md](CLAUDE.md) primero (invariantes, stack, prohibiciones).
2. Ejecuta las fases en orden: `0 -> 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7`.
3. NO inicies una fase hasta que la anterior cumpla su Definition of Done.
4. Para cada fase: usa el slash command `/fase N` o copia el "Brief" de
   esa fase como prompt inicial.

---

## Indice

### MVP (Fases 0-7)
| Fase | Nombre | Paralelizable | Duracion estimada |
|------|--------|---------------|-------------------|
| 0 | Fundaciones de arquitectura y repositorio | Alta | 1-2 dias |
| 1 | Skeleton tecnico backend | Media | 2-3 dias |
| 2 | Identidad, roles, permisos, configuracion | Alta | 3-4 dias |
| 3 | Core dominio residencial | Alta | 2-3 dias |
| 4 | Operacion de porteria | Media | 2-3 dias |
| 5 | Correspondencia y paqueteria | Baja | 3-4 dias |
| 6 | Tablero de anuncios | Alta | 1-2 dias |
| 7 | Hardening para piloto | Media | 2-3 dias |

### POST-MVP (Fases 8-15) — requieren Discovery primero
| Fase | Nombre | Discovery? | Duracion estimada |
|------|--------|-----------|-------------------|
| 8 | Parqueaderos (asignacion, sorteos, visitantes) | Si | 4-5 dias |
| 9 | Modulo financiero (cargos, pagos, pasarela) | Si | 6-8 dias |
| 10 | Reservas de zonas comunes | Si | 3-4 dias |
| 11 | Asambleas, votaciones, actas | Si | 5-7 dias |
| 12 | Incidentes y novedades de seguridad | Si | 2-3 dias |
| 13 | Multas y sanciones | Si | 3-4 dias |
| 14 | PQRS | Si | 3-4 dias |
| 15 | Notificaciones multicanal (WhatsApp, SMS, Push, Email) | Si | 3-4 dias |

### Re-arquitectura de identidad (Fase 16) — bloqueante para escala real
| Fase | Nombre | Discovery? | Duracion estimada |
|------|--------|-----------|-------------------|
| 16 | Identidad cross-tenant + provisioning + selector de conjunto | **Spec frozen** | 7-10 dias |

> **CRITICO**: la Fase 16 reemplaza partes del ADR 0002 con el ADR 0007. Es
> requisito para vender a empresas administradoras y para que personal operativo
> rote entre conjuntos. Spec autocontenida en
> [docs/specs/fase-16-cross-tenant-identity-spec.md](docs/specs/fase-16-cross-tenant-identity-spec.md)
> con 11 pasos ejecutables (16.0 a 16.10) resistentes a sesiones.

> Para fases 8-15, ejecutar `/descubrir <N>` ANTES de `/fase <N>`. El protocolo
> de discovery hace que Claude entreviste al usuario con rondas estructuradas
> y consolide las respuestas en `/docs/specs/fase-<N>-spec.md` antes de codear.

---

## Glosario rapido

- **Tenant** = un conjunto residencial. Tiene su propia DB Postgres aislada.
- **Control Plane** = base central de plataforma (tabla `tenants`, dominios, billing SaaS).
- **Data Plane** = base por tenant. NO contiene `tenant_id` en tablas.
- **Unidad** = apartamento/casa/local/oficina dentro de un tenant.
- **Residente** = persona vinculada operativamente a una unidad (propietario, inquilino, autorizado).
- **Modulo** = carpeta `internal/modules/<nombre>/` con Clean Architecture.
- **DoD** = Definition of Done (lista de aceptacion de fase).
- **ADR** = Architectural Decision Record en `/docs/adr/`.

---

## Convenciones para Claude Code

### TodoWrite
Antes de iniciar una fase, crear un TodoWrite con TODAS las tareas listadas
en la seccion "Tareas" de la fase. Marcar `in_progress` solo una a la vez.

### Multi-agente
Donde se indique "paralelizable", lanzar UNA sola tool call con multiples
invocaciones a `Agent`. Ejemplo conceptual:

```
[Agent ADR-1, Agent ADR-2, Agent ADR-3, Agent ADR-4, Agent ADR-5]
```

Cada agente recibe un prompt autocontenido. NO compartir estado entre agentes
durante su ejecucion. La sintesis se hace despues, al recibir todos los resultados.

### Verificacion al cerrar fase
Ejecutar TODOS los comandos del bloque "Verificacion" antes de marcar DoD.
Si alguno falla: arreglar antes de pasar de fase.

---

## PROTOCOLO DE DISCOVERY (para fases 8-15)

Las fases POST-MVP no se programan a ciegas. Cada una requiere primero una
ronda de entrevistas con el usuario para capturar reglas de negocio
especificas que NO estan en este documento. Claude actua como **Senior
Product Manager + Arquitecto** y conduce la entrevista.

### Cuando ejecutar Discovery
- Al iniciar una fase 8-15 por primera vez.
- Cuando una decision de negocio cambie a mitad de fase (re-discovery parcial).

### Como ejecutar Discovery
Usar `/descubrir <N>`. El comando:
1. Lee la seccion `## FASE N` del plan.
2. Presenta al usuario las rondas de preguntas (A, B, C, D, E).
3. Espera respuestas por bloque numerado.
4. Al recibir todas las respuestas, sintetiza en `/docs/specs/fase-<N>-spec.md`
   con: decisiones tomadas, supuestos, preguntas abiertas, modelo de datos
   propuesto, endpoints, permisos, casos extremos, riesgos.
5. Pide al usuario que valide la spec antes de marcarla como "frozen".

### Rondas estandar de Discovery (estructura comun)

Cada fase POST-MVP tiene 5 bloques de preguntas:

- **A. Modelo de datos y entidades**: que se modela, relaciones,
  cardinalidades, identificadores, historial.
- **B. Reglas de negocio y flujos operativos**: estados, transiciones,
  validaciones, prevalencias, edge cases.
- **C. Permisos y roles**: quien puede que, scopes, delegaciones,
  prevalencias propietario vs inquilino.
- **D. Concurrencia, transaccionalidad e idempotencia**: que operaciones
  son criticas, que conflictos preocupan, que debe ser idempotente.
- **E. Configuracion por tenant, notificaciones, reportes**: que se
  configura, que canales, que reportes/metricas.

### Reglas para Claude durante Discovery
- **NO inferir** reglas criticas de negocio. Si falta info: preguntar.
- **NO codear** durante Discovery. Solo entrevistar y sintetizar.
- **NO saltarse rondas**. Si el usuario responde solo A, NO avanzar a B
  hasta tener todas las respuestas de A.
- **Proponer defaults seguros** cuando sea posible, con racional. El
  usuario los acepta o cambia.
- **Detectar dependencias** con fases anteriores (ej. "esto requiere
  modulo financiero, que esta en Fase 9 -- ¿implementar antes o stub?").
- **Marcar OPEN QUESTIONS** que requieren input externo (legal,
  contable, regulatorio).

### Output de Discovery: spec frozen

Estructura obligatoria de `/docs/specs/fase-<N>-spec.md`:

```markdown
# Fase <N> — Spec frozen — <Nombre del modulo>

**Estado**: Frozen | Validado por: <usuario> | Fecha: <YYYY-MM-DD>

## 1. Resumen ejecutivo
<3-5 lineas>

## 2. Decisiones tomadas
- ...

## 3. Supuestos adoptados (no bloqueantes)
- ...

## 4. Open Questions (resolverse antes de programar)
- ...

## 5. Modelo de datos propuesto
- Tablas, columnas, indices, FK.

## 6. Endpoints
- Verbo + path + permiso requerido + status codes.

## 7. Permisos nuevos a registrar
- Namespace: descripcion.

## 8. Casos extremos (edge cases)
- ...

## 9. Operaciones transaccionales / idempotentes
- ...

## 10. Configuracion por tenant
- Keys nuevas en `tenant_settings` y defaults.

## 11. Notificaciones / eventos
- Que dispara que canal y a quien.

## 12. Reportes / metricas
- ...

## 13. Riesgos y mitigaciones
- ...

## 14. Multi-agente sugerido
- Como subdividir el trabajo y cuantos agentes lanzar en paralelo.

## 15. DoD adicional especifico de la fase
- Items que extienden el DoD universal.
```

Una vez "Frozen" y validada, ejecutar `/fase <N>` para la implementacion.

---

## FASE 0 — Fundaciones de arquitectura y repositorio

**Objetivo**: congelar decisiones, crear esqueleto de directorios, configurar
herramientas. Cero logica de negocio.

### Pre-condiciones
- Carpeta `/workspace` existe.
- Docker disponible para levantar Postgres.

### Brief (prompt autocontenido)

> Ejecuta la Fase 0 del PLAN_MAESTRO.md. Lee CLAUDE.md primero.
>
> Objetivo: crear la estructura del repositorio para un SaaS de Propiedad
> Horizontal con backend Go modular monolith + multi-tenant DB-por-tenant.
> NO escribir logica de negocio en esta fase. Solo esqueleto, herramientas
> y ADRs.
>
> Entregables:
> 1. Estructura `/apps/api`, `/apps/web`, `/apps/mobile`, `/deployments`,
>    `/docs/adr`, `/docs/openapi`, `/migrations/central`, `/migrations/tenant`.
> 2. `go.mod` en `/apps/api` con dependencias: chi, pgx/v5, golang-migrate.
> 3. `docker-compose.local.yml` en `/deployments/` con dos servicios Postgres:
>    `pg-central` (puerto 5432) y `pg-tenant-template` (puerto 5433).
> 4. `.golangci.yml` en raiz con configuracion estricta.
> 5. Hooks de pre-commit (`.pre-commit-config.yaml` o `lefthook.yml`) que
>    ejecuten `gofmt`, `goimports`, `golangci-lint`.
> 6. Cinco ADRs (ver lista abajo) en `/docs/adr/`. Cada ADR sigue el formato
>    Markdown con secciones: Contexto, Decision, Consecuencias, Alternativas
>    consideradas.
> 7. README.md raiz con instrucciones para levantar el entorno local.
>
> Estrategia: spawn 5 agentes en paralelo, uno por ADR. Mientras esos corren,
> tu ejecutas la creacion de directorios y configuracion local en el hilo
> principal.

### Tareas (TodoWrite)
1. Crear arbol de directorios completo.
2. Inicializar `go.mod` y dependencias en `/apps/api`.
3. Escribir `docker-compose.local.yml`.
4. Configurar linter y pre-commit hooks.
5. Spawn 5 agentes para los 5 ADRs (paralelo).
6. Escribir README raiz.
7. Verificar que `docker compose up -d` levanta ambos Postgres.
8. Verificar que `go build ./...` no falla (aunque no haya codigo).

### Estrategia multi-agente — ADRs en paralelo

Lanzar en una sola tool call (5 agentes `general-purpose` simultaneos):

| Agente | Archivo | Foco del prompt |
|--------|---------|-----------------|
| ADR-1 | `0001-architecture-multi-tenant-strategy.md` | Justificar DB por tenant, Control Plane vs Data Plane, resolucion por subdominio, prohibicion de tenant_id en tablas operativas |
| ADR-2 | `0002-authentication-and-identity.md` | Login por tenant, identidad global solo para Superadmin, MFA obligatorio, PK UUID + identificador de negocio (document_type + document_number), email nullable |
| ADR-3 | `0003-authorization-rbac-scopes.md` | RBAC con roles + permisos granulares + scopes (tenant, torre, unidad, modulo, horario), namespaces de permisos (`package.deliver`, `visit.create`, etc.) |
| ADR-4 | `0004-audit-and-soft-delete-strategy.md` | Soft delete generalizado, campos estandar, tabla `audit_logs` inmutable, trazabilidad de cambios de permisos |
| ADR-5 | `0005-transactional-and-idempotency-strategy.md` | Transaccionalidad obligatoria, bloqueo optimista con `version`, idempotency keys en webhooks/pagos, outbox pattern para eventos |

Brief base para CADA agente ADR (personalizar por foco):

> Eres un Senior Software Architect. Escribe el ADR `<NUMERO>-<SLUG>.md` para
> un SaaS de Propiedad Horizontal multi-tenant en Go.
>
> Contexto del producto: SaaS para conjuntos residenciales en Colombia.
> Backend Go 1.26+, chi, pgx + sqlc, PostgreSQL 18. Multi-tenant DB-por-tenant.
> Foco del ADR: <FOCO ESPECIFICO>.
>
> Estructura obligatoria del ADR:
> 1. Titulo y numero
> 2. Estado (Accepted)
> 3. Contexto (problema y restricciones)
> 4. Decision (que se decide concretamente)
> 5. Consecuencias (positivas y negativas)
> 6. Alternativas consideradas y por que se descartaron
> 7. Implicaciones tecnicas concretas (ejemplos de codigo / SQL si aplica)
>
> Escribir en espanol tecnico. Maximo 2 paginas. Concreto, no filosofico.
> Output: archivo en `/docs/adr/<NUMERO>-<SLUG>.md`.

### Salidas esperadas
```
/apps/api/go.mod
/apps/web/                          (vacio, placeholder)
/apps/mobile/                       (vacio, placeholder)
/deployments/docker-compose.local.yml
/docs/adr/0001-architecture-multi-tenant-strategy.md
/docs/adr/0002-authentication-and-identity.md
/docs/adr/0003-authorization-rbac-scopes.md
/docs/adr/0004-audit-and-soft-delete-strategy.md
/docs/adr/0005-transactional-and-idempotency-strategy.md
/migrations/central/.gitkeep
/migrations/tenant/.gitkeep
/.golangci.yml
/lefthook.yml (o .pre-commit-config.yaml)
/README.md
```

### Verificacion
```bash
cd apps/api && go build ./...
docker compose -f deployments/docker-compose.local.yml up -d
docker compose -f deployments/docker-compose.local.yml ps   # ambos UP
golangci-lint run ./apps/api/...
test -f docs/adr/0001-architecture-multi-tenant-strategy.md
test -f docs/adr/0005-transactional-and-idempotency-strategy.md
```

### DoD Fase 0
- [ ] Arbol de directorios completo.
- [ ] 5 ADRs creados con contenido sustantivo (no placeholders).
- [ ] Docker compose levanta `pg-central` y `pg-tenant-template`.
- [ ] `go build ./...` sin errores.
- [ ] Linter sin warnings.
- [ ] README explica como levantar el entorno.

---

## FASE 1 — Skeleton tecnico backend

**Objetivo**: tuberia central, conexiones, middlewares, sin entidades de negocio.

### Pre-condiciones
- Fase 0 con DoD cumplido.
- Postgres central levantado.

### Brief

> Ejecuta la Fase 1. Lee CLAUDE.md y PLAN_MAESTRO.md (Fase 1).
>
> Objetivo: construir el bootstrap del servidor Go, pool de conexiones, y
> los middlewares base. SIN entidades de negocio.
>
> Entregables:
> 1. `cmd/api/main.go` que levanta servidor `chi` en puerto configurable.
> 2. Carga de env vars (`os.Getenv` o `viper`/`envconfig`).
> 3. `internal/platform/db/` con `pgxpool` para Control Plane y para Tenant DBs
>    (con cache de conexiones por tenant).
> 4. `internal/platform/middleware/` con:
>    - `Logging` (OpenTelemetry, zerolog o slog)
>    - `Recovery` (recupera panics, devuelve 500 RFC7807)
>    - `RequestID` (genera/propaga `X-Request-ID`)
>    - `RateLimit` (token bucket basico, en memoria por ahora)
>    - `TenantResolver` (extrae subdominio, consulta Control Plane, inyecta
>       conexion al Tenant DB en el contexto)
> 5. `internal/platform/errors/` con tipos de error mapeables a RFC7807.
> 6. `internal/platform/migrations/` con configuracion de `golang-migrate`.
> 7. Endpoint `/health` (publico, sin tenant) y `/ready` (con tenant resuelto).
>
> Estrategia: el grueso es secuencial (bootstrap, db, middleware base), pero
> los 5 middlewares se pueden escribir en paralelo (un agente por middleware).

### Tareas
1. Crear `cmd/api/main.go` con bootstrap minimo.
2. Implementar `internal/platform/db/pool.go` (Control Plane).
3. Implementar `internal/platform/db/tenant_pool.go` (cache de pools por tenant).
4. Implementar paquete de errores RFC7807.
5. Spawn 5 agentes en paralelo, uno por middleware.
6. Cablear todo en `main.go`.
7. Implementar endpoints `/health` y `/ready`.
8. Configurar `golang-migrate` con drivers para central y tenant.
9. Verificacion completa.

### Estrategia multi-agente — Middlewares en paralelo

Lanzar 5 agentes `general-purpose` simultaneos. Cada uno escribe UN solo archivo
en `internal/platform/middleware/`. Brief base:

> Implementa el middleware `<NOMBRE>` para un servidor `chi` en Go 1.26+.
> Foco: <DESCRIPCION>. Output: archivo `internal/platform/middleware/<nombre>.go`
> + tests en `<nombre>_test.go`. Sigue Clean Architecture y los principios de
> CLAUDE.md. NO escribas en main.go (eso lo hace el orquestador).

Variantes:
- `logging.go` — OpenTelemetry traces + slog estructurado.
- `recovery.go` — recover panic, log, devolver 500 RFC7807.
- `request_id.go` — leer/generar `X-Request-ID`, exponer en contexto.
- `rate_limit.go` — token bucket por IP, configurable.
- `tenant_resolver.go` — parse host, consultar `tenants` en Control Plane,
  inyectar `*pgxpool.Pool` del tenant en `context.WithValue`.

### Salidas esperadas
```
/apps/api/cmd/api/main.go
/apps/api/internal/platform/db/pool.go
/apps/api/internal/platform/db/tenant_pool.go
/apps/api/internal/platform/db/tenant_pool_test.go
/apps/api/internal/platform/errors/problem.go
/apps/api/internal/platform/middleware/logging.go
/apps/api/internal/platform/middleware/recovery.go
/apps/api/internal/platform/middleware/request_id.go
/apps/api/internal/platform/middleware/rate_limit.go
/apps/api/internal/platform/middleware/tenant_resolver.go
/apps/api/internal/platform/migrations/migrate.go
/apps/api/internal/handlers/health.go
```

### Verificacion
```bash
cd apps/api && go build ./...
go test ./internal/platform/...
# Levantar y probar:
go run ./cmd/api &
curl -i http://localhost:8080/health                      # 200
curl -i -H "Host: demo.localhost:8080" http://localhost:8080/ready  # depende del tenant en DB
```

### DoD Fase 1
- [ ] Servidor levanta y responde `/health` con 200.
- [ ] Middleware `TenantResolver` resuelve un subdominio y conecta a la DB del tenant.
- [ ] Errores devuelven `application/problem+json`.
- [ ] Logs son JSON estructurado con `request_id`.
- [ ] Migraciones se pueden correr (`migrate -path ... up`).
- [ ] Tests unitarios de cada middleware pasan.

---

## FASE 2 — Identidad, roles, permisos, configuracion

**Objetivo**: quien es quien y que puede hacer.

### Pre-condiciones
- Fase 1 DoD cumplido. Middleware tenant_resolver funcional.

### Brief

> Ejecuta la Fase 2. Lee CLAUDE.md y PLAN_MAESTRO.md (Fase 2).
>
> Objetivo: crear los modulos `identity`, `authorization`, `tenant_config`
> en `/apps/api/internal/modules/` siguiendo Clean Architecture estricta
> (ver CLAUDE.md seccion 3).
>
> Entregables por modulo (ver lista de tablas y endpoints abajo). Migraciones
> en `/migrations/tenant/`. Roles semilla insertados via migracion.
>
> Estrategia: 3 modulos independientes -> 3 agentes en paralelo.

### Tablas (Tenant DB)

`identity`:
- `users` (id, document_type, document_number, names, last_names, email NULL, phone, password_hash, mfa_secret, status, + estandar)
- `user_sessions` (id, user_id, token_hash, expires_at, ip, user_agent)
- `user_mfa_recovery_codes` (id, user_id, code_hash, used_at)

`authorization`:
- `roles` (id, name, description, is_system_seed, + estandar)
- `permissions` (id, namespace, description) — seed estatico
- `role_permissions` (role_id, permission_id)
- `user_role_assignments` (user_id, role_id, scope_type NULL, scope_id NULL)

`tenant_config`:
- `tenant_settings` (key, value JSONB, updated_at)
- `tenant_branding` (logo_url, primary_color, secondary_color, display_name)

### Endpoints

`identity`:
- `POST /auth/login` — body: `{identifier, password}` (identifier = email o `<doc_type>:<doc_number>`)
- `POST /auth/mfa/verify` — body: `{session_token, code}`
- `POST /auth/logout`
- `GET /me` — usuario actual

`authorization`:
- `GET /roles`
- `POST /roles` (requiere permiso `role.create`)
- `PUT /roles/:id`
- `DELETE /roles/:id`
- `GET /permissions` — catalogo de namespaces
- `POST /users/:id/roles` (asignacion)
- `DELETE /users/:id/roles/:role_id`

`tenant_config`:
- `GET /settings`
- `PUT /settings/:key`
- `GET /branding`
- `PUT /branding`

### Estrategia multi-agente — 3 modulos en paralelo

Brief base por agente:

> Implementa el modulo `<MODULO>` en `/apps/api/internal/modules/<MODULO>/`
> siguiendo Clean Architecture (ver CLAUDE.md). Tablas: <TABLAS>. Endpoints:
> <ENDPOINTS>. Migraciones SQL Up/Down en `/migrations/tenant/<NNN>_<modulo>.up.sql`
> y `.down.sql`. Queries en `infrastructure/persistence/queries/*.sql` para sqlc.
>
> Generar codigo con `sqlc` (config en `/apps/api/sqlc.yaml`). Tests unitarios
> para policies y usecases. Tests de integracion para repositorio con
> Testcontainers.
>
> Roles semilla a insertar (via migracion): platform_superadmin (en Control Plane),
> tenant_admin, guard, accountant, owner, tenant_resident, authorized_resident,
> board_member, auditor_or_revisor.
>
> Permisos semilla (namespaces): identity.read, identity.write, role.create,
> role.read, role.update, role.delete, permission.read, settings.read,
> settings.write, branding.read, branding.write.
>
> NO hagas el trabajo de los otros dos modulos. NO toques `cmd/api/main.go`
> (el orquestador cablea las rutas despues).

### Salidas esperadas
```
/apps/api/internal/modules/identity/
/apps/api/internal/modules/authorization/
/apps/api/internal/modules/tenant_config/
/apps/api/sqlc.yaml
/migrations/tenant/001_identity.up.sql
/migrations/tenant/001_identity.down.sql
/migrations/tenant/002_authorization.up.sql
/migrations/tenant/002_authorization.down.sql
/migrations/tenant/003_tenant_config.up.sql
/migrations/tenant/003_tenant_config.down.sql
/migrations/tenant/seed_001_roles_permissions.up.sql
/docs/openapi/identity.yaml
/docs/openapi/authorization.yaml
/docs/openapi/tenant_config.yaml
```

### Verificacion
```bash
cd apps/api && go build ./...
go test ./internal/modules/...
# Migrar tenant template y validar:
migrate -path ../../migrations/tenant -database $TENANT_DB_URL up
psql $TENANT_DB_URL -c "SELECT name FROM roles;"     # 8 filas semilla
# Login flow:
curl -X POST http://demo.localhost:8080/auth/login -d '{"identifier":"...","password":"..."}'
```

### DoD Fase 2
- [ ] 3 modulos compilan y pasan tests.
- [ ] Migraciones Up/Down reversibles.
- [ ] Roles semilla insertados en `roles`.
- [ ] Permisos semilla insertados en `permissions`.
- [ ] Endpoint `POST /auth/login` retorna JWT/session valido.
- [ ] Middleware `RequirePermission(...)` bloquea peticiones sin el permiso.
- [ ] OpenAPI generado para los 3 modulos.

---

## FASE 3 — Core dominio residencial

**Objetivo**: mapa fisico del conjunto y relaciones de ocupacion.

### Pre-condiciones
- Fase 2 DoD cumplido. Modulo `identity` operativo.

### Brief

> Ejecuta la Fase 3. Modulos: `residential_structure`, `units`, `people`.
> Tres agentes en paralelo.
>
> Entidades clave:
> - `residential_structures` (Torre A, Bloque 1, Etapa 2)
> - `units` (Apto 101, Local 5, Casa 23) con `structure_id`
> - `unit_owners` (unit_id, user_id, percentage)
> - `unit_occupancies` (unit_id, user_id, role_in_unit, is_primary, move_in_date, move_out_date)
> - `vehicles` (id, plate, type)
> - `unit_vehicle_assignments` (unit_id, vehicle_id)
>
> Caso de uso critico: dado `unit_id`, devolver TODAS las personas autorizadas
> ahora mismo (propietarios + ocupantes activos sin `move_out_date`).

### Tareas
1. Spawn 3 agentes paralelos (uno por modulo).
2. Cablear rutas en `main.go` despues.
3. Tests E2E del caso critico.

### Endpoints
- `POST /structures` — crear torre/bloque
- `GET /structures` — listar
- `POST /units` — crear unidad
- `GET /units/:id` — detalle
- `GET /units/:id/people` — caso critico
- `POST /units/:id/owners`
- `POST /units/:id/occupants`
- `POST /vehicles`
- `POST /units/:id/vehicles`

### Salidas esperadas
```
/apps/api/internal/modules/residential_structure/
/apps/api/internal/modules/units/
/apps/api/internal/modules/people/
/migrations/tenant/004_residential_structure.up.sql + down
/migrations/tenant/005_units.up.sql + down
/migrations/tenant/006_people.up.sql + down
```

### DoD Fase 3
- [ ] Endpoint `GET /units/:id/people` devuelve propietarios e inquilinos activos.
- [ ] `move_out_date` excluye correctamente ocupantes pasados.
- [ ] Tests de integracion contra Postgres real (Testcontainers).
- [ ] OpenAPI actualizado.

---

## FASE 4 — Operacion base de porteria

**Objetivo**: herramientas para el guarda en el flujo de ingreso.

### Pre-condiciones
- Fase 3 DoD cumplido.

### Brief

> Ejecuta la Fase 4. Modulo: `access_control`.
>
> Reglas de negocio CRITICAS (no negociables):
> 1. El guarda NO puede ver estado de cartera/deudas (filtrar en authz).
> 2. Visitantes sin pre-registro: capturar document_type, document_number,
>    full_name, destination_unit_id, photo_evidence (URL).
> 3. Domicilios = visitantes estandar (no categoria especial).
> 4. Validar contra `blacklisted_persons` ANTES de aceptar registro. Si
>    hit -> bloquear UI con mensaje claro y registrar intento.
> 5. QR de pre-registro: tabla con `expires_at` y `max_uses`. Cada uso
>    incrementa contador y verifica expiracion.

### Tablas
- `visitor_pre_registrations` (id, unit_id, visitor_name, visitor_document, qr_code_hash, expires_at, max_uses, uses_count, status, + estandar)
- `visitor_entries` (id, unit_id, visitor_name, document_number, photo_url, guard_id, entry_time, exit_time, source ENUM[QR, MANUAL])
- `blacklisted_persons` (id, document_number, reason, reported_by_unit_id, + estandar)

### Endpoints
- `POST /visitor-preregistrations` (residente crea, genera QR firmado)
- `POST /visits/checkin-by-qr` (guarda escanea QR)
- `POST /visits/checkin-manual` (guarda registra sin QR, requiere foto)
- `POST /visits/:id/checkout`
- `GET /visits/active`
- `POST /blacklist` (admin agrega)
- `GET /blacklist`

### Estrategia multi-agente
Modulo unico pero subtareas independientes -> 3 agentes:
- Agente A: tablas + migraciones + queries sqlc.
- Agente B: usecases (preregistro, checkin QR, checkin manual, blacklist check).
- Agente C: handlers HTTP + DTOs + OpenAPI.

Sincronizar al final: handler usa usecase, usecase usa repositorio.

### DoD Fase 4
- [ ] QR pre-registro respeta `expires_at` y `max_uses`.
- [ ] Blacklist bloquea checkin manual y por QR.
- [ ] Guarda no ve cartera (verificar en test de autorizacion).
- [ ] Foto de evidencia es obligatoria en checkin manual.
- [ ] Cada visita queda en `audit_logs` con `guard_id`.

---

## FASE 5 — Correspondencia y paqueteria (modulo critico MVP)

**Objetivo**: digitalizar paqueteria con cero perdida de trazabilidad.

### Pre-condiciones
- Fase 4 DoD cumplido.

### Brief

> Ejecuta la Fase 5. Modulo: `packages`. Es el modulo de mayor riesgo
> operacional — concurrencia y trazabilidad criticas.
>
> Reglas de negocio CRITICAS:
> 1. Categorias opcionales: si vacio -> "Estandar". Lista configurable
>    en `tenant_settings`.
> 2. Foto de evidencia opcional al recibir.
> 3. Al recibir -> emitir evento async (outbox pattern) para notificar
>    residente. NO bloquear el response del guarda esperando la notificacion.
> 4. Entrega via QR del residente (flujo feliz).
> 5. Entrega manual (excepcion): requiere `signature_image_url` o
>    `photo_evidence_url`. Severity HIGH en audit_logs.
> 6. Estado RECEIVED -> DELIVERED via UPDATE con `WHERE version = $oldVersion`.
>    Si rowsAffected == 0 -> conflicto, devolver 409.
> 7. Cron diario 8:00 AM (timezone del tenant) -> re-notificar paquetes
>    `RECEIVED` con > 3 dias.

### Tablas
- `packages` (id, unit_id, recipient_name, category, status ENUM, received_evidence_url NULL, version, + estandar)
- `package_delivery_events` (id, package_id, delivered_to_user_id NULL, recipient_name_manual NULL, delivery_method ENUM[QR, MANUAL], signature_url NULL, photo_evidence_url NULL, guard_id, delivered_at)
- `package_categories` (id, name, requires_evidence BOOL) — semilla con "Estandar", "Refrigerado", "Sobre", "Caja Grande"

### Endpoints
- `POST /packages` (guarda registra)
- `GET /packages?unit_id=&status=`
- `POST /packages/:id/deliver-by-qr` (residente escanea QR, guarda confirma)
- `POST /packages/:id/deliver-manual` (firma o foto obligatoria)
- `POST /packages/:id/return` (devolucion al remitente)

### Estrategia multi-agente
Subdivision por capa (4 agentes):
- Agente A: schema SQL + migraciones + queries sqlc + outbox table.
- Agente B: usecases con bloqueo optimista + idempotency.
- Agente C: cron job (`internal/jobs/package_reminder.go`).
- Agente D: handlers + DTOs + OpenAPI + tests de concurrencia (race con
  `go test -race`).

### Verificacion especifica
```bash
# Test de concurrencia: dos peticiones simultaneas para entregar el mismo paquete
go test -race ./internal/modules/packages/...
# Una debe ganar (200), otra debe recibir 409.
```

### DoD Fase 5
- [ ] Bloqueo optimista probado: 2 deliver simultaneos -> 1 OK + 1 409.
- [ ] Outbox emite evento de notificacion al recibir.
- [ ] Cron re-notifica paquetes > 3 dias.
- [ ] Entrega manual graba `signature_url` o `photo_evidence_url`.
- [ ] `audit_logs` tiene `guard_id` exacto en cada entrega.
- [ ] `go test -race ./...` sin avisos.

---

## FASE 6 — Tablero de anuncios

**Objetivo**: comunicado unidireccional admin -> residentes.

### Pre-condiciones
- Fase 5 DoD cumplido.

### Brief

> Ejecuta la Fase 6. Modulo: `announcements`. Es el modulo mas pequeno y
> sirve como warmup post-modulo critico.
>
> Reglas:
> 1. Solo permiso `announcement.create` puede publicar.
> 2. Audiencia: GLOBAL, TOWER (id), ROLE (id), o lista combinada.
> 3. `expires_at` opcional. Cuando expira -> oculto en feed pero conservado.
> 4. Feed ordenado: `pinned` primero (boolean), luego `published_at DESC`.

### Tablas
- `announcements` (id, title, body, published_by, pinned, expires_at, + estandar)
- `announcement_audiences` (announcement_id, target_type ENUM, target_id NULL)
- `announcement_acknowledgments` (announcement_id, user_id, acknowledged_at)

### Endpoints
- `POST /announcements`
- `GET /announcements/feed` (filtra por audiencia del usuario)
- `POST /announcements/:id/ack`
- `DELETE /announcements/:id`

### Estrategia multi-agente
Modulo pequeno -> 1 agente solo. Si hay tiempo, paralelizar con Fase 7
(hardening) que es independiente.

### DoD Fase 6
- [ ] Anuncio "Torre A" no visible en feed de residente de Torre B.
- [ ] `expires_at` oculta del feed pero NO borra.
- [ ] `pinned` aparece primero.
- [ ] OpenAPI actualizado.

---

## FASE 7 — Hardening para piloto

**Objetivo**: que el sistema no colapse en operacion real.

### Pre-condiciones
- Fase 6 DoD cumplido.

### Brief

> Ejecuta la Fase 7. NO hay logica de negocio nueva. Solo:
> 1. Indices compuestos: ver lista abajo.
> 2. Rate limiting reforzado en `/auth/login` (max 5 intentos / 15 min / IP).
> 3. Auditoria inmutable: tabla `audit_logs` con trigger que rechaza UPDATE/DELETE.
> 4. Tests E2E Playwright para flujo: login guarda -> registro paquete ->
>    login residente -> ver paquete -> entrega QR.
> 5. Documento de runbook operativo (`/docs/runbook.md`).

### Indices criticos
```sql
CREATE INDEX idx_packages_unit_status ON packages (unit_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_packages_status_received_at ON packages (status, created_at) WHERE status = 'RECEIVED';
CREATE INDEX idx_visitor_entries_unit ON visitor_entries (unit_id, entry_time DESC);
CREATE INDEX idx_user_role_assignments_user ON user_role_assignments (user_id);
CREATE INDEX idx_announcements_published ON announcements (pinned DESC, created_at DESC) WHERE deleted_at IS NULL;
```

### Estrategia multi-agente
4 agentes paralelos:
- Agente A: indices + EXPLAIN ANALYZE benchmarks.
- Agente B: rate limit + tests de carga (vegeta o k6).
- Agente C: auditoria inmutable (trigger SQL + migracion + tests).
- Agente D: Playwright E2E + runbook.

### DoD Fase 7
- [ ] Indices creados, EXPLAIN muestra usage.
- [ ] Login resiste fuerza bruta (5 intentos -> bloqueo 15 min).
- [ ] `UPDATE audit_logs` falla con error de trigger.
- [ ] Playwright pasa el flujo E2E completo.
- [ ] Runbook documenta: backup, restore, rotacion de logs, on-call.

---

## Matriz de Roles Semilla (insertar via migracion en Fase 2)

| Rol | Scope | Permisos clave | Notas |
|-----|-------|---------------|-------|
| `platform_superadmin` | Control Plane | crear tenants, suspender, impersonar | Solo en DB central |
| `tenant_admin` | Tenant | configuracion total, gestion de residentes, reportes, branding | |
| `guard` | Tenant | `visit.create`, `visit.checkin`, `package.receive`, `package.deliver` | NO acceso a cartera |
| `accountant` | Tenant | finanzas (post-MVP) | Preparado para fase futura |
| `owner` | Unidad propia | `unit.read`, `package.read`, `visit.authorize` | Prevalencia sobre inquilino |
| `tenant_resident` | Unidad ocupada | similar a owner pero limitado | Sujeto a prevalencia |
| `authorized_resident` | Unidad asociada | datos no sensibles, retiro de paquetes si autorizado | Familiar/hijo |
| `board_member` | Tenant | solo lectura de reportes no operativos | Consejo |
| `auditor_or_revisor` | Tenant | lectura total + auditoria, mutacion prohibida | Revisor fiscal |

---

## Anti-patrones (que NO hacer)

- NO incluir `tenant_id` en tablas operativas del Tenant DB.
- NO usar GORM, ent, beego ni ningun ORM pesado.
- NO escribir SQL inline en codigo Go (usar `.sql` + sqlc).
- NO mezclar logica de dominio con handlers HTTP.
- NO hacer hard delete en tablas operativas (soft delete con `deleted_at`).
- NO procesar notificaciones sincronicamente bloqueando el response.
- NO almacenar contrasenas con SHA simple (usar bcrypt o argon2id).
- NO emitir tokens largos sin refresh + revocacion.
- NO commitear `.env` con secretos.
- NO saltarse migraciones Down (deben ser reversibles).
- NO hacer features beyond del scope MVP (sin parqueaderos avanzados, sin
  pagos, sin reservas — eso es post-MVP).

---

## Comandos utiles

```bash
# Levantar entorno local
docker compose -f deployments/docker-compose.local.yml up -d

# Migrar control plane
migrate -path migrations/central -database $CENTRAL_DB_URL up

# Migrar tenant (template)
migrate -path migrations/tenant -database $TENANT_DB_URL up

# Generar codigo sqlc
cd apps/api && sqlc generate

# Build + lint + test (full check)
cd apps/api && go build ./... && golangci-lint run ./... && go test -race ./...

# Levantar API
cd apps/api && go run ./cmd/api

# Crear tenant nuevo (despues de tener el modulo de provisioning)
curl -X POST http://localhost:8080/platform/tenants \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
  -d '{"name":"Conjunto Demo","subdomain":"demo"}'
```

---

---

# FASES POST-MVP (8-15) — Discovery + ejecucion

> **IMPORTANTE**: estas fases requieren `/descubrir <N>` ANTES de `/fase <N>`.
> Las preguntas de cada fase son la fuente para que Claude consolide la spec
> frozen en `/docs/specs/`. El bloque "Plan template" es solo orientativo —
> la spec frozen prevalece sobre cualquier sugerencia preliminar.

---

## FASE 8 — Parqueaderos

**Pre-condiciones**: Fase 7 DoD cumplido. Modulo `units` y `people` operativos.

### Discovery — preguntas estructuradas

#### Bloque A. Modelo de datos
1. ¿Los parqueaderos son entidades independientes o atributos de una unidad?
2. ¿Que tipos manejas? (cubierto, descubierto, doble, moto, bicicleta, visitantes, discapacitados, electrico).
3. ¿Hay parqueaderos comunes (sin propietario fijo) y privados (asignados a unidad)?
4. ¿Quien es el "dueno" del parqueadero — la unidad, el propietario, el conjunto?
5. ¿Necesitas historial de asignaciones (quien lo tuvo, desde cuando hasta cuando)?
6. ¿Hay numeracion fisica? ¿Por torre, por nivel, por zona?
7. ¿Soporta parqueaderos con tarifa de alquiler (no esenciales)?
8. ¿Necesitas vincular vehiculos a parqueaderos especificos o solo a la unidad?

#### Bloque B. Reglas de negocio y flujos
1. ¿Asignacion fija (1 unidad = 1 parqueadero) o pool (rotacion)?
2. ¿Una unidad puede tener varios parqueaderos? ¿Cuantos max?
3. ¿Un parqueadero puede tener varios "co-usuarios" o solo uno?
4. ¿Como manejas reasignaciones? ¿Aprobacion del consejo? ¿Solo admin?
5. ¿Sorteos? Si si: ¿periodicidad, criterios (antiguedad, dependientes), publicacion de resultados?
6. ¿Parqueaderos de visitantes: reservacion previa o por llegada (FCFS)?
7. ¿Tiempo maximo de uso de visitante? ¿Multas por exceso?
8. ¿Existe alquiler temporal entre residentes (subarriendo controlado)?

#### Bloque C. Permisos y roles
1. ¿Quien asigna parqueaderos? (admin, consejo aprobado, sorteo automatico).
2. ¿El residente puede ver SU parqueadero pero no el de otros?
3. ¿El guarda valida ingreso vehicular contra asignacion? ¿Tiene UI especifica?
4. ¿El propietario puede ceder su parqueadero al inquilino o requiere aprobacion?
5. ¿Permisos temporales (visitas con vehiculo > X horas)?

#### Bloque D. Concurrencia y transaccionalidad
1. ¿Que pasa si dos personas reservan el mismo parqueadero de visitantes simultaneamente?
2. ¿Bloqueo optimista o pesimista para asignaciones?
3. ¿Idempotencia en reservas (mismo request 2 veces)?
4. ¿Que evento dispara cuando se asigna? (notif al residente).
5. ¿Como manejas cancelacion / no-show de visitante?

#### Bloque E. Configuracion por tenant + notificaciones + reportes
1. ¿Tarifas configurables por tenant?
2. ¿Reglas de sorteo configurables (frecuencia, semilla, criterios)?
3. ¿Notif al residente cuando: asignan, reasignan, vence reserva visitante?
4. ¿Reporte de ocupacion en tiempo real para guarda y admin?
5. ¿Exportacion de historial para asambleas?

### Plan template (post-discovery)

Tablas tentativas: `parking_spaces`, `parking_assignments`, `parking_assignment_history`, `parking_visitor_reservations`, `parking_lottery_runs`, `parking_lottery_results`, `parking_rules`.

Endpoints tentativos: `GET /parking-spaces`, `POST /parking-spaces/:id/assign`, `POST /parking-visitor-reservations`, `POST /parking-lotteries/run`.

Multi-agente sugerido: 4 agentes paralelos:
- Agente A: schema + migraciones + sqlc.
- Agente B: usecases asignacion + sorteo (algoritmo determinista con seed).
- Agente C: usecases reservas visitantes + cron de expiracion.
- Agente D: handlers + OpenAPI + tests de concurrencia (`go test -race`).

DoD especifico:
- [ ] Doble reserva visitante simultanea -> 1 OK + 1 409.
- [ ] Sorteo es reproducible con misma seed.
- [ ] Historial de asignaciones queryable por unidad y por parqueadero.
- [ ] Guarda no ve datos sensibles del residente, solo asignacion.

---

## FASE 9 — Modulo financiero base

**Pre-condiciones**: Fase 8 DoD cumplido. Es el modulo de mas riesgo regulatorio
(Ley 1581 datos personales, normas contables, pasarela de pagos).

### Discovery — preguntas estructuradas

#### Bloque A. Modelo de datos
1. ¿La cuenta contable se asigna a unidad, propietario, ocupante o "cuenta contrato"?
2. ¿Manejas plan unico de cuentas o plan por tenant?
3. ¿Centros de costo? (administracion, fondo imprevistos, mantenimiento, parqueaderos).
4. ¿Que tipo de cargos: cuota administracion, multas, intereses, servicios, alquileres, otros?
5. ¿Periodicidad: mensual, bimestral, configurable por tenant?
6. ¿Manejo de saldo a favor / pagos anticipados?
7. ¿Reverso de pagos: quien autoriza, requiere doble validacion?
8. ¿Conciliacion bancaria automatica (extracto vs sistema) en V1 o post?

#### Bloque B. Reglas de negocio
1. ¿Quien puede pagar? (propietario, inquilino, tercero con link). Ya hay regla "configurable por tenant".
2. ¿Quien recibe el recibo de pago? ¿Email del pagador o del titular?
3. ¿Aplicacion de pagos: FIFO automatico, LIFO, manual, por preferencia del residente?
4. ¿Politica de moras: tasa configurable, gracia X dias, capitalizable?
5. ¿Acuerdos de pago: pueden generar cuotas adicionales?
6. ¿Bloqueo de servicios (reservas, autorizaciones) por mora? ¿A partir de cuanto?
7. ¿El guarda NO ve cartera (regla ya fija). ¿Pero el residente bloqueado debe quedar marcado para porteria?
8. ¿Certificados de paz y salvo: generacion automatica, firmados por admin?

#### Bloque C. Permisos y roles
1. Permisos del `accountant`: ¿total sobre cargos, pagos, reversos?
2. Permisos del `tenant_admin` sobre finanzas: ¿solo aprobacion?
3. ¿Limite de monto sin aprobacion (umbral)? ¿Quien aprueba sobre umbral?
4. ¿Auditor/revisor: lectura total + exportacion?
5. ¿Propietario ve cartera de su unidad. Inquilino: depende del tenant?

#### Bloque D. Concurrencia, transaccionalidad, idempotencia
1. Pagos por pasarela: webhook idempotente (idempotency_key obligatorio).
2. Doble cobro: ¿como prevenir? Constraint unique sobre (cargo_id, gateway_txn_id).
3. Reverso simultaneo del mismo pago: bloqueo optimista.
4. Estado del pago: PENDING -> AUTHORIZED -> CAPTURED -> SETTLED -> (REVERSED).
5. ¿Outbox pattern para emitir eventos a contabilidad?

#### Bloque E. Pasarelas, configuracion, reportes
1. ¿Pasarela: PSE, tarjeta, PayU/Wompi/Mercado Pago/Stripe?
2. ¿Multitenancy de pasarela: cada tenant con su propia cuenta merchant?
3. ¿Reportes obligatorios por ley en Colombia? (informes financieros publicos para asambleas).
4. ¿Periodicidad de cierre? (mensual con cierre soft, anual con cierre hard).
5. ¿Exportacion: PDF de estados de cuenta, Excel de cartera, formato DIAN/contable?

### Plan template

Tablas: `chart_of_accounts`, `cost_centers`, `charges`, `charge_items`, `payment_methods`, `payments`, `payment_allocations`, `payment_reversals`, `accounting_entries`, `accounting_entry_lines`, `payment_gateway_configs`, `payment_webhook_idempotency`.

Multi-agente sugerido: 6 agentes paralelos (es el modulo mas grande):
- A: schema + migraciones (con triggers de inmutabilidad para entries posteados).
- B: cargos (creacion, generacion masiva mensual, intereses).
- C: pagos + aplicacion FIFO/manual + reversos.
- D: integracion pasarela (adapter + webhook + idempotency).
- E: reportes + exportaciones + certificados PDF.
- F: handlers + OpenAPI + tests de race + tests de webhook duplicado.

DoD especifico:
- [ ] Webhook duplicado -> 1 sola aplicacion (verificar via `payment_webhook_idempotency`).
- [ ] Doble pago en pasarela -> rechazado por constraint unique.
- [ ] Reverso requiere doble validacion sobre umbral.
- [ ] Cierre mensual marca entries como inmutables.
- [ ] Certificado de paz y salvo en PDF firmado.
- [ ] Auditor puede leer todo, mutar nada (test explicito).

---

## FASE 10 — Reservas de zonas comunes

**Pre-condiciones**: Fase 9 DoD cumplido (para bloqueo por mora si aplica).

### Discovery — preguntas estructuradas

#### Bloque A. Modelo de datos
1. ¿Que zonas comunes hay? (salon social, BBQ, piscina, gym, cancha, sala estudio).
2. ¿Cada zona tiene capacidad maxima, horario, costo por uso, deposito de garantia?
3. ¿Reglas distintas por zona? (BBQ requiere deposito, piscina solo horario).
4. ¿Aforo simultaneo: 1 reserva exclusiva o varias (compartido)?

#### Bloque B. Reglas de negocio
1. ¿Quien reserva: propietario y/o inquilino? Configurable por tenant.
2. ¿Anticipacion minima/maxima? ¿Reserva mismo dia permitida?
3. ¿Cupos por unidad/mes? ¿Penalizaciones por no-show?
4. ¿Aprobacion previa requerida o reserva automatica?
5. ¿Bloqueo si la unidad tiene mora (>X dias)?
6. ¿Cancelacion: hasta cuando, con que penalizacion?

#### Bloque C. Permisos y roles
1. ¿Admin puede bloquear fechas (mantenimiento)?
2. ¿Guarda valida ingreso a la zona reservada (ver QR de reserva)?
3. ¿Consejo aprueba reservas si configurado asi?

#### Bloque D. Concurrencia
1. Doble reserva del mismo slot: bloqueo optimista + constraint unique sobre (zona, fecha, slot).
2. Idempotencia en creacion (key cliente).

#### Bloque E. Config + notif + reportes
1. ¿Tarifas y depositos configurables?
2. ¿Notif al reservar, recordatorio dia anterior, post-uso para devolver deposito?
3. ¿Reporte de ocupacion por zona / mes?

### Plan template
Tablas: `common_areas`, `common_area_rules`, `reservations`, `reservation_payments`, `reservation_blackouts`.

Multi-agente: 3 agentes (schema, usecases, handlers+notif).

DoD especifico:
- [ ] Doble reserva slot -> 1 OK + 1 409.
- [ ] Bloqueo por mora funciona si tenant lo activa.
- [ ] QR de reserva validable por guarda.

---

## FASE 11 — Asambleas, votaciones y actas

**Pre-condiciones**: Fase 10 DoD cumplido. Identidad solida + auditoria fuerte.

> Aplica Ley 527 de 1999 (mensajes de datos / firma electronica). Discovery
> debe capturar requisitos legales especificos.

### Discovery — preguntas estructuradas

#### Bloque A. Modelo de datos
1. ¿Tipos de asamblea: ordinaria, extraordinaria, virtual, mixta?
2. ¿Padron: una unidad = un voto, o coeficiente de copropiedad?
3. ¿Apoderados/poderes? ¿Limite por persona?
4. ¿Quorum minimo configurable por tipo de decision?
5. ¿Mayorias: simple, calificada, especial — configurables?

#### Bloque B. Reglas de negocio
1. ¿Convocatoria: anticipacion legal minima, canales obligatorios?
2. ¿Pueden votar inquilinos? (en ph normalmente NO, solo propietarios).
3. ¿Voto secreto vs nominal por tipo de decision?
4. ¿Modificacion de voto antes del cierre? ¿Trazabilidad?
5. ¿Acta: generada automaticamente, validada por presidente y secretario?

#### Bloque C. Permisos y roles
1. ¿`board_member` puede convocar? ¿Solo `tenant_admin`?
2. ¿Auditor/revisor presencia con voz sin voto?
3. ¿Quien firma el acta (Ley 527)?

#### Bloque D. Concurrencia, integridad, evidencia digital
1. ¿Voto idempotente (mismo voto 2 veces no cuenta doble)?
2. ¿Hash chain de votos para integridad? (cada voto referencia hash del anterior).
3. ¿Conservacion de evidencia: IP, user-agent, timestamp con NTP, hash del voto).
4. ¿Acta firmada digitalmente (PKI o firma simple con trazabilidad)?

#### Bloque E. Notificaciones, reportes, exportacion
1. ¿Convocatoria por: email, push, WhatsApp (si fase 15 lista), publicacion en cartelera?
2. ¿Resultados: en tiempo real durante votacion o solo al cierre?
3. ¿Acta en PDF firmable, archivable indefinidamente?
4. ¿Exportacion para administrador municipal / camara?

### Plan template
Tablas: `assemblies`, `assembly_calls`, `assembly_attendances`, `assembly_proxies`, `assembly_motions`, `votes`, `vote_evidence`, `acts`, `act_signatures`.

Multi-agente: 5 agentes (schema, padron+convocatoria, votacion+integridad, acta+PDF+firma, handlers+notif).

DoD especifico:
- [ ] Quorum se calcula con coeficientes correctamente.
- [ ] Voto duplicado -> rechazado, hash chain mantiene integridad.
- [ ] Acta PDF generada incluye anexos de evidencia.
- [ ] Inquilinos NO pueden votar (test explicito).
- [ ] Apoderados no exceden limite configurado.

---

## FASE 12 — Incidentes y novedades de seguridad

**Pre-condiciones**: Fase 11 DoD cumplido (o Fase 7 si se prioriza esta antes).

### Discovery — preguntas

#### Bloque A. Modelo de datos
1. Tipos: ruido, fuga, dano, robo intento, accidente, mascota, otros.
2. Severidad: bajo, medio, alto, critico.
3. ¿Adjuntar fotos/videos? ¿Numero max?
4. ¿Geolocalizacion (zona del conjunto)?

#### Bloque B. Reglas de negocio
1. ¿Quien reporta: residente, guarda, admin?
2. ¿Workflow: reportado -> asignado -> en proceso -> resuelto -> cerrado?
3. ¿SLAs por severidad?
4. ¿Escalamiento automatico si no se atiende en X horas?

#### Bloque C. Permisos
1. ¿Quien asigna? ¿Quien cierra?
2. ¿Visibilidad: el residente que reporto + admin + guarda asignado?

#### Bloque D. Concurrencia
1. ¿Estado idempotente en transiciones (no permitir cierre 2 veces)?

#### Bloque E. Config + notif + reportes
1. ¿Notif por escalamiento?
2. ¿Reporte mensual de incidentes para consejo?

### Plan template
Tablas: `incidents`, `incident_attachments`, `incident_status_history`, `incident_assignments`.

Multi-agente: 2 agentes (schema+usecases, handlers+notif).

DoD especifico:
- [ ] SLAs disparan escalamiento.
- [ ] Cierre requiere resolucion textual.

---

## FASE 13 — Multas y sanciones

**Pre-condiciones**: Fase 9 DoD cumplido (financiero) + Fase 11 (asambleas para sancionar formalmente).

### Discovery — preguntas

#### Bloque A. Modelo de datos
1. Tipos de sancion: amonestacion, multa monetaria, suspension de servicios.
2. ¿Catalogo de infracciones configurable por tenant (con monto base)?
3. ¿Reincidencia: aumenta monto?
4. ¿Vinculacion con incidentes (Fase 12) o independiente?

#### Bloque B. Reglas de negocio
1. ¿Quien impone? (admin, consejo, asamblea segun gravedad).
2. ¿Notificacion legal con tiempo para descargos?
3. ¿Apelacion: workflow propio?
4. ¿Conversion a cargo en cartera (Fase 9)?

#### Bloque C. Permisos
1. ¿`tenant_admin` impone hasta X monto, sobre eso pasa a consejo?
2. ¿Auditor lee todo?

#### Bloque D. Concurrencia
1. ¿Estado idempotente en imposicion?

#### Bloque E. Config + reportes
1. ¿Plantillas de notificacion legal por tenant?
2. ¿Reporte mensual de sanciones para asamblea?

### Plan template
Tablas: `penalty_catalog`, `penalties`, `penalty_appeals`, `penalty_status_history`.

Multi-agente: 3 agentes (schema, workflow + apelaciones, integracion con cartera).

DoD especifico:
- [ ] Multa monetaria genera cargo en cartera atomicamente.
- [ ] Apelacion suspende cobro hasta resolucion.
- [ ] Reincidencia incrementa monto segun catalogo.

---

## FASE 14 — PQRS

**Pre-condiciones**: Fase 13 DoD cumplido.

### Discovery — preguntas

#### Bloque A. Modelo de datos
1. Tipos: peticion, queja, reclamo, sugerencia, solicitud documental.
2. ¿Anonimo permitido?
3. ¿Categorizacion (administrativo, tecnico, financiero, convivencia)?

#### Bloque B. Reglas de negocio
1. ¿SLAs legales por tipo (15 dias habiles para reclamos en colombia).
2. ¿Workflow: radicado -> en estudio -> respondido -> cerrado/escalado?
3. ¿Calificacion del residente al cerrar?

#### Bloque C. Permisos
1. ¿Asignacion automatica por categoria a rol responsable?
2. ¿Visibilidad: solo el solicitante + responsable + admin?

#### Bloque D. Concurrencia
1. ¿Numero de radicado unico, atomico?

#### Bloque E. Config + notif + reportes
1. ¿Plantillas de respuesta?
2. ¿Reporte de SLAs y satisfaccion para consejo?

### Plan template
Tablas: `pqrs_tickets`, `pqrs_categories`, `pqrs_responses`, `pqrs_sla_alerts`.

Multi-agente: 2 agentes (schema+usecases, handlers+notif+SLAs).

DoD especifico:
- [ ] SLA dispara alerta antes de incumplimiento.
- [ ] Numero de radicado unico secuencial por tenant/anio.
- [ ] Anonimo no expone identidad ni en logs.

---

## FASE 15 — Notificaciones multicanal (WhatsApp, SMS, Email, Push)

**Pre-condiciones**: idealmente despues de Fase 7 (es transversal). Puede
adelantarse si el costo operativo de no tener WhatsApp es alto.

### Discovery — preguntas

#### Bloque A. Modelo de datos / canales
1. Canales en V1: email + push (movil) + WhatsApp + SMS?
2. ¿Cada residente elige sus canales preferidos por tipo de evento?
3. ¿Plantillas multilingue? (no esperado en V1, pero diseno preparado).

#### Bloque B. Reglas de negocio
1. ¿Que evento dispara que canal? (ej. paquete recibido -> push + WhatsApp).
2. ¿Anuncios criticos: forzar todos los canales?
3. ¿Horario silencioso (no enviar 22:00-7:00)?
4. ¿Opt-in legal para WhatsApp/SMS (requerido en colombia)?

#### Bloque C. Permisos
1. ¿Quien puede crear/editar plantillas? (admin).
2. ¿Quien dispara envio masivo? (admin con MFA + log).

#### Bloque D. Concurrencia, idempotencia, fallos
1. ¿Outbox pattern obligatorio?
2. ¿Reintentos con backoff exponencial?
3. ¿Idempotencia: si la cola reentrega el evento, no enviar de nuevo?

#### Bloque E. Proveedores, costos, reportes
1. ¿Provider WhatsApp: Twilio, Meta Business API, otros?
2. ¿SMS: Twilio, AWS SNS, local?
3. ¿Email: AWS SES, Postmark, SendGrid?
4. ¿Reporte de entregabilidad por canal?
5. ¿Costo por tenant: facturacion del SaaS sobre consumo?

### Plan template
Tablas: `notification_templates`, `notification_preferences`, `notification_outbox`, `notification_deliveries`, `notification_provider_configs`.

Multi-agente: 4 agentes (schema, outbox+worker, adapters por canal, handlers+templates).

DoD especifico:
- [ ] Outbox + worker entregan al menos una vez (at-least-once).
- [ ] Idempotencia evita reenvios duplicados.
- [ ] Horario silencioso respetado.
- [ ] Opt-in registrado y consultable.
- [ ] Reintentos con backoff exponencial.

---

## FASE 16 — Identidad cross-tenant + provisioning + selector

**Pre-condiciones**: Fase 7 (hardening MVP) cumplido. ADR 0007 mergeado.

### Por que esta fuera del orden 8-15

Las fases 8-15 son **modulos de negocio**. La Fase 16 es **re-arquitectura de
identidad** que afecta a TODOS los modulos. Se introduce como ruptura
controlada cuando hay evidencia de que el modelo "una identidad por tenant" del
ADR 0002 no escala a empresas administradoras y personal que rota.

### Brief autocontenido

> Ejecuta la Fase 16 segun la spec frozen en
> [docs/specs/fase-16-cross-tenant-identity-spec.md](docs/specs/fase-16-cross-tenant-identity-spec.md)
> y el ADR base en [docs/adr/0007-cross-tenant-identity.md](docs/adr/0007-cross-tenant-identity.md).
>
> 11 pasos numerados 16.0 a 16.10 con DoD por paso. Cada paso es autocontenido
> y puede ejecutarse en una sesion aislada. La spec referencia la memoria
> persistente del proyecto en `~/.claude/projects/.../memory/` (4 archivos
> de decisiones del usuario).
>
> Estrategia: 5 agentes paralelos (A-E) trabajando en archivos disjuntos:
> - A: migraciones DB central (002-005)
> - B: modulo `platform_identity` (login, switch-tenant, me, push-devices)
> - C: modulo `superadmin` + `provisioning` (CreateTenant transaccional)
> - D: middleware tenant_resolver reescrito + migraciones tenant 019-020 (FK realign)
> - E: frontend web (`/select-tenant`, `<TenantSwitcher>`, login 3-campos)
>
> Mobile Flutter (paso 16.8) va despues como agente F secuencial.

### Ruptura intencional

- ADR 0002 queda **Superseded by 0007**.
- Se rompe la convencion de subdominio por tenant. Pasa a URL raiz unica con
  `current_tenant` en JWT.
- El `cmd/seed-demo` actual se reescribe; `demo` se borra y resiembra.
- Las tablas tenant que tenian FK a `users(id)` migran a `tenant_user_links(id)`.

### DoD especifico

Ver seccion 15 de la spec. Resumen:
- ADR 0002 marcado Superseded.
- Migraciones central 002-005 reversibles.
- Migraciones tenant 019-020 reversibles.
- `cmd/seed-demo` reescrito.
- E2E Playwright login + selector + switcher verde.
- OpenAPI 3.0 actualizado.
- Memoria persistente referenciada desde MEMORY.md (ya hecho).

---

## Cierre

Al completar Fase 16 (y luego Fase 15 si se difirio), el producto es
funcionalmente completo. Pasos siguientes:

1. **Hardening cross-cutting** post-cada-fase: revisar performance, seguridad,
   observabilidad. Repetir el bloque de Fase 7 con foco en lo nuevo.
2. **Modulos no listados** (intercom virtual, marketplace de servicios,
   integraciones fisicas con talanqueras/biometria, BI/analitica): seguir
   el mismo patron Discovery -> Spec frozen -> /fase con multi-agentes.
3. **Refactor controlado**: si una fase rompe abstracciones, NO hackear.
   Crear ADR explicando, ajustar, mantener Clean Architecture.

### Comandos por fase (resumen)

| Tipo de fase | Comando 1 (descubrir) | Comando 2 (ejecutar) | Comando 3 (verificar) |
|--------------|----------------------|----------------------|-----------------------|
| MVP (0-7) | (no necesario) | `/fase <N>` | `/verificar-fase <N>` |
| POST-MVP (8-15) | `/descubrir <N>` | `/fase <N>` | `/verificar-fase <N>` |

Cada modulo POST-MVP sigue el patron: Discovery -> Spec frozen -> ADR (si la
decision es estructural) -> Tablas + migraciones -> Modulo Clean Architecture
-> Tests -> OpenAPI -> Hardening.
