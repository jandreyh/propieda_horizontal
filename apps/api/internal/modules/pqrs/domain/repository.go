// Package domain define los puertos del modulo pqrs.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/pqrs/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrCategoryNotFound se devuelve cuando una categoria por id no existe.
// La capa HTTP lo mapea a 404.
var ErrCategoryNotFound = errors.New("pqrs: category not found")

// ErrCategoryCodeDuplicate se devuelve cuando se intenta crear o
// actualizar una categoria con un codigo que ya existe. Mapea a 409.
var ErrCategoryCodeDuplicate = errors.New("pqrs: category code already exists")

// ErrTicketNotFound se devuelve cuando un ticket por id no existe.
// Mapea a 404.
var ErrTicketNotFound = errors.New("pqrs: ticket not found")

// ErrResponseNotFound se devuelve cuando una respuesta por id no existe.
// Mapea a 404.
var ErrResponseNotFound = errors.New("pqrs: response not found")

// ErrOfficialResponseExists se devuelve cuando ya existe una respuesta
// oficial para el ticket. Mapea a 409.
var ErrOfficialResponseExists = errors.New("pqrs: official response already exists for this ticket")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio. La capa HTTP lo mapea a 409.
var ErrVersionConflict = errors.New("pqrs: version conflict")

// ErrInvalidTransition se devuelve cuando se intenta transicionar el
// status de una entidad a un estado no permitido. Mapea a 409.
var ErrInvalidTransition = errors.New("pqrs: invalid status transition")

// ---------------------------------------------------------------------------
// CategoryRepository
// ---------------------------------------------------------------------------

// CreateCategoryInput agrupa los datos para persistir una categoria nueva.
type CreateCategoryInput struct {
	Code                  string
	Name                  string
	DefaultAssigneeRoleID *string
	ActorID               string
}

// UpdateCategoryInput agrupa los datos para actualizar una categoria.
type UpdateCategoryInput struct {
	ID                    string
	Code                  string
	Name                  string
	DefaultAssigneeRoleID *string
	Status                entities.CategoryStatus
	ExpectedVersion       int32
	ActorID               string
}

// CategoryRepository es el puerto que persiste categorias de PQRS.
type CategoryRepository interface {
	// Create inserta una categoria nueva en estado 'active'.
	Create(ctx context.Context, in CreateCategoryInput) (entities.Category, error)
	// GetByID devuelve una categoria por id. Si no existe, devuelve
	// ErrCategoryNotFound.
	GetByID(ctx context.Context, id string) (entities.Category, error)
	// List devuelve las categorias activas (no soft-deleted) ordenadas
	// por code.
	List(ctx context.Context) ([]entities.Category, error)
	// Update actualiza una categoria existente con concurrencia
	// optimista. Si la version no coincide, devuelve
	// ErrVersionConflict.
	Update(ctx context.Context, in UpdateCategoryInput) (entities.Category, error)
}

// ---------------------------------------------------------------------------
// TicketRepository
// ---------------------------------------------------------------------------

// CreateTicketInput agrupa los datos para persistir un ticket nuevo.
type CreateTicketInput struct {
	TicketYear      int32
	SerialNumber    int32
	PQRType         entities.PQRType
	CategoryID      *string
	Subject         string
	Body            string
	RequesterUserID string
	IsAnonymous     bool
	SLADueAt        *time.Time
	ActorID         string
}

// TicketListFilter permite filtrar tickets.
type TicketListFilter struct {
	Status           *entities.TicketStatus
	PQRType          *entities.PQRType
	RequesterUserID  *string
	AssignedToUserID *string
}

