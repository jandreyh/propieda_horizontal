// Package persistence implementa los repositorios del modulo
// platform_identity sobre el pool de la DB central.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
	platformiddb "github.com/saas-ph/api/internal/modules/platform_identity/infrastructure/persistence/sqlcgen"
)

// PlatformUserRepository envuelve sqlcgen.Queries con conversiones a/desde
// el dominio.
type PlatformUserRepository struct {
	q *platformiddb.Queries
}

// NewPlatformUserRepository construye el repo. La interface compliance se
// fuerza con `var _ domain.PlatformUserRepository = (*PlatformUserRepository)(nil)`.
func NewPlatformUserRepository(pool *pgxpool.Pool) *PlatformUserRepository {
	return &PlatformUserRepository{q: platformiddb.New(pool)}
}

var _ domain.PlatformUserRepository = (*PlatformUserRepository)(nil)

// FindByEmail busca un PlatformUser activo por email (case-insensitive).
func (r *PlatformUserRepository) FindByEmail(ctx context.Context, email string) (*entities.PlatformUser, error) {
	row, err := r.q.GetPlatformUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("FindByEmail: %w", err)
	}
	return rowToUser(row), nil
}

// FindByDocument busca por document_type + document_number.
func (r *PlatformUserRepository) FindByDocument(ctx context.Context, docType, docNumber string) (*entities.PlatformUser, error) {
	row, err := r.q.GetPlatformUserByDocument(ctx, platformiddb.GetPlatformUserByDocumentParams{
		DocumentType:   docType,
		DocumentNumber: docNumber,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("FindByDocument: %w", err)
	}
	return rowToUser(row), nil
}

// FindByID busca por UUID.
func (r *PlatformUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.PlatformUser, error) {
	row, err := r.q.GetPlatformUserByID(ctx, uuidToPg(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("FindByID: %w", err)
	}
	return rowToUser(row), nil
}

// FindByPublicCode busca por codigo unico publico.
func (r *PlatformUserRepository) FindByPublicCode(ctx context.Context, code string) (*entities.PlatformUser, error) {
	row, err := r.q.GetPlatformUserByPublicCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("FindByPublicCode: %w", err)
	}
	return rowToUser(row), nil
}

// MarkLoginSuccess sella login exitoso (last_login_at = now, resetea fallos).
func (r *PlatformUserRepository) MarkLoginSuccess(ctx context.Context, id uuid.UUID, when time.Time) error {
	if err := r.q.UpdatePlatformUserLastLogin(ctx, platformiddb.UpdatePlatformUserLastLoginParams{
		ID:          uuidToPg(id),
		LastLoginAt: timeToTs(when),
	}); err != nil {
		return fmt.Errorf("MarkLoginSuccess: %w", err)
	}
	return nil
}

// IncrementFailedLogin suma 1 al contador y bloquea tras 5 intentos.
// Devuelve el contador resultante y, si aplica, hasta cuando esta bloqueado.
func (r *PlatformUserRepository) IncrementFailedLogin(ctx context.Context, id uuid.UUID) (int32, *time.Time, error) {
	row, err := r.q.IncrementFailedLogin(ctx, uuidToPg(id))
	if err != nil {
		return 0, nil, fmt.Errorf("IncrementFailedLogin: %w", err)
	}
	return row.FailedLoginAttempts, tsToPtrTime(row.LockedUntil), nil
}

// ListMemberships devuelve las membresias activas del usuario para el
// selector y el JWT.
func (r *PlatformUserRepository) ListMemberships(ctx context.Context, userID uuid.UUID) ([]entities.Membership, error) {
	rows, err := r.q.ListMembershipsForUser(ctx, uuidToPg(userID))
	if err != nil {
		return nil, fmt.Errorf("ListMemberships: %w", err)
	}
	out := make([]entities.Membership, 0, len(rows))
	for _, row := range rows {
		out = append(out, entities.Membership{
			TenantID:     pgToUUID(row.TenantID),
			TenantSlug:   row.TenantSlug,
			TenantName:   row.TenantName,
			LogoURL:      row.LogoUrl,
			PrimaryColor: row.PrimaryColor,
			Role:         row.Role,
			Status:       row.MembershipStatus,
		})
	}
	return out, nil
}

// HasMembership verifica si un usuario tiene acceso activo a un tenant slug.
func (r *PlatformUserRepository) HasMembership(ctx context.Context, userID uuid.UUID, slug string) (bool, error) {
	ok, err := r.q.HasMembership(ctx, platformiddb.HasMembershipParams{
		PlatformUserID: uuidToPg(userID),
		Slug:           slug,
	})
	if err != nil {
		return false, fmt.Errorf("HasMembership: %w", err)
	}
	return ok, nil
}

// SessionRepository envuelve sqlcgen para `platform_user_sessions`.
type SessionRepository struct {
	q *platformiddb.Queries
}

// NewSessionRepository construye el repo.
func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{q: platformiddb.New(pool)}
}

var _ domain.SessionRepository = (*SessionRepository)(nil)

// Create inserta una sesion con token_hash + expires_at.
func (r *SessionRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, userAgent *string, expiresAt time.Time) (*entities.PlatformSession, error) {
	row, err := r.q.CreatePlatformUserSession(ctx, platformiddb.CreatePlatformUserSessionParams{
		PlatformUserID: uuidToPg(userID),
		TokenHash:      tokenHash,
		IP:             nil,
		UserAgent:      userAgent,
		ExpiresAt:      timeToTs(expiresAt),
	})
	if err != nil {
		return nil, fmt.Errorf("Create session: %w", err)
	}
	return rowToSession(row), nil
}

