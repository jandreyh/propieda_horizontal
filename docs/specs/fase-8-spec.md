# Fase 8 — Spec — Parqueaderos

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

## 1. Resumen ejecutivo

Modulo `parking` que modela parqueaderos como entidades fisicas independientes
del conjunto, asignables a unidades (residentes) o reservables por visitantes.
Soporta historial de asignaciones, sorteos deterministas con semilla, reservas
de visitantes con prevencion de doble reserva via UNIQUE parcial sobre slots,
y reglas configurables por tenant. NO maneja talanqueras fisicas en V1; la
verificacion la realiza el guarda contra la asignacion vigente.

## 2. Decisiones tomadas

Todas las decisiones siguientes son adoptadas por defecto y deben ser
validadas posteriormente por el usuario.

- **ASSUMPTION**: parqueaderos son entidades independientes (`parking_spaces`),
  no atributos de la unidad.
- **ASSUMPTION**: tipos soportados: `covered`, `uncovered`, `motorcycle`,
  `bicycle`, `visitor`, `disabled`, `electric`, `double`.
- **ASSUMPTION**: existen parqueaderos privados (asignados a una unidad) y
  comunes/visitantes (no asignados, reservables).
- **ASSUMPTION**: el "dueno" administrativo es el conjunto (tenant); la
  asignacion vincula `parking_space -> unit`. La unidad puede tener varios
  parqueaderos.
- **ASSUMPTION**: numeracion fisica libre (`code`) con opcional `level`,
  `zone`, `structure_id`. UNIQUE por (`code`) entre activos.
- **ASSUMPTION**: V1 NO soporta tarifas de alquiler. La columna `monthly_fee`
  queda nullable como hook para Fase 9.
- **ASSUMPTION**: vehiculos se asocian a la unidad (modulo people existente),
  no al parqueadero. La asignacion permite anotar opcionalmente un
  `vehicle_id` esperado.
- **ASSUMPTION**: una unidad puede tener N parqueaderos; el limite es
  configurable por tenant (`parking.max_spaces_per_unit`, default `null`).
- **ASSUMPTION**: 1 parqueadero -> 1 asignacion activa. UNIQUE parcial sobre
  `parking_assignments.parking_space_id WHERE deleted_at IS NULL AND
  until_date IS NULL`.
- **ASSUMPTION**: reasignacion = cerrar la asignacion actual (`until_date`)
  y crear una nueva. Aprobacion se mapea a permiso `parking.assign`.
- **ASSUMPTION**: visitantes operan en modo reserva previa con `slot_start_at`
  y `slot_end_at`. FCFS lo cubre el cliente al elegir slot disponible.
- **ASSUMPTION**: tiempo maximo configurable (`parking.visitor_max_hours`,
  default 12). No multas en V1.
- **ASSUMPTION**: subarriendo entre residentes NO en V1.
- **ASSUMPTION**: sorteos opcionales con algoritmo determinista
  `SHA-256(seed || ordered_unit_ids) -> shuffle`. Resultado almacenado
  con la seed para reproducibilidad.
- **ASSUMPTION**: el guarda ve solo la asignacion actual (placa esperada,
  unidad), nunca cartera ni datos sensibles.
- **ASSUMPTION**: notificaciones se emiten via outbox modulo-local
  (`parking_outbox_events`).

## 3. Supuestos adoptados (no bloqueantes)

Mismos items de la seccion 2 (todos marcados ASSUMPTION). Se consolidan aqui
como recordatorio de revision.

- **ASSUMPTION**: ningun decreto local exige un campo adicional en V1.
- **ASSUMPTION**: el sorteo se ejecuta por solicitud manual del admin y
  publica resultados en feed de anuncios (Fase 6).

## 4. Open Questions

- **OPEN Q**: ¿el tenant requiere flujo de aprobacion del consejo antes de
  oficializar asignacion? (configurable, default `false`).
- **OPEN Q**: ¿hay obligacion legal local (ej. RPH) de publicar el listado
  de asignaciones en cartelera fisica? (impacta Fase 6).
- **OPEN Q**: ¿se admite cesion del parqueadero del propietario al
  inquilino sin aprobar admin? (afecta permisos).
- **OPEN Q**: ¿politica de no-show: penalizacion monetaria? (depende Fase 9
  + Fase 13).

## 5. Modelo de datos propuesto

Tablas (todas con campos estandar: `id`, `status`, `created_at`,
`updated_at`, `deleted_at`, `created_by`, `updated_by`, `deleted_by`,
`version`).

- `parking_spaces`: `code`, `type`, `structure_id`, `level`, `zone`,
  `monthly_fee` NULL, `is_visitor` BOOL, `notes`. UNIQUE(`code`) parcial.
- `parking_assignments`: `parking_space_id`, `unit_id`, `vehicle_id` NULL,
  `since_date`, `until_date` NULL, `assigned_by_user_id`. UNIQUE parcial
  sobre `parking_space_id` con `until_date IS NULL`.
- `parking_assignment_history`: snapshot append-only (no soft delete) con
  `parking_space_id`, `unit_id`, `since_date`, `until_date`, `closed_reason`.
- `parking_visitor_reservations`: `parking_space_id`, `unit_id` (anfitrion),
  `visitor_name`, `visitor_document`, `vehicle_plate`, `slot_start_at`,
  `slot_end_at`, `status` (`pending|confirmed|cancelled|no_show|consumed`).
  UNIQUE parcial (`parking_space_id`, `slot_start_at`) WHERE
  `status='confirmed' AND deleted_at IS NULL`.
