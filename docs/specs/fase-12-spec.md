# Fase 12 — Spec — Incidentes y novedades de seguridad

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

---

## 1. Resumen ejecutivo

Modulo para registrar, asignar, escalar y resolver incidentes y novedades
operativas dentro del conjunto residencial (ruido, fugas, danos, robo,
accidentes, mascotas, otros). Permite a residentes, guardas y admin
reportar; a admin asignar a un guarda o equipo; y mantiene historial
auditable del workflow. Incluye SLAs por severidad y escalamiento
automatico. La capa de notificaciones se delega a Fase 15 — esta fase
solo emite eventos a un outbox modulo-local (ADR-0005, patron ya
establecido en `packages`).

## 2. Decisiones tomadas

- **ASSUMPTION**: Tipos de incidente catalogados como enum cerrado en CHECK
  constraint: `noise`, `leak`, `damage`, `theft_attempt`, `accident`,
  `pet_issue`, `other`. Editar el catalogo requiere migracion (NO tabla
  separada en V1 para reducir complejidad).
- **ASSUMPTION**: Severidades fijas: `low`, `medium`, `high`, `critical`.
- **ASSUMPTION**: Adjuntos permitidos hasta 10 por incidente (foto/video),
  validacion en handler. Se almacenan URLs (storage S3-compatible
  externo); el modulo solo persiste la URL y metadatos.
- **ASSUMPTION**: Geolocalizacion modelada como referencia opcional a
  `residential_structures(id)` (zona del conjunto) + texto libre
  `location_detail`. NO se usa lat/long en V1.
- **ASSUMPTION**: Quien reporta puede ser cualquier usuario autenticado
  con permiso `incidents.report`: residente, guarda, admin.
- **ASSUMPTION**: Workflow fijo: `reported -> assigned -> in_progress ->
  resolved -> closed`. Tambien estado terminal `cancelled` para reportes
  invalidos. Transiciones validas se aplican en policy del dominio.
- **ASSUMPTION**: SLAs por severidad (configurables por tenant en V2,
  hardcoded en V1):
  - `critical`: 1 hora a `assigned`, 4 horas a `resolved`.
  - `high`: 4 horas / 24 horas.
  - `medium`: 24 horas / 72 horas.
  - `low`: 72 horas / 168 horas.
- **ASSUMPTION**: Escalamiento automatico: si SLA de `assigned` se
  vence, el incidente se marca `escalated=true` y se emite evento
  `incident.escalated`. Job batch corre cada 15 minutos.
- **ASSUMPTION**: Asignacion la realiza un `tenant_admin` o usuario con
  permiso `incidents.assign`. Cierre lo realiza el asignado o admin.
- **ASSUMPTION**: Visibilidad: el reportante, el asignado, admins y
  auditores. Otros residentes NO ven incidentes ajenos.
- **ASSUMPTION**: Idempotencia en transiciones via columna `version`
  (concurrencia optimista) + verificacion de estado anterior valido.
- **ASSUMPTION**: Cierre exige `resolution_notes` no vacio (CHECK).

## 3. Supuestos adoptados (no bloqueantes)

- Adjuntos: solo URL y mime_type; el upload se hace por presigned URL
  fuera del modulo.
- Notificaciones se entregan via `notification_outbox` (Fase 15) cuando
  exista; mientras tanto, `incident_outbox_events` modulo-local.
- El reporte mensual se construye via query de lectura sobre
  `incidents` + `incident_status_history`; no hay tabla agregada.
- Anonimato en V1 NO soportado para incidentes (se requiere identidad
  para responsabilidad y seguimiento).

## 4. Open Questions

- Plazo legal para conservar evidencia de incidentes (varia por
  reglamento del conjunto; default 5 anios). Confirmar con consejo.
- Provider final de almacenamiento de adjuntos (S3 / GCS / Cloudflare
  R2): decision externa al modulo.
- Si se requiere notificacion a autoridades publicas (ej. policia) en
  incidentes `theft_attempt`/`critical`: fuera del alcance backend en
  V1, pendiente decision de producto.
- Configuracion de SLAs por tenant: V2.

## 5. Modelo de datos propuesto

- `incidents`: incidente principal. Campos: `id`, `incident_type`,
  `severity`, `title`, `description`, `reported_by_user_id`,
  `reported_at`, `structure_id` (NULL), `location_detail`,
  `assigned_to_user_id` (NULL), `assigned_at`, `resolved_at`,
  `closed_at`, `resolution_notes`, `escalated`, `sla_due_at`,
  `status`, campos estandar + `version`.
- `incident_attachments`: adjuntos (url, mime, size_bytes, FK
  incident_id, sin `version`).
- `incident_status_history`: append-only audit del workflow. Campos:
  `id`, `incident_id`, `from_status`, `to_status`,
  `transitioned_by_user_id`, `transitioned_at`, `notes`.
