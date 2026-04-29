package entities

import "time"

// CommonAreaRule representa un override de regla de configuracion por
// zona comun (almacenada como clave-valor JSONB).
type CommonAreaRule struct {
	ID           string
	CommonAreaID string
	RuleKey      string
	RuleValue    []byte
	Description  *string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}
