package entities

import "time"

// CostCenterStatus enumera los estados validos de un centro de costo.
type CostCenterStatus string

// Possible values for CostCenterStatus.
const (
	CostCenterStatusActive   CostCenterStatus = "active"
	CostCenterStatusInactive CostCenterStatus = "inactive"
	CostCenterStatusArchived CostCenterStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s CostCenterStatus) IsValid() bool {
	switch s {
	case CostCenterStatusActive, CostCenterStatusInactive, CostCenterStatusArchived:
		return true
	}
	return false
}

// CostCenter representa un centro de costo del tenant.
type CostCenter struct {
	ID        string
	Code      string
	Name      string
	Status    CostCenterStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	CreatedBy *string
	UpdatedBy *string
	DeletedBy *string
	Version   int32
}

// IsActive indica si el centro de costo esta activo y no soft-deleted.
func (c CostCenter) IsActive() bool {
	return c.Status == CostCenterStatusActive && c.DeletedAt == nil
}
