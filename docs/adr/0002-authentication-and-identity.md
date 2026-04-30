# ADR 0002 — Autenticacion e identidad

- **Estado:** **Superseded by [ADR 0007](0007-cross-tenant-identity.md)** (2026-04-30)
- **Fecha original:** 2026-04-28
- **Autor:** Equipo Plataforma (Senior Software Architect)
- **Relacionado:** ADR 0001 (Modelo multi-tenant database-per-tenant), ADR 0003 (RBAC), ADR 0007 (Identidad cross-tenant — supersede del actual).

> **Nota de superseding:** Esta ADR asumio que la identidad operativa
> vivia DENTRO de cada tenant DB (`users` por conjunto). En la fase de
> Discovery POST-MVP el usuario decidio modelar identidad GLOBAL — una
> persona = un `platform_users` en la DB central que aparece en N
> tenants via `tenant_user_links`. ADR 0007 documenta la decision y la
> Fase 16 implementa el cambio (login de 3 factores, selector,
> switch-tenant). El contenido a continuacion se conserva como contexto
> historico pero **no refleja el sistema actual**.

## Contexto

La plataforma SaaS de Propiedad Horizontal (PH) es **multi-tenant con base de datos por tenant** (ver ADR 0001): cada conjunto/edificio tiene su propia base PostgreSQL, y existe ademas una **base central de Control Plane** que conoce el catalogo de tenants y la identidad global del operador SaaS. Los usuarios finales son residentes, propietarios, administradores del conjunto, contadores y vigilantes; ninguno de ellos pertenece simultaneamente a multiples tenants en el mismo rol operativo, y ninguno requiere una identidad global.

Restricciones reales del dominio colombiano de PH:

1. La identidad legal de las personas se acredita por **tipo + numero de documento** (CC, CE, PA, NIT, TI). El email no es universal: residentes mayores, personal de portería y contratistas frecuentemente carecen de correo confiable.
2. Existen dos personajes con identidad **plataforma-global**: el `platform_superadmin` (operador de la empresa SaaS) y, eventualmente, soporte interno. Todos los demas viven dentro de la base de un tenant.
3. Roles con poder financiero u operativo critico (admin de tenant, contador, vigilante) deben llevar MFA obligatorio por compliance (Habeas Data Ley 1581, Circular SFC para administracion de recursos comunes).
4. El stack es Go (chi + pgx + sqlc) sobre PostgreSQL 18; las primary keys del dominio ya estan estandarizadas en UUIDv7.
5. **Regla dura del modelo (CLAUDE.md y ADR 0001):** PROHIBIDO incluir columna `tenant_id` en tablas operativas del Tenant DB. La pertenencia al tenant es **implicita** por la base; un connection pool por tenant garantiza el aislamiento fisico.

El reto: ¿como modelar la identidad para que (a) un mismo numero de cedula pueda existir en multiples tenants sin colision (resuelto trivialmente porque viven en bases distintas), (b) el login sea simple para el usuario final, (c) el superadmin pueda cruzar tenants para soporte/impersonation, y (d) MFA sea aplicable selectivamente por rol?

## Decision

### 1. Particion de identidad

- **No existe identidad global de residente.** Cada Tenant DB tiene su propia tabla `users`. La unicidad del documento se garantiza por `(document_type, document_number)` **dentro de la base del tenant** (no se requiere `tenant_id` como columna porque la base entera es de ese tenant).
- **Solo `platform_superadmin` vive en una tabla aparte** `platform_users` en la base **central** (Control Plane), con identidad global y unicidad por email (obligatorio en este caso). Esta tabla **no lleva `tenant_id`** porque es identidad de plataforma, no pertenece a ningun tenant.
- **Opcional:** si el superadmin necesita acceso explicito por tenant para impersonation/soporte, se modela una tabla auxiliar `platform_user_tenant_grants(platform_user_id, tenant_id, granted_at)` en el Control Plane. No es obligatoria en MVP; el superadmin puede operar con un grant universal por defecto.
- El login de aplicacion siempre ocurre **por tenant**: el subdominio (`acacias.ph.app`) lo resuelve el middleware `TenantResolver` consultando el Control Plane, lo que selecciona el connection pool correcto antes de validar credenciales contra la `users` del Tenant DB.

