// Package persistence implementa los repositorios del modulo
// tenant_members. LinkRepository va contra el pool del tenant
// (tenant_user_links) y EnricherRepository va contra el pool central
// (platform_users).
//
// Se usa pgx directo (sin sqlc) para evitar agregar otra entrada al
// codegen y porque las queries son sencillas.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/modules/tenant_members/domain"
	"github.com/saas-ph/api/internal/modules/tenant_members/domain/entities"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// LinkRepository va contra el pool del tenant resuelto en runtime via
// tenantctx. Cada metodo se cobra el pool desde el contexto.
type LinkRepository struct{}

// NewLinkRepository construye el repo.
func NewLinkRepository() *LinkRepository { return &LinkRepository{} }

var _ domain.LinkRepository = (*LinkRepository)(nil)

func tenantPool(ctx context.Context) (*pgxpool.Pool, error) {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant_members: tenant pool: %w", err)
	}
	if t == nil || t.Pool == nil {
		return nil, errors.New("tenant_members: tenant pool nil")
	}
	return t.Pool, nil
}

// Create inserta un link nuevo. Conflicto por UNIQUE → ErrAlreadyLinked.
func (r *LinkRepository) Create(ctx context.Context, in domain.CreateLink) (*entities.TenantMember, error) {
	pool, err := tenantPool(ctx)
	if err != nil {
		return nil, err
	}
	var (
		id        pgtype.UUID
		status    string
		createdAt pgtype.Timestamptz
		updatedAt pgtype.Timestamptz
		version   int32
	)
	row := pool.QueryRow(ctx, `
		INSERT INTO tenant_user_links (platform_user_id, role, primary_unit_id, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, status, created_at, updated_at, version
	`, in.PlatformUserID, in.Role, in.PrimaryUnitID)
	if err := row.Scan(&id, &status, &createdAt, &updatedAt, &version); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrAlreadyLinked
		}
		return nil, fmt.Errorf("create link: %w", err)
	}
	return &entities.TenantMember{
		ID:             uuid.UUID(id.Bytes),
		PlatformUserID: in.PlatformUserID,
		Role:           in.Role,
		PrimaryUnitID:  in.PrimaryUnitID,
		Status:         status,
		CreatedAt:      createdAt.Time,
		UpdatedAt:      updatedAt.Time,
		Version:        version,
	}, nil
}

// List devuelve los links no soft-deleted del tenant.
func (r *LinkRepository) List(ctx context.Context) ([]entities.TenantMember, error) {
	pool, err := tenantPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT id, platform_user_id, role, primary_unit_id, status, created_at, updated_at, version
		FROM tenant_user_links
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()
	out := make([]entities.TenantMember, 0)
	for rows.Next() {
		var (
			m       entities.TenantMember
			id      pgtype.UUID
			puid    pgtype.UUID
			unit    pgtype.UUID
			created pgtype.Timestamptz
			updated pgtype.Timestamptz
		)
		if err := rows.Scan(&id, &puid, &m.Role, &unit, &m.Status, &created, &updated, &m.Version); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		m.ID = uuid.UUID(id.Bytes)
		m.PlatformUserID = uuid.UUID(puid.Bytes)
		if unit.Valid {
			u := uuid.UUID(unit.Bytes)
			m.PrimaryUnitID = &u
		}
		m.CreatedAt = created.Time
		m.UpdatedAt = updated.Time
		out = append(out, m)
	}
	return out, nil
}

