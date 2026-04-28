package entities

import "time"

// DeliveryMethod enumera los origenes validos de una entrega de
// paquete.
type DeliveryMethod string

const (
	// DeliveryMethodQR es el residente presento un QR firmado en porteria.
	DeliveryMethodQR DeliveryMethod = "qr"
	// DeliveryMethodManual es el guarda registro la entrega manualmente
	// (debe acompanar firma o foto).
	DeliveryMethodManual DeliveryMethod = "manual"
)

// IsValid indica si el metodo es uno de los enumerados.
func (m DeliveryMethod) IsValid() bool {
	return m == DeliveryMethodQR || m == DeliveryMethodManual
}

// DeliveryEventStatus enumera los estados de un evento de entrega.
type DeliveryEventStatus string

const (
	// DeliveryEventStatusCompleted es el evento se completo correctamente.
	DeliveryEventStatusCompleted DeliveryEventStatus = "completed"
	// DeliveryEventStatusVoided es el evento fue anulado por error
	// administrativo (auditoria).
	DeliveryEventStatusVoided DeliveryEventStatus = "voided"
)

// DeliveryEvent representa el registro auditable de una entrega de
// paquete (sea por QR o manual).
type DeliveryEvent struct {
	ID                  string
	PackageID           string
	DeliveredToUserID   *string
	RecipientNameManual *string
	DeliveryMethod      DeliveryMethod
	SignatureURL        *string
	PhotoEvidenceURL    *string
	DeliveredByUserID   string
	DeliveredAt         time.Time
	Notes               *string
	Status              DeliveryEventStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
	CreatedBy           *string
	UpdatedBy           *string
	DeletedBy           *string
	Version             int32
}
