package entities

import "time"

// ProxyStatus enumera los estados validos de un poder (proxy).
type ProxyStatus string

const (
	// ProxyStatusPending el poder esta pendiente de validacion.
	ProxyStatusPending ProxyStatus = "pending"
	// ProxyStatusValidated el poder fue validado.
	ProxyStatusValidated ProxyStatus = "validated"
	// ProxyStatusRejected el poder fue rechazado.
	ProxyStatusRejected ProxyStatus = "rejected"
	// ProxyStatusRevoked el poder fue revocado por el otorgante.
	ProxyStatusRevoked ProxyStatus = "revoked"
	// ProxyStatusArchived el poder fue archivado.
	ProxyStatusArchived ProxyStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s ProxyStatus) IsValid() bool {
	switch s {
	case ProxyStatusPending, ProxyStatusValidated,
		ProxyStatusRejected, ProxyStatusRevoked, ProxyStatusArchived:
		return true
	}
	return false
}

// AssemblyProxy representa un poder otorgado por un propietario a un
// apoderado para votar en su nombre en una asamblea determinada.
type AssemblyProxy struct {
	ID            string
	AssemblyID    string
	GrantorUserID string
	ProxyUserID   string
	UnitID        string
	DocumentURL   *string
	DocumentHash  *string
	ValidatedAt   *time.Time
	ValidatedBy   *string
	RevokedAt     *time.Time
	Status        ProxyStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
	CreatedBy     *string
	UpdatedBy     *string
	DeletedBy     *string
	Version       int32
}
