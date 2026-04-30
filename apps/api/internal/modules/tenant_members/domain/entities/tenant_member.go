// Package entities define las entidades del modulo tenant_members.
package entities

import (
	"time"

	"github.com/google/uuid"
)

// TenantMember representa una fila de `tenant_user_links` enriquecida
// con datos del PlatformUser asociado (nombre, email, codigo) que el
// servidor compone consultando la DB central.
//
// Los permisos se derivan del campo Role; el cliente NO los infiere.
type TenantMember struct {
	ID             uuid.UUID
	PlatformUserID uuid.UUID
	Role           string
	PrimaryUnitID  *uuid.UUID
	CarteraStatus  *string
	FechaIngreso   *time.Time
	Status         string // active | blocked
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int32

	// Datos del PlatformUser (cache server-side por request).
	Names      string
	LastNames  string
	Email      string
	PublicCode string
}
