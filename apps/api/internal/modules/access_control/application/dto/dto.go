// Package dto contiene los Data Transfer Objects del modulo
// access_control. Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ----------------------------------------------------------------------------
// Pre-registros
// ----------------------------------------------------------------------------

// CreatePreRegistrationRequest es el body de POST /visitor-preregistrations.
//
// El cliente NO envia QRCodeHash; lo genera el servidor y devuelve el
// codigo plano UNA SOLA VEZ en la respuesta.
type CreatePreRegistrationRequest struct {
	UnitID                string     `json:"unit_id"`
	VisitorFullName       string     `json:"visitor_full_name"`
	VisitorDocumentType   *string    `json:"visitor_document_type,omitempty"`
	VisitorDocumentNumber *string    `json:"visitor_document_number,omitempty"`
	ExpectedAt            *time.Time `json:"expected_at,omitempty"`
	ExpiresAt             time.Time  `json:"expires_at"`
	MaxUses               *int32     `json:"max_uses,omitempty"`
}

// CreatePreRegistrationResponse incluye el QR plano (UNICA VEZ).
type CreatePreRegistrationResponse struct {
	ID        string    `json:"id"`
	QRCode    string    `json:"qr_code"`
	ExpiresAt time.Time `json:"expires_at"`
	MaxUses   int32     `json:"max_uses"`
}

// ----------------------------------------------------------------------------
// Checkin / checkout
// ----------------------------------------------------------------------------

// CheckinByQRRequest es el body de POST /visits/checkin-by-qr.
type CheckinByQRRequest struct {
	QRCode   string  `json:"qr_code"`
	GuardID  string  `json:"guard_id"`
	PhotoURL *string `json:"photo_url,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

// CheckinManualRequest es el body de POST /visits/checkin-manual.
//
// PhotoURL es OBLIGATORIO en el flujo manual.
type CheckinManualRequest struct {
	UnitID                *string `json:"unit_id,omitempty"`
	VisitorFullName       string  `json:"visitor_full_name"`
	VisitorDocumentType   *string `json:"visitor_document_type,omitempty"`
	VisitorDocumentNumber string  `json:"visitor_document_number"`
	PhotoURL              string  `json:"photo_url"`
	GuardID               string  `json:"guard_id"`
	Notes                 *string `json:"notes,omitempty"`
}

// CheckoutRequest es el body de POST /visits/{id}/checkout.
//
// EntryID viene en la URL; el body es opcional (kept por simetria).
type CheckoutRequest struct {
	EntryID string `json:"entry_id,omitempty"`
}

// VisitorEntryResponse es la representacion HTTP de una VisitorEntry.
type VisitorEntryResponse struct {
	ID                    string     `json:"id"`
	UnitID                *string    `json:"unit_id,omitempty"`
	PreRegistrationID     *string    `json:"pre_registration_id,omitempty"`
	VisitorFullName       string     `json:"visitor_full_name"`
	VisitorDocumentType   *string    `json:"visitor_document_type,omitempty"`
	VisitorDocumentNumber string     `json:"visitor_document_number"`
	PhotoURL              *string    `json:"photo_url,omitempty"`
	GuardID               string     `json:"guard_id"`
	EntryTime             time.Time  `json:"entry_time"`
	ExitTime              *time.Time `json:"exit_time,omitempty"`
	Source                string     `json:"source"`
	Notes                 *string    `json:"notes,omitempty"`
	Status                string     `json:"status"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	Version               int32      `json:"version"`
}

// ListVisitorEntriesResponse es el sobre del listado de visitas.
type ListVisitorEntriesResponse struct {
	Items []VisitorEntryResponse `json:"items"`
	Total int                    `json:"total"`
}

// ----------------------------------------------------------------------------
// Blacklist
// ----------------------------------------------------------------------------

// CreateBlacklistRequest es el body de POST /blacklist.
type CreateBlacklistRequest struct {
	DocumentType     string     `json:"document_type"`
	DocumentNumber   string     `json:"document_number"`
	FullName         *string    `json:"full_name,omitempty"`
	Reason           string     `json:"reason"`
	ReportedByUnitID *string    `json:"reported_by_unit_id,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

// BlacklistResponse es la representacion HTTP de una BlacklistEntry.
type BlacklistResponse struct {
	ID               string     `json:"id"`
	DocumentType     string     `json:"document_type"`
	DocumentNumber   string     `json:"document_number"`
	FullName         *string    `json:"full_name,omitempty"`
	Reason           string     `json:"reason"`
	ReportedByUnitID *string    `json:"reported_by_unit_id,omitempty"`
	ReportedByUserID *string    `json:"reported_by_user_id,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Version          int32      `json:"version"`
}

// ListBlacklistResponse es el sobre del listado de blacklist.
type ListBlacklistResponse struct {
	Items []BlacklistResponse `json:"items"`
	Total int                 `json:"total"`
}