- `incident_assignments`: historial de asignaciones (multiples). Campos:
  `id`, `incident_id`, `assigned_to_user_id`, `assigned_by_user_id`,
  `assigned_at`, `unassigned_at` (NULL), `status`.
- `incident_outbox_events`: outbox modulo-local (ADR-0005). Eventos:
  `incident.reported`, `incident.assigned`, `incident.escalated`,
  `incident.resolved`, `incident.closed`.

## 6. Endpoints propuestos

- `POST /api/v1/incidents` — reportar.
- `GET /api/v1/incidents` — listar (filtros: status, severity, mine).
- `GET /api/v1/incidents/{id}` — detalle.
- `POST /api/v1/incidents/{id}/assign` — admin asigna.
- `POST /api/v1/incidents/{id}/start` — asignado marca `in_progress`.
- `POST /api/v1/incidents/{id}/resolve` — registrar resolucion.
- `POST /api/v1/incidents/{id}/close` — cerrar.
- `POST /api/v1/incidents/{id}/cancel` — cancelar (admin).
- `POST /api/v1/incidents/{id}/attachments` — anadir adjunto.
- `GET /api/v1/incidents/{id}/history` — auditoria de transiciones.
- `GET /api/v1/reports/incidents/monthly?year=&month=` — reporte.

Errores RFC 7807. Concurrencia optimista en transiciones.

## 7. Permisos nuevos (namespaces)

- `incidents.report` — crear reporte.
- `incidents.read` — listar/leer (con filtro de visibilidad).
- `incidents.read_all` — admin/auditor lee todo.
- `incidents.assign` — asignar.
- `incidents.transition` — start/resolve/close (asignado o admin).
- `incidents.cancel` — cancelar.
- `incidents.report_monthly` — leer reporte mensual.

## 8. Casos extremos

- Reporte duplicado en pocos segundos por mismo usuario y misma unit:
  permitir (no bloquear), el admin discrimina.
- Cerrar un incidente ya cerrado: 409 Conflict via `version` o estado.
- Asignar a usuario eliminado (soft-deleted): 422 con error de dominio.
- Adjunto > limite: 413.
- SLA ya vencido al crearse (severidad bajo + creado en horario fuera):
  el job de escalamiento no escala dos veces (idempotente via flag
  `escalated`).
- Reportante intenta cerrar (no es asignado/admin): 403.

## 9. Operaciones transaccionales / idempotentes

- Toda transicion de estado: misma TX inserta fila en
  `incident_status_history` + UPDATE `incidents` con `version` +
  INSERT en `incident_outbox_events`.
- Asignacion: cierra fila previa de `incident_assignments` (set
  `unassigned_at`) y abre nueva, en TX.
- Job de escalamiento: `UPDATE ... WHERE escalated=false AND
  sla_due_at < now()` + insert outbox; idempotente.

## 10. Configuracion por tenant

- `incidents.sla_minutes_by_severity` (JSON): override de SLAs por
  severidad. (V2; en V1 hardcoded.)
- `incidents.attachments_max_per_incident`: default 10.
- `incidents.allowed_mime_types`: default `image/jpeg`, `image/png`,
  `video/mp4`.

## 11. Notificaciones / eventos

Eventos emitidos al outbox (consumidos por Fase 15):

- `incident.reported` -> admin + guardas.
- `incident.assigned` -> asignado + reportante.
- `incident.escalated` -> admin.
- `incident.resolved` -> reportante.
- `incident.closed` -> reportante.

## 12. Reportes / metricas

- Reporte mensual: total por tipo, severidad, tiempo promedio a
  resolucion, tasa de cumplimiento de SLA.
- Metricas OTel: `incidents_reported_total`, `incidents_escalated_total`,
  `incident_resolution_seconds` (histogram).

## 13. Riesgos y mitigaciones

- **Riesgo**: incidente no asignado se pierde. **Mitigacion**: SLA +
  escalamiento.
- **Riesgo**: spam de reportes. **Mitigacion**: rate limiting por
  usuario en handler (50/dia).
- **Riesgo**: evidencia se elimina. **Mitigacion**: soft delete +
  `incident_status_history` append-only.

## 14. Multi-agente sugerido

2 agentes en paralelo:
- Agente 1: schema (sqlc queries) + entidades + usecases.
- Agente 2: handlers HTTP + adapters outbox + job de escalamiento.

## 15. DoD adicional

- [ ] SLAs disparan escalamiento (test de integracion con tiempo
  controlado).
- [ ] Cierre requiere `resolution_notes` no vacio (test 422).
- [ ] Concurrencia optimista verificada (test de doble cierre).
- [ ] Reporte mensual funcional (query SQL + endpoint).
- [ ] Eventos de outbox emitidos en cada transicion.
