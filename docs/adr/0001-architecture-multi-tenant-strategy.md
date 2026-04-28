# ADR 0001 — Estrategia multi-tenant: DB por tenant

- **Estado:** Accepted
- **Fecha:** 2026-04-28
- **Autor:** Equipo Plataforma

## 1. Contexto

Construimos un SaaS de Propiedad Horizontal (PH) para conjuntos residenciales en Colombia (administracion, asambleas, cuotas, PQRS, reservas, accesos). El backend es un Modular Monolith en Go 1.26+, `chi`, `pgx` + `sqlc`, sobre PostgreSQL 18.

**Problema:** definir como aislar los datos de cada conjunto (tenant) garantizando confidencialidad, cumplimiento normativo y operacion sostenible.

**Restricciones y requisitos:**

- **Aislamiento fuerte:** los datos de un conjunto no deben poder filtrarse a otro ni siquiera por bug de aplicacion (defense in depth). Una asamblea, un acta o un estado de cuenta jamas debe cruzarse.
- **Cumplimiento legal:** Ley 1581 de 2012 (habeas data / proteccion de datos personales) y Ley 675 de 2001 (regimen de PH). Necesitamos poder responder solicitudes de supresion/exportacion por titular y por conjunto, y aislar custodias.
- **Escalabilidad por tenant:** la carga es heterogenea (20 a 5.000 unidades). No queremos que un conjunto grande degrade a los demas (noisy neighbor).
- **Billing por uso:** medicion por tenant (storage, requests, modulos activos).
- **On-prem futuro:** algunos conjuntos grandes o administradoras pueden exigir despliegue dedicado o residencia local de datos. La arquitectura debe permitirlo sin reescribir el dominio.
- **Operacion:** equipo de plataforma pequeno. Migraciones, backups y observabilidad deben ser automatizables.

## 2. Decision

- **DB-por-tenant:** cada conjunto tiene su propia base PostgreSQL fisica (o logica en cluster compartido en plan basico, pero siempre `database` separada). Ninguna tabla operativa lleva columna `tenant_id`: la base entera *es* el tenant.
- **Separacion Control Plane / Data Plane:** existe una base central (`platform`) que es el Control Plane. Contiene `tenants`, `tenant_domains`, `branding`, `plans`, `subscriptions`, `superadmins`, `platform_audit_log` e `impersonation_grants`. Cada Tenant DB es Data Plane puro y solo conoce su propio dominio (residentes, unidades, cuotas, etc.).
- **Resolucion de tenant por subdominio en web (`conjunto.dominio.com`) y por header `X-Tenant-Slug` en movil**, resuelto en middleware temprano con cache en memoria del pool y metadata. Login se valida contra la Tenant DB; no existe identidad global de usuario final, solo Superadmin vive en el Control Plane.

## 3. Consecuencias

**Positivas**

- Aislamiento fisico: un `GRANT` mal hecho o un `WHERE` olvidado no expone otro tenant. Reduce drasticamente el blast radius de bugs y de incidentes de seguridad.
- Backups, restores y `PITR` por tenant son triviales (`pg_dump`/`pg_restore` por DB). Cumple bien la Ley 1581 ante solicitudes de supresion masiva.
- Migracion a on-prem o a cluster dedicado es mover una DB, no extraer filas.
- Tuning por tenant (indices, `work_mem`, extensiones) sin afectar al resto.
- Billing por uso es directo: metricas a nivel DB.
- Esquema simple: las queries de dominio no cargan predicado `tenant_id` en cada `JOIN`, `sqlc` genera codigo limpio.

**Negativas**

- **Costo operativo:** N bases implican N migraciones, N monitorizaciones, N conexiones. Mitigado con orquestador de migraciones y pool LRU.
- **Migraciones:** desplegar un cambio de schema implica iterar tenants. Requiere herramienta idempotente, observabilidad y *canary tenants*.
- **Cross-tenant analytics dificil:** reportes de plataforma exigen ETL (CDC -> warehouse). Asumido: lo operativo nunca cruza, lo analitico sale del warehouse.
- **Conexiones:** un pool por tenant escala mal a miles. Mitigado con `pgbouncer` en modo transaction y cache LRU de pools en el binario.
- **Provisioning mas complejo** que un `INSERT` en tabla compartida: hay que crear DB, correr migraciones, sembrar datos base, registrar en Control Plane.

## 4. Alternativas consideradas