### 2. Identidad de negocio

- Clave natural: `(document_type, document_number)`, unica dentro de la Tenant DB.
- `document_type` es enum: `CC`, `CE`, `PA`, `NIT`, `TI`.
- `email` es **nullable**, pero si esta presente debe ser unico dentro de la base del tenant.
- `phone_e164` se almacena normalizado para canales secundarios (WhatsApp, SMS OTP de respaldo).

### 3. PK interna

- `id UUID` generado en el servidor con **UUIDv7** (`uuid_generate_v7()` via `pg_uuidv7` o generacion en Go con `github.com/google/uuid` v1.6+). Da orden temporal natural, mejor localidad de indices B-tree y menos fragmentacion que UUIDv4.

### 4. Hash de contrasenas

- **argon2id** como default (`m=64MiB, t=3, p=2`), parametros versionados en columna `password_algo`.
- bcrypt cost 12 permitido solo como fallback de migracion. Rehash transparente en login exitoso si `password_algo` no es el actual.

### 5. MFA

- **Obligatorio** para roles: `platform_superadmin`, `tenant_admin`, `accountant`, `guard`.
- **Opt-in** para `resident`, `owner`, `visitor`.
- Mecanismo primario: **TOTP RFC 6238** (SHA-1, 6 digitos, ventana 30s, drift +/-1).
- Secreto cifrado en reposo con AES-256-GCM (KEK en KMS).
- 10 **recovery codes** de un solo uso, hasheados con argon2id, en tabla `user_mfa_recovery_codes` del Tenant DB.
- WebAuthn queda como evolucion futura (no en este ADR).

### 6. Sesiones y tokens

- **Cookie de sesion** `__Host-ph_session`: `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/`.
- **Access token JWT** corto: 15 minutos, firmado EdDSA (Ed25519), claims minimos.
- **Refresh token opaco** (32 bytes random base64url) con TTL 7 dias, **rotativo**: cada uso emite uno nuevo e invalida el anterior; deteccion de reuso revoca toda la familia.
- Tokens persistidos como `sha256` hash en `user_sessions` del Tenant DB (nunca en claro). Logout = `revoked_at = now()`.
- El JWT lleva `tid` (tenant id) en sus claims: este `tid` es un **identificador de runtime**, no una columna persistida en una tabla operativa. Sirve para validar coherencia entre el subdominio resuelto y el tenant para el que el access token fue emitido.

### 7. Flujo de login en dos pasos

1. `POST /auth/login` recibe `{document_type, document_number, password}` o `{email, password}`. El tenant lo determina el subdominio (`TenantResolver` ya seteo el pool en el contexto). Si las credenciales son validas y el rol exige MFA, responde `{pre_auth_token, mfa_required: true}`. El `pre_auth_token` es JWT de 5 min, audience `mfa`, no autoriza ningun recurso.
2. `POST /auth/mfa/verify` recibe `{pre_auth_token, code}`. Si el TOTP o recovery code es valido emite la cookie de sesion + access token + refresh token.

Si el usuario no requiere MFA, el paso 1 emite directamente la sesion.

## Consecuencias

**Positivas**

- Aislamiento fuerte: un breach en un tenant no compromete identidades de otros (las bases estan fisicamente separadas).
- Esquemas de tabla mas limpios: sin la columna `tenant_id` redundante; los indices son mas pequenos y los queries no necesitan filtros extra.
- El modelo refleja la realidad legal colombiana (documento) sin forzar email.
- UUIDv7 + argon2id + cookies `__Host-` cubren los baselines de OWASP ASVS L2.
- MFA por rol minimiza friccion para residentes manteniendo seguridad donde duele.

**Negativas / costos**

- Un mismo humano con apartamentos en dos conjuntos tendra **dos cuentas distintas** (aceptable: lo pidio el negocio).
- Implementar rotacion de refresh con deteccion de reuso requiere disciplina (familia de tokens, `parent_session_id`).
- argon2id consume CPU/RAM; el endpoint de login debe ir detras de rate limiting (token bucket por IP + por identifier).
- El paso 2 de login obliga a UI de dos pantallas; mas codigo que un login monolitico.
- El middleware `TenantResolver` es critico: si falla la resolucion del subdominio no hay login posible. Debe tener cache local de `slug -> tenant` con TTL razonable.

