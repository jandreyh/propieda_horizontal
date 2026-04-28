// Package persistence implementa los puertos del modulo access_control
// usando el codigo generado por sqlc.
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

	"github.com/saas-ph/api/internal/modules/access_control/domain"
	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	accessdb "github.com/saas-ph/api/internal/modules/access_control/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// --- Repositorios ---

// BlacklistRepository implementa domain.BlacklistRepository.
type BlacklistRepository struct{}

// NewBlacklistRepository construye una instancia stateless.
func NewBlacklistRepository() *BlacklistRepository { return &BlacklistRepository{} }

// PreRegistrationRepository implementa domain.PreRegistrationRepository.
type PreRegistrationRepository struct{}

// NewPreRegistrationRepository construye una instancia stateless.
func NewPreRegistrationRepository() *PreRegistrationRepository { return &PreRegistrationRepository{} }

// VisitorEntryRepository implementa domain.VisitorEntryRepository.
type VisitorEntryRepository struct{}

// NewVisitorEntryRepository construye una instancia stateless.
func NewVisitorEntryRepository() *VisitorEntryRepository { return &VisitorEntryRepository{} }

func querier(ctx context.Context) (*accessdb.Queries, error) {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("access_control: tenant pool is nil")
	}
	return accessdb.New(t.Pool), nil
}

// --- BlacklistRepository ---

// Get implementa domain.BlacklistRepository. Devuelve nil si no hay match.
func (r *BlacklistRepository) Get(ctx context.Context, dt entities.DocumentType, dn string) (*entities.BlacklistEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	row, err := q.GetBlacklistByDocument(ctx, accessdb.GetBlacklistByDocumentParams{
		DocumentType:   string(dt),
		DocumentNumber: dn,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	out := mapBlacklist(row)
	return &out, nil
}

// Create implementa domain.BlacklistRepository.
func (r *BlacklistRepository) Create(ctx context.Context, in domain.CreateBlacklistInput) (entities.BlacklistEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.BlacklistEntry{}, err
	}
	row, err := q.CreateBlacklistEntry(ctx, accessdb.CreateBlacklistEntryParams{
		DocumentType:     string(in.DocumentType),
		DocumentNumber:   in.DocumentNumber,
		FullName:         in.FullName,
		Reason:           in.Reason,
		ReportedByUnitID: uuidPtrToPgtype(strPtrToString(in.ReportedByUnitID)),
		ReportedByUserID: uuidPtrToPgtype(strPtrToString(in.ReportedByUserID)),
		ExpiresAt:        tsFromPtr(in.ExpiresAt),
		CreatedBy:        uuidPtrToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.BlacklistEntry{}, domain.ErrBlacklistAlreadyExists
		}
		return entities.BlacklistEntry{}, err
	}
	return mapBlacklist(row), nil
}

// List implementa domain.BlacklistRepository.
func (r *BlacklistRepository) List(ctx context.Context) ([]entities.BlacklistEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListBlacklist(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.BlacklistEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapBlacklist(row))
	}
	return out, nil
}

// Archive implementa domain.BlacklistRepository.
func (r *BlacklistRepository) Archive(ctx context.Context, id, actorID string) (entities.BlacklistEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.BlacklistEntry{}, err
	}
	row, err := q.ArchiveBlacklistEntry(ctx, accessdb.ArchiveBlacklistEntryParams{
		ID:        uuidPtrToPgtype(id),
		DeletedBy: uuidPtrToPgtype(actorID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.BlacklistEntry{}, domain.ErrBlacklistNotFound
		}
		return entities.BlacklistEntry{}, err
	}
	return mapBlacklist(row), nil
}

// --- PreRegistrationRepository ---

// Create implementa domain.PreRegistrationRepository.
func (r *PreRegistrationRepository) Create(ctx context.Context, in domain.CreatePreRegistrationInput) (entities.PreRegistration, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PreRegistration{}, err
	}
	row, err := q.CreatePreRegistration(ctx, accessdb.CreatePreRegistrationParams{
		UnitID:                uuidPtrToPgtype(in.UnitID),
		CreatedByUserID:       uuidPtrToPgtype(in.CreatedByUserID),
		VisitorFullName:       in.VisitorFullName,
		VisitorDocumentType:   in.VisitorDocumentType,
		VisitorDocumentNumber: in.VisitorDocumentNumber,
		ExpectedAt:            tsFromPtr(in.ExpectedAt),
		ExpiresAt:             pgtype.Timestamptz{Time: in.ExpiresAt, Valid: true},
		MaxUses:               in.MaxUses,
		QrCodeHash:            in.QRCodeHash,
	})
	if err != nil {
		return entities.PreRegistration{}, err
	}
	return mapPreReg(row), nil
}

