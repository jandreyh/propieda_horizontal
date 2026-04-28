// Package persistence implementa los puertos del modulo people usando el
// codigo generado por sqlc.
//
// Reglas:
//   - El pool del Tenant DB se obtiene del contexto via tenantctx.FromCtx.
//   - NO se usa database/sql ni SQL inline. Toda query vive en .sql.
//   - El repo mapea entre tipos pgtype (sqlc) y los tipos de dominio.
package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saas-ph/api/internal/modules/people/domain"
	"github.com/saas-ph/api/internal/modules/people/domain/entities"
	peopledb "github.com/saas-ph/api/internal/modules/people/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// VehicleRepository implementa domain.VehicleRepository sobre la Tenant
// DB resuelta del contexto.
type VehicleRepository struct{}

// NewVehicleRepository construye una instancia stateless.
func NewVehicleRepository() *VehicleRepository {
	return &VehicleRepository{}
}

// AssignmentRepository implementa domain.AssignmentRepository sobre la
// Tenant DB resuelta del contexto.
type AssignmentRepository struct{}

// NewAssignmentRepository construye una instancia stateless.
func NewAssignmentRepository() *AssignmentRepository {
	return &AssignmentRepository{}
}

func querier(ctx context.Context) (*peopledb.Queries, error) {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("people: tenant pool is nil")
	}
	return peopledb.New(t.Pool), nil
}

// --- VehicleRepository ---

// Create implementa domain.VehicleRepository.
func (r *VehicleRepository) Create(ctx context.Context, in domain.CreateVehicleInput) (entities.Vehicle, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Vehicle{}, err
	}
	row, err := q.CreateVehicle(ctx, peopledb.CreateVehicleParams{
		Plate:     in.Plate,
		Type:      string(in.Type),
		Brand:     in.Brand,
		Model:     in.Model,
		Color:     in.Color,
		Year:      in.Year,
		CreatedBy: uuidPtrToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Vehicle{}, domain.ErrPlateAlreadyExists
		}
		return entities.Vehicle{}, err
	}
	return mapVehicle(row), nil
}

// GetByID implementa domain.VehicleRepository.
func (r *VehicleRepository) GetByID(ctx context.Context, id string) (entities.Vehicle, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Vehicle{}, err
	}
	row, err := q.GetVehicleByID(ctx, uuidPtrToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Vehicle{}, domain.ErrVehicleNotFound
		}
		return entities.Vehicle{}, err
	}
	return mapVehicle(row), nil
}

// GetByPlate implementa domain.VehicleRepository.
func (r *VehicleRepository) GetByPlate(ctx context.Context, plate string) (entities.Vehicle, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Vehicle{}, err
	}
	row, err := q.GetVehicleByPlate(ctx, plate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Vehicle{}, domain.ErrVehicleNotFound
		}
		return entities.Vehicle{}, err
	}
	return mapVehicle(row), nil
}

// ListAll implementa domain.VehicleRepository.
func (r *VehicleRepository) ListAll(ctx context.Context) ([]entities.Vehicle, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAllVehicles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Vehicle, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapVehicle(row))
	}
	return out, nil
}

// --- AssignmentRepository ---

// Assign implementa domain.AssignmentRepository.
func (r *AssignmentRepository) Assign(ctx context.Context, in domain.AssignInput) (entities.UnitVehicleAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.UnitVehicleAssignment{}, err
	}
	row, err := q.AssignVehicleToUnit(ctx, peopledb.AssignVehicleToUnitParams{
		UnitID:    uuidPtrToPgtype(in.UnitID),
		VehicleID: uuidPtrToPgtype(in.VehicleID),
		SinceDate: dateFromPtr(in.SinceDate),
		CreatedBy: uuidPtrToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.UnitVehicleAssignment{}, domain.ErrVehicleAlreadyAssigned
		}
		if isFKViolation(err) {
			// FK violation puede ser por unit_id o vehicle_id inexistente.
			// Para diferenciar deberiamos parsear el detalle; en MVP
			// devolvemos NotFound generico via Vehicle (mas comun).
			return entities.UnitVehicleAssignment{}, domain.ErrVehicleNotFound
		}
		return entities.UnitVehicleAssignment{}, err
	}
	return mapAssignment(row), nil
}

