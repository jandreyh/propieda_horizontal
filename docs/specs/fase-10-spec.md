# Fase 10 — Spec — Reservas de zonas comunes

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

## 1. Resumen ejecutivo

Modulo `reservations` que permite a residentes reservar zonas comunes
(salon social, BBQ, piscina, gym, cancha) con reglas configurables por
zona (capacidad, horario, costo, deposito) y por tenant (anticipacion,
cupos, bloqueo por mora). Garantiza no-doble-reserva via UNIQUE parcial
sobre `(common_area_id, slot_start_at) WHERE status='confirmed'`. Soporta
QR de validacion para guarda y notificaciones de recordatorio.

## 2. Decisiones tomadas

Todas marcadas ASSUMPTION.

- **ASSUMPTION**: zonas semilla: `salon_social`, `bbq`, `piscina`, `gym`,
  `cancha`, `sala_estudio`. Tenant puede agregar mas.
- **ASSUMPTION**: cada zona tiene `max_capacity`, `opening_time`,
  `closing_time`, `slot_duration_minutes`, `cost_per_use`,
  `security_deposit`, `requires_approval`.
- **ASSUMPTION**: aforo simultaneo: por defecto `1 reserva exclusiva por
  slot`. Configurable a `compartido` (sin UNIQUE estricto) en futuro.
- **ASSUMPTION**: reserva por `propietario` o `inquilino`, configurable
  por tenant (`reservations.who_can_reserve`).
- **ASSUMPTION**: anticipacion minima 0h, maxima 30 dias (configurables).
- **ASSUMPTION**: cupos por unidad/mes configurables (default `null`).
- **ASSUMPTION**: aprobacion previa solo si `common_area.requires_approval`.
- **ASSUMPTION**: bloqueo por mora si Fase 9 lo activa (consulta a
  `billing_accounts` o flag cacheado).
- **ASSUMPTION**: cancelacion permitida hasta N horas antes (default 24h)
  con devolucion total; despues, 50% devolucion.
- **ASSUMPTION**: el guarda valida ingreso escaneando QR de la reserva.
- **ASSUMPTION**: deposito devolucion automatica post-uso si no hay
  incidente (Fase 12).
- **ASSUMPTION**: outbox modulo-local emite eventos para notificacion.

## 3. Supuestos adoptados (no bloqueantes)

Mismos items de seccion 2.

- **ASSUMPTION**: integracion con Fase 9 (cargo por uso) opcional via
  `reservation.charge_billing_account_id`.

## 4. Open Questions

- **OPEN Q**: ¿algun tenant requiere aforo simultaneo nativo (varias
  reservas en el mismo slot)? Si aplica, refactor a tabla
  `reservation_attendees`.
- **OPEN Q**: ¿politica de penalizacion por no-show es monetaria
  (cargo Fase 9) o suspension (cuentas Fase 13)?
- **OPEN Q**: ¿integracion con calendario externo (Google, Outlook) en V1?

## 5. Modelo de datos propuesto

Tablas (con campos estandar).

- `common_areas`: `name`, `code`, `kind`, `max_capacity`,
  `opening_time`, `closing_time`, `slot_duration_minutes`, `cost_per_use`,
  `security_deposit`, `requires_approval` BOOL, `is_active`.
- `common_area_rules`: extension de reglas configurables por area
  (`rule_key`, `rule_value` JSONB).
- `reservations`: `common_area_id`, `unit_id`, `requested_by_user_id`,
  `slot_start_at`, `slot_end_at`, `attendees_count`, `status`
  (`pending|confirmed|cancelled|consumed|no_show`), `cost`,
  `security_deposit`, `qr_code_hash`, `idempotency_key`.
- `reservation_payments`: vincula `reservation_id` con `payment_id`
  (Fase 9) o `voucher_url` para pago manual.
- `reservation_blackouts`: `common_area_id`, `from_at`, `to_at`,
  `reason` (mantenimiento, asamblea).
- `reservation_status_history`: log de transiciones.
- `reservations_outbox_events`.

UNIQUE parcial: `(common_area_id, slot_start_at)` WHERE
`status='confirmed' AND deleted_at IS NULL`.

## 6. Endpoints