// GetByQRHash implementa domain.PreRegistrationRepository.
func (r *PreRegistrationRepository) GetByQRHash(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PreRegistration{}, err
	}
	row, err := q.GetPreRegistrationByQRHash(ctx, qrHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.PreRegistration{}, domain.ErrPreregistrationNotFound
		}
		return entities.PreRegistration{}, err
	}
	return mapPreReg(row), nil
}

// ConsumeOne implementa domain.PreRegistrationRepository. Si la query
// atomica afecta 0 filas, busca el motivo (no encontrado, expirado,
// agotado) para devolver el sentinel apropiado.
func (r *PreRegistrationRepository) ConsumeOne(ctx context.Context, qrHash string) (entities.PreRegistration, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.PreRegistration{}, err
	}
	row, err := q.IncrementPreRegistrationUses(ctx, qrHash)
	if err == nil {
		return mapPreReg(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return entities.PreRegistration{}, err
	}
	// 0 filas afectadas: diagnostica el motivo concreto.
	cur, gerr := q.GetPreRegistrationByQRHash(ctx, qrHash)
	if gerr != nil {
		if errors.Is(gerr, pgx.ErrNoRows) {
			return entities.PreRegistration{}, domain.ErrPreregistrationNotFound
		}
		return entities.PreRegistration{}, gerr
	}
	now := time.Now()
	if cur.ExpiresAt.Valid && !cur.ExpiresAt.Time.After(now) {
		return entities.PreRegistration{}, domain.ErrPreregistrationExpired
	}
	if cur.UsesCount >= cur.MaxUses || cur.Status != "active" {
		return entities.PreRegistration{}, domain.ErrPreregistrationExhausted
	}
	return entities.PreRegistration{}, domain.ErrPreregistrationExhausted
}

// --- VisitorEntryRepository ---

// Create implementa domain.VisitorEntryRepository.
func (r *VisitorEntryRepository) Create(ctx context.Context, in domain.CreateVisitorEntryInput) (entities.VisitorEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorEntry{}, err
	}
	row, err := q.CreateVisitorEntry(ctx, accessdb.CreateVisitorEntryParams{
		UnitID:                uuidPtrToPgtype(strPtrToString(in.UnitID)),
		PreRegistrationID:     uuidPtrToPgtype(strPtrToString(in.PreRegistrationID)),
		VisitorFullName:       in.VisitorFullName,
		VisitorDocumentType:   in.VisitorDocumentType,
		VisitorDocumentNumber: in.VisitorDocumentNumber,
		PhotoUrl:              in.PhotoURL,
		GuardID:               uuidPtrToPgtype(in.GuardID),
		Source:                string(in.Source),
		Notes:                 in.Notes,
		Status:                string(in.Status),
	})
	if err != nil {
		return entities.VisitorEntry{}, err
	}
	return mapEntry(row), nil
}

// Close implementa domain.VisitorEntryRepository.
func (r *VisitorEntryRepository) Close(ctx context.Context, entryID, actorID string) (entities.VisitorEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorEntry{}, err
	}
	row, err := q.CloseVisitorEntry(ctx, accessdb.CloseVisitorEntryParams{
		ID:        uuidPtrToPgtype(entryID),
		UpdatedBy: uuidPtrToPgtype(actorID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.VisitorEntry{}, domain.ErrEntryNotFound
		}
		return entities.VisitorEntry{}, err
	}
	return mapEntry(row), nil
}

// ListActive implementa domain.VisitorEntryRepository.
func (r *VisitorEntryRepository) ListActive(ctx context.Context) ([]entities.VisitorEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveVisits(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.VisitorEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapEntry(row))
	}
	return out, nil
}

