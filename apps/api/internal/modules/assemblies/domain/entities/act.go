package entities

import "time"

// ActStatus enumera los estados validos de un acta.
type ActStatus string

const (
	// ActStatusDraft el acta esta en borrador (editable).
	ActStatusDraft ActStatus = "draft"
	// ActStatusSigned el acta fue firmada (inmutable).
	ActStatusSigned ActStatus = "signed"
	// ActStatusArchived el acta fue archivada.
	ActStatusArchived ActStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s ActStatus) IsValid() bool {
	switch s {
	case ActStatusDraft, ActStatusSigned, ActStatusArchived:
		return true
	}
	return false
}

// Act representa el acta de una asamblea. Inmutable despues de firmada.
type Act struct {
	ID           string
	AssemblyID   string
	BodyMD       string
	PDFURL       *string
	PDFHash      *string
	SealedAt     *time.Time
	ArchiveUntil *time.Time
	Status       ActStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	CreatedBy    *string
	UpdatedBy    *string
	DeletedBy    *string
	Version      int32
}

// IsSigned indica si el acta ya fue firmada y es inmutable.
func (a Act) IsSigned() bool {
	return a.Status == ActStatusSigned || a.Status == ActStatusArchived
}

// SignatureRole enumera los roles de firma del acta.
type SignatureRole string

const (
	// SignatureRolePresident presidente de la asamblea.
	SignatureRolePresident SignatureRole = "president"
	// SignatureRoleSecretary secretario de la asamblea.
	SignatureRoleSecretary SignatureRole = "secretary"
	// SignatureRoleWitness testigo.
	SignatureRoleWitness SignatureRole = "witness"
	// SignatureRoleAuditor auditor/revisor fiscal.
	SignatureRoleAuditor SignatureRole = "auditor"
)

// IsValid indica si el rol de firma es valido.
func (r SignatureRole) IsValid() bool {
	switch r {
	case SignatureRolePresident, SignatureRoleSecretary,
		SignatureRoleWitness, SignatureRoleAuditor:
		return true
	}
	return false
}

// SignatureMethod enumera los metodos de firma validos.
type SignatureMethod string

const (
	// SignatureMethodSimpleOTP firma por OTP (low security).
	SignatureMethodSimpleOTP SignatureMethod = "simple_otp"
	// SignatureMethodSimpleTraceable firma simple trazable.
	SignatureMethodSimpleTraceable SignatureMethod = "simple_traceable"
	// SignatureMethodPKICertified firma con certificado PKI.
	SignatureMethodPKICertified SignatureMethod = "pki_certified"
)

// IsValid indica si el metodo de firma es valido.
func (m SignatureMethod) IsValid() bool {
	switch m {
	case SignatureMethodSimpleOTP, SignatureMethodSimpleTraceable,
		SignatureMethodPKICertified:
		return true
	}
	return false
}

// SignatureStatus enumera los estados de una firma.
type SignatureStatus string

const (
	// SignatureStatusValid la firma es valida.
	SignatureStatusValid SignatureStatus = "valid"
	// SignatureStatusRevoked la firma fue revocada.
	SignatureStatusRevoked SignatureStatus = "revoked"
	// SignatureStatusArchived la firma fue archivada.
	SignatureStatusArchived SignatureStatus = "archived"
)

// IsValid indica si el status de firma es valido.
func (s SignatureStatus) IsValid() bool {
	switch s {
	case SignatureStatusValid, SignatureStatusRevoked,
		SignatureStatusArchived:
		return true
	}
	return false
}

// ActSignature representa una firma individual en un acta.
type ActSignature struct {
	ID              string
	ActID           string
	SignerUserID    string
	Role            SignatureRole
	SignedAt        time.Time
	SignatureMethod SignatureMethod
	EvidenceHash    string
	ClientIP        *string
	UserAgent       *string
	Status          SignatureStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	CreatedBy       *string
	UpdatedBy       *string
	DeletedBy       *string
	Version         int32
}
