# Runbook operativo — SaaS Propiedad Horizontal

Este documento es la referencia rapida para operar el sistema en piloto y
mas adelante. Esta organizado por escenario.

> Convencion: los comandos asumen que estas en la raiz del repositorio
> (`/workspace`) y que tienes acceso al cluster Postgres del Control
> Plane y al de cada tenant via las URLs en el secret manager.

## 1. Inventario rapido

| Componente | Donde vive | Como lo despliego |
|-----------|------------|--------------------|
| API Go | `apps/api/cmd/api` | binario `./api`, container `ghcr.io/saas-ph/api:<tag>` |
| Web Next | `apps/web` | Vercel/CDN o container Node 22 |
| Movil Expo | `apps/mobile` | EAS Build / OTA updates |
| Postgres Control Plane | tabla `tenants` | gestionado (RDS/Cloud SQL) |
| Postgres por Tenant | una DB aislada | gestionado, naming `ph_tenant_<slug>` |
| Postgres Tenant Template | clonado al provisionar | una DB "plantilla" en el cluster del Data Plane |

## 2. Provisioning de un tenant nuevo

1. **Reservar slug**: subdominio en `<slug>.dominio.com`. Validar contra
   regex `^[a-z0-9](-?[a-z0-9])*$`, longitud 1-63.
2. **Crear DB**: `CREATE DATABASE ph_tenant_<slug> WITH TEMPLATE ph_tenant_template OWNER ph;`.
3. **Migrar**: `migrate -path migrations/tenant -database "$URL" up`.
4. **Seed inicial**: `psql -f migrations/tenant/seed_001_roles_permissions.up.sql "$URL"`.
5. **Crear tenant en Control Plane**: `INSERT INTO tenants (slug, display_name, database_url, status) VALUES ('<slug>','<Nombre>','<URL>','provisioning');`.
6. **Crear tenant_admin inicial**: insertar en `users` un superusuario operativo (con MFA pendiente). Comunicar credenciales temporales por canal seguro.
7. **Activar**: `UPDATE tenants SET status='active', activated_at=now() WHERE slug='<slug>';`.
8. **Health check**: `curl https://<slug>.dominio.com/tenant/ready` debe responder 200.

Tiempo objetivo: <5 minutos desde paso 1 a paso 8 con scripts.

## 3. Backup y restore

### Backup diario por tenant

```bash
pg_dump --format=custom \
        --no-owner --no-privileges \
        --file=/backups/$(date +%F)/ph_tenant_<slug>.dump \
        "$URL"
```

Programar via cron del operador o servicio gestionado.
Retencion: 30 dias en almacenamiento estandar + 1 ano en frio.

### Restore por tenant

```bash
# 1. Crear DB temporal vacia.
createdb ph_tenant_<slug>_restore
# 2. Restore.
pg_restore --no-owner --no-privileges \
           --dbname=ph_tenant_<slug>_restore \
           /backups/<fecha>/ph_tenant_<slug>.dump
# 3. Switch atomico (cambiar database_url en Control Plane).
UPDATE tenants
   SET database_url = '<URL_RESTORE>'
 WHERE slug = '<slug>';
# 4. Invalidar cache de pools (reiniciar API o llamar endpoint /admin/invalidate-tenant).
```

### Backup del Control Plane

```bash
pg_dump --format=custom --file=/backups/control-plane-$(date +%F).dump "$CENTRAL_URL"
```

## 4. Rotacion de logs y observabilidad

- **Logs estructurados** salen a stdout en JSON (slog). El operador del
  pod los captura via su agregador (Loki, CloudWatch, Stackdriver).
- **Retencion sugerida**: 14 dias online, 90 dias en frio.
- **Niveles**: error (incidentes), warn (degradacion), info (operacion
  normal). En piloto deja `LOG_LEVEL=info`.
- **request_id** se inyecta por middleware y se propaga en todos los
  logs del request — usalo para correlacionar.
- **OpenTelemetry**: el codigo emite spans; conectar a Tempo/Jaeger/X-Ray
  via `OTEL_EXPORTER_OTLP_ENDPOINT`.

### Alertas minimas (piloto)

