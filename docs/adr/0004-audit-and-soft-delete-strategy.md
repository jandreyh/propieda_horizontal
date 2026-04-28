# ADR 0004 — Auditoria y soft-delete

- **Estado:** Accepted
- **Fecha:** 2026-04-28
- **Autor:** Senior Software Architect
- **Stack:** Go 1.23, chi, pgx + sqlc, PostgreSQL 18
- **Ambito:** SaaS multi-tenant de Propiedad Horizontal (Control Plane + Tenant DB por copropiedad)

---

## 1. Contexto

El dominio de Propiedad Horizontal exige trazabilidad estricta sobre quien, cuando y como modifico cada entidad operativa: residentes, unidades, paquetes, visitantes, asignaciones de roles, configuraciones de seguridad, expensas. Las regulaciones aplicables (habeas data, normativa contable y reglamentos de copropiedad) y la naturaleza adversarial de algunos eventos (disputas entre copropietarios, reclamos por paqueteria, reversion de cargos) obligan a:

1. **No perder informacion historica** ante una eliminacion accidental o malintencionada.
2. **Reconstruir el estado** de cualquier entidad en un punto del tiempo durante investigaciones.
3. **Detectar concurrencia** entre operadores simultaneos (porteria, administracion, residente desde la app).
4. **Diferenciar auditoria operativa** (Tenant DB) de **auditoria de plataforma** (Control Plane: superadmin, impersonation, lifecycle de tenants).

Se descartan estrategias destructivas por defecto. Los borrados duros son excepcionales y requieren ADR especifico por tabla con justificacion (ej.: purga GDPR/habeas data tras vencimiento de retencion).

---

## 2. Decision

- **Soft delete generalizado** en todas las tablas operativas del Tenant DB. La columna `deleted_at IS NOT NULL` marca el registro como borrado logicamente. Hard delete prohibido salvo ADR explicito por tabla.
- **Esquema estandar OBLIGATORIO** en cada tabla operativa (Tenant DB y Control Plane cuando aplique):

  ```sql
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  status      TEXT        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at  TIMESTAMPTZ NULL,
  created_by  UUID        NULL REFERENCES users(id),
  updated_by  UUID        NULL REFERENCES users(id),
  deleted_by  UUID        NULL REFERENCES users(id),
  version    INTEGER      NOT NULL DEFAULT 1
  ```

- **Indice parcial estandar** en cada tabla: `CREATE INDEX ... ON t (...) WHERE deleted_at IS NULL` para queries del hot path.
- **Concurrencia optimista** mediante `version`: todo `UPDATE` debe incrementar `version` y validar `WHERE version = $expected`. Si el `RowsAffected = 0`, el repo retorna `ErrStaleVersion` y la capa HTTP responde `409 Conflict`.
- **`audit_logs` por Tenant DB**, append-only, con trigger que rechaza `UPDATE` y `DELETE`. Captura `actor_user_id`, `action` (`user.login`, `role.assign`, `package.deliver`, `permission.update`, ...), `entity_type`, `entity_id`, `before` JSONB, `after` JSONB, `ip`, `user_agent`, `occurred_at`.
- **`platform_audit_logs` en Control Plane** con la misma forma append-only para acciones de superadmin, impersonation y tenant lifecycle (provision, suspension, eliminacion).
- **Trazabilidad de permisos**: cualquier mutacion sobre `role_permissions` (o tablas equivalentes) emite obligatoriamente un evento de auditoria con snapshot completo `before`/`after` del set de permisos del rol.
- **Retencion minima 5 anios** para `audit_logs` y `platform_audit_logs`. Particionado **mensual** por `occurred_at` (`PARTITION BY RANGE`) cuando una tabla supere 50M filas o 20 GB.
- **Convencion de repos sqlc**: todos los `SELECT` operativos incluyen `AND deleted_at IS NULL`. Existe variante `*WithDeleted` solo para casos de auditoria/restore explicitos.
- **Restore** se modela como `UPDATE ... SET deleted_at = NULL, deleted_by = NULL, version = version + 1` y emite evento `entity.restore`.

---

## 3. Consecuencias

**Positivas**

- Recuperacion total ante borrado accidental sin recurrir a backups.
- Auditoria forense con `before`/`after` reconstruible sin event store dedicado.
- Concurrencia optimista barata (un INTEGER), sin row locks de larga duracion.
- Modelo homogeneo: generadores (sqlc, codegen interno) pueden emitir CRUD seguro sin casos especiales.

**Negativas / mitigaciones**

- **Espacio en disco**: las tablas crecen monotonamente. Mitigacion: indices parciales `WHERE deleted_at IS NULL`, `pg_partman` mensual en `audit_logs`, job de archivado a almacenamiento frio (`audit_logs_cold` o S3 Parquet) tras 24 meses dejando metadata referenciable.
- **Filtro `deleted_at IS NULL` olvidable**: riesgo de filtrar registros borrados a usuarios. Mitigacion en orden de preferencia:
  1. **Convencion en repos sqlc**: ningun `SELECT` operativo se escribe sin el filtro; revisado en code review y por linter custom (`scripts/lint/sqlc-soft-delete.go`).
  2. **Vistas SQL** `v_<tabla>` que ya aplican el filtro, usadas por queries ad-hoc y reportes BI.
  3. Vista materializada solo si una agregacion concreta lo justifica por performance (no por defecto: invalidacion costosa).
