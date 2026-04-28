# ADR 0005 — Transaccionalidad, idempotencia, outbox

- **Estado:** Accepted
- **Fecha:** 2026-04-28
- **Componentes:** API Go (chi, pgx/v5, sqlc), PostgreSQL 18, worker de relay, integraciones externas (WhatsApp/SMS/Email, pasarelas de pago, webhooks).

## Contexto

El SaaS multi-tenant de Propiedad Horizontal coordina operaciones que tocan multiples agregados (asambleas, votaciones, cartera, reservas, paquetes) y, ademas, dispara efectos externos (notificaciones, pagos). En este escenario son inaceptables tres clases de fallo: (a) escrituras parciales que dejan un agregado en estado invalido, (b) duplicacion de efectos al reintentar peticiones HTTP o webhooks entrantes, y (c) perdida silenciosa de eventos cuando el broker externo esta caido. ADR 0004 definio control de concurrencia optimista con columna `version`; este ADR estandariza como se aplica esa estrategia dentro de transacciones explicitas, como se garantizan operaciones idempotentes en la frontera HTTP, y como se desacoplan los efectos externos via outbox.

## Decision

- **Transaccion explicita obligatoria** en toda operacion de negocio que (i) escriba en mas de una tabla, (ii) emita un evento de dominio, o (iii) cruce agregados. La transaccion (`pgx.Tx`) se abre en la capa *use case / service* y se inyecta hacia abajo. Los handlers HTTP nunca abren transacciones.
- **Repositorios polimorficos sobre `Querier`**: cada repo recibe un `sqlc.Querier` (interfaz comun a `*pgx.Conn`, `*pgxpool.Pool` y `pgx.Tx`). Permite reutilizar el mismo metodo de repo dentro o fuera de transaccion sin duplicar codigo. El servicio decide el alcance transaccional con un helper `db.WithTx(ctx, pool, opts, fn)`.
- **Bloqueo optimista** (ADR 0004): los `UPDATE` van con `WHERE id = $1 AND tenant_id = $2 AND version = $3` y `RETURNING version`. Si `RowsAffected = 0` el repo devuelve `ErrConflict` (HTTP 409). Toda mutacion incrementa `version = version + 1` dentro de la misma sentencia.
- **Idempotency-Key obligatoria** en endpoints externos sensibles: `POST /webhooks/*`, futuros `POST /payments/*`, y `POST /packages` (recepcion de paquetes en porteria). El cliente envia header `Idempotency-Key: <uuid>`. Un middleware materializa la respuesta en `idempotency_records` (clave compuesta por `tenant_id + endpoint + key`) con ventana de 24 h. Repeticion -> se devuelve la respuesta original byte-a-byte (mismo status, body y headers relevantes); colision con cuerpo distinto -> HTTP 422 `idempotency_key_reused`.
- **Outbox pattern** para todo efecto externo (notificaciones Fase 15, callbacks a pasarelas, integraciones): el caso de uso inserta una fila en `outbox_events` dentro de la *misma* `pgx.Tx` que muta el dominio. Un worker (`outbox-relay`) lee filas pendientes con `FOR UPDATE SKIP LOCKED`, publica al broker, y marca `delivered_at`. Garantia: **at-least-once**; los consumidores deben ser idempotentes usando `event_id`.
- **Reintentos seguros**: la API expone (y registra) la `Idempotency-Key` para que el cliente pueda reintentar sin duplicar. El worker outbox aplica backoff exponencial (`attempts`, `next_attempt_at`) y, tras N=10 fallos, mueve el evento a `outbox_events_dead`.
- **Aislamiento**: por defecto `READ COMMITTED`. Se eleva a `SERIALIZABLE` para: sorteos de parqueadero, transiciones de estado de asamblea (abrir/cerrar/quorum), cierre de periodo contable y conciliacion de pagos. El helper `WithTx` recibe el nivel como parametro y reintenta automaticamente sobre `40001 serialization_failure` (max 3 intentos).

## Consecuencias

**Positivas**

- Consistencia fuerte intra-agregado: nunca hay escrituras parciales.
- Reintentos HTTP y de webhook son seguros sin coordinacion adicional con el cliente.
- Desacople total con el broker: una caida de Rabbit/SQS/Twilio no rompe el flujo de negocio; el worker drena al recuperarse.
- `version` + `SERIALIZABLE` selectivo cubre tanto contienda baja (mayoria de casos) como invariantes globales criticos sin pagar el costo en todo el sistema.

**Negativas**

- Complejidad operativa: el worker `outbox-relay` es un nuevo proceso que requiere monitoreo (lag, dead-letter, tasa de reintentos).
- Latencia anadida en notificaciones (segundos en vez de milisegundos) por el ciclo insert -> poll -> publish.
- Doble escritura logica (dominio + outbox) incrementa I/O en PostgreSQL; mitigado con indice parcial sobre `delivered_at IS NULL`.
- Tabla `idempotency_records` crece linealmente; requiere job de purga 24 h.

## Alternativas consideradas

