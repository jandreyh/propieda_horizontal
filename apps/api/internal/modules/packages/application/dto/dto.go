// Package dto contiene los Data Transfer Objects del modulo packages.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ----------------------------------------------------------------------------
// Categorias
// ----------------------------------------------------------------------------

// CategoryResponse es la representacion HTTP de una PackageCategory.
type CategoryResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	RequiresEvidence bool   `json:"requires_evidence"`
}

// ListCategoriesResponse es el sobre del listado de categorias.
type ListCategoriesResponse struct {
	Items []CategoryResponse `json:"items"`
	Total int                `json:"total"`
}

// ----------------------------------------------------------------------------
// Packages
// ----------------------------------------------------------------------------

// CreatePackageRequest es el body de POST /packages.
//
// Una de CategoryID o CategoryName puede venir; si ambas vienen, prevalece
// CategoryID.
type CreatePackageRequest struct {
	UnitID              string  `json:"unit_id"`
	RecipientName       string  `json:"recipient_name"`
	CategoryID          *string `json:"category_id,omitempty"`
	CategoryName        *string `json:"category_name,omitempty"`
	Carrier             *string `json:"carrier,omitempty"`
	TrackingNumber      *string `json:"tracking_number,omitempty"`
	ReceivedEvidenceURL *string `json:"received_evidence_url,omitempty"`
	ReceivedByUserID    string  `json:"received_by_user_id"`
}

// DeliverByQRRequest es el body de POST /packages/{id}/deliver-by-qr.
type DeliverByQRRequest struct {
	DeliveredToUserID string  `json:"delivered_to_user_id"`
	GuardID           string  `json:"guard_id"`
	IdempotencyKey    *string `json:"idempotency_key,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

// DeliverManualRequest es el body de POST /packages/{id}/deliver-manual.
//
// Una de SignatureURL o PhotoEvidenceURL es OBLIGATORIA.
type DeliverManualRequest struct {
	RecipientNameManual *string `json:"recipient_name_manual,omitempty"`
	SignatureURL        *string `json:"signature_url,omitempty"`
	PhotoEvidenceURL    *string `json:"photo_evidence_url,omitempty"`
	GuardID             string  `json:"guard_id"`
	IdempotencyKey      *string `json:"idempotency_key,omitempty"`
	Notes               *string `json:"notes,omitempty"`
}

// ReturnPackageRequest es el body de POST /packages/{id}/return.
type ReturnPackageRequest struct {
	GuardID string  `json:"guard_id"`
	Notes   *string `json:"notes,omitempty"`
}

// PackageResponse es la representacion HTTP de un paquete.
type PackageResponse struct {
	ID                  string     `json:"id"`
	UnitID              string     `json:"unit_id"`
	RecipientName       string     `json:"recipient_name"`
	CategoryID          *string    `json:"category_id,omitempty"`
	ReceivedEvidenceURL *string    `json:"received_evidence_url,omitempty"`
	Carrier             *string    `json:"carrier,omitempty"`
	TrackingNumber      *string    `json:"tracking_number,omitempty"`
	ReceivedByUserID    string     `json:"received_by_user_id"`
	ReceivedAt          time.Time  `json:"received_at"`
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`
	ReturnedAt          *time.Time `json:"returned_at,omitempty"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Version             int32      `json:"version"`
}

// ListPackagesResponse es el sobre del listado de paquetes.
type ListPackagesResponse struct {
	Items []PackageResponse `json:"items"`
	Total int               `json:"total"`
}

// DeliveryEventResponse es la representacion HTTP del evento de entrega
// (incluido en la respuesta de un deliver exitoso).
type DeliveryEventResponse struct {
	ID                  string    `json:"id"`
	PackageID           string    `json:"package_id"`
	DeliveredToUserID   *string   `json:"delivered_to_user_id,omitempty"`
	RecipientNameManual *string   `json:"recipient_name_manual,omitempty"`
	DeliveryMethod      string    `json:"delivery_method"`
	SignatureURL        *string   `json:"signature_url,omitempty"`
	PhotoEvidenceURL    *string   `json:"photo_evidence_url,omitempty"`
	DeliveredByUserID   string    `json:"delivered_by_user_id"`
	DeliveredAt         time.Time `json:"delivered_at"`
	Notes               *string   `json:"notes,omitempty"`
	Status              string    `json:"status"`
}

// DeliverResponse es la respuesta de un deliver-by-qr o deliver-manual:
// incluye el paquete con su nuevo status y el evento de entrega.
type DeliverResponse struct {
	Package PackageResponse       `json:"package"`
	Event   DeliveryEventResponse `json:"event"`
}
