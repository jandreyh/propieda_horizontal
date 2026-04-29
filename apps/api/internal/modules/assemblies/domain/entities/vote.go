package entities

import "time"

// VoteStatus enumera los estados validos de un voto.
type VoteStatus string

const (
	// VoteStatusCast el voto fue emitido y esta vigente.
	VoteStatusCast VoteStatus = "cast"
	// VoteStatusChanged el voto fue reemplazado por uno nuevo.
	VoteStatusChanged VoteStatus = "changed"
	// VoteStatusVoided el voto fue anulado.
	VoteStatusVoided VoteStatus = "voided"
	// VoteStatusArchived el voto fue archivado.
	VoteStatusArchived VoteStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s VoteStatus) IsValid() bool {
	switch s {
	case VoteStatusCast, VoteStatusChanged,
		VoteStatusVoided, VoteStatusArchived:
		return true
	}
	return false
}

// Vote representa un voto individual emitido por un propietario o
// apoderado en una mocion. Mantiene un hash chain para auditoria.
type Vote struct {
	ID              string
	MotionID        string
	VoterUserID     string
	UnitID          string
	CoefficientUsed float64
	Option          string
	CastAt          time.Time
	PrevVoteHash    *string
	VoteHash        string
	Nonce           string
	IsProxyVote     bool
	Status          VoteStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	CreatedBy       *string
	UpdatedBy       *string
	DeletedBy       *string
	Version         int32
}

// VoteEvidence representa la evidencia digital append-only de un voto,
// incluyendo la cadena de hashes para verificacion.
type VoteEvidence struct {
	ID           string
	VoteID       string
	MotionID     string
	PrevVoteHash *string
	VoteHash     string
	PayloadJSON  []byte
	ClientIP     *string
	UserAgent    *string
	NTPOffsetMS  *int32
	SealedAt     time.Time
	CreatedAt    time.Time
}
