# Fase 15 — Spec — Notificaciones multicanal

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

---

## 1. Resumen ejecutivo

Modulo transversal que entrega notificaciones por email, push, WhatsApp
y SMS. Implementa **outbox pattern** con worker (at-least-once),
idempotencia por (`event_type`, `recipient_user_id`, `idempotency_key`),
preferencias por usuario y por tipo de evento, plantillas por canal,
reintentos con backoff exponencial y respeto al horario silencioso
configurable por tenant. Los proveedores concretos (WhatsApp, SMS,
email, push) son adaptadores intercambiables; la decision final de
proveedor queda en Open Questions.

## 2. Decisiones tomadas

- **ASSUMPTION**: Canales V1: `email`, `push`, `whatsapp`, `sms`.
  Cada uno es un adapter detras de una interface
  `ChannelSender`.
- **ASSUMPTION**: Cada residente puede elegir canales preferidos por
  tipo de evento via `notification_preferences`. Default: todos los
  canales activos para eventos criticos; solo `email` y `push` para
  no criticos.
- **ASSUMPTION**: Plantillas (`notification_templates`): por
  `event_type` + `channel` + `locale` (default `es-CO`). Multilingue
  preparado pero V1 solo `es-CO`.
- **ASSUMPTION**: Mapping evento->canales default (sobre-escribible
  por preferencia de usuario):
  - `package.received` -> push, whatsapp.
  - `incident.assigned` -> email, push.
  - `pqrs.responded` -> email, push.
  - `announcement.critical` -> email, push, whatsapp, sms (forzado).
  - `penalty.notified` -> email + canal legal del tenant.
- **ASSUMPTION**: Anuncios criticos (`is_critical=true` + flag de
  forzado) ignoran preferencias y horario silencioso. Resto respeta.
- **ASSUMPTION**: Horario silencioso: configurable por tenant (default
  22:00-07:00, zona horaria del tenant). Mensajes durante silencio se
  encolan con `scheduled_at` al fin del horario.
- **ASSUMPTION**: Opt-in para WhatsApp/SMS: tabla
  `notification_consents` con `user_id`, `channel`, `consented_at`,
  `revoked_at`, `consent_proof_url`. Sin consentimiento: el adapter
  rechaza envio (`status='blocked_no_consent'`).
- **ASSUMPTION**: Plantillas las crea/edita `tenant_admin` con permiso
  dedicado. Envio masivo manual requiere MFA reciente + log de
  auditoria con motivo.
- **ASSUMPTION**: Outbox obligatorio. Worker batch cada 5s, polling
  con `FOR UPDATE SKIP LOCKED`.
- **ASSUMPTION**: Reintentos backoff exponencial: 1m, 5m, 15m, 1h,
  6h, 24h. Tras 6 fallos -> `status='failed_permanent'`. Configurable.
- **ASSUMPTION**: Idempotencia: UNIQUE
  `(event_type, recipient_user_id, idempotency_key) WHERE deleted_at
  IS NULL`. El productor del evento provee `idempotency_key` (ej.
  `package.received:{package_id}`).
- **ASSUMPTION**: Configuraciones de proveedores por tenant via
  `notification_provider_configs` (credentials encriptadas con KMS;
  almacenadas como JSONB cifrado en V1, integracion KMS V2).

## 3. Supuestos adoptados (no bloqueantes)

- Estados outbox: `pending`, `scheduled`, `sending`, `sent`,
  `failed_retry`, `failed_permanent`, `blocked_no_consent`,
  `cancelled`.
- Una entrega = una fila en `notification_deliveries` (con id del
  proveedor, timestamp, status detallado).
- Costos por tenant (medicion para facturacion SaaS) se exponen via
  query agregada sobre `notification_deliveries`.
- Push: usar tokens en `notification_push_tokens` (FK users), uno o
  mas por usuario.

## 4. Open Questions

- **Provider WhatsApp**: Twilio vs Meta WhatsApp Business API
  (oficial, mas barato a escala) vs Gupshup. **Decision externa**.
- **Provider SMS**: Twilio vs AWS SNS vs proveedor local Colombia
  (Hablame, Infobip).
- **Provider Email**: AWS SES vs Postmark vs SendGrid vs Resend.
- **Provider Push**: FCM (Android + iOS via APNs proxy) vs APNs
  directo + FCM (split por OS).
