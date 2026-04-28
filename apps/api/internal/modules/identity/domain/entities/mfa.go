package entities

import "time"

// MFARecoveryCode es un codigo de recuperacion single-use que un usuario
// puede utilizar cuando perdio acceso al segundo factor (TOTP). El
// hash es la unica representacion persistida; el codigo en claro solo
// existe en el momento de la enrolacion.
type MFARecoveryCode struct {
	ID        string
	UserID    string
	CodeHash  string
	UsedAt    *time.Time
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy *string
	UpdatedBy *string
	Version   int
}

// IsUsed indica si el codigo ya fue redimido.
func (c *MFARecoveryCode) IsUsed() bool {
	return c != nil && c.UsedAt != nil
}
