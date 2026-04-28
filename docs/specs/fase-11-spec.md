# Fase 11 — Spec — Asambleas, votaciones y actas

**Estado**: Frozen-Auto
**Validado por**: auto
**Fecha de freeze**: 2026-04-28
**Version**: 1.0

## 1. Resumen ejecutivo

Modulo `assemblies` para convocatoria, registro de asistencia, manejo de
poderes/apoderados, votaciones (secretas o nominales) y generacion de
actas. Aplica Ley 675 de 2001 (regimen de propiedad horizontal) y
Ley 527 de 1999 (mensajes de datos / firma electronica). Padron por
coeficiente de copropiedad. Integridad de votos via hash chain
(`prev_vote_hash` + `vote_hash`). Inquilinos NO votan en V1 (regla PH).
Acta exportable a PDF con anexos de evidencia.

## 2. Decisiones tomadas

Todas marcadas ASSUMPTION.

- **ASSUMPTION**: tipos de asamblea: `ordinaria`, `extraordinaria`,
  `virtual`, `mixta`.
- **ASSUMPTION**: padron usa coeficiente de copropiedad (Ley 675);
  fallback a "1 unidad = 1 voto" si tenant lo activa
  (`assemblies.voting_mode`).
- **ASSUMPTION**: apoderados permitidos con limite configurable por
  tenant (`assemblies.max_proxies_per_attendee`, default 1, segun
  reglamento de cada copropiedad).
- **ASSUMPTION**: quorum minimo configurable por tipo de decision
  (simple: 51%, calificada: 70%, especial: 80%) segun Ley 675.
- **ASSUMPTION**: convocatoria con anticipacion minima 15 dias (Ley 675
  ordinaria) configurable por tenant; canales obligatorios: email +
  cartelera (anuncio Fase 6).
- **ASSUMPTION**: solo propietarios votan (regla PH default). Inquilinos
  pueden votar SOLO si tienen poder firmado del propietario.
- **ASSUMPTION**: voto secreto vs nominal configurable por moccion.
- **ASSUMPTION**: modificacion de voto antes del cierre permitida; cada
  cambio queda en hash chain con sello de tiempo.
- **ASSUMPTION**: acta generada automaticamente, validada (firmada) por
  presidente y secretario via firma simple con trazabilidad (Ley 527).
- **ASSUMPTION**: PKI/firma digital certificada NO en V1; pendiente
  validacion legal.
- **ASSUMPTION**: hash chain SHA-256: `vote_hash = hash(prev_vote_hash ||
  motion_id || voter_id || option || timestamp || nonce)`.
- **ASSUMPTION**: evidencia digital almacenada: IP, user_agent, timestamp
  con NTP del servidor, hash del payload.
- **ASSUMPTION**: outbox modulo-local emite eventos.

## 3. Supuestos adoptados (no bloqueantes)

Mismos items de seccion 2.

- **ASSUMPTION**: actas archivadas indefinidamente (NO soft delete despues
  de firma).
- **ASSUMPTION**: voto solo durante ventana de votacion; sistema rechaza
  fuera de tiempo.

## 4. Open Questions

- **OPEN Q**: Ley 527 — ¿que tipo concreto de firma electronica usar?
  Opciones: (1) simple con OTP + trazabilidad; (2) firma digital
  certificada con entidad acreditada (Certicamara, Andes SCD, GSE).
  La spec asume opcion (1) en V1; pendiente concepto legal.
- **OPEN Q**: ¿asambleas virtuales requieren video grabado y archivado?
  Estandar post-pandemia es si; pendiente confirmar.
- **OPEN Q**: ¿padron debe excluir morosos (Ley 675 art. 23)? Default
  asume si: morosos tienen voz pero NO voto. Configurable.
- **OPEN Q**: ¿reglamento interno de cada copropiedad puede definir
  mayorias diferentes? Si si, exponer todos como configurables.
- **OPEN Q**: ¿almacenamiento de actas debe cumplir conservacion 10 anos
  (Codigo de Comercio)? Confirmar.

## 5. Modelo de datos propuesto

Tablas (campos estandar; actas firmadas se sellan via trigger).

- `assemblies`: `name`, `assembly_type`, `scheduled_at`, `status`
  (`draft|called|in_progress|closed|archived`), `quorum_required_pct`,
  `voting_mode` (`coefficient|one_unit_one_vote`), `notes`.
- `assembly_calls`: convocatorias formales con `published_at`, `channels`
  (JSONB), `agenda` (text/JSON).
- `assembly_attendances`: `assembly_id`, `unit_id`,
  `represented_by_user_id`, `arrival_at`, `is_remote`, `coefficient_at_event`.
- `assembly_proxies`: poderes registrados; `assembly_id`, `grantor_user_id`,
  `proxy_user_id`, `unit_id`, `document_url`, `validated_at`,
  `validated_by_user_id`.
- `assembly_motions`: `assembly_id`, `title`, `description`,
  `decision_type` (`simple|qualified|special`),
  `voting_method` (`secret|nominal`), `opens_at`, `closes_at`, `status`.
- `votes`: `motion_id`, `voter_user_id`, `unit_id`, `coefficient_used`,
  `option` (yes|no|abstain|other_id), `cast_at`, `prev_vote_hash`,
  `vote_hash`, `nonce`.
- `vote_evidence`: `vote_id`, `prev_vote_hash`, `vote_hash`, `payload_json`,
  `client_ip`, `user_agent`, `ntp_offset_ms`, `created_at` (append-only).
- `acts`: `assembly_id`, `body_md`, `pdf_url`, `pdf_hash`, `status`
  (`draft|signed|archived`).
- `act_signatures`: `act_id`, `signer_user_id`, `role` (`president|
  secretary|witness`), `signed_at`, `signature_method`, `evidence_hash`.
