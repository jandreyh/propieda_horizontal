# Fase 9 — Spec — Modulo financiero base

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

## 1. Resumen ejecutivo

Modulo `finance` que gestiona plan de cuentas, centros de costo, cargos
(cuotas administracion, multas, intereses), pagos (manual, pasarela,
PSE/tarjeta) con aplicacion FIFO o manual, reversos con doble validacion,
cierre mensual con asientos inmutables, y certificados de paz y salvo en PDF.
Cumple Ley 1581 (datos personales). El webhook de la pasarela es idempotente
via tabla dedicada, y se previene doble cobro con UNIQUE parcial en
`payments(gateway_txn_id)`.

## 2. Decisiones tomadas

Todas marcadas ASSUMPTION para revision posterior.

- **ASSUMPTION**: la cuenta contable se asigna al par (`unit_id`,
  `account_holder_user_id`) creando el concepto "cuenta contrato".
- **ASSUMPTION**: plan de cuentas por tenant (cada conjunto puede personalizar).
- **ASSUMPTION**: centros de costo: `administracion`, `fondo_imprevistos`,
  `mantenimiento`, `parqueaderos`, `zonas_comunes`, `otros`.
- **ASSUMPTION**: tipos de cargo: `admin_fee`, `late_fee`, `interest`,
  `service`, `rental`, `penalty` (link a Fase 13), `other`.
- **ASSUMPTION**: periodicidad mensual por defecto, configurable por tenant.
- **ASSUMPTION**: saldo a favor permitido (registrado como `payment` con
  `unallocated_amount > 0`).
- **ASSUMPTION**: reverso requiere doble validacion sobre umbral
  configurable por tenant. Entry inmutable post-cierre.
- **ASSUMPTION**: conciliacion bancaria automatica POST-V1 (manual en V1).
- **ASSUMPTION**: pagos abiertos a `propietario`, `inquilino` y `tercero por
  link`. Configurable por tenant.
- **ASSUMPTION**: recibo email al pagador y CC al titular.
- **ASSUMPTION**: aplicacion de pagos por defecto FIFO; opcion `manual`
  por residente con permiso `payment.allocate`.
- **ASSUMPTION**: tasa de mora configurable por tenant
  (`finance.late_fee_rate_monthly`); gracia X dias configurable.
- **ASSUMPTION**: bloqueo de servicios (reservas, autorizaciones) por mora
  >= N dias configurable; el guarda NO ve cartera, solo flag binario
  "unidad bloqueada por mora" en metadata cacheada.
- **ASSUMPTION**: certificado paz y salvo automatico, firmado por
  `tenant_admin` con sello digital (no PKI en V1).
- **ASSUMPTION**: estado de pago: `PENDING -> AUTHORIZED -> CAPTURED ->
  SETTLED -> (REVERSED|FAILED)`.
- **ASSUMPTION**: outbox modulo-local emite eventos `charge.created`,
  `payment.captured`, `payment.reversed`, `period.closed`.
- **ASSUMPTION**: cierre mensual `soft` (revertible con auditoria) hasta
  cierre anual `hard` que sella entries con trigger anti-UPDATE/DELETE.
- **ASSUMPTION**: cada tenant tiene su merchant propio en la pasarela
  (multitenancy strict, credenciales encriptadas en
  `payment_gateway_configs`).

## 3. Supuestos adoptados (no bloqueantes)

Mismos items de seccion 2.

- **ASSUMPTION**: numeros consecutivos de recibo/factura por tenant y anio.
- **ASSUMPTION**: pdf de estados de cuenta y certificados generados con
  plantilla HTML + headless render (no PDFKit propietario).

## 4. Open Questions

- **OPEN Q**: ¿pasarela de pagos concreta? Candidatos colombianos: **Wompi**,
  **PayU**, **Mercado Pago**, **ePayco**. La spec asume contrato adapter
  generico; el adapter concreto se decide en discovery posterior.
- **OPEN Q**: ¿cumplimiento DIAN aplica? (facturacion electronica obligatoria
  para administradoras pero NO para copropiedad — confirmar).
- **OPEN Q**: ¿reglas contables Decreto 2706 / NIIF para entidades sin
  animo de lucro? confirmar perfil contable.
- **OPEN Q**: ¿certificado de paz y salvo requiere firma PKI segun Ley 527?
  En V1 se asume firma simple con trazabilidad; pendiente validacion legal.
- **OPEN Q**: ¿retencion en la fuente sobre cobros? (tipico no aplica para
  copropiedades pero confirmar).

