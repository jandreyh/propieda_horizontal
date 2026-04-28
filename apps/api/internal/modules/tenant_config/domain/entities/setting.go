// Package entities define las entidades de dominio del modulo tenant_config.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos; nunca aparece como columna
//     ni como campo de dominio.
package entities

import (
	"time"
)

// SettingStatus enumera los estados validos de una Setting.
type SettingStatus string

const (
	// SettingStatusActive marca la setting como vigente.
	SettingStatusActive SettingStatus = "active"
	// SettingStatusArchived marca la setting como soft-deleted.
	SettingStatusArchived SettingStatus = "archived"
)

// Setting representa un par (key, value JSONB) de configuracion del tenant.
//
// `Value` es un JSON crudo en bytes (lo que viene/va a JSONB). El dominio no
// interpreta el contenido; los usecases lo serializan/deserializan segun
// el contrato del DTO.
type Setting struct {
	ID          string
	Key         string
	Value       []byte
	Description string
	Category    string
	Status      SettingStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}

// IsArchived indica si la Setting esta soft-deleted.
func (s Setting) IsArchived() bool {
	return s.Status == SettingStatusArchived || s.DeletedAt != nil
}