- `GET /common-areas` — `reservation.area.read`
- `POST /common-areas` — `reservation.area.write`
- `PUT /common-areas/:id` — `reservation.area.write`
- `POST /common-areas/:id/blackouts` — `reservation.area.write`
- `GET /common-areas/:id/availability?date=` — `reservation.area.read`
- `POST /reservations` — `reservation.create` (header
  `Idempotency-Key`)
- `POST /reservations/:id/cancel` — owner o `reservation.write`
- `POST /reservations/:id/approve` — `reservation.approve`
- `POST /reservations/:id/reject` — `reservation.approve`
- `POST /reservations/:id/checkin` — `reservation.guard.checkin`
- `GET /reservations?unit_id=&date=` — segun permiso
- `GET /reservations/mine` — autenticado

Errores RFC 7807.

## 7. Permisos nuevos a registrar

- `reservation.area.read` / `reservation.area.write`
- `reservation.create`
- `reservation.approve`
- `reservation.guard.checkin`
- `reservation.write` (cancelar de terceros)
- `reservation.report.read`

## 8. Casos extremos

- Doble reserva slot mismo segundo -> UNIQUE parcial -> 1 OK + 1 409.
- Reserva en blackout -> rechazada con RFC 7807.
- Reserva fuera de horario -> rechazada.
- Cancelacion tardia -> aplica regla devolucion parcial.
- Bloqueo por mora -> rechazo con `type=mora_block`.
- Cambio de horario despues de confirmacion -> politica: NO permitido,
  cancelar y crear nuevo.
- No-show -> cron marca status `no_show` 30 min despues del slot_end.

## 9. Operaciones transaccionales / idempotentes

- Crear reserva: TX que (1) verifica disponibilidad, (2) inserta
  reservation con status `confirmed` o `pending`, (3) genera QR hash,
  (4) emite outbox.
- `Idempotency-Key`: cache en outbox o tabla dedicada por 24h. Repetir
  request -> 200 con misma reserva.
- Aprobacion: bloqueo optimista via `version`.

## 10. Configuracion por tenant

- `reservations.who_can_reserve` (`owner|tenant|both`, default `both`)
- `reservations.advance_min_hours` (int, default 0)
- `reservations.advance_max_days` (int, default 30)
- `reservations.cap_per_unit_per_month` (int|null, default null)
- `reservations.block_if_overdue_days` (int, default 0; 0 = desactivado)
- `reservations.cancel_full_refund_hours` (int, default 24)
- `reservations.cancel_partial_refund_pct` (decimal, default 0.50)
- `reservations.no_show_grace_minutes` (int, default 30)
- `reservations.qr_validity_minutes_before` (int, default 30)

## 11. Notificaciones / eventos

- `reservation.created` -> residente + admin si `requires_approval`.
- `reservation.approved` / `reservation.rejected` -> residente.
- `reservation.reminder_24h` -> residente.
- `reservation.checkin` -> admin (audit).
- `reservation.cancelled` -> residente + lista de espera (V2).
- `reservation.deposit_refunded` -> residente.

## 12. Reportes / metricas

- Ocupacion por zona/mes (%).
- Top zonas mas reservadas.
- Reservas canceladas vs efectivas.
- Ingresos por uso.
- No-show rate.

## 13. Riesgos y mitigaciones

- **Doble reserva**: UNIQUE parcial + retry.
- **QR falsificado**: hash firmado con secreto del tenant.
- **Bloqueo por mora con datos stale**: consulta directa a Fase 9 al
  momento de validar.
- **Notificacion duplicada**: outbox con dedup.

## 14. Multi-agente sugerido

3 agentes paralelos:
- A: schema + migraciones + sqlc.
- B: usecases (crear, aprobar, cancelar, checkin, cron no-show).
- C: handlers + OpenAPI + tests race + cron expiracion.

## 15. DoD adicional especifico

- [ ] Doble reserva slot -> 1 OK + 1 409 (test concurrencia).
- [ ] UNIQUE parcial sobre `(common_area_id, slot_start_at)` WHERE
  `status='confirmed' AND deleted_at IS NULL` activo.
- [ ] Bloqueo por mora aplicable cuando tenant lo activa.
- [ ] QR validable por guarda con `reservation.guard.checkin`.
- [ ] Cron marca `no_show` y libera deposito segun regla.
- [ ] OpenAPI completo, RFC 7807.
