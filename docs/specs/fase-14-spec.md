# Fase 14 — Spec — PQRS

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

---

## 1. Resumen ejecutivo

Modulo de Peticiones, Quejas, Reclamos y Sugerencias (PQRS). Permite a
residentes radicar tickets categorizados con numero de radicado unico
secuencial por anio. Workflow: radicado -> en estudio -> respondido ->
cerrado/escalado. SLA por tipo de PQR (configurable por tenant; el
plazo legal exacto queda en Open Questions). Asignacion automatica
por categoria. Calificacion del solicitante al cerrar. Genera alertas
internas previas al vencimiento de SLA.

## 2. Decisiones tomadas

- **ASSUMPTION**: Tipos (CHECK enum): `peticion`, `queja`, `reclamo`,
  `sugerencia`, `solicitud_documental`.
- **ASSUMPTION**: Anonimo NO permitido en V1 (se requiere identidad
  para responder y para auditoria). Si en V2 se admite, se anade flag
  `is_anonymous` y se omite identidad de la vista publica del ticket.
- **ASSUMPTION**: Categorizacion via tabla `pqrs_categories`
  (configurable por tenant): `code`, `name`,
  `default_assignee_role_id` (FK roles).
- **ASSUMPTION**: SLA por tipo (default sugerido, configurable):
  - `peticion`: 15 dias habiles.
  - `queja`: 15 dias habiles.
  - `reclamo`: 15 dias habiles.
  - `sugerencia`: 30 dias habiles.
  - `solicitud_documental`: 10 dias habiles.
  **Estos plazos NO se hardcodean** — se almacenan en
  `tenant_settings` y son ajustables. Ver Open Questions.
- **ASSUMPTION**: Workflow (CHECK): `radicado -> en_estudio ->
  respondido -> cerrado` + estados terminales `escalado`, `cancelado`.
- **ASSUMPTION**: Calificacion (1-5) se solicita al solicitante al
  pasar a `respondido`; se almacena en el cierre. Es opcional.
- **ASSUMPTION**: Asignacion automatica via `default_assignee_role_id`
  de la categoria. Si hay multiples usuarios con ese rol, asigna por
  round-robin simple (orden alfabetico de user_id, alternancia con
  contador en config). Reasignacion manual permitida.
- **ASSUMPTION**: Visibilidad: solicitante + responsable asignado +
  `tenant_admin` + auditor.
- **ASSUMPTION**: Numero de radicado unico atomico:
  `serial_number` INT secuencial dentro del par
  `(ticket_year, deleted_at IS NULL)`. UNIQUE constraint
  `(ticket_year, serial_number) WHERE deleted_at IS NULL`. Generacion
  via `SELECT COALESCE(MAX(serial_number),0)+1 FROM pqrs_tickets WHERE
  ticket_year=$1 FOR UPDATE` dentro de TX (advisory lock por anio
  como alternativa preferida).
- **ASSUMPTION**: Formato visible: `PQRS-{YYYY}-{NNNNNN}` calculado en
  presentation (no almacenado).
- **ASSUMPTION**: Alertas SLA: job batch cada 15 minutos. Inserta en
  `pqrs_sla_alerts` cuando faltan 24 horas o el plazo se vencio.
- **ASSUMPTION**: Plantillas de respuesta: tabla `pqrs_response_templates`
  (no en esta migracion - V2). En V1, respuesta libre.
- **ASSUMPTION**: Dia habil = lunes a viernes excluyendo festivos
  colombianos. Tabla de festivos NO incluida en V1; se calculara
  via libreria/funcion auxiliar (`pqrs_business_days_add`).
  **NOTA**: la tabla de festivos es Open Question.

## 3. Supuestos adoptados (no bloqueantes)

- Una sola respuesta oficial por ticket (en V1). Multiples
  comunicaciones internas via `pqrs_responses` (tipo `internal_note`
  vs `official_response`).
- Adjuntos: hasta 5 por respuesta y 5 por ticket.
- Escalamiento manual (admin marca `escalado` con motivo). Auto-
  escalado por SLA NO en V1.

## 4. Open Questions

- **Plazos legales exactos por tipo**: Ley 1755 de 2015 / Decreto 1166
  de 2016 (Colombia) regulan PQRS para entidades publicas; para
  propiedad horizontal regida por Ley 675 de 2001 los plazos son los
  del reglamento del conjunto. **No hardcodear**. Default 15 dias
  habiles. Confirmar con asesoria juridica del tenant.
- Tabla de festivos colombianos: requiere fuente oficial actualizada.
- Si el solicitante no califica, ¿se le envia recordatorio? V2.
- Plantillas oficiales de respuesta firmadas digitalmente: V2.
- Anonimato real con anonimizacion de logs: V2 (impacta toda la
  observabilidad, incluso `request.user_id`).

## 5. Modelo de datos propuesto

- `pqrs_categories`: `id`, `code` (UNIQUE WHERE deleted_at IS NULL),
  `name`, `default_assignee_role_id` (FK roles), campos estandar.
- `pqrs_tickets`: `id`, `ticket_year` INT NOT NULL, `serial_number`
  INT NOT NULL, `pqr_type`, `category_id`, `subject`, `body`,
  `requester_user_id` (FK users), `assigned_to_user_id` NULL,
  `assigned_at`, `responded_at`, `closed_at`, `escalated_at`,
  `cancelled_at`, `sla_due_at`, `requester_rating` (INT NULL CHECK
  1..5), `requester_feedback` TEXT NULL, `is_anonymous` BOOLEAN
  DEFAULT false, `status`, campos estandar + `version`. UNIQUE
  `(ticket_year, serial_number) WHERE deleted_at IS NULL`.
