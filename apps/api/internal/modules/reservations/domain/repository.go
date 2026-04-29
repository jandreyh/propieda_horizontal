// Package domain define los puertos del modulo reservations.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/reservations/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrCommonAreaNotFound se devuelve cuando una zona comun por id no existe.
var ErrCommonAreaNotFound = errors.New("reservations: common area not found")

// ErrCommonAreaCodeDuplicate se devuelve cuando se intenta crear o
// actualizar una zona comun con un codigo que ya existe.
var ErrCommonAreaCodeDuplicate = errors.New("reservations: common area code already exists")

// ErrReservationNotFound se devuelve cuando una reserva por id no existe.
var ErrReservationNotFound = errors.New("reservations: reservation not found")

// ErrReservationSlotConflict se devuelve cuando se intenta reservar un
// slot ya confirmado para la misma zona comun.
var ErrReservationSlotConflict = errors.New("reservations: reservation slot conflict")

// ErrIdempotencyKeyConflict se devuelve cuando la clave de idempotencia
// ya esta en uso.
var ErrIdempotencyKeyConflict = errors.New("reservations: idempotency key conflict")

// ErrBlackoutNotFound se devuelve cuando un bloqueo por id no existe.
var ErrBlackoutNotFound = errors.New("reservations: blackout not found")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio.
var ErrVersionConflict = errors.New("reservations: version conflict")

// ErrInvalidTransition se devuelve cuando se intenta transicionar el
// status de una entidad a un estado no permitido.
var ErrInvalidTransition = errors.New("reservations: invalid status transition")

// ---------------------------------------------------------------------------
// CommonAreaRepository
// ---------------------------------------------------------------------------

// CreateCommonAreaInput agrupa los datos para persistir una zona comun.
type CreateCommonAreaInput struct {
	Code                string
	Name                string
	Kind                entities.CommonAreaKind
	MaxCapacity         *int32
	OpeningTime         *string
	ClosingTime         *string
	SlotDurationMinutes int32
	CostPerUse          float64
	SecurityDeposit     float64
	RequiresApproval    bool
	IsActive            bool
	Description         *string
}

// UpdateCommonAreaInput agrupa los datos para actualizar una zona comun.
type UpdateCommonAreaInput struct {
	ID                  string
	Code                string
	Name                string
	Kind                entities.CommonAreaKind
	MaxCapacity         *int32
	OpeningTime         *string
	ClosingTime         *string
	SlotDurationMinutes int32
	CostPerUse          float64
	SecurityDeposit     float64
	RequiresApproval    bool
	IsActive            bool
	Description         *string
	Status              entities.CommonAreaStatus
	ExpectedVersion     int32
	ActorID             string
}

// CommonAreaRepository es el puerto que persiste zonas comunes.
type CommonAreaRepository interface {
	Create(ctx context.Context, in CreateCommonAreaInput) (entities.CommonArea, error)
	GetByID(ctx context.Context, id string) (entities.CommonArea, error)
	List(ctx context.Context) ([]entities.CommonArea, error)
	Update(ctx context.Context, in UpdateCommonAreaInput) (entities.CommonArea, error)
}

// ---------------------------------------------------------------------------
// BlackoutRepository
// ---------------------------------------------------------------------------

// CreateBlackoutInput agrupa los datos para persistir un bloqueo.
type CreateBlackoutInput struct {
	CommonAreaID string
	FromAt       time.Time
	ToAt         time.Time
	Reason       string
	ActorID      string
}

// BlackoutRepository es el puerto que persiste bloqueos.
type BlackoutRepository interface {
	Create(ctx context.Context, in CreateBlackoutInput) (entities.ReservationBlackout, error)
	ListActiveByCommonArea(ctx context.Context, commonAreaID string) ([]entities.ReservationBlackout, error)
	ListByCommonAreaAndWindow(ctx context.Context, commonAreaID string, fromAt, toAt time.Time) ([]entities.ReservationBlackout, error)
}

// ---------------------------------------------------------------------------
// ReservationRepository
// ---------------------------------------------------------------------------

// CreateReservationInput agrupa los datos para persistir una reserva.
type CreateReservationInput struct {
	CommonAreaID      string
	UnitID            string
	RequestedByUserID string
	SlotStartAt       time.Time
	SlotEndAt         time.Time
	AttendeesCount    *int32
	Cost              float64
	SecurityDeposit   float64
	QRCodeHash        *string
	IdempotencyKey    *string
	Notes             *string
	Status            entities.ReservationStatus
}

// ReservationRepository es el puerto que persiste reservas.
type ReservationRepository interface {
	Create(ctx context.Context, in CreateReservationInput) (entities.Reservation, error)
	GetByID(ctx context.Context, id string) (entities.Reservation, error)
	GetByIdempotencyKey(ctx context.Context, key string) (entities.Reservation, error)
	GetByQRCodeHash(ctx context.Context, hash string) (entities.Reservation, error)
	List(ctx context.Context) ([]entities.Reservation, error)
	ListByUnit(ctx context.Context, unitID string) ([]entities.Reservation, error)
	ListByCommonAreaAndDate(ctx context.Context, commonAreaID string, start, end time.Time) ([]entities.Reservation, error)
	UpdateStatus(ctx context.Context, id string, expectedVersion int32, newStatus entities.ReservationStatus, actorID string) (entities.Reservation, error)
	Approve(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error)
	Cancel(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error)
	Reject(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error)
	Checkin(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Reservation, error)
}

// ---------------------------------------------------------------------------
// StatusHistoryRepository
// ---------------------------------------------------------------------------

// RecordStatusHistoryInput agrupa los datos para registrar un cambio de
// estado.
type RecordStatusHistoryInput struct {
	ReservationID string
	FromStatus    *string
	ToStatus      string
	ChangedBy     *string
	Reason        *string
}

// StatusHistoryRepository es el puerto que persiste el historial de
// cambios de estado (append-only).
type StatusHistoryRepository interface {
	Record(ctx context.Context, in RecordStatusHistoryInput) (entities.ReservationStatusHistory, error)
	ListByReservation(ctx context.Context, reservationID string) ([]entities.ReservationStatusHistory, error)
}

// ---------------------------------------------------------------------------
// OutboxRepository
// ---------------------------------------------------------------------------

// EnqueueOutboxInput agrupa los datos para encolar un evento.
type EnqueueOutboxInput struct {
	AggregateID string
	EventType   entities.OutboxEventType
	Payload     []byte
}

// OutboxRepository es el puerto que persiste eventos del outbox.
type OutboxRepository interface {
	Enqueue(ctx context.Context, in EnqueueOutboxInput) (entities.OutboxEvent, error)
	LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error)
	MarkDelivered(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error
}