- **Shared-DB con `tenant_id`:** una sola base, todas las tablas con `tenant_id` y RLS. *Descartada:* el riesgo de fuga por bug de aplicacion o por olvido de `RLS FORCE` es alto; el blast radius cubre toda la base; `pg_dump` selectivo por tenant es costoso; on-prem implica reescritura.
- **Schema-per-tenant:** una DB, N schemas. *Descartada:* sigue siendo un solo `pg_catalog`, un solo `WAL`, un solo punto de falla. PostgreSQL degrada con miles de schemas (catalog bloat, autovacuum). Backups granulares siguen siendo dificiles. Ganancia de aislamiento intermedia, costo similar al de DB-por-tenant.
- **Hibrido (shared para free, dedicated para enterprise):** *descartada como default* por duplicar capa de acceso a datos y mantener dos modelos de seguridad. Se reserva como excepcion futura solo si aparece un plan freemium masivo; hoy no aplica.

## 5. Implicaciones tecnicas

### 5.1 Estructura de migraciones

```
/migrations/
  central/   # Control Plane: tenants, plans, superadmins, audit, impersonation
  tenant/    # Data Plane: residentes, unidades, cuotas, asambleas, PQRS...
```

`central/` corre una vez por despliegue. `tenant/` corre N veces, una por DB registrada en `tenants`, orquestado por un job (`migrate-tenants`) que itera con concurrencia limitada y registra version aplicada en `tenants.schema_version`.

### 5.2 Middleware de resolucion (pseudo-Go)

```go
func TenantResolver(reg *tenant.Registry) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            slug := slugFromHost(r.Host) // conjunto.dominio.com -> "conjunto"
            if slug == "" {
                slug = r.Header.Get("X-Tenant-Slug") // movil
            }
            if slug == "" {
                http.Error(w, "tenant required", http.StatusBadRequest)
                return
            }
            t, err := reg.Lookup(r.Context(), slug) // cache LRU + TTL
            if err != nil || t.Status != "active" {
                http.Error(w, "tenant not found", http.StatusNotFound)
                return
            }
            ctx := tenant.WithContext(r.Context(), t)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### 5.3 Cache de metadata y pools

```go
type Registry struct {
    central *pgxpool.Pool
    meta    *lru.Cache[string, Tenant]   // slug -> metadata (TTL 5 min)
    pools   *lru.Cache[uuid.UUID, *pgxpool.Pool] // tenantID -> pool
    sf      singleflight.Group
}
```

- `meta` evita golpear `platform.tenants` en cada request. Invalidacion por evento (`tenant.updated`) via NOTIFY/LISTEN.
- `pools` mantiene como max ~200 pools calientes; los frios se cierran por LRU. `singleflight` evita stampede al abrir el pool de un tenant nuevo.
- Conexiones reales detras de `pgbouncer` (transaction pooling) para acotar sockets.

### 5.4 Login y sesion

- Login se hace contra la Tenant DB resuelta por subdominio: `POST conjunto.dominio.com/auth/login`. El JWT incluye `tenant_id` y se valida que coincida con el resuelto en cada request.
- Superadmin se autentica contra `platform.superadmins` en `admin.dominio.com`. La impersonation emite un token corto firmado por el Control Plane y registrado en `platform_audit_log` (quien, a que tenant, cuando, por que).

### 5.5 Backups

- `pg_dump` programado por tenant a object storage cifrado, retencion segun plan.
- `PITR` por cluster fisico; restore selectivo a una DB temporal y `pg_restore` al tenant afectado.

### 5.6 Runbook abreviado de provisioning

1. `INSERT` en `platform.tenants` (slug, nombre, plan, region) en estado `provisioning`.
2. `CREATE DATABASE tenant_<uuid>` en el cluster correspondiente al plan/region.
3. Aplicar `/migrations/tenant/` hasta `HEAD`; setear `schema_version`.
4. Seed minimo: roles, modulos del plan, admin inicial del conjunto.
5. Registrar dominio en `platform.tenant_domains` (`conjunto.dominio.com`) y emitir certificado.
6. Cambiar estado a `active`, emitir evento `tenant.created`, invalidar cache de `Registry`.
7. Smoke test automatizado contra `https://conjunto.dominio.com/healthz`.

Fallback: si cualquier paso falla, el job marca `failed`, conserva la DB para diagnostico y notifica a plataforma; nunca queda un tenant `active` a medio provisionar.
