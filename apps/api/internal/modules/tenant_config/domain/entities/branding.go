package entities

import "time"

// BrandingStatus enumera los estados validos de Branding.
type BrandingStatus string

const (
	// BrandingStatusActive marca el branding como vigente.
	BrandingStatusActive BrandingStatus = "active"
	// BrandingStatusArchived no se usa hoy (singleton), pero se conserva por
	// consistencia con el modelo estandar.
	BrandingStatusArchived BrandingStatus = "archived"
)

// Branding representa la fila singleton de identidad visual del tenant.
//
// La regla de "una sola fila por tenant" se enforza en la base con la
// columna `singleton` UNIQUE; el dominio asume que solo existe una.
type Branding struct {
	ID             string
	DisplayName    string
	LogoURL        *string
	PrimaryColor   *string // hex (#RRGGBB) si presente
	SecondaryColor *string // hex (#RRGGBB) si presente
	Timezone       string
	Locale         string
	Status         BrandingStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
	CreatedBy      *string
	UpdatedBy      *string
	DeletedBy      *string
	Version        int32
}
