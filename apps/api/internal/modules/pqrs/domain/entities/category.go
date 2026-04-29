package entities

import "time"

// CategoryStatus enumera los estados validos de una categoria PQRS.
type CategoryStatus string

const (
	// CategoryStatusActive indica que la categoria esta activa.
	CategoryStatusActive CategoryStatus = "active"
	// CategoryStatusArchived indica que la categoria fue archivada.
	CategoryStatusArchived CategoryStatus = "archived"
)

// IsValid indica si el status es uno de los enumerados.
func (s CategoryStatus) IsValid() bool {
	switch s {
	case CategoryStatusActive, CategoryStatusArchived:
		return true
	}
	return false
}

// Category representa una categoria configurable de PQRS.
type Category struct {
	ID                    string
	Code                  string
	Name                  string
	DefaultAssigneeRoleID *string
	Status                CategoryStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
	CreatedBy             *string
	UpdatedBy             *string
	DeletedBy             *string
	Version               int32
}

// IsActive indica si la categoria esta activa y no soft-deleted.
func (c Category) IsActive() bool {
	return c.Status == CategoryStatusActive && c.DeletedAt == nil
}
