// Package domain define los puertos del modulo incidents.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/incidents/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrIncidentNotFound se devuelve cuando un incidente por id no existe.
// La capa HTTP lo mapea a 404.
var ErrIncidentNotFound = errors.New("incidents: incident not found")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio. La capa HTTP lo mapea a 409.
var ErrVersionConflict = errors.New("incidents: version conflict")

// ErrInvalidTransition se devuelve cuando se intenta transicionar el
// status de un incidente a un estado no permitido. Mapea a 409.
var ErrInvalidTransition = errors.New("incidents: invalid status transition")

// ErrAttachmentNotFound se devuelve cuando un adjunto por id no existe.
// Mapea a 404.
var ErrAttachmentNotFound = errors.New("incidents: attachment not found")

// ErrAttachmentLimitReached se devuelve cuando se intenta agregar un
// adjunto y el incidente ya alcanzo el maximo. Mapea a 409.
var ErrAttachmentLimitReached = errors.New("incidents: attachment limit reached")

// ErrAssignmentAlreadyActive se devuelve cuando ya existe una asignacion
// activa para un incidente. Mapea a 409.
var ErrAssignmentAlreadyActive = errors.New("incidents: incident already has an active assignment")

// ---------------------------------------------------------------------------
// IncidentRepository
// ---------------------------------------------------------------------------

// CreateIncidentInput agrupa los datos para persistir un incidente nuevo.
type CreateIncidentInput struct {
	IncidentType     entities.IncidentType
	Severity         entities.Severity
	Title            string
	Description      string
	ReportedByUserID string
	ReportedAt       time.Time
	StructureID      *string
	LocationDetail   *string
	SLAAssignDueAt   *time.Time
	SLAResolveDueAt  *time.Time
	ActorID          string
}

// UpdateIncidentStatusInput agrupa los datos para una transicion de
// estado de incidente.
type UpdateIncidentStatusInput struct {
	ID               string
	NewStatus        entities.IncidentStatus
	ExpectedVersion  int32
	ActorID          string
	ResolutionNotes  *string
	AssignedToUserID *string
	AssignedAt       *time.Time
	StartedAt        *time.Time
	ResolvedAt       *time.Time
	ClosedAt         *time.Time
	CancelledAt      *time.Time
	Escalated        *bool
}

// IncidentListFilter define los filtros para listar incidentes.
type IncidentListFilter struct {
	Status           *entities.IncidentStatus
	Severity         *entities.Severity
	ReportedByUserID *string
}

// IncidentRepository es el puerto que persiste incidentes.
type IncidentRepository interface {
	// Create inserta un incidente nuevo en estado 'reported'.
	Create(ctx context.Context, in CreateIncidentInput) (entities.Incident, error)
	// GetByID devuelve un incidente por id. Si no existe, devuelve
	// ErrIncidentNotFound.
	GetByID(ctx context.Context, id string) (entities.Incident, error)
	// List devuelve los incidentes activos (no soft-deleted) con filtros
	// opcionales.
	List(ctx context.Context, filter IncidentListFilter) ([]entities.Incident, error)
	// UpdateStatus actualiza el status y campos temporales con
	// concurrencia optimista. Si la version no coincide, devuelve
	// ErrVersionConflict.
	UpdateStatus(ctx context.Context, in UpdateIncidentStatusInput) (entities.Incident, error)
}

// ---------------------------------------------------------------------------
// AttachmentRepository
// ---------------------------------------------------------------------------

// CreateAttachmentInput agrupa los datos para persistir un adjunto.
type CreateAttachmentInput struct {
	IncidentID string
	URL        string
	MimeType   string
	SizeBytes  int64
	UploadedBy string
}

// AttachmentRepository es el puerto que persiste adjuntos de incidentes.
type AttachmentRepository interface {
	// Create inserta un adjunto nuevo.
	Create(ctx context.Context, in CreateAttachmentInput) (entities.Attachment, error)
	// ListByIncidentID devuelve los adjuntos activos de un incidente.
	ListByIncidentID(ctx context.Context, incidentID string) ([]entities.Attachment, error)
	// CountByIncidentID devuelve la cantidad de adjuntos activos de un
	// incidente.
	CountByIncidentID(ctx context.Context, incidentID string) (int, error)
}

// ---------------------------------------------------------------------------
// StatusHistoryRepository
// ---------------------------------------------------------------------------

// RecordStatusHistoryInput agrupa los datos para registrar una transicion
// de estado.
type RecordStatusHistoryInput struct {
	IncidentID           string
	FromStatus           *string
	ToStatus             string
	TransitionedByUserID string
	Notes                *string
}

// StatusHistoryRepository es el puerto que persiste el historial de
// transiciones de estado.
type StatusHistoryRepository interface {
	// Record inserta un registro de historial.
	Record(ctx context.Context, in RecordStatusHistoryInput) (entities.StatusHistory, error)
	// ListByIncidentID devuelve el historial de un incidente ordenado
	// por transitioned_at descendente.
	ListByIncidentID(ctx context.Context, incidentID string) ([]entities.StatusHistory, error)
}

// ---------------------------------------------------------------------------
// IncidentAssignmentRepository
// ---------------------------------------------------------------------------

// CreateAssignmentInput agrupa los datos para persistir una asignacion.
type CreateAssignmentInput struct {
	IncidentID       string
	AssignedToUserID string
	AssignedByUserID string
}

// IncidentAssignmentRepository es el puerto que persiste asignaciones de
// incidentes.
type IncidentAssignmentRepository interface {
	// Create inserta una asignacion nueva en estado 'active'. Si el
	// incidente ya tiene una asignacion activa, devuelve
	// ErrAssignmentAlreadyActive.
	Create(ctx context.Context, in CreateAssignmentInput) (entities.IncidentAssignment, error)
	// UnassignActive desactiva la asignacion activa de un incidente.
	UnassignActive(ctx context.Context, incidentID string, actorID string) error
	// GetActiveByIncidentID devuelve la asignacion activa de un
	// incidente. Si no hay asignacion activa, devuelve
	// ErrAttachmentNotFound (se reutiliza 404).
	GetActiveByIncidentID(ctx context.Context, incidentID string) (entities.IncidentAssignment, error)
}

// ---------------------------------------------------------------------------
// OutboxRepository
// ---------------------------------------------------------------------------

// EnqueueOutboxInput agrupa los datos para encolar un evento.
type EnqueueOutboxInput struct {
	IncidentID string
	EventType  entities.OutboxEventType
	Payload    []byte
}

// OutboxRepository es el puerto que persiste eventos del outbox.
type OutboxRepository interface {
	// Enqueue inserta un evento pendiente de despacho.
	Enqueue(ctx context.Context, in EnqueueOutboxInput) (entities.OutboxEvent, error)
	// LockPending bloquea un lote de eventos pendientes para procesar
	// (FOR UPDATE SKIP LOCKED). DEBE invocarse dentro de una transaccion
	// del caller.
	LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error)
	// MarkDelivered marca un evento como entregado.
	MarkDelivered(ctx context.Context, id string) error
	// MarkFailed registra un fallo: incrementa attempts, fija last_error,
	// reagenda next_attempt_at con el delta provisto por el caller.
	MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error
}
