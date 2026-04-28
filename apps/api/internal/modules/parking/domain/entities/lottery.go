package entities

import "time"

// LotteryStatus enumera los estados validos de una ejecucion de sorteo.
type LotteryStatus string

const (
	// LotteryStatusDraft indica que el sorteo esta en borrador.
	LotteryStatusDraft LotteryStatus = "draft"
	// LotteryStatusCompleted indica que el sorteo fue ejecutado
	// exitosamente.
	LotteryStatusCompleted LotteryStatus = "completed"
	// LotteryStatusCancelled indica que el sorteo fue cancelado.
	LotteryStatusCancelled LotteryStatus = "cancelled"
	// LotteryStatusArchived indica que el sorteo fue archivado.
	LotteryStatusArchived LotteryStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s LotteryStatus) IsValid() bool {
	switch s {
	case LotteryStatusDraft, LotteryStatusCompleted,
		LotteryStatusCancelled, LotteryStatusArchived:
		return true
	}
	return false
}

// LotteryResultStatus enumera los estados validos de un resultado de
// sorteo por unidad.
type LotteryResultStatus string

const (
	// LotteryResultStatusAllocated indica que la unidad recibio un espacio.
	LotteryResultStatusAllocated LotteryResultStatus = "allocated"
	// LotteryResultStatusWaitlist indica que la unidad quedo en lista de
	// espera.
	LotteryResultStatusWaitlist LotteryResultStatus = "waitlist"
	// LotteryResultStatusDeclined indica que la unidad declino el espacio
	// asignado.
	LotteryResultStatusDeclined LotteryResultStatus = "declined"
	// LotteryResultStatusArchived indica que el resultado fue archivado.
	LotteryResultStatusArchived LotteryResultStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s LotteryResultStatus) IsValid() bool {
	switch s {
	case LotteryResultStatusAllocated, LotteryResultStatusWaitlist,
		LotteryResultStatusDeclined, LotteryResultStatusArchived:
		return true
	}
	return false
}

// LotteryRun representa una ejecucion de sorteo de parqueaderos con
// seed reproducible (determinista).
type LotteryRun struct {
	ID         string
	Name       string
	SeedHash   string
	Criteria   []byte
	ExecutedAt time.Time
	ExecutedBy string
	Status     LotteryStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
	CreatedBy  *string
	UpdatedBy  *string
	DeletedBy  *string
	Version    int32
}

// LotteryResult representa el resultado de un sorteo para una unidad
// inmobiliaria, incluyendo su posicion y espacio asignado (si aplica).
type LotteryResult struct {
	ID             string
	LotteryRunID   string
	UnitID         string
	ParkingSpaceID *string
	Position       int32
	Status         LotteryResultStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
	CreatedBy      *string
	UpdatedBy      *string
	DeletedBy      *string
	Version        int32
}
