// Package usecases agrupa la orquestacion de los casos de uso del modulo
// platform_identity. Cada usecase recibe sus dependencias por inyeccion
// y no conoce HTTP ni la implementacion concreta del repositorio.
//
// Ver docs/specs/fase-16-cross-tenant-identity-spec.md y ADR 0007.
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

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/platform/jwtsign"
	"github.com/saas-ph/api/internal/platform/passwords"
)

// Errores publicos de los usecases. Los handlers HTTP los traducen a
// Problem+JSON con el codigo de estado adecuado. Mensajes son
// intencionalmente genericos para no filtrar info al cliente.
var (
	ErrInvalidInput       = errors.New("platform_identity: invalid input")
	ErrInvalidCredentials = errors.New("platform_identity: invalid credentials")
	ErrAccountLocked      = errors.New("platform_identity: account locked")
	ErrAccountInactive    = errors.New("platform_identity: account inactive")
	ErrInternal           = errors.New("platform_identity: internal error")
)

// Constantes de tiempo y formato.
const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	preAuthTokenTTL = 5 * time.Minute
	tokenTypeBearer = "Bearer"

	// PreAuthSessionMarker es el valor que se coloca en la claim `sid`
	// del JWT pre-auth (cuando la persona tiene MFA enrolado y debe
	// resolver el segundo factor antes de obtener un access token real).
	PreAuthSessionMarker = "pre-auth"
	// PreAuthRole se incluye en `roles` del JWT pre-auth para impedir que
	// un access token completo se reuse como pre-auth.
	PreAuthRole = "pre-auth:mfa"
)

// LoginDeps agrupa las dependencias del usecase de login.
//
// Sessions es opcional. Si es nil, Login no persiste sesion ni emite
// refresh_token (modo "stateless" para tests). Si esta presente, Login
// crea una fila platform_user_sessions y devuelve refresh_token.
type LoginDeps struct {
	Users    domain.PlatformUserRepository
	Sessions domain.SessionRepository
	Signer   *jwtsign.Signer
	Now      func() time.Time
}

// LoginUseCase implementa POST /auth/login segun ADR 0007.
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

