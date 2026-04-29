// Package domain define los puertos del modulo penalties.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/penalties/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrCatalogNotFound se devuelve cuando una entrada del catalogo no existe.
var ErrCatalogNotFound = errors.New("penalties: catalog entry not found")

// ErrCatalogCodeDuplicate se devuelve cuando se intenta crear o actualizar
// una entrada con un codigo que ya existe (UNIQUE parcial). Mapea a 409.
var ErrCatalogCodeDuplicate = errors.New("penalties: catalog code already exists")

// ErrPenaltyNotFound se devuelve cuando una sancion por id no existe.
var ErrPenaltyNotFound = errors.New("penalties: penalty not found")

// ErrPenaltyIdempotencyConflict se devuelve cuando ya existe una sancion
// con la misma idempotency_key.
var ErrPenaltyIdempotencyConflict = errors.New("penalties: penalty idempotency conflict")

// ErrAppealNotFound se devuelve cuando una apelacion por id no existe.
var ErrAppealNotFound = errors.New("penalties: appeal not found")

// ErrAppealAlreadyActive se devuelve cuando se intenta crear una
// apelacion activa para un penalty que ya tiene una.
var ErrAppealAlreadyActive = errors.New("penalties: appeal already active for this penalty")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio. Mapea a 409.
var ErrVersionConflict = errors.New("penalties: version conflict")

// ErrInvalidTransition se devuelve cuando se intenta transicionar el
// status de una entidad a un estado no permitido. Mapea a 409.
var ErrInvalidTransition = errors.New("penalties: invalid status transition")

// ---------------------------------------------------------------------------
// CatalogRepository
// ---------------------------------------------------------------------------

// CreateCatalogInput agrupa los datos para persistir una entrada de catalogo.
type CreateCatalogInput struct {
	Code                     string
	Name                     string
	Description              *string
	DefaultSanctionType      entities.SanctionType
	BaseAmount               float64
	RecurrenceMultiplier     float64
	RecurrenceCAPMultiplier  float64
	RequiresCouncilThreshold *float64
	ActorID                  string
}

// UpdateCatalogInput agrupa los datos para actualizar una entrada de catalogo.
type UpdateCatalogInput struct {
	ID                       string
	Code                     string
	Name                     string
	Description              *string
	DefaultSanctionType      entities.SanctionType
	BaseAmount               float64
	RecurrenceMultiplier     float64
	RecurrenceCAPMultiplier  float64
	RequiresCouncilThreshold *float64
	Status                   entities.CatalogStatus
	ExpectedVersion          int32
	ActorID                  string
}

// CatalogRepository es el puerto que persiste entradas del catalogo de
// sanciones.
type CatalogRepository interface {
	// Create inserta una entrada nueva en estado 'active'.
	Create(ctx context.Context, in CreateCatalogInput) (entities.PenaltyCatalog, error)
	// GetByID devuelve una entrada por id. Si no existe, devuelve
	// ErrCatalogNotFound.
	GetByID(ctx context.Context, id string) (entities.PenaltyCatalog, error)
	// List devuelve las entradas activas (no soft-deleted) ordenadas por
	// code.
	List(ctx context.Context) ([]entities.PenaltyCatalog, error)
	// Update actualiza una entrada con concurrencia optimista. Si la
	// version no coincide, devuelve ErrVersionConflict.
	Update(ctx context.Context, in UpdateCatalogInput) (entities.PenaltyCatalog, error)
}

// ---------------------------------------------------------------------------
// PenaltyRepository
// ---------------------------------------------------------------------------

// CreatePenaltyInput agrupa los datos para persistir una sancion.
type CreatePenaltyInput struct {
	CatalogID               string
	DebtorUserID            string
	UnitID                  *string
	SourceIncidentID        *string
	SanctionType            entities.SanctionType
	Amount                  float64
	Reason                  string
	ImposedByUserID         string
	RequiresCouncilApproval bool
	IdempotencyKey          *string
	ActorID                 string
}

