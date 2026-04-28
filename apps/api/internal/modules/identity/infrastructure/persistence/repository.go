// Package persistence implementa los repositorios del modulo identity.
//
// Diseno: stateless. Cada metodo del repositorio resuelve el pool del
// tenant a partir del contexto del request via `tenantctx.FromCtx`.
// Esto encaja con el modelo multi-tenant DB-por-tenant: una unica
// instancia del repo sirve a todos los tenants y la conexion correcta
// se elige por request.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saas-ph/api/internal/modules/identity/domain"
	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
	identitydb "github.com/saas-ph/api/internal/modules/identity/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// UserRepository implementa domain.UserRepository.
type UserRepository struct{}

// NewUserRepository construye un UserRepository sin estado. La conexion
// se resuelve por-request desde tenantctx.
func NewUserRepository() *UserRepository { return &UserRepository{} }

// SessionRepository implementa domain.SessionRepository.
type SessionRepository struct{}

// NewSessionRepository construye un SessionRepository sin estado.
func NewSessionRepository() *SessionRepository { return &SessionRepository{} }

func queriesFromCtx(ctx context.Context) (*identitydb.Queries, error) {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("identity persistence: tenant pool nil")
	}
	return identitydb.New(t.Pool), nil
}

// GetByID retorna un usuario por id.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*entities.User, error) {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	row, err := q.GetUserByID(ctx, mustUUID(id))
	if err != nil {
		return nil, mapNotFound(err, domain.ErrUserNotFound)
	}
	return userFromRow(row), nil
}

// GetByEmail retorna un usuario por email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	em := email
	row, err := q.GetUserByEmail(ctx, &em)
	if err != nil {
		return nil, mapNotFound(err, domain.ErrUserNotFound)
	}
	return userFromRow(row), nil
}

// GetByDocument retorna un usuario por (document_type, document_number).
func (r *UserRepository) GetByDocument(ctx context.Context, docType entities.DocumentType, docNumber string) (*entities.User, error) {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	row, err := q.GetUserByDocument(ctx, identitydb.GetUserByDocumentParams{
		DocumentType:   string(docType),
		DocumentNumber: docNumber,
	})
	if err != nil {
		return nil, mapNotFound(err, domain.ErrUserNotFound)
	}
	return userFromRow(row), nil
}

// IncrementFailedAttempts suma 1 al contador del usuario.
func (r *UserRepository) IncrementFailedAttempts(ctx context.Context, userID string) (int, error) {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return 0, err
	}
	v, err := q.IncrementFailedAttempts(ctx, mustUUID(userID))
	if err != nil {
		return 0, fmt.Errorf("identity: incr failed attempts: %w", err)
	}
	return int(v), nil
}

// ResetFailedAttempts deja el contador en 0.
func (r *UserRepository) ResetFailedAttempts(ctx context.Context, userID string) error {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return err
	}
	if err := q.ResetFailedAttempts(ctx, mustUUID(userID)); err != nil {
		return fmt.Errorf("identity: reset failed: %w", err)
	}
	return nil
}

// LockUser fija locked_until.
func (r *UserRepository) LockUser(ctx context.Context, userID string, until time.Time) error {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return err
	}
	return q.LockUser(ctx, identitydb.LockUserParams{
		ID:          mustUUID(userID),
		LockedUntil: pgtype.Timestamptz{Time: until, Valid: true},
	})
}

// UnlockUser limpia locked_until.
func (r *UserRepository) UnlockUser(ctx context.Context, userID string) error {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return err
	}
	return q.UnlockUser(ctx, mustUUID(userID))
}

// UpdateLastLoginAt marca el ultimo login.
func (r *UserRepository) UpdateLastLoginAt(ctx context.Context, userID string, at time.Time) error {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return err
	}
	return q.UpdateLastLoginAt(ctx, identitydb.UpdateLastLoginAtParams{
		ID:          mustUUID(userID),
		LastLoginAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}

// Create persiste una sesion nueva (refresh token).
func (r *SessionRepository) Create(ctx context.Context, s *entities.Session) error {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return err
	}
	params := identitydb.CreateSessionParams{
		UserID:    mustUUID(s.UserID),
		TokenHash: s.TokenHash,
		IssuedAt:  pgtype.Timestamptz{Time: s.IssuedAt, Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: s.ExpiresAt, Valid: true},
	}
	if s.ParentSessionID != nil && *s.ParentSessionID != "" {
		params.ParentSessionID = mustUUID(*s.ParentSessionID)
	}
	if s.IP != nil {
		if ip, err := netip.ParseAddr(*s.IP); err == nil {
			params.Ip = &ip
		}
	}
	if s.UserAgent != nil {
		params.UserAgent = s.UserAgent
	}
	row, err := q.CreateSession(ctx, params)
	if err != nil {
		return fmt.Errorf("identity: create session: %w", err)
	}
	if id, err := uuidString(row.ID); err == nil {
		s.ID = id
	}
	if row.CreatedAt.Valid {
		s.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		s.UpdatedAt = row.UpdatedAt.Time
	}
	s.Status = entities.SessionStatus(row.Status)
	return nil
}