- **JSONB `before`/`after` voluminoso**: limitar a campos relevantes mediante allowlist por `entity_type`; comprimido con `TOAST` por defecto.
- **`audit_logs` append-only complica fixes**: aceptado por diseno. Correcciones se modelan como nuevos eventos `action = '*.correction'` referenciando el evento original.

---

## 4. Alternativas consideradas

- **Hard delete + tabla `tombstones`**: descartado. Duplica el modelo, exige sincronizacion entre dos tablas y dos rutas de query, y pierde referencias FK al borrar fisicamente.
- **Event sourcing puro**: descartado para MVP. Aporta reconstruccion temporal pero impone CQRS, projections, snapshots y reescritura de toda la capa de persistencia. Coste no justificado frente a `audit_logs` + soft delete + `version`. Reevaluable post-MVP para subdominios de alta auditabilidad (cobranza, asambleas).
- **Triggers genericos de auditoria sobre todas las tablas**: descartado como unica fuente. Capturan datos crudos pero pierden el contexto semantico (`action`, `actor_user_id` real, IP, user-agent). Se usan solo como red de seguridad complementaria, no como fuente primaria.
- **`xmin`/system columns de Postgres para concurrencia**: descartado. No portable, no expresable en API publica; `version` explicito viaja con el DTO al cliente.

---

## 5. Implicaciones

### 5.1 DDL de `audit_logs` con trigger anti-modificacion

```sql
CREATE TABLE audit_logs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id   UUID        NULL REFERENCES users(id),
    action          TEXT        NOT NULL,
    entity_type     TEXT        NOT NULL,
    entity_id       UUID        NULL,
    before          JSONB       NULL,
    after           JSONB       NULL,
    ip              INET        NULL,
    user_agent      TEXT        NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (occurred_at);

CREATE INDEX idx_audit_entity   ON audit_logs (entity_type, entity_id, occurred_at DESC);
CREATE INDEX idx_audit_actor    ON audit_logs (actor_user_id, occurred_at DESC);
CREATE INDEX idx_audit_action   ON audit_logs (action, occurred_at DESC);

CREATE OR REPLACE FUNCTION audit_logs_immutable()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only: % rejected', TG_OP
        USING ERRCODE = 'check_violation';
END;
$$;

CREATE TRIGGER trg_audit_logs_no_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_immutable();

CREATE TRIGGER trg_audit_logs_no_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_immutable();

-- Particiones mensuales gestionadas por pg_partman o job propio.
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
```

`platform_audit_logs` en Control Plane sigue el mismo patron, sustituyendo `entity_type` por `tenant_id` + `entity_type`.

### 5.2 UPDATE con chequeo de `version` (sqlc)

```sql
-- name: UpdatePackageStatus :execrows
UPDATE packages
SET status      = $2,
    updated_at  = now(),
    updated_by  = $3,
    version     = version + 1
WHERE id        = $1
  AND version   = $4
  AND deleted_at IS NULL;
```

Soft delete equivalente:

```sql
-- name: SoftDeletePackage :execrows
UPDATE packages
SET deleted_at  = now(),
    deleted_by  = $2,
    updated_at  = now(),
    updated_by  = $2,
    version     = version + 1
WHERE id        = $1
  AND version   = $3
  AND deleted_at IS NULL;
```

### 5.3 Helper Go en `internal/repo`

```go
// ErrStaleVersion indica conflicto de concurrencia optimista.
var ErrStaleVersion = errors.New("repo: stale version, reload entity")

// CheckAffected traduce el RowsAffected de un UPDATE con version check.
// Convencion: 0 filas afectadas -> conflicto; 1 -> OK; >1 -> invariante rota.
func CheckAffected(n int64) error {
    switch {
    case n == 0:
        return ErrStaleVersion
    case n > 1:
        return fmt.Errorf("repo: unexpected rows affected: %d", n)
    default:
        return nil
    }
}

// WriteAudit inserta un evento de auditoria dentro de la misma tx que la mutacion.
// Obligatorio para mutaciones de role_permissions, users, tenants y entidades $.
func WriteAudit(ctx context.Context, q *db.Queries, e AuditEvent) error {
    return q.InsertAuditLog(ctx, db.InsertAuditLogParams{
        ActorUserID: e.ActorID,
        Action:      e.Action,
        EntityType:  e.EntityType,
        EntityID:    e.EntityID,
        Before:      e.Before, // []byte JSONB
        After:       e.After,
        Ip:          e.IP,
        UserAgent:   e.UserAgent,
    })
}
```

Regla: toda mutacion pasa por un `Service` que abre `tx`, ejecuta el `UPDATE`/`INSERT`/soft delete con `version`, y antes del `Commit` invoca `WriteAudit`. Si el `audit_logs` falla, la tx hace rollback: no hay mutacion sin rastro.

### 5.4 Checklist por tabla operativa nueva

- [ ] Incluye los 9 campos estandar.
- [ ] Indice parcial `WHERE deleted_at IS NULL` sobre los lookups del hot path.
- [ ] Queries sqlc sin variantes "raw"; existe `*WithDeleted` solo si se justifica.
- [ ] `UPDATE`/soft delete con `version = $expected`.
- [ ] Service emite `audit_logs` con `before`/`after` dentro de la tx.
- [ ] Migracion documentada en ADR si requiere hard delete.