// PenaltyRepository es el puerto que persiste sanciones.
type PenaltyRepository interface {
	// Create inserta una sancion nueva en estado 'drafted'.
	Create(ctx context.Context, in CreatePenaltyInput) (entities.Penalty, error)
	// GetByID devuelve una sancion por id. Si no existe, devuelve
	// ErrPenaltyNotFound.
	GetByID(ctx context.Context, id string) (entities.Penalty, error)
	// List devuelve las sanciones (no soft-deleted) ordenadas por
	// created_at desc.
	List(ctx context.Context) ([]entities.Penalty, error)
	// UpdateStatus actualiza el status de una sancion con concurrencia
	// optimista. Si la version no coincide, devuelve ErrVersionConflict.
	UpdateStatus(ctx context.Context, id string, expectedVersion int32, newStatus entities.PenaltyStatus, actorID string) (entities.Penalty, error)
	// SetNotified establece notified_at, appeal_deadline_at y status='notified'.
	SetNotified(ctx context.Context, id string, expectedVersion int32, notifiedAt time.Time, appealDeadlineAt time.Time, actorID string) (entities.Penalty, error)
	// SetCouncilApproved establece council_approved_by y council_approved_at.
	SetCouncilApproved(ctx context.Context, id string, expectedVersion int32, approvedByUserID string, approvedAt time.Time, actorID string) (entities.Penalty, error)
	// SetConfirmed establece confirmed_at y status='confirmed'.
	SetConfirmed(ctx context.Context, id string, expectedVersion int32, confirmedAt time.Time, actorID string) (entities.Penalty, error)
	// SetSettled establece settled_at y status='settled'.
	SetSettled(ctx context.Context, id string, expectedVersion int32, settledAt time.Time, actorID string) (entities.Penalty, error)
	// SetDismissed establece dismissed_at y status='dismissed'.
	SetDismissed(ctx context.Context, id string, expectedVersion int32, dismissedAt time.Time, actorID string) (entities.Penalty, error)
	// SetCancelled establece cancelled_at y status='cancelled'.
	SetCancelled(ctx context.Context, id string, expectedVersion int32, cancelledAt time.Time, actorID string) (entities.Penalty, error)
	// CountReincidence cuenta sanciones confirmed/settled para el mismo
	// (debtor, catalog) en una ventana temporal.
	CountReincidence(ctx context.Context, debtorUserID, catalogID string, since time.Time) (int, error)
}

// ---------------------------------------------------------------------------
// AppealRepository
// ---------------------------------------------------------------------------

// CreateAppealInput agrupa los datos para persistir una apelacion.
type CreateAppealInput struct {
	PenaltyID         string
	SubmittedByUserID string
	Grounds           string
	ActorID           string
}

// ResolveAppealInput agrupa los datos para resolver una apelacion.
type ResolveAppealInput struct {
	ID               string
	ResolvedByUserID string
	Resolution       string
	NewStatus        entities.AppealStatus
	ExpectedVersion  int32
	ActorID          string
}

// AppealRepository es el puerto que persiste apelaciones.
type AppealRepository interface {
	// Create inserta una apelacion nueva en estado 'submitted'. Si ya
	// existe una apelacion activa para el penalty, devuelve
	// ErrAppealAlreadyActive.
	Create(ctx context.Context, in CreateAppealInput) (entities.PenaltyAppeal, error)
	// GetByID devuelve una apelacion por id. Si no existe, devuelve
	// ErrAppealNotFound.
	GetByID(ctx context.Context, id string) (entities.PenaltyAppeal, error)
	// GetActiveByPenaltyID devuelve la apelacion activa de un penalty.
	// Si no existe, devuelve ErrAppealNotFound.
	GetActiveByPenaltyID(ctx context.Context, penaltyID string) (entities.PenaltyAppeal, error)
	// Resolve resuelve una apelacion con concurrencia optimista.
	Resolve(ctx context.Context, in ResolveAppealInput) (entities.PenaltyAppeal, error)
}

// ---------------------------------------------------------------------------
// StatusHistoryRepository
// ---------------------------------------------------------------------------

// RecordStatusHistoryInput agrupa los datos para registrar una transicion.
type RecordStatusHistoryInput struct {
	PenaltyID            string
	FromStatus           *string
	ToStatus             string
	TransitionedByUserID string
	Notes                *string
}

// StatusHistoryRepository es el puerto que persiste el historial append-only.
type StatusHistoryRepository interface {
	// Record inserta un registro de historial.
	Record(ctx context.Context, in RecordStatusHistoryInput) (entities.PenaltyStatusHistory, error)
	// ListByPenaltyID devuelve el historial de un penalty ordenado por
	// transitioned_at desc.
	ListByPenaltyID(ctx context.Context, penaltyID string) ([]entities.PenaltyStatusHistory, error)
}

// ---------------------------------------------------------------------------
// OutboxRepository
// ---------------------------------------------------------------------------

// EnqueueOutboxInput agrupa los datos para encolar un evento.
type EnqueueOutboxInput struct {
	PenaltyID      string
	EventType      entities.OutboxEventType
	Payload        []byte
	IdempotencyKey *string
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