// FindByID lee un link por id.
func (r *LinkRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.TenantMember, error) {
	pool, err := tenantPool(ctx)
	if err != nil {
		return nil, err
	}
	var (
		m       entities.TenantMember
		pid     pgtype.UUID
		puid    pgtype.UUID
		unit    pgtype.UUID
		created pgtype.Timestamptz
		updated pgtype.Timestamptz
	)
	err = pool.QueryRow(ctx, `
		SELECT id, platform_user_id, role, primary_unit_id, status, created_at, updated_at, version
		FROM tenant_user_links
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&pid, &puid, &m.Role, &unit, &m.Status, &created, &updated, &m.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrLinkNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find link: %w", err)
	}
	m.ID = uuid.UUID(pid.Bytes)
	m.PlatformUserID = uuid.UUID(puid.Bytes)
	if unit.Valid {
		u := uuid.UUID(unit.Bytes)
		m.PrimaryUnitID = &u
	}
	m.CreatedAt = created.Time
	m.UpdatedAt = updated.Time
	return &m, nil
}

// Update modifica role/unit con optimistic lock por columna version.
func (r *LinkRepository) Update(ctx context.Context, in domain.UpdateLink) (*entities.TenantMember, error) {
	pool, err := tenantPool(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := pool.Exec(ctx, `
		UPDATE tenant_user_links
		SET role = $1, primary_unit_id = $2, version = version + 1, updated_at = now()
		WHERE id = $3 AND version = $4 AND deleted_at IS NULL
	`, in.Role, in.PrimaryUnitID, in.ID, in.Version)
	if err != nil {
		return nil, fmt.Errorf("update link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Distinguir: si no existe o si version mismatched.
		existing, ferr := r.FindByID(ctx, in.ID)
		if ferr != nil {
			return nil, ferr
		}
		_ = existing
		return nil, errors.New("update link: version mismatch")
	}
	return r.FindByID(ctx, in.ID)
}

// Block fija status=blocked.
func (r *LinkRepository) Block(ctx context.Context, id uuid.UUID) error {
	pool, err := tenantPool(ctx)
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, `
		UPDATE tenant_user_links
		SET status = 'blocked', updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("block link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrLinkNotFound
	}
	return nil
}

// EnricherRepository va contra el pool central.
type EnricherRepository struct {
	central *pgxpool.Pool
}

// NewEnricherRepository construye el repo.
func NewEnricherRepository(central *pgxpool.Pool) *EnricherRepository {
	return &EnricherRepository{central: central}
}

var _ domain.EnricherRepository = (*EnricherRepository)(nil)

// FindPlatformUserIDByCode resuelve un public_code en la DB central.
func (r *EnricherRepository) FindPlatformUserIDByCode(ctx context.Context, code string) (uuid.UUID, string, string, string, error) {
	var (
		id    pgtype.UUID
		names string
		last  string
		email string
	)
	err := r.central.QueryRow(ctx, `
		SELECT id, names, last_names, email
		FROM platform_users
		WHERE public_code = $1 AND deleted_at IS NULL AND status = 'active'
	`, code).Scan(&id, &names, &last, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", "", "", domain.ErrPlatformUserNotFound
	}
	if err != nil {
		return uuid.Nil, "", "", "", fmt.Errorf("find by code: %w", err)
	}
	return uuid.UUID(id.Bytes), names, last, email, nil
}

// Hydrate hace UN query ANY($1::uuid[]) contra platform_users y mergea
// los datos con los miembros. Si un id no resuelve queda con campos
// vacios.
func (r *EnricherRepository) Hydrate(ctx context.Context, members []entities.TenantMember) ([]entities.TenantMember, error) {
	if len(members) == 0 {
		return members, nil
	}
	ids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.PlatformUserID)
	}
	rows, err := r.central.Query(ctx, `
		SELECT id, names, last_names, email, public_code
		FROM platform_users
		WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("hydrate query: %w", err)
	}
	defer rows.Close()
	type row struct {
		Names      string
		LastNames  string
		Email      string
		PublicCode string
	}
	byID := make(map[uuid.UUID]row, len(members))
	for rows.Next() {
		var pid pgtype.UUID
		var rec row
		if err := rows.Scan(&pid, &rec.Names, &rec.LastNames, &rec.Email, &rec.PublicCode); err != nil {
			return nil, fmt.Errorf("hydrate scan: %w", err)
		}
		byID[uuid.UUID(pid.Bytes)] = rec
	}
	out := make([]entities.TenantMember, 0, len(members))
	for _, m := range members {
		if rec, ok := byID[m.PlatformUserID]; ok {
			m.Names = rec.Names
			m.LastNames = rec.LastNames
			m.Email = rec.Email
			m.PublicCode = rec.PublicCode
		}
		out = append(out, m)
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}