- **Two-Phase Commit (XA) entre Postgres y broker**: descartado. Los brokers objetivo (RabbitMQ, SQS, proveedores SMS) no soportan XA o lo hacen de forma fragil; ademas penaliza la latencia y agrega un coordinador.
- **Kafka transactional producer / exactly-once**: descartado en Fase 15. Anade un componente pesado para un volumen de eventos modesto y no resuelve la idempotencia del *consumer* externo (Twilio, pasarelas) que de todas formas exige claves idempotentes.
- **Sin Idempotency-Key, confiar en deduplicacion del cliente**: descartado. Webhooks de terceros y conexiones moviles inestables generan reentradas; trasladar la responsabilidad al cliente es contractualmente debil y produce cobros/notificaciones duplicados.
- **CDC (Debezium sobre WAL) en lugar de outbox**: viable a futuro pero introduce dependencia operativa fuerte (Kafka Connect, schemas) desproporcionada para Fase 15.

## Implicaciones

### Esquema SQL minimo

```sql
-- NOTA: ambas tablas viven en cada **Tenant DB** (data plane). El tenant
-- es implicito porque la base entera ya es del tenant — NO se incluye
-- columna `tenant_id` (ver CLAUDE.md y ADR 0001).
CREATE TABLE idempotency_records (
    endpoint        TEXT         NOT NULL,
    idempotency_key TEXT         NOT NULL,
    request_hash    BYTEA        NOT NULL,
    status_code     INT          NOT NULL,
    response_body   JSONB        NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ  NOT NULL DEFAULT now() + INTERVAL '24 hours',
    PRIMARY KEY (endpoint, idempotency_key)
);
CREATE INDEX idx_idem_expires ON idempotency_records (expires_at);

CREATE TABLE outbox_events (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type   TEXT         NOT NULL,
    aggregate_id     UUID         NOT NULL,
    event_type       TEXT         NOT NULL,
    payload          JSONB        NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    next_attempt_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attempts         INT          NOT NULL DEFAULT 0,
    delivered_at     TIMESTAMPTZ,
    last_error       TEXT
);
CREATE INDEX idx_outbox_pending
    ON outbox_events (next_attempt_at)
    WHERE delivered_at IS NULL;
```

### Pseudocodigo del worker `outbox-relay`

```go
func (w *Relay) tick(ctx context.Context) error {
    return db.WithTx(ctx, w.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(q sqlc.Querier) error {
        rows, err := q.LockPendingOutbox(ctx, sqlc.LockPendingOutboxParams{
            Limit: 100, Now: time.Now(),
        }) // SELECT ... WHERE delivered_at IS NULL AND next_attempt_at <= $now
           // ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $limit
        if err != nil { return err }

        for _, ev := range rows {
            if err := w.broker.Publish(ctx, ev.EventType, ev.ID, ev.Payload); err != nil {
                backoff := expBackoff(ev.Attempts)         // 5s, 15s, 45s, ...
                _ = q.MarkOutboxFailed(ctx, ev.ID, err.Error(), time.Now().Add(backoff))
                if ev.Attempts+1 >= maxAttempts { _ = q.MoveOutboxToDLQ(ctx, ev.ID) }
                continue
            }
            _ = q.MarkOutboxDelivered(ctx, ev.ID)         // SET delivered_at = now()
        }
        return nil
    })
}
```

El worker corre en N replicas; `FOR UPDATE SKIP LOCKED` permite paralelismo seguro sin coordinacion externa.

### Ejemplo de uso del `Querier` en repo y servicio

```go
// repo recibe Querier (Conn|Pool|Tx); no abre transaccion.
func (r *AssemblyRepo) UpdateState(ctx context.Context, q sqlc.Querier, in UpdateStateIn) error {
    n, err := q.UpdateAssemblyState(ctx, sqlc.UpdateAssemblyStateParams{
        ID: in.ID, TenantID: in.TenantID, NewState: in.State, ExpectedVersion: in.Version,
    })
    if err != nil { return err }
    if n == 0    { return domain.ErrConflict }
    return nil
}

// service abre la transaccion y compone repo + outbox.
func (s *AssemblyService) Close(ctx context.Context, cmd CloseAssembly) error {
    return db.WithTx(ctx, s.pool, pgx.TxOptions{IsoLevel: pgx.Serializable}, func(q sqlc.Querier) error {
        if err := s.repo.UpdateState(ctx, q, UpdateStateIn{
            ID: cmd.ID, TenantID: cmd.TenantID, State: "closed", Version: cmd.Version,
        }); err != nil { return err }

        return s.outbox.Enqueue(ctx, q, OutboxEvent{
            TenantID: cmd.TenantID, AggregateType: "assembly", AggregateID: cmd.ID,
            EventType: "assembly.closed.v1", Payload: cmd.AsPayload(),
        })
    })
}
```

El middleware de Idempotency-Key envuelve el handler: si encuentra un registro vigente con mismo `request_hash` corta antes del servicio; si no, ejecuta el handler y persiste la respuesta en `idempotency_records` dentro de la misma `pgx.Tx` que el cambio de dominio (mismo `WithTx`), garantizando atomicidad entre efecto y registro de idempotencia.
