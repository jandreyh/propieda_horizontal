package entities

import "time"

// VisitorEntryStatus enumera los estados validos de una entrada de
// visitante.
type VisitorEntryStatus string

const (
	// VisitorEntryStatusActive es el visitante esta dentro (sin exit_time).
	VisitorEntryStatusActive VisitorEntryStatus = "active"
	// VisitorEntryStatusClosed es el visitante salio (exit_time fijado).
	VisitorEntryStatusClosed VisitorEntryStatus = "closed"
	// VisitorEntryStatusRejected es intento bloqueado por blacklist u otra
	// regla; queda registrado por auditoria.
	VisitorEntryStatusRejected VisitorEntryStatus = "rejected"
)

// VisitorEntrySource enumera los origenes validos de una entrada.
type VisitorEntrySource string

const (
	// VisitorEntrySourceQR es el visitante presento un QR de pre-registro
	// firmado y aun valido.
	VisitorEntrySourceQR VisitorEntrySource = "qr"
	// VisitorEntrySourceManual es el guarda registro la entrada a mano (foto
	// del documento obligatoria).
	VisitorEntrySourceManual VisitorEntrySource = "manual"
)

// VisitorEntry representa una entrada (potencialmente cerrada con
// exit_time) de un visitante a una unidad.
type VisitorEntry struct {
	ID                    string
	UnitID                *string
	PreRegistrationID     *string
	VisitorFullName       string
	VisitorDocumentType   *string
	VisitorDocumentNumber string
	PhotoURL              *string
	GuardID               string
	EntryTime             time.Time
	ExitTime              *time.Time
	Source                VisitorEntrySource
	Notes                 *string
	Status                VisitorEntryStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
	CreatedBy             *string
	UpdatedBy             *string
	DeletedBy             *string
	Version               int32
}

// IsOpen indica si la visita sigue activa (dentro del conjunto).
func (v VisitorEntry) IsOpen() bool {
	return v.Status == VisitorEntryStatusActive && v.ExitTime == nil && v.DeletedAt == nil
}