**Riesgos mitigados**

- Stuffing de credenciales: rate limit + lockout exponencial por `(subdominio, identifier)`.
- Phishing de TOTP: ventana corta + binding del `pre_auth_token` a IP/UA hash.
- Robo de cookie: `SameSite=Strict` + `Secure` + rotacion de refresh.
- Cross-tenant via JWT manipulado: el `tid` del claim se valida contra el tenant resuelto por subdominio en cada request.

## Alternativas consideradas

1. **Tabla `users` global con `tenant_id` en una sola base.** Descartada: contradice el modelo database-per-tenant del ADR 0001 y la regla dura de CLAUDE.md. Ademas obliga a filtros `WHERE tenant_id = $1` en cada query y abre la puerta a fugas cross-tenant por bug de programador.
2. **Identidad federada con OIDC global propio (un IdP central tipo Keycloak/Auth0).** Descartada: agrega un punto de fallo critico, complica el aislamiento por tenant, encarece la operacion para un mercado de PH donde el residente no quiere "una cuenta mas". Reevaluar solo si aparece demanda B2B de SSO empresarial.
3. **Login solo con email.** Descartada: ~25% de la base usuaria estimada (adultos mayores, personal operativo) no tiene email confiable. Excluiria usuarios reales.
4. **MFA opcional para todos.** Descartada: dejar al `tenant_admin` o `accountant` sin MFA es inaceptable; manejan recaudo y sanciones. El costo de UX de TOTP obligatorio para esos roles es marginal frente al riesgo.
5. **JWT de larga duracion sin refresh rotativo.** Descartada: imposible revocar antes de la expiracion sin lista negra global, y el blast radius de un token filtrado es inaceptable.

## Implicaciones tecnicas

### Esquema de las tablas en el Tenant DB

Estas tablas viven **dentro de la base de cada tenant**. La pertenencia al tenant es implicita por la base; **no llevan columna `tenant_id`** ni claves UNIQUE compuestas con ella.

```sql
CREATE TYPE document_type AS ENUM ('CC','CE','PA','NIT','TI');
CREATE TYPE password_algo AS ENUM ('argon2id','bcrypt');

CREATE TABLE users (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v7(),
    document_type   document_type NOT NULL,
    document_number TEXT         NOT NULL,
    email           CITEXT       NULL,
    phone_e164      TEXT         NULL,
    full_name       TEXT         NOT NULL,
    password_hash   TEXT         NOT NULL,
    password_algo   password_algo NOT NULL DEFAULT 'argon2id',
    mfa_enabled     BOOLEAN      NOT NULL DEFAULT FALSE,
    mfa_secret_enc  BYTEA        NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT users_doc_unique
        UNIQUE (document_type, document_number),
    CONSTRAINT users_email_unique
        UNIQUE (email)
);

CREATE TABLE user_sessions (
    id                 UUID        PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id            UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash BYTEA       NOT NULL,            -- sha256(refresh)
    parent_session_id  UUID        NULL,                -- familia de rotacion
    user_agent_hash    BYTEA       NOT NULL,
    ip_inet            INET        NOT NULL,
    issued_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    revoked_at         TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX ON user_sessions (refresh_token_hash);

CREATE TABLE user_mfa_recovery_codes (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash   TEXT        NOT NULL,        -- argon2id
    used_at     TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Esquema en el Control Plane (base central)

Solo el superadmin global vive aqui. La tabla `platform_users` **no lleva `tenant_id`** porque es identidad global de plataforma.

```sql
-- Base central / Control Plane
CREATE TABLE platform_users (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v7(),
    email           CITEXT       NOT NULL UNIQUE,
    full_name       TEXT         NOT NULL,
    password_hash   TEXT         NOT NULL,
    password_algo   password_algo NOT NULL DEFAULT 'argon2id',
    mfa_enabled     BOOLEAN      NOT NULL DEFAULT TRUE,  -- obligatorio
    mfa_secret_enc  BYTEA        NULL,
    status          TEXT         NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Opcional: grants explicitos de superadmin sobre tenants concretos
-- (no requerido en MVP; el superadmin puede tener acceso universal por defecto)
CREATE TABLE platform_user_tenant_grants (
    platform_user_id UUID        NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    granted_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (platform_user_id, tenant_id)
);
```

Nota: la mencion de `tenant_id` en `platform_user_tenant_grants` y en la tabla `tenants` del Control Plane es **legitima** — esas tablas viven en la base central, donde el `tenant_id` es la PK/FK natural del catalogo. La regla solo prohibe `tenant_id` en tablas operativas del **Tenant DB**.

### Pseudocodigo del flujo de login (Go / chi)

```go
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest // (doc o email) + password ; el tenant viene del subdominio
    _ = json.NewDecoder(r.Body).Decode(&req)

    // TenantResolver ya inyecto el tenant resuelto y el pool del Tenant DB
    tenant := tenantctx.From(r.Context())
    tdb    := dbctx.From(r.Context())  // *pgxpool.Pool del Tenant DB

    user, err := h.q.WithDB(tdb).FindUserForLogin(r.Context(), req.Identifier())
    if err != nil || !verifyPassword(user.PasswordHash, req.Password) {
        h.rateLimiter.Penalize(r, req.Identifier())
        writeErr(w, 401); return
    }
    rehashIfNeeded(h.q.WithDB(tdb), user, req.Password)

    if requiresMFA(user.Role) || user.MFAEnabled {
        preAuth := signPreAuthJWT(user.ID, tenant.ID, 5*time.Minute)
        writeJSON(w, 200, map[string]any{
            "pre_auth_token": preAuth, "mfa_required": true,
        })
        return
    }
    issueSession(w, r, h.q.WithDB(tdb), user, tenant) // cookie + access + refresh
}