| Alerta | Trigger | Severity |
|--------|---------|----------|
| API 5xx burst | `>5 errores 500 en 1 min` | high |
| Login lockouts | `>20 lockouts en 5 min` (posible ataque) | warn |
| Postgres central down | ping `/ready` 200 -> fail por 3 ciclos | critical |
| Tenant DB down | `/tenant/ready` falla por slug | high (notificar al tenant) |
| Disco DB | `> 80%` | warn |
| Outbox stuck | `outbox_events` con `delivered_at IS NULL AND created_at < now() - interval '10 min'` | high |

## 5. On-call y escalado

- **Turnos**: rotacion semanal entre 2 ingenieros minimo. Compartir un
  numero unico via PagerDuty/Opsgenie.
- **Severidad** dispara cuando se cumpla cualquiera de:
  - critical: API caida >2 min, fuga de datos sospechada, perdida de
    transaccion confirmada.
  - high: degradacion afectando > 1 tenant, blacklist circumvented,
    audit trigger fallando.
  - warn: alertas no bloqueantes, intentos de fuerza bruta detectados.
- **Runbook por incidente**: cada alerta linkea aqui (TODO en piloto).

## 6. Hardening aplicado en Fase 7

- Tabla `audit_logs` con trigger anti-modificacion: cualquier UPDATE o
  DELETE genera `RAISE EXCEPTION` con `ERRCODE='check_violation'`.
- Indices compuestos sobre packages, visitor_entries, user_role_assignments,
  announcements.
- Lockout de login: 5 intentos fallidos -> 15 minutos bloqueado (modulo
  identity, Fase 2 — sigue vigente).
- RateLimit en memoria global (50 rps, burst 100). En produccion mover a
  Redis si el cluster API escala horizontalmente.

## 7. Rotacion de secretos

- `JWT_SIGNING_KEY` (Ed25519): rotar trimestralmente. Fase de transicion
  con `kid` en el header del JWT (TODO: implementar JWKS multi-key).
- DB passwords: rotar semestralmente. Postgres soporta `ALTER ROLE`
  sin reinicio.
- Secretos en `tenant_settings.value` JSONB: actualmente texto plano;
  envolverlos con KMS pre-paso a produccion.

## 8. Comandos de troubleshooting frecuentes

```bash
# Ver sesiones activas de un usuario
psql "$TENANT_URL" -c "
  SELECT id, user_id, created_at, expires_at, revoked_at
    FROM user_sessions
   WHERE user_id = '<uuid>'
   ORDER BY created_at DESC LIMIT 20;"

# Ver paquetes pendientes > 3 dias
psql "$TENANT_URL" -c "
  SELECT id, unit_id, recipient_name, received_at
    FROM packages
   WHERE status = 'received' AND deleted_at IS NULL
     AND received_at < now() - interval '3 days';"

# Ver outbox atascado
psql "$TENANT_URL" -c "
  SELECT id, event_type, attempts, last_error, created_at
    FROM package_outbox_events
   WHERE delivered_at IS NULL
   ORDER BY created_at LIMIT 20;"

# Forzar reset de lockout (administrador)
psql "$TENANT_URL" -c "
  UPDATE users SET locked_until = NULL, failed_login_attempts = 0
   WHERE id = '<uuid>';"

# Audit log: que cambio en una entidad
psql "$TENANT_URL" -c "
  SELECT occurred_at, actor_user_id, action, before, after
    FROM audit_logs
   WHERE entity_type = '<table>' AND entity_id = '<uuid>'
   ORDER BY occurred_at DESC LIMIT 50;"
```

## 9. Checklist pre-piloto (Go-live)

- [ ] Backups configurados y un restore probado en staging.
- [ ] Alertas conectadas a PagerDuty.
- [ ] Indices revisados con `EXPLAIN ANALYZE` sobre datos sinteticos.
- [ ] Limites de rate limit y lockout validados.
- [ ] Secretos en KMS, no en `.env`.
- [ ] Runbook revisado por el ingeniero on-call.
- [ ] Plan de rollback documentado (downgrade de migracion + restore desde dump).
- [ ] Soporte L1 entrenado en este runbook.
