// Package domain define las interfaces (puertos) del modulo tenant_config
// que la capa de aplicacion consume y que infra implementa.
package domain

import (
	"context"
	"errors"

	"github.com/saas-ph/api/internal/modules/tenant_config/domain/entities"
)

// ErrSettingNotFound se devuelve cuando una key consultada no existe (o
// esta archivada).
var ErrSettingNotFound = errors.New("tenant_config: setting not found")

// ErrBrandingNotFound se devuelve si la fila singleton no esta presente.
// En la practica el provisioning siembra siempre la fila, pero el repo
// debe contemplar el caso para tests/fixtures.
var ErrBrandingNotFound = errors.New("tenant_config: branding not found")

// ErrVersionMismatch se devuelve cuando el update optimista no encuentra
// la version esperada (otro proceso actualizo entre la lectura y este
// write).
var ErrVersionMismatch = errors.New("tenant_config: version mismatch (concurrent update)")

// ListSettingsFilter agrupa los filtros y la paginacion del listado.
type ListSettingsFilter struct {
	// Category filtra por categoria exacta. Vacio = sin filtro.
	Category string
	// Limit es el tamano de pagina. Si <=0, el repo aplica un default.
	Limit int32
	// Offset es el desplazamiento en filas (0-based).
	Offset int32
}

// SettingsRepository es el puerto que persiste tenant_settings.
type SettingsRepository interface {
	// List devuelve la pagina de settings activas y el total de filas
	// (para construir paginacion).
	List(ctx context.Context, f ListSettingsFilter) (items []entities.Setting, total int64, err error)
	// Get devuelve la setting activa con esa key, o ErrSettingNotFound.
	Get(ctx context.Context, key string) (entities.Setting, error)
	// Upsert crea o actualiza la setting, incrementando version si ya
	// existia. `actorID` es el user_id que origina la operacion (para
	// auditar `created_by`/`updated_by`); puede ser vacio en operaciones
	// internas/sistema.
	Upsert(ctx context.Context, in UpsertSettingInput) (entities.Setting, error)
	// Archive marca la setting como archived (soft-delete). Devuelve
	// ErrSettingNotFound si no existe o ya estaba archivada.
	Archive(ctx context.Context, key string, actorID string) (entities.Setting, error)
}

// UpsertSettingInput agrupa los datos de un upsert.
type UpsertSettingInput struct {
	Key         string
	Value       []byte // JSON crudo (JSONB)
	Description string // vacio = no cambiar (en update)
	Category    string // vacio = no cambiar (en update)
	ActorID     string // user_id que ejecuta (para audit)
}

// BrandingRepository es el puerto que persiste la fila singleton de
// tenant_branding.
type BrandingRepository interface {
	// Get devuelve la fila singleton vigente o ErrBrandingNotFound.
	Get(ctx context.Context) (entities.Branding, error)
	// Update actualiza el branding usando concurrencia optimista. Si
	// `expectedVersion` no coincide con la fila actual, devuelve
	// ErrVersionMismatch.
	Update(ctx context.Context, in UpdateBrandingInput) (entities.Branding, error)
}

// UpdateBrandingInput agrupa los datos de un update completo del branding.
type UpdateBrandingInput struct {
	DisplayName     string
	LogoURL         *string
	PrimaryColor    *string
	SecondaryColor  *string
	Timezone        string
	Locale          string
	ActorID         string
	ExpectedVersion int32
}