- Plantillas WhatsApp deben pre-aprobarse en Meta (HSM templates):
  proceso operativo fuera del scope tecnico.
- Marco legal opt-in en Colombia: consentimiento previo expreso e
  informado segun habeas data y normas de proteccion al consumidor.
  **No hardcodear**; consentimiento se exige siempre y se prueba con
  `consent_proof_url`.
- Encriptacion de credenciales de proveedor: KMS provider (AWS KMS
  vs HashiCorp Vault).
- Costo por canal y rate limits por tenant: V2.

## 5. Modelo de datos propuesto

- `notification_templates`: `id`, `event_type`, `channel`, `locale`,
  `subject` NULL (email), `body_template`, `provider_template_ref`
  NULL (HSM ID en WhatsApp), `status`, campos estandar + `version`.
  UNIQUE `(event_type, channel, locale) WHERE deleted_at IS NULL`.
- `notification_preferences`: `id`, `user_id` (FK users(id)),
  `event_type`, `channel`, `enabled` BOOLEAN, campos estandar.
  UNIQUE `(user_id, event_type, channel) WHERE deleted_at IS NULL`.
- `notification_consents`: `id`, `user_id`, `channel`,
  `consented_at`, `revoked_at` NULL, `consent_proof_url`,
  `legal_basis`, campos estandar. UNIQUE `(user_id, channel) WHERE
  deleted_at IS NULL`.
- `notification_outbox`: `id`, `event_type`, `recipient_user_id`,
  `channel`, `payload` JSONB, `idempotency_key`, `scheduled_at`,
  `sent_at` NULL, `status`, `attempts`, `last_error`, campos
  estandar. UNIQUE `(event_type, recipient_user_id, idempotency_key)
  WHERE deleted_at IS NULL`. Indice `(status, scheduled_at) WHERE
  status IN ('pending','scheduled','failed_retry')` para el worker.
- `notification_deliveries`: `id`, `outbox_id` (FK),
  `provider_message_id`, `provider_status`, `delivered_at` NULL,
  `failure_reason` NULL, campos estandar.
- `notification_provider_configs`: `id`, `channel`, `provider_name`,
  `config` JSONB cifrado, `is_active`, campos estandar. UNIQUE
  `(channel, provider_name) WHERE deleted_at IS NULL`.
- `notification_push_tokens`: `id`, `user_id`, `platform`
  (`ios`/`android`/`web`), `token`, `last_seen_at`, campos estandar.
  UNIQUE `(user_id, token) WHERE deleted_at IS NULL`.

## 6. Endpoints propuestos

- `GET /api/v1/notifications/preferences` (mias).
- `PATCH /api/v1/notifications/preferences`.
- `POST /api/v1/notifications/consents` — opt-in por canal.
- `DELETE /api/v1/notifications/consents/{channel}` — revoke.
- `POST /api/v1/notifications/push-tokens` — registrar token.
- `DELETE /api/v1/notifications/push-tokens/{id}`.
- `GET /api/v1/notifications/templates` / `POST` / `PATCH /{id}`.
- `GET /api/v1/notifications/provider-configs` / `PATCH`.
- `POST /api/v1/notifications/broadcast` — envio masivo (admin +
  MFA).
- `GET /api/v1/reports/notifications/deliverability`.

(Eventos internos NO tienen endpoint publico; se publican via outbox.)

## 7. Permisos nuevos (namespaces)

- `notifications.preferences.read_mine` /
  `notifications.preferences.write_mine`.
- `notifications.consents.write_mine`.
- `notifications.push_tokens.write_mine`.
- `notifications.templates.read` / `notifications.templates.write`.
- `notifications.providers.read` / `notifications.providers.write`.
- `notifications.broadcast` — envio masivo (requiere MFA reciente).
- `notifications.report` — reportes de entregabilidad.

## 8. Casos extremos

- Token push invalido / expirado: adapter marca delivery como
  `invalid_token`; worker desactiva token (set `last_seen_at` muy
  vieja) y NO reintenta.
- Usuario sin consentimiento WhatsApp: outbox se setea
  `blocked_no_consent`, no se reintenta.