// FindByTokenHash devuelve la sesion activa con ese hash.
func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*entities.PlatformSession, error) {
	row, err := r.q.GetSessionByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("FindByTokenHash: %w", err)
	}
	return rowToSession(row), nil
}

// Revoke marca la sesion como revocada con un motivo.
func (r *SessionRepository) Revoke(ctx context.Context, sessionID uuid.UUID, reason string) error {
	if err := r.q.RevokeSession(ctx, platformiddb.RevokeSessionParams{
		ID:               uuidToPg(sessionID),
		RevocationReason: &reason,
	}); err != nil {
		return fmt.Errorf("Revoke: %w", err)
	}
	return nil
}

// RevokeAllForUser revoca todas las sesiones activas de un usuario.
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, reason string) error {
	if err := r.q.RevokeAllUserSessions(ctx, platformiddb.RevokeAllUserSessionsParams{
		PlatformUserID:   uuidToPg(userID),
		RevocationReason: &reason,
	}); err != nil {
		return fmt.Errorf("RevokeAllForUser: %w", err)
	}
	return nil
}

func rowToSession(r platformiddb.PlatformUserSession) *entities.PlatformSession {
	s := &entities.PlatformSession{
		ID:               pgToUUID(r.ID),
		PlatformUserID:   pgToUUID(r.PlatformUserID),
		TokenHash:        r.TokenHash,
		UserAgent:        r.UserAgent,
		IssuedAt:         tsToTime(r.IssuedAt),
		ExpiresAt:        tsToTime(r.ExpiresAt),
		RevokedAt:        tsToPtrTime(r.RevokedAt),
		RevocationReason: r.RevocationReason,
		Status:           r.Status,
	}
	if r.ParentSessionID.Valid {
		p := pgToUUID(r.ParentSessionID)
		s.ParentSessionID = &p
	}
	return s
}

// PushDeviceRepository envuelve sqlcgen para `platform_push_devices`.
type PushDeviceRepository struct {
	q *platformiddb.Queries
}

// NewPushDeviceRepository construye el repo. Reutiliza el pool central.
func NewPushDeviceRepository(pool *pgxpool.Pool) *PushDeviceRepository {
	return &PushDeviceRepository{q: platformiddb.New(pool)}
}

var _ domain.PushDeviceRepository = (*PushDeviceRepository)(nil)

// Register hace upsert del token (es UNIQUE por usuario+token).
func (r *PushDeviceRepository) Register(ctx context.Context, userID uuid.UUID, token, platform string, label *string) (*entities.PushDevice, error) {
	row, err := r.q.RegisterPushDevice(ctx, platformiddb.RegisterPushDeviceParams{
		PlatformUserID: uuidToPg(userID),
		DeviceToken:    token,
		Platform:       platform,
		DeviceLabel:    label,
	})
	if err != nil {
		return nil, fmt.Errorf("Register: %w", err)
	}
	return rowToPushDevice(row), nil
}

// Revoke marca el device como revocado.
func (r *PushDeviceRepository) Revoke(ctx context.Context, deviceID, userID uuid.UUID) error {
	if err := r.q.RevokePushDevice(ctx, platformiddb.RevokePushDeviceParams{
		ID:             uuidToPg(deviceID),
		PlatformUserID: uuidToPg(userID),
	}); err != nil {
		return fmt.Errorf("Revoke: %w", err)
	}
	return nil
}

// List devuelve los devices activos del usuario.
func (r *PushDeviceRepository) List(ctx context.Context, userID uuid.UUID) ([]entities.PushDevice, error) {
	rows, err := r.q.ListPushDevicesForUser(ctx, uuidToPg(userID))
	if err != nil {
		return nil, fmt.Errorf("List: %w", err)
	}
	out := make([]entities.PushDevice, 0, len(rows))
	for _, row := range rows {
		out = append(out, *rowToPushDevice(row))
	}
	return out, nil
}

// rowToUser convierte el row sqlc al dominio.
func rowToUser(r platformiddb.PlatformUser) *entities.PlatformUser {
	return &entities.PlatformUser{
		ID:                  pgToUUID(r.ID),
		DocumentType:        r.DocumentType,
		DocumentNumber:      r.DocumentNumber,
		Names:               r.Names,
		LastNames:           r.LastNames,
		Email:               r.Email,
		Phone:               r.Phone,
		PhotoURL:            r.PhotoUrl,
		PasswordHash:        r.PasswordHash,
		MFASecret:           r.MfaSecret,
		MFAEnrolledAt:       tsToPtrTime(r.MfaEnrolledAt),
		PublicCode:          r.PublicCode,
		FailedLoginAttempts: r.FailedLoginAttempts,
		LockedUntil:         tsToPtrTime(r.LockedUntil),
		LastLoginAt:         tsToPtrTime(r.LastLoginAt),
		Status:              r.Status,
		CreatedAt:           tsToTime(r.CreatedAt),
		UpdatedAt:           tsToTime(r.UpdatedAt),
	}
}

func rowToPushDevice(r platformiddb.PlatformPushDevice) *entities.PushDevice {
	return &entities.PushDevice{
		ID:             pgToUUID(r.ID),
		PlatformUserID: pgToUUID(r.PlatformUserID),
		DeviceToken:    r.DeviceToken,
		Platform:       r.Platform,
		DeviceLabel:    r.DeviceLabel,
		LastSeenAt:     tsToTime(r.LastSeenAt),
		CreatedAt:      tsToTime(r.CreatedAt),
		RevokedAt:      tsToPtrTime(r.RevokedAt),
	}
}

// --- Helpers de conversion entre uuid.UUID y pgtype.* ---

func uuidToPg(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func pgToUUID(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func tsToPtrTime(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func timeToTs(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}