// ListActiveByUnit implementa domain.AssignmentRepository.
func (r *AssignmentRepository) ListActiveByUnit(ctx context.Context, unitID string) ([]entities.UnitVehicleAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveAssignmentsByUnit(ctx, uuidPtrToPgtype(unitID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.UnitVehicleAssignment, 0, len(rows))
	for _, row := range rows {
		a := entities.UnitVehicleAssignment{
			ID:        uuidString(row.ID),
			UnitID:    uuidString(row.UnitID),
			VehicleID: uuidString(row.VehicleID),
			SinceDate: dateToTime(row.SinceDate),
			Status:    entities.AssignmentStatus(row.Status),
			CreatedAt: tsToTime(row.CreatedAt),
			UpdatedAt: tsToTime(row.UpdatedAt),
			Version:   row.Version,
		}
		if row.UntilDate.Valid {
			t := row.UntilDate.Time
			a.UntilDate = &t
		}
		if row.DeletedAt.Valid {
			t := row.DeletedAt.Time
			a.DeletedAt = &t
		}
		if s := uuidStringPtr(row.CreatedBy); s != nil {
			a.CreatedBy = s
		}
		if s := uuidStringPtr(row.UpdatedBy); s != nil {
			a.UpdatedBy = s
		}
		if s := uuidStringPtr(row.DeletedBy); s != nil {
			a.DeletedBy = s
		}
		// Materializa el vehiculo asociado (join).
		v := entities.Vehicle{
			ID:     uuidString(row.VehicleID),
			Plate:  row.Plate,
			Type:   entities.VehicleType(row.VehicleType),
			Brand:  row.Brand,
			Model:  row.Model,
			Color:  row.Color,
			Year:   row.Year,
			Status: entities.VehicleStatusActive,
		}
		a.Vehicle = &v
		out = append(out, a)
	}
	return out, nil
}

// End implementa domain.AssignmentRepository.
func (r *AssignmentRepository) End(ctx context.Context, in domain.EndAssignmentInput) (entities.UnitVehicleAssignment, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.UnitVehicleAssignment{}, err
	}
	row, err := q.EndVehicleAssignment(ctx, peopledb.EndVehicleAssignmentParams{
		ID:        uuidPtrToPgtype(in.AssignmentID),
		UntilDate: dateFromPtr(in.UntilDate),
		UpdatedBy: uuidPtrToPgtype(in.ActorID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.UnitVehicleAssignment{}, domain.ErrAssignmentNotFound
		}
		return entities.UnitVehicleAssignment{}, err
	}
	return mapAssignment(row), nil
}

// --- helpers de mapeo ---

func mapVehicle(r peopledb.Vehicle) entities.Vehicle {
	out := entities.Vehicle{
		ID:        uuidString(r.ID),
		Plate:     r.Plate,
		Type:      entities.VehicleType(r.Type),
		Brand:     r.Brand,
		Model:     r.Model,
		Color:     r.Color,
		Year:      r.Year,
		Status:    entities.VehicleStatus(r.Status),
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
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

func mapAssignment(r peopledb.UnitVehicleAssignment) entities.UnitVehicleAssignment {
	out := entities.UnitVehicleAssignment{
		ID:        uuidString(r.ID),
		UnitID:    uuidString(r.UnitID),
		VehicleID: uuidString(r.VehicleID),
		SinceDate: dateToTime(r.SinceDate),
		Status:    entities.AssignmentStatus(r.Status),
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
	}
	if r.UntilDate.Valid {
		t := r.UntilDate.Time
		out.UntilDate = &t
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

func dateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

func dateFromPtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return formatUUID(b)
}

func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidString(u)
	return &s
}

func uuidPtrToPgtype(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	b, err := parseUUID(s)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: b, Valid: true}
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
	hexByte := func(c byte) (byte, error) {
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
		hi, err := hexByte(s[i])
		if err != nil {
			return b, err
		}
		i++
		lo, err := hexByte(s[i])
		if err != nil {
			return b, err
		}
		b[idx] = hi<<4 | lo
		idx++
	}
	return b, nil
}

// isUniqueViolation devuelve true si err es un pg "23505 unique_violation".
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// isFKViolation devuelve true si err es un pg "23503 foreign_key_violation".
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}
