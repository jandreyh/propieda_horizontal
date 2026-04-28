// Package persistence implementa los puertos del modulo
// residential_structure usando el codigo generado por sqlc.
//
// Reglas:
//   - El pool del Tenant DB se obtiene del contexto via tenantctx.FromCtx
//     (repo STATELESS — no almacena el pool).
//   - NO se usa SQL inline. Toda query vive en .sql para sqlc.
//   - El repo mapea entre tipos pgtype (sqlc) y los tipos de dominio.
package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saas-ph/api/internal/modules/residential_structure/domain"
	"github.com/saas-ph/api/internal/modules/residential_structure/domain/entities"
	residentialdb "github.com/saas-ph/api/internal/modules/residential_structure/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// StructureRepository implementa domain.StructureRepository sobre la
// Tenant DB resuelta del contexto. Es stateless: cada metodo resuelve
// el pool por-request.
type StructureRepository struct{}

// NewStructureRepository construye una instancia stateless.
func NewStructureRepository() *StructureRepository {
	return &StructureRepository{}
}

func querier(ctx context.Context) (*residentialdb.Queries, error) {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("residential_structure: tenant pool is nil")
	}
	return residentialdb.New(t.Pool), nil
}

// ListActive implementa domain.StructureRepository.
func (r *StructureRepository) ListActive(ctx context.Context) ([]entities.Structure, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveStructures(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Structure, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapStructure(row))
	}
	return out, nil
}

// GetByID implementa domain.StructureRepository.
func (r *StructureRepository) GetByID(ctx context.Context, id string) (entities.Structure, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Structure{}, err
	}
	row, err := q.GetStructureByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Structure{}, domain.ErrStructureNotFound
		}
		return entities.Structure{}, err
	}
	return mapStructure(row), nil
}

// Create implementa domain.StructureRepository.
func (r *StructureRepository) Create(ctx context.Context, in domain.CreateStructureInput) (entities.Structure, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Structure{}, err
	}
	var desc *string
	if in.Description != "" {
		d := in.Description
		desc = &d
	}
	row, err := q.CreateStructure(ctx, residentialdb.CreateStructureParams{
		Name:        in.Name,
		Type:        string(in.Type),
		ParentID:    optionalUUID(in.ParentID),
		Description: desc,
		OrderIndex:  in.OrderIndex,
		CreatedBy:   uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.Structure{}, err
	}
	return mapStructure(row), nil
}

// Update implementa domain.StructureRepository.
func (r *StructureRepository) Update(ctx context.Context, in domain.UpdateStructureInput) (entities.Structure, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Structure{}, err
	}
	var desc *string
	if in.Description != "" {
		d := in.Description
		desc = &d
	}
	row, err := q.UpdateStructure(ctx, residentialdb.UpdateStructureParams{
		Name:        in.Name,
		Type:        string(in.Type),
		ParentID:    optionalUUID(in.ParentID),
		Description: desc,
		OrderIndex:  in.OrderIndex,
		UpdatedBy:   uuidToPgtype(in.ActorID),
		ID:          uuidToPgtype(in.ID),
		Version:     in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Distinguir "no existe" vs "version mismatch": si la fila
			// existe (GetByID OK) pero version no calza, devolvemos
			// ErrVersionMismatch; si tampoco existe, ErrStructureNotFound.
			if _, gerr := r.GetByID(ctx, in.ID); gerr == nil {
				return entities.Structure{}, domain.ErrVersionMismatch
			}
			return entities.Structure{}, domain.ErrStructureNotFound
		}
		return entities.Structure{}, err
	}
	return mapStructure(row), nil
}

// Archive implementa domain.StructureRepository.
func (r *StructureRepository) Archive(ctx context.Context, id, actorID string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	_, err = q.ArchiveStructure(ctx, residentialdb.ArchiveStructureParams{
		ID:        uuidToPgtype(id),
		DeletedBy: uuidToPgtype(actorID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrStructureNotFound
		}
		return err
	}
	return nil
}

// ListChildren implementa domain.StructureRepository.
func (r *StructureRepository) ListChildren(ctx context.Context, parentID string) ([]entities.Structure, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListChildStructures(ctx, uuidToPgtype(parentID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Structure, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapStructure(row))
	}
	return out, nil
}

// --- helpers de mapeo ---

func mapStructure(r residentialdb.ResidentialStructure) entities.Structure {
	out := entities.Structure{
		ID:         uuidString(r.ID),
		Name:       r.Name,
		Type:       entities.StructureType(r.Type),
		OrderIndex: r.OrderIndex,
		Status:     entities.StructureStatus(r.Status),
		CreatedAt:  tsToTime(r.CreatedAt),
		UpdatedAt:  tsToTime(r.UpdatedAt),
		Version:    r.Version,
	}
	if s := uuidStringPtr(r.ParentID); s != nil {
		out.ParentID = s
	}
	if r.Description != nil {
		out.Description = *r.Description
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	if s := uuidStringPtr(r.CreatedBy); s != nil {
		out.CreatedBy = s
	}
	if s := uuidStringPtr(r.UpdatedBy); s != nil {
		out.UpdatedBy = s
	}
	if s := uuidStringPtr(r.DeletedBy); s != nil {
		out.DeletedBy = s
	}
	return out
}

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return formatUUID(u.Bytes)
}

func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidString(u)
	return &s
}

func uuidToPgtype(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	b, err := parseUUID(s)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: b, Valid: true}
}

func optionalUUID(s *string) pgtype.UUID {
	if s == nil || *s == "" {
		return pgtype.UUID{Valid: false}
	}
	return uuidToPgtype(*s)
}

func formatUUID(b [16]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	idx := 0
	for i := 0; i < 16; i++ {
		out[idx] = hex[b[i]>>4]
		out[idx+1] = hex[b[i]&0x0f]
		idx += 2
		if i == 3 || i == 5 || i == 7 || i == 9 {
			out[idx] = '-'
			idx++
		}
	}
	return string(out)
}

func parseUUID(s string) ([16]byte, error) {
	var b [16]byte
	if len(s) != 36 {
		return b, errors.New("invalid uuid length")
	}
	hex := func(c byte) (byte, error) {
		switch {
		case c >= '0' && c <= '9':
			return c - '0', nil
		case c >= 'a' && c <= 'f':
			return c - 'a' + 10, nil
		case c >= 'A' && c <= 'F':
			return c - 'A' + 10, nil
		}
		return 0, errors.New("invalid uuid char")
	}
	idx := 0
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if s[i] != '-' {
				return b, errors.New("invalid uuid format")
			}
			continue
		}
		hi, err := hex(s[i])
		if err != nil {
			return b, err
		}
		i++
		lo, err := hex(s[i])
		if err != nil {
			return b, err
		}
		b[idx] = hi<<4 | lo
		idx++
	}
	return b, nil
}

// compile-time guard: ensure *StructureRepository satisfies the port.
var _ domain.StructureRepository = (*StructureRepository)(nil)
