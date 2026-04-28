// Package entities define las entidades de dominio del modulo packages.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

import "time"

// PackageStatus enumera los estados validos del ciclo de vida de un
// paquete.
type PackageStatus string

const (
	// PackageStatusReceived es el paquete fue recibido por porteria y
	// espera ser entregado al residente.
	PackageStatusReceived PackageStatus = "received"
	// PackageStatusDelivered es el paquete fue entregado al residente.
	PackageStatusDelivered PackageStatus = "delivered"
	// PackageStatusReturned es el paquete fue devuelto al transportador
	// (no se entrega; ej. paquete sospechoso o destinatario incorrecto).
	PackageStatusReturned PackageStatus = "returned"
)

// IsValid indica si el status es uno de los enumerados.
func (s PackageStatus) IsValid() bool {
	switch s {
	case PackageStatusReceived, PackageStatusDelivered, PackageStatusReturned:
		return true
	}
	return false
}

// Package representa un paquete recibido por porteria.
type Package struct {
	ID                  string
	UnitID              string
	RecipientName       string
	CategoryID          *string
	ReceivedEvidenceURL *string
	Carrier             *string
	TrackingNumber      *string
	ReceivedByUserID    string
	ReceivedAt          time.Time
	DeliveredAt         *time.Time
	ReturnedAt          *time.Time
	Status              PackageStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
	CreatedBy           *string
	UpdatedBy           *string
	DeletedBy           *string
	Version             int32
}

// IsReceived indica si el paquete sigue en porteria.
func (p Package) IsReceived() bool {
	return p.Status == PackageStatusReceived && p.DeletedAt == nil
}