// GetByID implementa domain.VisitorEntryRepository.
func (r *VisitorEntryRepository) GetByID(ctx context.Context, id string) (entities.VisitorEntry, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VisitorEntry{}, err
	}
	row, err := q.GetEntryByID(ctx, uuidPtrToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.VisitorEntry{}, domain.ErrEntryNotFound
		}
		return entities.VisitorEntry{}, err
	}
	return mapEntry(row), nil
}

// --- helpers de mapeo ---

func mapBlacklist(r accessdb.BlacklistedPerson) entities.BlacklistEntry {
	out := entities.BlacklistEntry{
		ID:             uuidString(r.ID),
		DocumentType:   entities.DocumentType(r.DocumentType),
		DocumentNumber: r.DocumentNumber,
		FullName:       r.FullName,
		Reason:         r.Reason,
		Status:         entities.BlacklistStatus(r.Status),
		CreatedAt:      tsToTime(r.CreatedAt),
		UpdatedAt:      tsToTime(r.UpdatedAt),
		Version:        r.Version,
	}
	if s := uuidStringPtr(r.ReportedByUnitID); s != nil {
		out.ReportedByUnitID = s
	}
	if s := uuidStringPtr(r.ReportedByUserID); s != nil {
		out.ReportedByUserID = s
	}
	if r.ExpiresAt.Valid {
		t := r.ExpiresAt.Time
		out.ExpiresAt = &t
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

func mapPreReg(r accessdb.VisitorPreRegistration) entities.PreRegistration {
	out := entities.PreRegistration{
		ID:                    uuidString(r.ID),
		UnitID:                uuidString(r.UnitID),
		CreatedByUserID:       uuidString(r.CreatedByUserID),
		VisitorFullName:       r.VisitorFullName,
		VisitorDocumentType:   r.VisitorDocumentType,
		VisitorDocumentNumber: r.VisitorDocumentNumber,
		MaxUses:               r.MaxUses,
		UsesCount:             r.UsesCount,
		QRCodeHash:            r.QrCodeHash,
		Status:                entities.PreRegistrationStatus(r.Status),
		CreatedAt:             tsToTime(r.CreatedAt),
		UpdatedAt:             tsToTime(r.UpdatedAt),
		Version:               r.Version,
	}
	if r.ExpectedAt.Valid {
		t := r.ExpectedAt.Time
		out.ExpectedAt = &t
	}
	if r.ExpiresAt.Valid {
		out.ExpiresAt = r.ExpiresAt.Time
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

func mapEntry(r accessdb.VisitorEntry) entities.VisitorEntry {
	out := entities.VisitorEntry{
		ID:                    uuidString(r.ID),
		VisitorFullName:       r.VisitorFullName,
		VisitorDocumentType:   r.VisitorDocumentType,
		VisitorDocumentNumber: r.VisitorDocumentNumber,
		PhotoURL:              r.PhotoUrl,
		GuardID:               uuidString(r.GuardID),
		EntryTime:             tsToTime(r.EntryTime),
		Source:                entities.VisitorEntrySource(r.Source),
		Notes:                 r.Notes,
		Status:                entities.VisitorEntryStatus(r.Status),
		CreatedAt:             tsToTime(r.CreatedAt),
		UpdatedAt:             tsToTime(r.UpdatedAt),
		Version:               r.Version,
	}
	if s := uuidStringPtr(r.UnitID); s != nil {
		out.UnitID = s
	}
	if s := uuidStringPtr(r.PreRegistrationID); s != nil {
		out.PreRegistrationID = s
	}
	if r.ExitTime.Valid {
		t := r.ExitTime.Time
		out.ExitTime = &t
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

// --- pgtype helpers ---

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func tsFromPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func strPtrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	v, err := u.Value()
	if err != nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidString(u)
	if s == "" {
		return nil
	}
	return &s
}

func uuidPtrToPgtype(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{Valid: false}
	}
	return u
}

// isUniqueViolation devuelve true si err es un pg "23505 unique_violation".
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// Compile-time checks: el repo implementa el puerto del dominio.
var (
	_ domain.BlacklistRepository       = (*BlacklistRepository)(nil)
	_ domain.PreRegistrationRepository = (*PreRegistrationRepository)(nil)
	_ domain.VisitorEntryRepository    = (*VisitorEntryRepository)(nil)
)
