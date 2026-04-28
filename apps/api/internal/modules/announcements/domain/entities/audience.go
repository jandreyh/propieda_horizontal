package entities

import "time"

// TargetType enumera los tipos validos de una audiencia.
type TargetType string

const (
	// TargetGlobal es el anuncio es visible para todos los usuarios.
	TargetGlobal TargetType = "global"
	// TargetStructure es el anuncio se restringe a una estructura
	// residencial (torre, bloque, etapa).
	TargetStructure TargetType = "structure"
	// TargetRole es el anuncio se restringe a usuarios con un rol dado.
	TargetRole TargetType = "role"
	// TargetUnit es el anuncio se restringe a una unidad concreta.
	TargetUnit TargetType = "unit"
)

// IsValid indica si t es un TargetType admitido.
func (t TargetType) IsValid() bool {
	switch t {
	case TargetGlobal, TargetStructure, TargetRole, TargetUnit:
		return true
	}
	return false
}

// Audience representa una audiencia destinataria de un anuncio. Si
// TargetType=='global', TargetID debe ser nil; en cualquier otro caso,
// TargetID es obligatorio.
type Audience struct {
	ID             string
	AnnouncementID string
	TargetType     TargetType
	TargetID       *string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
	CreatedBy      *string
	UpdatedBy      *string
	DeletedBy      *string
}
