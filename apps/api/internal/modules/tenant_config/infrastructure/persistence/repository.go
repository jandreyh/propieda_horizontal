// Package persistence implementa los puertos del modulo tenant_config
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
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saas-ph/api/internal/modules/tenant_config/domain"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/entities"
	tenantcfgdb "github.com/saas-ph/api/internal/modules/tenant_config/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// SettingsRepository implementa domain.SettingsRepository sobre la
// Tenant DB resuelta del contexto.
type SettingsRepository struct{}

// NewSettingsRepository construye una instancia stateless.
func NewSettingsRepository() *SettingsRepository {
	return &SettingsRepository{}
}

func querier(ctx context.Context) (*tenantcfgdb.Queries, error) {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("tenant_config: tenant pool is nil")
	}
	return tenantcfgdb.New(t.Pool), nil
}

// List implementa domain.SettingsRepository.
func (r *SettingsRepository) List(ctx context.Context, f domain.ListSettingsFilter) ([]entities.Setting, int64, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, 0, err
	}
	var cat *string
	if f.Category != "" {
		c := f.Category
		cat = &c
	}
	rows, err := q.ListSettings(ctx, tenantcfgdb.ListSettingsParams{
		Limit:    f.Limit,
		Offset:   f.Offset,
		Category: cat,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountSettings(ctx, cat)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entities.Setting, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapSetting(row))
	}
	return out, total, nil
}

// Get implementa domain.SettingsRepository.
func (r *SettingsRepository) Get(ctx context.Context, key string) (entities.Setting, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Setting{}, err
	}
	row, err := q.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Setting{}, domain.ErrSettingNotFound
		}
		return entities.Setting{}, err
	}
	return mapSetting(row), nil
}

// Upsert implementa domain.SettingsRepository.
func (r *SettingsRepository) Upsert(ctx context.Context, in domain.UpsertSettingInput) (entities.Setting, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Setting{}, err
	}
	var desc, cat *string
	if in.Description != "" {
		d := in.Description
		desc = &d
	}
	if in.Category != "" {
		c := in.Category
		cat = &c
	}
	row, err := q.UpsertSetting(ctx, tenantcfgdb.UpsertSettingParams{
		Key:         in.Key,
		Value:       in.Value,
		Description: desc,
		Category:    cat,
		CreatedBy:   uuidPtrToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.Setting{}, err
	}
	return mapSetting(row), nil
}

// Archive implementa domain.SettingsRepository.
func (r *SettingsRepository) Archive(ctx context.Context, key, actorID string) (entities.Setting, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Setting{}, err
	}
	row, err := q.ArchiveSetting(ctx, tenantcfgdb.ArchiveSettingParams{
		Key:       key,
		DeletedBy: uuidPtrToPgtype(actorID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Setting{}, domain.ErrSettingNotFound
		}
		return entities.Setting{}, err
	}
	return mapSetting(row), nil
}

// BrandingRepository implementa domain.BrandingRepository.
type BrandingRepository struct{}

// NewBrandingRepository construye una instancia stateless.
func NewBrandingRepository() *BrandingRepository {
	return &BrandingRepository{}
}

// Get implementa domain.BrandingRepository.
func (r *BrandingRepository) Get(ctx context.Context) (entities.Branding, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Branding{}, err
	}
	row, err := q.GetBranding(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Branding{}, domain.ErrBrandingNotFound
		}
		return entities.Branding{}, err
	}
	return mapBranding(row), nil
}

// Update implementa domain.BrandingRepository.
func (r *BrandingRepository) Update(ctx context.Context, in domain.UpdateBrandingInput) (entities.Branding, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Branding{}, err
	}
	row, err := q.UpdateBranding(ctx, tenantcfgdb.UpdateBrandingParams{
		DisplayName:    in.DisplayName,
		LogoUrl:        in.LogoURL,
		PrimaryColor:   in.PrimaryColor,
		SecondaryColor: in.SecondaryColor,
		Timezone:       in.Timezone,
		Locale:         in.Locale,
		UpdatedBy:      uuidPtrToPgtype(in.ActorID),
		Version:        in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Distinguir entre "no existe" y "version mismatch": si la fila
			// existe (Get devuelve algo) pero version no calza, devolvemos
			// ErrVersionMismatch; si tampoco existe, ErrBrandingNotFound.
			if _, gerr := r.Get(ctx); gerr == nil {
				return entities.Branding{}, domain.ErrVersionMismatch
			}
			return entities.Branding{}, domain.ErrBrandingNotFound
		}
		return entities.Branding{}, err
	}
	return mapBranding(row), nil
}

// --- helpers de mapeo ---

func mapSetting(r tenantcfgdb.TenantSetting) entities.Setting {
	out := entities.Setting{
		ID:        uuidString(r.ID),
		Key:       r.Key,
		Value:     append([]byte(nil), r.Value...),
		Status:    entities.SettingStatus(r.Status),
		CreatedAt: tsToTime(r.CreatedAt),
		UpdatedAt: tsToTime(r.UpdatedAt),
		Version:   r.Version,
	}
	if r.Description != nil {
		out.Description = *r.Description
	}
	if r.Category != nil {
		out.Category = *r.Category
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

func mapBranding(r tenantcfgdb.TenantBranding) entities.Branding {
	out := entities.Branding{
		ID:             uuidString(r.ID),
		DisplayName:    r.DisplayName,
		LogoURL:        r.LogoUrl,
		PrimaryColor:   r.PrimaryColor,
		SecondaryColor: r.SecondaryColor,
		Timezone:       r.Timezone,
		Locale:         r.Locale,
		Status:         entities.BrandingStatus(r.Status),
		CreatedAt:      tsToTime(r.CreatedAt),
		UpdatedAt:      tsToTime(r.UpdatedAt),
		Version:        r.Version,
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
	// pgtype.UUID expone un []byte de 16; usamos el formato canonico via
	// la libreria de pgx (Bytes). Construimos el string manualmente para
	// evitar dependencia extra de google/uuid.
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