func (h *AuthHandler) MFAVerify(w http.ResponseWriter, r *http.Request) {
    var req MFARequest // pre_auth_token + code
    _ = json.NewDecoder(r.Body).Decode(&req)

    claims, err := parsePreAuthJWT(req.PreAuthToken)
    if err != nil { writeErr(w, 401); return }

    // El tenant del pre_auth_token debe coincidir con el resuelto por subdominio
    tenant := tenantctx.From(r.Context())
    if claims.TenantID != tenant.ID { writeErr(w, 401); return }

    tdb := dbctx.From(r.Context())
    user, _ := h.q.WithDB(tdb).GetUserByID(r.Context(), claims.UserID)
    if !totp.Validate(req.Code, decryptSecret(user.MFASecretEnc)) &&
       !consumeRecoveryCode(h.q.WithDB(tdb), user.ID, req.Code) {
        writeErr(w, 401); return
    }
    issueSession(w, r, h.q.WithDB(tdb), user, tenant)
}
```

### Formato del JWT corto (access token)

Header: `{"alg":"EdDSA","typ":"JWT","kid":"2026-04"}`

Payload (claims minimos, todos requeridos salvo `email`):

```json
{
  "iss": "https://api.ph.app",
  "aud": "ph-api",
  "sub": "0190e3a8-7b1a-7c2f-9a4e-2b6c1d5f0a11",
  "tid": "0190e3a8-1111-7000-9000-000000000001",
  "rol": ["tenant_admin"],
  "amr": ["pwd","totp"],
  "sid": "0190e3a8-7b1a-7c2f-9a4e-2b6c1d5f0a99",
  "iat": 1714291200,
  "nbf": 1714291200,
  "exp": 1714292100,
  "jti": "0190e3a8-7b1a-7c2f-9a4e-2b6c1d5f0aaa"
}
```

- `sub` = `users.id` del Tenant DB, `tid` es el identificador de runtime del tenant (NO una columna en `users`), `sid` = `user_sessions.id` (permite revocacion server-side).
- En cada request, el middleware compara `claims.tid` contra el tenant resuelto por subdominio; si difieren, 401.
- `amr` documenta el factor (`pwd`, `totp`, `recovery`) para auditoria.
- Claves rotadas trimestralmente; `kid` apunta al JWKS publicado en `/.well-known/jwks.json`.
- El refresh token nunca es JWT: es opaco y solo significa algo contra la fila de `user_sessions` del Tenant DB correspondiente.