## 5. Modelo de datos propuesto

Tablas (todas con campos estandar, salvo asientos contables sellados que
mantienen `created_at` pero rechazan UPDATE/DELETE post-cierre).

- `chart_of_accounts`: `code`, `name`, `account_type` (asset|liability|
  equity|income|expense), `parent_id` NULL.
- `cost_centers`: `code`, `name`.
- `billing_accounts`: `unit_id`, `holder_user_id`, `opened_at`,
  `closed_at` NULL. UNIQUE(`unit_id`,`holder_user_id`) parcial.
- `charges`: `billing_account_id`, `concept` ENUM, `period_year`,
  `period_month`, `amount`, `due_date`, `cost_center_id`, `account_id`,
  `idempotency_key` (para generacion masiva).
- `charge_items`: detalle linea (`charge_id`, `description`, `amount`).
- `payment_methods`: catalogo (`cash`, `bank_transfer`, `pse`,
  `credit_card`, `debit_card`, `voucher`).
- `payments`: `billing_account_id`, `gateway`, `gateway_txn_id` NULL,
  `idempotency_key`, `amount`, `currency`, `status`, `captured_at`,
  `unallocated_amount`. UNIQUE(`gateway_txn_id`) parcial WHERE
  `gateway_txn_id IS NOT NULL AND deleted_at IS NULL`.
- `payment_allocations`: `payment_id`, `charge_id`, `amount`.
- `payment_reversals`: `payment_id`, `reason`, `requested_by`,
  `approved_by` NULL, `approved_at` NULL, `status`.
- `accounting_entries`: `period`, `posted_at`, `source_type`,
  `source_id`, `posted` BOOL, `sealed` BOOL.
- `accounting_entry_lines`: `entry_id`, `account_id`, `cost_center_id`,
  `debit`, `credit`.
- `payment_gateway_configs`: `gateway`, `merchant_id`, `secrets_kms_ref`,
  `enabled`.
- `payment_webhook_idempotency`: `gateway`, `idempotency_key`, `received_at`,
  `payload_hash`. UNIQUE(`gateway`, `idempotency_key`).
- `late_fee_runs`: registro de generacion de intereses (idempotente por
  `(period_year, period_month)`).
- `period_closures`: `period_year`, `period_month`, `closed_soft_at`,
  `closed_hard_at`, `closed_by`.
- `paid_in_full_certificates`: `unit_id`, `issued_at`, `valid_until`,
  `pdf_url`, `signed_by_user_id`, `hash`.
- `finance_outbox_events`.

## 6. Endpoints

- `GET /chart-of-accounts` / `POST /chart-of-accounts` — `finance.coa.*`
- `GET /cost-centers` / `POST /cost-centers` — `finance.coa.*`
- `POST /charges` (creacion individual) — `finance.charge.create`
- `POST /charges/generate-monthly` — `finance.charge.create` (idempotente)
- `GET /charges?unit_id=&period=` — `finance.charge.read`
- `POST /payments` (manual) — `finance.payment.create`
- `POST /payments/webhook/:gateway` — publico, idempotente
- `POST /payments/:id/reverse` — `finance.payment.reverse`
- `POST /payments/:id/reverse/:reversal_id/approve` —
  `finance.payment.reverse.approve`
- `POST /payments/:id/allocate` — `finance.payment.allocate`
- `GET /billing-accounts/:id/statement` — `finance.read.own` o
  `finance.read.all`
- `POST /periods/:year/:month/close-soft` — `finance.period.close`
- `POST /periods/:year/:month/close-hard` — `finance.period.close.hard`
- `POST /units/:id/paid-in-full-certificate` —
  `finance.certificate.issue`
- `GET /reports/portfolio` — `finance.report.read`
- `GET /reports/financial-statements?period=` — `finance.report.read`

Errores RFC 7807.

## 7. Permisos nuevos a registrar

- `finance.coa.read` / `finance.coa.write`
- `finance.charge.create` / `finance.charge.read` / `finance.charge.write`
- `finance.payment.create` / `finance.payment.read.own` /
  `finance.payment.read.all`
- `finance.payment.reverse` / `finance.payment.reverse.approve`
- `finance.payment.allocate`
- `finance.period.close` / `finance.period.close.hard`
- `finance.certificate.issue`
- `finance.report.read`
- `finance.gateway.config`

## 8. Casos extremos

- Webhook duplicado -> deduplicado por
  `payment_webhook_idempotency(gateway, idempotency_key)`.