// Execute ejecuta el flujo de login con tres factores de identificacion
// (email + document_type + document_number) ademas del password.
//
// Pasos:
//  1. Valida input.
//  2. Resuelve el usuario por email.
//  3. Compara contra el documento provisto. Si no coincide, devuelve
//     ErrInvalidCredentials sin pista de cual factor fallo.
//  4. Verifica status activo.
//  5. Verifica lockout vigente.
//  6. Verifica password con argon2id; en fallo incrementa contador.
//  7. Si OK y MFA enrolado, emite pre_auth_token (ttl 5min).
//  8. Si OK y MFA no enrolado, lista memberships y emite access JWT
//     con SignPlatform (current_tenant vacio = el cliente debe usar el
//     selector o llamar /auth/switch-tenant).
func (uc *LoginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	docType := strings.TrimSpace(strings.ToUpper(req.DocumentType))
	docNumber := strings.TrimSpace(req.DocumentNumber)
	password := req.Password

	if email == "" || docType == "" || docNumber == "" || password == "" {
		return dto.LoginResponse{}, ErrInvalidInput
	}

	user, err := uc.deps.Users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.LoginResponse{}, ErrInvalidCredentials
		}
		return dto.LoginResponse{}, fmt.Errorf("%w: find by email: %w", ErrInternal, err)
	}

	if user.DocumentType != docType || user.DocumentNumber != docNumber {
		return dto.LoginResponse{}, ErrInvalidCredentials
	}

	now := uc.deps.Now()

	if !user.IsActive() {
		return dto.LoginResponse{}, ErrAccountInactive
	}
	if user.IsLocked(now) {
		return dto.LoginResponse{}, ErrAccountLocked
	}

	if err := passwords.Verify(password, user.PasswordHash); err != nil {
		// Password incorrecto: el repo decide si incrementar y bloquear.
		// El cliente sigue viendo invalid credentials; si quedo bloqueado
		// recien lo sabra al siguiente intento.
		if _, _, incErr := uc.deps.Users.IncrementFailedLogin(ctx, user.ID); incErr != nil {
			// No enmascaramos: el contador es importante.
			return dto.LoginResponse{}, fmt.Errorf("%w: increment failed: %w", ErrInternal, incErr)
		}
		return dto.LoginResponse{}, ErrInvalidCredentials
	}

	if user.MFAEnrolledAt != nil {
		preAuth, err := uc.deps.Signer.Sign(
			user.ID.String(),
			"",
			PreAuthSessionMarker,
			[]string{PreAuthRole},
			[]string{"pwd"},
			preAuthTokenTTL,
		)
		if err != nil {
			return dto.LoginResponse{}, fmt.Errorf("%w: sign pre-auth: %w", ErrInternal, err)
		}
		return dto.LoginResponse{
			MFARequired:  true,
			PreAuthToken: preAuth,
			TokenType:    tokenTypeBearer,
			ExpiresIn:    int(preAuthTokenTTL / time.Second),
		}, nil
	}

	if err := uc.deps.Users.MarkLoginSuccess(ctx, user.ID, now); err != nil {
		return dto.LoginResponse{}, fmt.Errorf("%w: mark login: %w", ErrInternal, err)
	}

	memberships, err := uc.deps.Users.ListMemberships(ctx, user.ID)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("%w: list memberships: %w", ErrInternal, err)
	}

	mclaims := make([]jwtsign.MembershipClaim, 0, len(memberships))
	mdtos := make([]dto.MembershipDTO, 0, len(memberships))
	for _, m := range memberships {
		mclaims = append(mclaims, jwtsign.MembershipClaim{
			TenantID:   m.TenantID.String(),
			TenantSlug: m.TenantSlug,
			TenantName: m.TenantName,
			Role:       m.Role,
		})
		mdtos = append(mdtos, dto.MembershipDTO{
			TenantID:     m.TenantID.String(),
			TenantSlug:   m.TenantSlug,
			TenantName:   m.TenantName,
			LogoURL:      m.LogoURL,
			PrimaryColor: m.PrimaryColor,
			Role:         m.Role,
			Status:       m.Status,
		})
	}

	sessionID, refreshPlain, err := uc.issueSession(ctx, user.ID, now)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	access, err := uc.deps.Signer.SignPlatform(
		user.ID.String(),
		sessionID,
		"",
		mclaims,
		[]string{"pwd"},
		accessTokenTTL,
	)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("%w: sign access: %w", ErrInternal, err)
	}

	return dto.LoginResponse{
		AccessToken:  access,
		RefreshToken: refreshPlain,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int(accessTokenTTL / time.Second),
		Memberships:  mdtos,
		NeedsTenant:  len(mdtos) != 1,
	}, nil
}

// issueSession crea una fila en platform_user_sessions (si Sessions repo
// esta presente) y devuelve (sessionID, refreshPlain). Si Sessions es
// nil, devuelve un sessionID sintetico y refreshPlain vacio (modo
// stateless para tests donde no nos importa la persistencia).
func (uc *LoginUseCase) issueSession(ctx context.Context, userID uuid.UUID, now time.Time) (string, string, error) {
	if uc.deps.Sessions == nil {
		return fmt.Sprintf("plat-%d", now.UnixNano()), "", nil
	}
	plain, hash, err := generateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("%w: generate refresh: %w", ErrInternal, err)
	}
	session, err := uc.deps.Sessions.Create(ctx, userID, hash, nil, now.Add(refreshTokenTTL))
	if err != nil {
		return "", "", fmt.Errorf("%w: create session: %w", ErrInternal, err)
	}
	return session.ID.String(), plain, nil
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
