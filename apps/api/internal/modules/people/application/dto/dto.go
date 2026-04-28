// Package dto contiene los Data Transfer Objects del modulo people.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// CreateVehicleRequest es el body de POST /vehicles.
type CreateVehicleRequest struct {
	Plate string  `json:"plate"`
	Type  string  `json:"type"`
	Brand *string `json:"brand,omitempty"`
	Model *string `json:"model,omitempty"`
	Color *string `json:"color,omitempty"`
	Year  *int32  `json:"year,omitempty"`
}

// VehicleResponse es la representacion HTTP de un Vehicle.
type VehicleResponse struct {
	ID        string    `json:"id"`
	Plate     string    `json:"plate"`
	Type      string    `json:"type"`
	Brand     *string   `json:"brand,omitempty"`
	Model     *string   `json:"model,omitempty"`
	Color     *string   `json:"color,omitempty"`
	Year      *int32    `json:"year,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int32     `json:"version"`
}

// ListVehiclesResponse es el sobre simple del listado.
type ListVehiclesResponse struct {
	Items []VehicleResponse `json:"items"`
	Total int               `json:"total"`
}

// AssignVehicleRequest es el body de POST /units/{unitID}/vehicles.
type AssignVehicleRequest struct {
	VehicleID string     `json:"vehicle_id"`
	SinceDate *time.Time `json:"since_date,omitempty"`
}

// AssignmentResponse es la representacion HTTP de una asignacion.
type AssignmentResponse struct {
	ID        string           `json:"id"`
	UnitID    string           `json:"unit_id"`
	VehicleID string           `json:"vehicle_id"`
	SinceDate time.Time        `json:"since_date"`
	UntilDate *time.Time       `json:"until_date,omitempty"`
	Status    string           `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Version   int32            `json:"version"`
	Vehicle   *VehicleResponse `json:"vehicle,omitempty"`
}

// ListAssignmentsResponse es el sobre del listado de asignaciones.
type ListAssignmentsResponse struct {
	Items []AssignmentResponse `json:"items"`
	Total int                  `json:"total"`
}