- Doble pago real (mismo `gateway_txn_id`) -> rechazado por UNIQUE.
- Pago parcial -> `unallocated_amount` queda como saldo a favor; aplicacion
  posterior FIFO o manual.
- Reverso sobre cierre hard -> rechazado (entry sellada).
- Generacion masiva de cargos lanzada 2 veces -> idempotente por
  `idempotency_key=year-month-account`.
- Mora con cambio retroactivo de tasa -> nuevo run de intereses, no
  modifica historicos.
- Inquilino paga pero el titular es otro: pago se acredita al
  `billing_account` correctamente; recibo a ambos.

## 9. Operaciones transaccionales / idempotentes

- Captura de pago + asiento contable: TX unica con escrituras a
  `payments`, `payment_allocations`, `accounting_entries`,
  `accounting_entry_lines`, `outbox`.
- Reverso: TX que crea `payment_reversals`, marca `payments.status=REVERSED`
  (con bloqueo optimista por `version`), genera entry contrario.
- Webhook: SELECT FOR UPDATE sobre `payment_webhook_idempotency` antes de
  procesar; si ya existe -> 200 OK no-op.
- Cierre soft: marca period, deshabilita escrituras nuevas en ese periodo
  excepto por usuarios con permiso `finance.period.close`.
- Cierre hard: trigger `tg_finance_entries_immutable` rechaza UPDATE/DELETE
  sobre `accounting_entries` con `sealed=true`.

## 10. Configuracion por tenant

- `finance.billing_period` (string, `monthly|bimonthly`, default `monthly`)
- `finance.late_fee_rate_monthly` (decimal, default 0.0150)
- `finance.late_fee_grace_days` (int, default 5)
- `finance.payment_allocation_default` (`fifo|manual`, default `fifo`)
- `finance.reverse_amount_threshold` (decimal, default 100000 COP)
- `finance.block_services_after_days_overdue` (int, default 60)
- `finance.allow_third_party_payment` (bool, default true)
- `finance.gateway.primary` (string, ej. `wompi`)
- `finance.tenant.merchant_id` (per gateway via
  `payment_gateway_configs`)
- `finance.certificate_validity_days` (int, default 30)

## 11. Notificaciones / eventos

- `charge.created` -> residente (email + push).
- `payment.captured` -> residente + admin.
- `payment.reversed` -> residente + admin.
- `period.closed_soft` -> contador + admin.
- `payment.failed` -> residente.
- `mora.threshold_crossed` -> residente + admin (90 dias antes de bloqueo
  total).

## 12. Reportes / metricas

- Cartera por unidad (PDF / Excel).
- Estado de cuenta historico.
- Estados financieros (PYG y balance) por periodo.
- Conciliacion mensual extracto vs sistema (V2).
- Recaudo por canal/pasarela.
- Top deudores (visible solo a `accountant`/`tenant_admin`).
- Reproducibilidad: export JSON de un periodo cerrado.

## 13. Riesgos y mitigaciones

- **Doble cobro**: UNIQUE parcial sobre `gateway_txn_id` + dedup webhook.
- **Reverso fraudulento**: doble validacion + audit log severity HIGH.
- **Mutacion post-cierre**: trigger SQL inmutabilidad + tests.
- **Discrepancia contable**: balance entry constraint
  (`SUM(debit)=SUM(credit)` por entry).
- **Datos personales**: cifrado at-rest de
  `payment_gateway_configs.secrets_kms_ref`.

## 14. Multi-agente sugerido

6 agentes paralelos (es el modulo mas grande):
- A: schema + migraciones + triggers de inmutabilidad.
- B: cargos (creacion, generacion masiva mensual idempotente, intereses).
- C: pagos + aplicacion FIFO/manual + reversos.
- D: integracion pasarela (adapter generico + webhook idempotente).
- E: reportes + exportaciones + certificado PDF.
- F: handlers + OpenAPI + tests race + tests webhook duplicado.

## 15. DoD adicional especifico

- [ ] Webhook duplicado -> 1 sola aplicacion (verificar via
  `payment_webhook_idempotency`).
- [ ] Doble pago en pasarela (`gateway_txn_id` repetido) rechazado por UNIQUE.
- [ ] Reverso sobre umbral requiere doble validacion (test explicito).
- [ ] Cierre hard marca entries inmutables; UPDATE falla por trigger.
- [ ] Certificado paz y salvo en PDF firmado y consultable.
- [ ] Auditor lee todo y mutacion prohibida (test explicito).
- [ ] OpenAPI completo, RFC 7807 en errores.