// GetByTokenHash busca por sha256 del refresh token.
func (r *SessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*entities.Session, error) {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	row, err := q.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, mapNotFound(err, domain.ErrSessionNotFound)
	}
	return sessionFromRow(row), nil
}

// GetByID busca sesion por id.
func (r *SessionRepository) GetByID(ctx context.Context, id string) (*entities.Session, error) {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	row, err := q.GetSessionByID(ctx, mustUUID(id))
	if err != nil {
		return nil, mapNotFound(err, domain.ErrSessionNotFound)
	}
	return sessionFromRow(row), nil
}

// Revoke marca la sesion como revocada.
func (r *SessionRepository) Revoke(ctx context.Context, sessionID, reason string, at time.Time) error {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return err
	}
	rs := reason
	return q.RevokeSession(ctx, identitydb.RevokeSessionParams{
		ID:               mustUUID(sessionID),
		RevokedAt:        pgtype.Timestamptz{Time: at, Valid: true},
		RevocationReason: &rs,
	})
}

// RevokeChain revoca toda la cadena (parents + descendants).
func (r *SessionRepository) RevokeChain(ctx context.Context, sessionID, reason string, at time.Time) error {
	q, err := queriesFromCtx(ctx)
	if err != nil {
		return err
	}
	rs := reason
	return q.RevokeSessionChain(ctx, identitydb.RevokeSessionChainParams{
		ID:               mustUUID(sessionID),
		RevokedAt:        pgtype.Timestamptz{Time: at, Valid: true},
		RevocationReason: &rs,
	})
}

func userFromRow(u identitydb.User) *entities.User {
	id, _ := uuidString(u.ID)
	out := &entities.User{
		ID:                  id,
		DocumentType:        entities.DocumentType(u.DocumentType),
		DocumentNumber:      u.DocumentNumber,
		Names:               u.Names,
		LastNames:           u.LastNames,
		Email:               u.Email,
		Phone:               u.Phone,
		PasswordHash:        u.PasswordHash,
		MFASecret:           u.MfaSecret,
		FailedLoginAttempts: int(u.FailedLoginAttempts),
		Status:              entities.UserStatus(u.Status),
		Version:             int(u.Version),
	}
	if u.MfaEnrolledAt.Valid {
		t := u.MfaEnrolledAt.Time
		out.MFAEnrolledAt = &t
	}
	if u.LockedUntil.Valid {
		t := u.LockedUntil.Time
		out.LockedUntil = &t
	}
	if u.LastLoginAt.Valid {
		t := u.LastLoginAt.Time
		out.LastLoginAt = &t
	}
	if u.CreatedAt.Valid {
		out.CreatedAt = u.CreatedAt.Time
	}
	if u.UpdatedAt.Valid {
		out.UpdatedAt = u.UpdatedAt.Time
	}
	if u.DeletedAt.Valid {
		t := u.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s, err := uuidString(u.CreatedBy); err == nil && s != "" {
		out.CreatedBy = &s
	}
	if s, err := uuidString(u.UpdatedBy); err == nil && s != "" {
		out.UpdatedBy = &s
	}
	if s, err := uuidString(u.DeletedBy); err == nil && s != "" {
		out.DeletedBy = &s
	}
	return out
}

func sessionFromRow(s identitydb.UserSession) *entities.Session {
	id, _ := uuidString(s.ID)
	uid, _ := uuidString(s.UserID)
	out := &entities.Session{
		ID:        id,
		UserID:    uid,
		TokenHash: s.TokenHash,
		Status:    entities.SessionStatus(s.Status),
		Version:   int(s.Version),
	}
	if s.ParentSessionID.Valid {
		if ps, err := uuidString(s.ParentSessionID); err == nil && ps != "" {
			out.ParentSessionID = &ps
		}
	}
	if s.Ip != nil {
		ipStr := s.Ip.String()
		out.IP = &ipStr
	}
	out.UserAgent = s.UserAgent
	if s.IssuedAt.Valid {
		out.IssuedAt = s.IssuedAt.Time
	}
	if s.ExpiresAt.Valid {
		out.ExpiresAt = s.ExpiresAt.Time
	}
	if s.RevokedAt.Valid {
		t := s.RevokedAt.Time
		out.RevokedAt = &t
	}
	if s.RevocationReason != nil {
		out.RevocationReason = s.RevocationReason
	}
	if s.CreatedAt.Valid {
		out.CreatedAt = s.CreatedAt.Time
	}
	if s.UpdatedAt.Valid {
		out.UpdatedAt = s.UpdatedAt.Time
	}
	return out
}

func mustUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func uuidString(u pgtype.UUID) (string, error) {
	if !u.Valid {
		return "", nil
	}
	v, err := u.Value()
	if err != nil {
		return "", err
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("identity: unexpected uuid value type %T", v)
}

func mapNotFound(err, sentinel error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return sentinel
	}
	return err
}

var (
	_ domain.UserRepository    = (*UserRepository)(nil)
	_ domain.SessionRepository = (*SessionRepository)(nil)
)