- Mensaje creado a las 23:30 con horario silencioso 22:00-07:00:
  `scheduled_at` se ajusta a 07:00 del dia siguiente (zona del
  tenant).
- Mismo evento producido 2 veces (productor reintenta): UNIQUE
  bloquea, retorna idempotente.
- Provider down: backoff exponencial, no se desbloquea worker entero
  (uso de SKIP LOCKED).
- Anuncio critico forzado: ignora horario silencioso, ignora
  preferencias `enabled=false`, pero NO ignora consentimiento legal
  (no se puede enviar SMS sin opt-in legal).
- Plantilla faltante (event/channel/locale): outbox -> `failed_permanent`
  con `last_error='template_missing'`, alerta a admin.

## 9. Operaciones transaccionales / idempotentes

- Productor (otro modulo): inserta en SU outbox local. Un relay envia
  a `notification_outbox` central de este modulo, manteniendo
  `idempotency_key`.
- Worker:
  1. `SELECT ... FOR UPDATE SKIP LOCKED LIMIT N WHERE status IN
     ('pending','scheduled','failed_retry') AND scheduled_at <= now()`.
  2. Marca `sending`.
  3. Llama adapter; en exito: insert `notification_deliveries`,
     update outbox `sent`. En fallo: incrementa `attempts`, calcula
     siguiente `scheduled_at`, set `failed_retry` o
     `failed_permanent`.
  4. Todo en TX corta para no bloquear filas.

## 10. Configuracion por tenant

- `notifications.quiet_hours_start` (default `22:00`).
- `notifications.quiet_hours_end` (default `07:00`).
- `notifications.timezone` (default `America/Bogota`).
- `notifications.retry_schedule_seconds` (default `[60, 300, 900,
  3600, 21600, 86400]`).
- `notifications.broadcast_requires_mfa` (default true).
- `notifications.default_locale` (default `es-CO`).

## 11. Notificaciones / eventos

Este es el modulo consumidor. Recibe eventos de:
- `package.*` (Fase 5).
- `announcement.*` (Fase 6).
- `incident.*` (Fase 12).
- `penalty.*` (Fase 13).
- `pqrs.*` (Fase 14).
- `reservation.*` (Fase 10), `assembly.*` (Fase 11), etc.

Emite eventos internos para auditoria: `notification.sent`,
`notification.failed`.

## 12. Reportes / metricas

- Reporte de entregabilidad por canal: `sent`/`failed`/
  `blocked_no_consent` por evento, por dia, por tenant.
- Costos por tenant para facturacion SaaS (count por canal).
- Metricas OTel: `notifications_sent_total{channel,event_type}`,
  `notifications_failed_total{channel,reason}`,
  `notification_delivery_seconds` (histogram),
  `notification_outbox_lag_seconds`.

## 13. Riesgos y mitigaciones

- **Riesgo**: Spam por bug del productor. **Mitigacion**:
  idempotency_key + UNIQUE.
- **Riesgo**: Provider lock-in. **Mitigacion**: interface
  `ChannelSender` + adapters intercambiables.
- **Riesgo**: Costo descontrolado. **Mitigacion**: rate limits por
  tenant (V2) + reporte de costo.
- **Riesgo**: Falla de cumplimiento opt-in. **Mitigacion**: gate
  estricto en adapter + auditoria.
- **Riesgo**: Worker stuck procesando una fila. **Mitigacion**:
  SKIP LOCKED + timeout en TX.

## 14. Multi-agente sugerido

4 agentes en paralelo:
- Agente A: schema + outbox + entidades + worker.
- Agente B: adapters por canal (4 adapters como sub-tareas; cada
  uno es interface implementada).
- Agente C: handlers HTTP + preferences + consents + push tokens.
- Agente D: templates + broadcast + reportes.

## 15. DoD adicional

- [ ] Outbox + worker entregan at-least-once (test de fallo
  reintento).
- [ ] Idempotencia evita duplicados (test inserta dos veces).
- [ ] Horario silencioso respetado (test con time freeze).
- [ ] Opt-in: sin consentimiento, no se envia (test).
- [ ] Reintentos con backoff exponencial (test mide intervalos).
- [ ] Token push invalido se desactiva (test).
- [ ] Plantilla faltante -> failed_permanent + alerta.
- [ ] Broadcast requiere MFA reciente (test 403).
