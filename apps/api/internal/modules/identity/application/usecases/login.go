// Package usecases agrupa la orquestacion de los casos de uso del
// modulo identity. Cada usecase recibe sus dependencias por inyeccion y
// no conoce HTTP ni la implementacion concreta de los repositorios.
package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
	"github.com/saas-ph/api/internal/modules/identity/domain"
	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
	"github.com/saas-ph/api/internal/modules/identity/domain/policies"
	"github.com/saas-ph/api/internal/platform/jwtsign"
	"github.com/saas-ph/api/internal/platform/passwords"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// Errores publicos de los usecases. Los handlers HTTP los traducen a
// Problem+JSON conservando el codigo de estado adecuado.
var (
	ErrInvalidInput       = errors.New("identity: invalid input")
	ErrInvalidCredentials = errors.New("identity: invalid credentials")
	ErrAccountLocked      = errors.New("identity: account locked")
	ErrAccountInactive    = errors.New("identity: account inactive")
	ErrInvalidPreAuth     = errors.New("identity: invalid pre-auth token")
	ErrInvalidMFACode     = errors.New("identity: invalid mfa code")
	ErrInvalidRefresh     = errors.New("identity: invalid refresh token")
	ErrTenantMissing      = errors.New("identity: tenant missing in context")
	ErrInternal           = errors.New("identity: internal error")
)

// Constantes de tiempo y formato.
const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	preAuthTokenTTL = 5 * time.Minute
	tokenTypeBearer = "Bearer"

	// PreAuthSessionMarker es el valor que se coloca en la claim `sid`
	// del JWT cuando solo cubre la primera etapa de autenticacion (MFA
	// pendiente). El platform `jwtsign.Signer` exige una unica audiencia
	// estatica, por lo que el "aud=mfa" del ADR se materializa con esta
	// marca + el role `pre-auth:mfa`. La capa MFAVerify rechaza tokens
	// que no traigan este marker para impedir que un access token
	// completo se reuse como pre-auth.
	PreAuthSessionMarker = "pre-auth"

	// PreAuthRole se incluye en `roles` del JWT pre-auth. Es la doble
	// verificacion que evita confundir un pre-auth con una sesion real.
	PreAuthRole = "pre-auth:mfa"
)

// LoginDeps agrupa las dependencias del usecase de login.
type LoginDeps struct {
	Users    domain.UserRepository
	Sessions domain.SessionRepository
	Signer   *jwtsign.Signer
	Now      func() time.Time
}

// LoginUseCase implementa POST /auth/login.
type LoginUseCase struct {
	deps LoginDeps
}

// NewLoginUseCase construye el usecase. Si Now es nil usa time.Now.
func NewLoginUseCase(deps LoginDeps) *LoginUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &LoginUseCase{deps: deps}
}

// Execute ejecuta el flujo de login segun el ADR de identidad.
//
// Pasos:
//  1. Valida input.
//  2. Resuelve el usuario por email o por (doc_type, doc_number).
//  3. Verifica lockout vigente.
//  4. Verifica password con argon2id; en fallo incrementa contador y
//     potencialmente bloquea.
//  5. Si OK y MFA enrolado, emite pre_auth_token (aud=mfa, ttl 5min).
//  6. Si OK y MFA no enrolado, crea session row + emite tokens reales.
func (uc *LoginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	if strings.TrimSpace(req.Identifier) == "" || strings.TrimSpace(req.Password) == "" {
		return dto.LoginResponse{}, ErrInvalidInput
	}

	user, err := uc.resolveUser(ctx, req.Identifier)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.LoginResponse{}, ErrInvalidCredentials
		}
		return dto.LoginResponse{}, fmt.Errorf("%w: resolve user: %w", ErrInternal, err)
	}

	now := uc.deps.Now()

	if !user.IsActive() {
		return dto.LoginResponse{}, ErrAccountInactive
	}
	if policies.IsLocked(user, now) {
		return dto.LoginResponse{}, ErrAccountLocked
	}

	if err := passwords.Verify(req.Password, user.PasswordHash); err != nil {
		// Password incorrecto: incrementar el contador y posiblemente
		// bloquear. Cualquier error en este paso debe seguir devolviendo
		// invalid credentials al cliente.
		if policies.WouldExceedFailedAttempts(user.FailedLoginAttempts) {
			lockUntil := policies.NextLockUntil(now)
			_ = uc.deps.Users.LockUser(ctx, user.ID, lockUntil)
			return dto.LoginResponse{}, ErrAccountLocked
		}
		_, _ = uc.deps.Users.IncrementFailedAttempts(ctx, user.ID)
		return dto.LoginResponse{}, ErrInvalidCredentials
	}

	// Password OK: limpiar contador antes de emitir nada.
	if err := uc.deps.Users.ResetFailedAttempts(ctx, user.ID); err != nil {
		return dto.LoginResponse{}, fmt.Errorf("%w: reset attempts: %w", ErrInternal, err)
	}

	if policies.ShouldRequireMFA(user) {
		token, err := uc.signPreAuthToken(ctx, user.ID, now)
		if err != nil {
			return dto.LoginResponse{}, fmt.Errorf("%w: sign pre-auth: %w", ErrInternal, err)
		}
		return dto.LoginResponse{
			MFARequired:  true,
			PreAuthToken: token,
		}, nil
	}

	access, refresh, err := IssueSession(ctx, uc.deps.Sessions, uc.deps.Signer, user.ID, "", now)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	if err := uc.deps.Users.UpdateLastLoginAt(ctx, user.ID, now); err != nil {
		// No bloqueamos el login por fallar el last_login_at; pero si el
		// signer emite tokens validos, los devolvemos. Caller decide.
		// Aqui preferimos visibilidad: no enmascaramos.
		return dto.LoginResponse{}, fmt.Errorf("%w: update last_login: %w", ErrInternal, err)
	}

	return dto.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(accessTokenTTL / time.Second),
		TokenType:    tokenTypeBearer,
	}, nil
}

