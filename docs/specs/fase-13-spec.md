# Fase 13 — Spec — Multas y sanciones

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

---

## 1. Resumen ejecutivo

Modulo de multas y sanciones a residentes/propietarios. Soporta
amonestaciones, multas monetarias y suspension de servicios. Catalogo
de infracciones configurable por tenant con monto base y politica de
reincidencia. Workflow legal: borrador -> notificado -> en apelacion ->
confirmado -> liquidado. Multas monetarias generan cargo en cartera
(integracion con Fase 9 financiero) atomicamente cuando son
confirmadas. Apelaciones suspenden el cobro hasta resolucion.

## 2. Decisiones tomadas

- **ASSUMPTION**: Tipos de sancion (CHECK enum):
  `warning`, `monetary`, `service_suspension`.
- **ASSUMPTION**: Catalogo de infracciones (`penalty_catalog`)
  configurable por tenant: `code`, `name`, `description`,
  `default_sanction_type`, `base_amount`, `recurrence_multiplier`
  (factor por reincidencia, default 1.5).
- **ASSUMPTION**: Reincidencia se calcula contando multas
  `confirmed`/`settled` para mismo usuario y mismo `catalog_id` en
  ventana de 365 dias previos. El monto resultante = `base_amount *
  recurrence_multiplier^(prior_count)`, capado en V1 a 5x.
- **ASSUMPTION**: Vinculacion opcional con `incidents` (Fase 12) via
  `source_incident_id` NULL.
- **ASSUMPTION**: Imposicion: `tenant_admin` puede crear hasta umbral
  (configurable, default 5 SMMLV). Sobre umbral, requiere flag
  `requires_council_approval=true` y aprobacion via endpoint adicional
  por usuario con permiso `penalties.approve_council`.
- **ASSUMPTION**: Notificacion legal con plazo para descargos: default
  10 dias habiles desde `notified_at`. Configurable por tenant.
  **NO se hardcodea regulacion legal** (ver Open Questions).
- **ASSUMPTION**: Apelacion: workflow propio (`penalty_appeals`),
  estados `submitted`, `under_review`, `accepted`, `rejected`. Una sola
  apelacion activa por sancion (UNIQUE WHERE deleted_at IS NULL AND
  status IN ('submitted','under_review')).
- **ASSUMPTION**: Cuando una sancion `monetary` se mueve a `confirmed`
  y NO esta en apelacion activa, se inserta cargo en cartera (Fase 9)
  en MISMA TX via outbox o llamada directa al usecase de cartera.
- **ASSUMPTION**: Workflow de la sancion (CHECK):
  `drafted -> notified -> in_appeal -> confirmed -> settled` +
  terminal `dismissed` (apelacion aceptada). El estado `cancelled` se
  permite solo desde `drafted`.
- **ASSUMPTION**: Apelacion suspende cobro: si `monetary`, el cargo en
  cartera se marca `on_hold` (campo en cartera, fuera del scope de
  esta migracion) hasta resolucion.
- **ASSUMPTION**: `tenant_admin` impone hasta umbral; `auditor` lee
  todo, no edita.
- **ASSUMPTION**: Idempotencia en imposicion via `version` +
  `idempotency_key` opcional en request.
- **ASSUMPTION**: Plantillas de notificacion legal: una plantilla por
  tenant, en JSON con placeholders. Almacenadas en `tenant_settings`
  (modulo 003), NO en este modulo.

## 3. Supuestos adoptados (no bloqueantes)

- Las sanciones `service_suspension` solo registran intencion; la
  ejecucion (bloquear acceso a zonas) la hace el modulo de control de
  accesos (007), via evento outbox.
- Reportes mensuales se construyen via query de lectura.
- La integracion con cartera de Fase 9 se hace via outbox (
  `penalty_outbox_events`) para evitar acoplamiento sincrono.

## 4. Open Questions

- Plazos legales exactos para descargos y apelacion en propiedad
  horizontal colombiana: depende del Reglamento Interno del conjunto y
  la Ley 675 de 2001. **NO hardcodear**; se exponen como
  `tenant_setting`. Default sugerido: 10 dias habiles para descargos,
  15 dias habiles para resolver apelacion.
- Plazo legal de prescripcion de multas: pendiente confirmacion juridica.
- Si la apelacion debe ser revisada por un consejo (multi-firmante):
  V2.
- Forma de notificacion legal valida (correo certificado, fisico,
  email): pendiente decision juridica del tenant.

## 5. Modelo de datos propuesto

- `penalty_catalog`: `id`, `code` (UNIQUE WHERE deleted_at IS NULL),
  `name`, `description`, `default_sanction_type`, `base_amount`
  (NUMERIC(14,2)), `recurrence_multiplier` (NUMERIC(5,2)),
  `requires_council_threshold` (NUMERIC(14,2) NULL), campos estandar.
- `penalties`: `id`, `catalog_id`, `debtor_user_id` (FK users(id) NOT
  NULL), `unit_id` NULL, `source_incident_id` NULL, `sanction_type`,
  `amount` (NUMERIC(14,2)), `reason`, `imposed_by_user_id`,
  `notified_at` NULL, `appeal_deadline_at` NULL, `confirmed_at` NULL,
  `settled_at` NULL, `dismissed_at` NULL, `requires_council_approval`,
  `council_approved_by_user_id` NULL, `council_approved_at` NULL,
  `idempotency_key` (UNIQUE WHERE deleted_at IS NULL), `status` con
  CHECK del workflow, campos estandar + `version`.