- `pqrs_responses`: `id`, `ticket_id`, `response_type` (`internal_note`
  / `official_response`), `body`, `responded_by_user_id`,
  `responded_at`, campos estandar.
- `pqrs_attachments`: `id`, `ticket_id` NULL, `response_id` NULL,
  `url`, `mime_type`, `size_bytes`, campos estandar (CHECK que
  exactamente uno de `ticket_id`/`response_id` no sea NULL).
- `pqrs_status_history`: append-only audit.
- `pqrs_sla_alerts`: `id`, `ticket_id`, `alert_type` (`24h_warning`,
  `breached`), `alerted_at`, UNIQUE `(ticket_id, alert_type)`.
- `pqrs_outbox_events`: outbox modulo-local.

## 6. Endpoints propuestos

- `POST /api/v1/pqrs` — radicar.
- `GET /api/v1/pqrs` — listar (filtros: status, type, mine).
- `GET /api/v1/pqrs/{id}` — detalle.
- `POST /api/v1/pqrs/{id}/assign` — admin reasigna.
- `POST /api/v1/pqrs/{id}/start-study` — pasar a `en_estudio`.
- `POST /api/v1/pqrs/{id}/respond` — respuesta oficial.
- `POST /api/v1/pqrs/{id}/notes` — notas internas.
- `POST /api/v1/pqrs/{id}/close` — solicitante o admin cierran +
  calificacion opcional.
- `POST /api/v1/pqrs/{id}/escalate` — admin escala.
- `POST /api/v1/pqrs/{id}/cancel` — admin cancela.
- `GET /api/v1/pqrs/{id}/history`.
- `GET /api/v1/pqrs/categories` / `POST` / `PATCH`.
- `GET /api/v1/reports/pqrs/sla-summary`.

## 7. Permisos nuevos (namespaces)

- `pqrs.create` — radicar (residente).
- `pqrs.read_mine` — solicitante.
- `pqrs.read_all` — admin/auditor.
- `pqrs.assign` — admin.
- `pqrs.respond` — asignado/admin.
- `pqrs.note` — agregar notas internas.
- `pqrs.close_admin` / `pqrs.close_self`.
- `pqrs.escalate` / `pqrs.cancel`.
- `pqrs.categories.write`.
- `pqrs.report` — reportes.

## 8. Casos extremos

- Generacion concurrente de `serial_number`: usar `pg_advisory_xact_lock`
  con hash de `ticket_year` para serializar.
- Reset de serial al cambiar de anio: `ticket_year = EXTRACT(YEAR
  FROM now())` al insertar.
- Solicitante intenta cerrar antes de respuesta: 422.
- Calificacion fuera de rango: 422.
- Adjunto sin `ticket_id` ni `response_id`: CHECK constraint bloquea.
- Categoria sin `default_assignee_role_id`: queda sin asignar (status
  `radicado`, `assigned_to_user_id NULL`); admin debe asignar.
- Anonimo en V1 (false siempre): si llega true, 422.

## 9. Operaciones transaccionales / idempotentes

- Radicar: TX con advisory lock por anio -> calcular serial -> insert
  ticket -> insert history -> outbox `pqrs.created`.
- Responder: TX inserta `pqrs_responses` (`official_response`) +
  transiciona ticket a `respondido` + history + outbox.
- Cierre: TX cambia status + inserta history + (si calificacion)
  setea `requester_rating`.

## 10. Configuracion por tenant

- `pqrs.sla_business_days_by_type` (JSON).
- `pqrs.business_holidays` (JSON array de fechas).
- `pqrs.attachments_max`: 5.
- `pqrs.allow_anonymous`: false (V1).
- `pqrs.assign_round_robin_counter` (interno).

## 11. Notificaciones / eventos

Eventos al outbox -> Fase 15:
- `pqrs.created` -> admin + asignado.
- `pqrs.assigned` -> asignado.
- `pqrs.responded` -> solicitante.
- `pqrs.sla_warning_24h` -> asignado + admin.
- `pqrs.sla_breached` -> admin.
- `pqrs.closed` -> solicitante.

## 12. Reportes / metricas

- Reporte SLA mensual: total radicados, respondidos en plazo, tasa
  cumplimiento, calificacion promedio, top categorias.
- Metricas OTel: `pqrs_created_total`, `pqrs_sla_breach_total`,
  `pqrs_response_seconds` (histogram).

## 13. Riesgos y mitigaciones

- **Riesgo**: race condition en serial. **Mitigacion**: advisory lock.
- **Riesgo**: SLA mal calculado por festivos. **Mitigacion**:
  funcion `pqrs_business_days_add` con tabla configurable.
- **Riesgo**: anonimato filtra identidad. **Mitigacion**: V1
  desactivado; V2 con audit separado.

## 14. Multi-agente sugerido

2 agentes en paralelo:
- Agente A: schema + entidades + categorias + radicacion (serial).
- Agente B: handlers + workflow + SLA job + reportes.

## 15. DoD adicional

- [ ] Numero de radicado unico secuencial por tenant/anio (test
  concurrente con goroutines).
- [ ] SLA dispara alerta `24h_warning` antes de incumplimiento.
- [ ] Anonimo en V1 rechaza con 422.
- [ ] Calificacion fuera de 1-5 rechaza con 422.
- [ ] Asignacion automatica usa `default_assignee_role_id`.