// resolveUser interpreta el identifier del request: email si contiene
// `@`, par documento si contiene `:`. Cualquier otro formato se
// considera invalido (mismo signal que credenciales malas para no
// filtrar info al cliente).
func (uc *LoginUseCase) resolveUser(ctx context.Context, identifier string) (*entities.User, error) {
	id := strings.TrimSpace(identifier)
	if strings.Contains(id, "@") {
		return uc.deps.Users.GetByEmail(ctx, strings.ToLower(id))
	}
	if i := strings.IndexByte(id, ':'); i > 0 && i < len(id)-1 {
		docType := entities.DocumentType(strings.ToUpper(id[:i]))
		docNumber := strings.TrimSpace(id[i+1:])
		if !docType.IsValid() || docNumber == "" {
			return nil, domain.ErrUserNotFound
		}
		return uc.deps.Users.GetByDocument(ctx, docType, docNumber)
	}
	return nil, domain.ErrUserNotFound
}

// signPreAuthToken firma un JWT corto que solo da acceso a /auth/mfa/verify.
//
// El platform Signer enforce una sola audiencia estatica, asi que el
// "aud=mfa" del ADR se modela con (sid=pre-auth, roles=[pre-auth:mfa]).
// MFAVerify rechaza tokens que no satisfagan ambos criterios.
func (uc *LoginUseCase) signPreAuthToken(ctx context.Context, userID string, _ time.Time) (string, error) {
	tenant, err := tenantctx.FromCtx(ctx)
	tenantID := ""
	if err == nil {
		tenantID = tenant.ID
	}
	return uc.deps.Signer.Sign(userID, tenantID, PreAuthSessionMarker, []string{PreAuthRole}, []string{"pwd"}, preAuthTokenTTL)
}

// IssueSession crea una session row con refresh token aleatorio y firma
// el access JWT correspondiente. parentSessionID puede ser vacio si es
// una sesion nueva (login normal); para refresh, contiene el id del
// padre.
func IssueSession(ctx context.Context, sessions domain.SessionRepository, signer *jwtsign.Signer, userID, parentSessionID string, now time.Time) (accessToken, refreshToken string, err error) {
	tenant, terr := tenantctx.FromCtx(ctx)
	if terr != nil {
		return "", "", fmt.Errorf("%w: %w", ErrTenantMissing, terr)
	}

	refreshPlain, refreshHash, rerr := generateRefreshToken()
	if rerr != nil {
		return "", "", fmt.Errorf("%w: refresh: %w", ErrInternal, rerr)
	}

	session := &entities.Session{
		UserID:    userID,
		TokenHash: refreshHash,
		IssuedAt:  now,
		ExpiresAt: now.Add(refreshTokenTTL),
		Status:    entities.SessionStatusActive,
	}
	if parentSessionID != "" {
		session.ParentSessionID = &parentSessionID
	}
	if err := sessions.Create(ctx, session); err != nil {
		return "", "", fmt.Errorf("%w: create session: %w", ErrInternal, err)
	}

	access, serr := signer.Sign(userID, tenant.ID, session.ID, nil, []string{"pwd"}, accessTokenTTL)
	if serr != nil {
		return "", "", fmt.Errorf("%w: sign access: %w", ErrInternal, serr)
	}
	return access, refreshPlain, nil
}

// generateRefreshToken devuelve `(plain, sha256-hex)` con 32 bytes
// aleatorios codificados en base64 url-safe.
func generateRefreshToken() (string, string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(raw[:])
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}

// HashRefreshToken expone la convencion de hashing usada para el refresh
// token (sha256 hex del valor base64). Compartido por usecases que
// necesitan buscar la sesion sin volver a hashear inline.
func HashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