- `parking_lottery_runs`: `name`, `seed_hash`, `criteria` JSONB, `executed_at`,
  `executed_by_user_id`.
- `parking_lottery_results`: `lottery_run_id`, `unit_id`, `parking_space_id`,
  `position`.
- `parking_rules`: `rule_key`, `rule_value` JSONB (override de defaults).
- `parking_outbox_events`: outbox modulo-local.

## 6. Endpoints

- `GET /parking-spaces` — `parking.read`
- `POST /parking-spaces` — `parking.write`
- `PUT /parking-spaces/:id` — `parking.write`
- `POST /parking-spaces/:id/assign` — `parking.assign` -> 201/409
- `POST /parking-assignments/:id/release` — `parking.assign`
- `GET /units/:id/parking` — `parking.read` (residente solo su unidad)
- `POST /parking-visitor-reservations` — `parking.visitor.create` -> 201/409
- `POST /parking-visitor-reservations/:id/cancel` — owner o `parking.write`
- `GET /parking-visitor-reservations?date=...` — `parking.read`
- `POST /parking-lotteries/run` — `parking.lottery.run`
- `GET /parking-lotteries/:id/results` — `parking.read`
- `GET /guard/parking/today` — `parking.guard.read` (vista del guarda)

Errores: RFC 7807 con `type=https://errors/parking/<slug>`.

## 7. Permisos nuevos a registrar

- `parking.read`: leer espacios, asignaciones, reservas.
- `parking.write`: crear/editar espacios.
- `parking.assign`: asignar/liberar espacios privados.
- `parking.visitor.create`: crear reservas de visitante.
- `parking.lottery.run`: ejecutar sorteo.
- `parking.guard.read`: lectura limitada para guarda.

## 8. Casos extremos

- Doble reserva visitante simultanea -> UNIQUE parcial fuerza 1 ganador, otro 409.
- Asignacion duplicada del mismo espacio -> UNIQUE parcial.
- Reasignacion en caliente: TX cierra `until_date` y abre nueva fila.
- Sorteo con menos espacios que unidades elegibles -> ranking parcial,
  publicar lista de espera.
- Visitante no llega: cron marca `no_show` al cumplir `slot_end_at + 30 min`.
- Vehiculo asociado eliminado: asignacion permanece (FK RESTRICT) o se
  re-asigna a NULL.

## 9. Operaciones transaccionales / idempotentes

- Asignacion: TX que (1) cierra previa con `until_date`, (2) inserta nueva,
  (3) graba `parking_assignment_history`, (4) emite outbox event.
- Reserva visitante: idempotente via `Idempotency-Key` header (cache 24h en
  outbox/redis). Conflicto -> 409.
- Sorteo: TX que (1) inserta `parking_lottery_runs`, (2) inserta resultados,
  (3) opcionalmente crea asignaciones derivadas. La seed se persiste para
  reproducibilidad.

## 10. Configuracion por tenant

Keys nuevas en `tenant_settings`:
- `parking.max_spaces_per_unit` (int|null, default null)
- `parking.visitor_max_hours` (int, default 12)
- `parking.visitor_advance_min_minutes` (int, default 0)
- `parking.visitor_advance_max_days` (int, default 7)
- `parking.allow_owner_to_tenant_cession` (bool, default false)
- `parking.lottery_publish_to_announcements` (bool, default true)
- `parking.no_show_grace_minutes` (int, default 30)

## 11. Notificaciones / eventos

- `parking.assigned` -> residente (todos los canales activos).
- `parking.released` -> residente.
- `parking.visitor_reservation_created` -> anfitrion + guarda turno.
- `parking.visitor_reservation_expiring` -> anfitrion 30 min antes.
- `parking.lottery_published` -> feed global de anuncios.

## 12. Reportes / metricas

- Ocupacion en tiempo real (`%` espacios privados con asignacion activa).
- Historial mensual de reasignaciones.
- Uso visitante por dia/semana.
- Reproduccion de sorteo (export JSON con seed + criterios).
- Export CSV de asignaciones para asambleas.

## 13. Riesgos y mitigaciones

- **Doble reserva**: UNIQUE parcial + retry con backoff.
- **Sorteo no reproducible**: persistir seed, criterios JSONB y orden de
  unidades elegibles.
- **Filtracion datos a guarda**: vista materializada o query especifica
  que excluye campos sensibles.
- **Race en reasignacion**: bloqueo optimista via `version` en
  `parking_assignments`.

## 14. Multi-agente sugerido

4 agentes paralelos:
- A: schema + migraciones + sqlc.
- B: usecases asignacion + sorteo (algoritmo determinista con seed).
- C: usecases reservas visitantes + cron de expiracion / no-show.
- D: handlers + OpenAPI + tests `go test -race`.

## 15. DoD adicional especifico

- [ ] Doble reserva visitante simultanea -> 1 OK + 1 409.
- [ ] Sorteo es reproducible con misma seed.
- [ ] Historial de asignaciones queryable por unidad y por parqueadero.
- [ ] Guarda no ve datos sensibles del residente, solo asignacion vigente.
- [ ] UNIQUE parcial sobre `parking_assignments` previene doble asignacion activa.
- [ ] OpenAPI completo y RFC 7807 en todos los errores.