- `assemblies_outbox_events`.

## 6. Endpoints

- `POST /assemblies` — `assembly.create`
- `GET /assemblies` — `assembly.read`
- `GET /assemblies/:id` — `assembly.read`
- `POST /assemblies/:id/call` — `assembly.call`
- `POST /assemblies/:id/start` — `assembly.run`
- `POST /assemblies/:id/close` — `assembly.run`
- `POST /assemblies/:id/attendances` — `assembly.attend`
- `POST /assemblies/:id/proxies` — `assembly.proxy.register`
- `POST /assemblies/:id/motions` — `assembly.run`
- `POST /motions/:id/open-voting` / `close-voting` — `assembly.run`
- `POST /motions/:id/votes` — `assembly.vote`
- `GET /motions/:id/results` — segun visibility (live vs cierre)
- `POST /assemblies/:id/act` — `assembly.act.draft`
- `POST /acts/:id/sign` — `assembly.act.sign`
- `GET /acts/:id/pdf` — `assembly.read`

Errores RFC 7807.

## 7. Permisos nuevos a registrar

- `assembly.create` / `assembly.read` / `assembly.run`
- `assembly.call`
- `assembly.attend`
- `assembly.proxy.register` / `assembly.proxy.validate`
- `assembly.vote`
- `assembly.act.draft` / `assembly.act.sign`

## 8. Casos extremos

- Voto duplicado mismo motion: rechazado por UNIQUE
  (`motion_id`, `voter_user_id`).
- Apoderado excede limite: rechazo en registro de poder.
- Quorum no alcanzado: marcar asamblea como `quorum_failed`, no permitir
  votaciones.
- Modificacion de voto: append nueva fila en `votes` con
  `prev_vote_hash` apuntando al previo del mismo voter+motion.
- Hash chain roto (manipulacion): job de auditoria detecta y alerta.
- Inquilino quiere votar sin poder: rechazo `403`.
- Asamblea virtual con caida de internet: voto pendiente `pending_sync`
  con timestamp local + reconciliacion.
- Acta firmada y luego corregida: NO se modifica; se anexa addendum.

## 9. Operaciones transaccionales / idempotentes

- Voto: TX que (1) calcula `prev_vote_hash` (ultimo del motion),
  (2) computa `vote_hash`, (3) inserta `votes` y `vote_evidence`,
  (4) emite outbox. SELECT FOR UPDATE sobre ultimo hash del motion para
  serializar.
- Cierre votacion: TX que sella `motion.status='closed'`, calcula
  totales, los persiste en `motions.results`.
- Firma de acta: TX que (1) calcula hash del PDF, (2) inserta
  `act_signatures`, (3) marca `acts.status='signed'`. Trigger anti-UPDATE
  sobre `acts` `signed/archived`.

## 10. Configuracion por tenant

- `assemblies.voting_mode` (`coefficient|one_unit_one_vote`, default
  `coefficient`)
- `assemblies.max_proxies_per_attendee` (int, default 1)
- `assemblies.advance_call_days_ordinary` (int, default 15)
- `assemblies.advance_call_days_extraordinary` (int, default 5)
- `assemblies.exclude_overdue_voters` (bool, default true)
- `assemblies.quorum_simple_pct` (decimal, default 0.51)
- `assemblies.quorum_qualified_pct` (decimal, default 0.70)
- `assemblies.quorum_special_pct` (decimal, default 0.80)
- `assemblies.signature_method` (`simple_otp|simple_traceable`, default
  `simple_traceable`)
- `assemblies.allow_tenant_vote_with_proxy` (bool, default true)
- `assemblies.act_archive_years` (int, default 10)

## 11. Notificaciones / eventos

- `assembly.called` -> todos los propietarios (email + push +
  cartelera).
- `assembly.starting_soon` -> 1h antes.
- `assembly.motion_opened` -> asistentes.
- `assembly.motion_closed` -> asistentes con resultado.
- `assembly.act_signed` -> propietarios + auditor.

## 12. Reportes / metricas

- Padron de votantes con coeficientes.
- Resultados por motion (votos + coeficientes).
- Verificacion de hash chain (informe de integridad).
- Acta PDF firmada con anexos (asistencia, poderes, evidencias).
- Export para administracion municipal/camara comercio.

## 13. Riesgos y mitigaciones

- **Voto duplicado**: UNIQUE + hash chain.
- **Manipulacion de votos**: hash chain + auditoria periodica.
- **Falta firma legal**: Ley 527 obliga trazabilidad — almacenamos IP,
  user_agent, timestamp NTP, hash payload.
- **Asamblea virtual sin red**: cliente offline-first + reconciliacion al
  reconectar (V2).
- **Conservacion**: archivo S3 con object lock + retencion 10 anos.

## 14. Multi-agente sugerido

5 agentes paralelos:
- A: schema + migraciones + triggers anti-UPDATE en actas firmadas.
- B: padron + convocatoria + asistencia + apoderados.
- C: votacion + integridad (hash chain) + sealing motion.
- D: acta + PDF + firma + archivado.
- E: handlers + OpenAPI + notificaciones.

## 15. DoD adicional especifico

- [ ] Quorum se calcula con coeficientes correctamente.
- [ ] Voto duplicado rechazado por UNIQUE; hash chain integro.
- [ ] Hash chain verificable por job auditoria (tampering -> alerta).
- [ ] Acta PDF generada incluye anexos (asistencias + poderes +
  evidencias).
- [ ] Inquilinos NO pueden votar sin poder valido (test explicito).
- [ ] Apoderados no exceden limite configurado por tenant.
- [ ] Trigger anti-UPDATE sobre actas firmadas activo.
- [ ] OpenAPI completo, RFC 7807.