- `penalty_appeals`: `id`, `penalty_id` (FK), `submitted_by_user_id`,
  `submitted_at`, `grounds`, `resolved_by_user_id` NULL,
  `resolved_at` NULL, `resolution`, `status`, campos estandar +
  `version`.
- `penalty_status_history`: append-only.
- `penalty_outbox_events`: outbox modulo-local. Eventos:
  `penalty.notified`, `penalty.appealed`, `penalty.confirmed`,
  `penalty.dismissed`, `penalty.settled`, `penalty.charge_requested`
  (-> cartera).

## 6. Endpoints propuestos

- `GET /api/v1/penalty-catalog` / `POST` / `PATCH /{id}`.
- `POST /api/v1/penalties` (drafted).
- `POST /api/v1/penalties/{id}/notify` (drafted -> notified, abre plazo).
- `POST /api/v1/penalties/{id}/council-approve` (si requerido).
- `POST /api/v1/penalties/{id}/confirm` (notified -> confirmed; emite
  charge_requested si monetary y sin apelacion).
- `POST /api/v1/penalties/{id}/settle` (confirmed -> settled).
- `POST /api/v1/penalties/{id}/cancel` (drafted -> cancelled).
- `POST /api/v1/penalties/{id}/appeals` (residente apela).
- `POST /api/v1/penalties/{id}/appeals/{aid}/resolve`.
- `GET /api/v1/penalties` (filtros: debtor, status, type).
- `GET /api/v1/penalties/{id}/history`.
- `GET /api/v1/reports/penalties/monthly`.

## 7. Permisos nuevos (namespaces)

- `penalties.catalog.read` / `penalties.catalog.write`.
- `penalties.impose` — crear y notificar.
- `penalties.approve_council` — aprobar sobre umbral.
- `penalties.confirm` / `penalties.settle` / `penalties.cancel`.
- `penalties.appeal` — residente apela su sancion.
- `penalties.appeal.resolve` — admin/consejo resuelve.
- `penalties.read_all` — auditor.
- `penalties.read_mine` — residente lee solo las suyas.

## 8. Casos extremos

- Imponer sancion a usuario soft-deleted: 422.
- Doble apelacion concurrente: UNIQUE constraint bloquea.
- Notificar dos veces: idempotente (no cambia estado si ya notificado).
- Confirmar una sancion en apelacion: 409.
- Catalogo eliminado mientras hay sancion `drafted`: ON DELETE
  RESTRICT en FK previene; se exige soft-delete.
- Reincidencia con sanciones soft-deleted: NO cuentan (filtrar
  `deleted_at IS NULL`).
- Cargo en cartera falla: outbox reintenta; sancion queda
  `confirmed` con flag `charge_pending`.

## 9. Operaciones transaccionales / idempotentes

- Imposicion: TX inserta `penalties` + `penalty_status_history` +
  `penalty_outbox_events('penalty.drafted')`. `idempotency_key`
  garantiza no duplicar.
- Confirmacion monetary: TX cambia status, inserta history, inserta
  outbox `penalty.charge_requested`.
- Apelacion: TX inserta `penalty_appeals` + transiciona penalty a
  `in_appeal` + history.

## 10. Configuracion por tenant

- `penalties.appeal_deadline_business_days` (default 10).
- `penalties.council_threshold_amount` (default equivalente a
  5 SMMLV; NUMERIC).
- `penalties.recurrence_window_days` (default 365).
- `penalties.recurrence_cap_multiplier` (default 5.0).
- `penalties.notification_template_id` (referencia a plantilla legal).

## 11. Notificaciones / eventos

Eventos al outbox -> Fase 15:
- `penalty.notified` -> debtor + admin.
- `penalty.appealed` -> admin/consejo.
- `penalty.appeal.resolved` -> debtor.
- `penalty.confirmed` -> debtor.
- `penalty.charge_requested` -> modulo financiero.

## 12. Reportes / metricas

- Reporte mensual: total por tipo, monto total, tasa de apelacion,
  tasa de aceptacion de apelaciones.
- Metricas OTel: `penalties_imposed_total`,
  `penalties_appealed_total`, `penalties_amount_total`.

## 13. Riesgos y mitigaciones

- **Riesgo**: error legal en notificacion. **Mitigacion**: plantilla
  configurable por tenant, no hardcoded.
- **Riesgo**: doble cobro a cartera. **Mitigacion**: outbox
  idempotente con `idempotency_key` por evento.
- **Riesgo**: apelacion no detiene cobro. **Mitigacion**: trigger en
  TX al cambiar a `in_appeal`.

## 14. Multi-agente sugerido

3 agentes en paralelo:
- Agente A: schema + catalogo + entidades.
- Agente B: workflow de penalty + apelaciones + history.
- Agente C: integracion con cartera (outbox) + reportes.

## 15. DoD adicional

- [ ] Multa monetaria genera evento `penalty.charge_requested`
  atomicamente al confirmar.
- [ ] Apelacion suspende cobro (test de integracion).
- [ ] Reincidencia incrementa monto segun catalogo (test).
- [ ] `idempotency_key` previene duplicacion (test).
- [ ] Umbral de consejo requiere aprobacion adicional (test 403).
