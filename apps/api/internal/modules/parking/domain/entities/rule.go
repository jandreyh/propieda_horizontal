package entities

import "time"

// ParkingRule representa un override de regla de configuracion del modulo
// de parking por tenant (almacenada como clave-valor JSONB).
type ParkingRule struct {
	ID          string
	RuleKey     string
	RuleValue   []byte
	Description *string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string
	Version     int32
}