// TicketRepository es el puerto que persiste tickets PQRS.
type TicketRepository interface {
	// Create inserta un ticket nuevo en estado 'radicado'.
	Create(ctx context.Context, in CreateTicketInput) (entities.Ticket, error)
	// GetByID devuelve un ticket por id. Si no existe, devuelve
	// ErrTicketNotFound.
	GetByID(ctx context.Context, id string) (entities.Ticket, error)
	// List devuelve tickets filtrados y ordenados por created_at desc.
	List(ctx context.Context, filter TicketListFilter) ([]entities.Ticket, error)
	// UpdateStatus actualiza el status con concurrencia optimista.
	// Si la version no coincide, devuelve ErrVersionConflict.
	UpdateStatus(ctx context.Context, id string, newStatus entities.TicketStatus, expectedVersion int32, actorID string) (entities.Ticket, error)
	// Assign establece assigned_to_user_id y assigned_at.
	Assign(ctx context.Context, id string, assigneeUserID string, expectedVersion int32, actorID string) (entities.Ticket, error)
	// SetResponded marca el ticket como respondido (responded_at).
	SetResponded(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Ticket, error)
	// Close cierra el ticket con rating y feedback opcionales.
	Close(ctx context.Context, id string, rating *int32, feedback *string, expectedVersion int32, actorID string) (entities.Ticket, error)
	// Escalate escala el ticket.
	Escalate(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Ticket, error)
	// Cancel cancela el ticket.
	Cancel(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Ticket, error)
	// NextSerialNumber obtiene el siguiente serial para el anio dado
	// usando advisory lock.
	NextSerialNumber(ctx context.Context, year int32) (int32, error)
}

// ---------------------------------------------------------------------------
// ResponseRepository
// ---------------------------------------------------------------------------

// CreateResponseInput agrupa los datos para persistir una respuesta.
type CreateResponseInput struct {
	TicketID          string
	ResponseType      entities.ResponseType
	Body              string
	RespondedByUserID string
}

// ResponseRepository es el puerto que persiste respuestas de tickets.
type ResponseRepository interface {
	// Create inserta una respuesta. Si ya existe una respuesta
	// oficial, devuelve ErrOfficialResponseExists.
	Create(ctx context.Context, in CreateResponseInput) (entities.Response, error)
	// ListByTicketID devuelve las respuestas de un ticket ordenadas
	// por responded_at desc.
	ListByTicketID(ctx context.Context, ticketID string) ([]entities.Response, error)
	// HasOfficialResponse indica si el ticket tiene respuesta oficial.
	HasOfficialResponse(ctx context.Context, ticketID string) (bool, error)
}

// ---------------------------------------------------------------------------
// StatusHistoryRepository
// ---------------------------------------------------------------------------

// RecordHistoryInput agrupa los datos para registrar una transicion de
// estado.
type RecordHistoryInput struct {
	TicketID             string
	FromStatus           *string
	ToStatus             string
	TransitionedByUserID string
	Notes                *string
}

// StatusHistoryRepository es el puerto que persiste el historial
// append-only de transiciones de estado.
type StatusHistoryRepository interface {
	// Record inserta un registro de historial.
	Record(ctx context.Context, in RecordHistoryInput) (entities.StatusHistory, error)
	// ListByTicketID devuelve el historial de un ticket ordenado por
	// transitioned_at desc.
	ListByTicketID(ctx context.Context, ticketID string) ([]entities.StatusHistory, error)
}

// ---------------------------------------------------------------------------
// OutboxRepository
// ---------------------------------------------------------------------------

// EnqueueOutboxInput agrupa los datos para encolar un evento.
type EnqueueOutboxInput struct {
	TicketID       string
	EventType      entities.OutboxEventType
	Payload        []byte
	IdempotencyKey *string
}

// OutboxRepository es el puerto que persiste eventos del outbox.
type OutboxRepository interface {
	// Enqueue inserta un evento pendiente de despacho.
	Enqueue(ctx context.Context, in EnqueueOutboxInput) (entities.OutboxEvent, error)
	// LockPending bloquea un lote de eventos pendientes para procesar
	// (FOR UPDATE SKIP LOCKED). DEBE invocarse dentro de una
	// transaccion del caller.
	LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error)
	// MarkDelivered marca un evento como entregado.
	MarkDelivered(ctx context.Context, id string) error
	// MarkFailed registra un fallo: incrementa attempts, fija
	// last_error, reagenda next_attempt_at.
	MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error
}
