// Package usecases orquesta la logica de aplicacion del modulo
// tenant_config. Cada usecase recibe sus dependencias por inyeccion
// (interfaces) y NO conoce HTTP ni la base.
package usecases

import (
	"context"
	"errors"

	apperrors "github.com/saas-ph/api/internal/platform/errors"

	"github.com/saas-ph/api/internal/modules/tenant_config/domain"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/entities"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/policies"
)

// defaultListLimit es el tamano de pagina por defecto si el caller no
// especifica.
const (
	defaultListLimit int32 = 50
	maxListLimit     int32 = 200
)

// GetSetting devuelve una setting por key.
type GetSetting struct {
	Repo domain.SettingsRepository
}

// Execute valida key y delega al repo.
func (u GetSetting) Execute(ctx context.Context, key string) (entities.Setting, error) {
	if err := policies.ValidateSettingKey(key); err != nil {
		return entities.Setting{}, apperrors.BadRequest(err.Error())
	}
	s, err := u.Repo.Get(ctx, key)
	if err != nil {
		if errors.Is(err, domain.ErrSettingNotFound) {
			return entities.Setting{}, apperrors.NotFound("setting not found")
		}
		return entities.Setting{}, apperrors.Internal("failed to load setting")
	}
	return s, nil
}

// ListSettings lista settings paginadas (opcional filtro por categoria).
type ListSettings struct {
	Repo domain.SettingsRepository
}

// ListInput agrupa input del listado.
type ListInput struct {
	Category string
	Limit    int32
	Offset   int32
}

// ListOutput agrupa salida del listado.
type ListOutput struct {
	Items  []entities.Setting
	Total  int64
	Limit  int32
	Offset int32
}

// Execute aplica defaults, valida y delega.
func (u ListSettings) Execute(ctx context.Context, in ListInput) (ListOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	items, total, err := u.Repo.List(ctx, domain.ListSettingsFilter{
		Category: in.Category,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return ListOutput{}, apperrors.Internal("failed to list settings")
	}
	return ListOutput{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// SetSetting crea o actualiza una setting (PUT idempotente).
type SetSetting struct {
	Repo domain.SettingsRepository
}

// SetInput agrupa input del upsert.
type SetInput struct {
	Key         string
	Value       []byte
	Description string
	Category    string
	ActorID     string
}

// Execute valida y delega.
func (u SetSetting) Execute(ctx context.Context, in SetInput) (entities.Setting, error) {
	if err := policies.ValidateSettingKey(in.Key); err != nil {
		return entities.Setting{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateSettingValue(in.Value); err != nil {
		return entities.Setting{}, apperrors.BadRequest(err.Error())
	}
	s, err := u.Repo.Upsert(ctx, domain.UpsertSettingInput{
		Key:         in.Key,
		Value:       in.Value,
		Description: in.Description,
		Category:    in.Category,
		ActorID:     in.ActorID,
	})
	if err != nil {
		return entities.Setting{}, apperrors.Internal("failed to upsert setting")
	}
	return s, nil
}

// ArchiveSetting hace soft-delete por key.
type ArchiveSetting struct {
	Repo domain.SettingsRepository
}

// ArchiveInput agrupa input de la operacion.
type ArchiveInput struct {
	Key     string
	ActorID string
}

// Execute valida y delega.
func (u ArchiveSetting) Execute(ctx context.Context, in ArchiveInput) (entities.Setting, error) {
	if err := policies.ValidateSettingKey(in.Key); err != nil {
		return entities.Setting{}, apperrors.BadRequest(err.Error())
	}
	s, err := u.Repo.Archive(ctx, in.Key, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrSettingNotFound) {
			return entities.Setting{}, apperrors.NotFound("setting not found")
		}
		return entities.Setting{}, apperrors.Internal("failed to archive setting")
	}
	return s, nil
}
