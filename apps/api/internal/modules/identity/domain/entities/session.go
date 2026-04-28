package entities

import "time"

// SessionStatus refleja el ciclo de vida de la sesion HTTP. La columna
// revoked_at es la fuente de verdad operacional; este enum es reflejo
// para queries y dashboards.
type SessionStatus string

// Estados permitidos en user_sessions.status.
const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusRevoked SessionStatus = "revoked"
	SessionStatusExpired SessionStatus = "expired"
)

// RevocationReason categoriza por que se revoco una sesion. Es texto
// libre persistido en revocation_reason; aqui declaramos las constantes
// usadas por el modulo identity.
const (
	RevocationReasonRotated         = "rotated"
	RevocationReasonLogout          = "logout"
	RevocationReasonReuseDetected   = "reuse_detected"
	RevocationReasonAdminRevocation = "admin_revocation"
)

// Session representa una sesion HTTP del usuario contra el tenant.
// El refresh token NO se almacena en claro: solo se persiste su
// SHA-256 (TokenHash). El access token JWT se firma con jwtsign y NO
// se persiste; lo unico que la base conoce es la SessionID que viaja en
// la claim `sid`.
type Session struct {
	ID               string
	UserID           string
	TokenHash        string
	ParentSessionID  *string
	IP               *string
	UserAgent        *string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	RevocationReason *string
	Status           SessionStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CreatedBy        *string
	UpdatedBy        *string
	Version          int
}

// IsRevoked indica si la sesion ya esta marcada como revocada.
func (s *Session) IsRevoked() bool {
	return s != nil && s.RevokedAt != nil
}

// IsExpiredAt indica si la sesion expiro respecto a la referencia now.
func (s *Session) IsExpiredAt(now time.Time) bool {
	return s != nil && !s.ExpiresAt.IsZero() && !now.Before(s.ExpiresAt)
}

// RevocationReasonValue devuelve el motivo de revocacion o "" si la
// sesion no ha sido revocada. Util para no manipular el puntero en el
// caller.
func (s *Session) RevocationReasonValue() string {
	if s == nil || s.RevocationReason == nil {
		return ""
	}
	return *s.RevocationReason
}
