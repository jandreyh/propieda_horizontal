// Package domain define los puertos del modulo parking.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/parking/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrSpaceNotFound se devuelve cuando un espacio por id no existe.
// La capa HTTP lo mapea a 404.
var ErrSpaceNotFound = errors.New("parking: space not found")

// ErrSpaceCodeDuplicate se devuelve cuando se intenta crear o actualizar
// un espacio con un codigo que ya existe (UNIQUE parcial). Mapea a 409.
var ErrSpaceCodeDuplicate = errors.New("parking: space code already exists")

// ErrAssignmentNotFound se devuelve cuando una asignacion por id no existe.
// Mapea a 404.
var ErrAssignmentNotFound = errors.New("parking: assignment not found")

// ErrAssignmentAlreadyActive se devuelve cuando se intenta crear una
// asignacion para un espacio que ya tiene una asignacion activa (UNIQUE
// parcial en parking_space_id WHERE until_date IS NULL). Mapea a 409.
var ErrAssignmentAlreadyActive = errors.New("parking: space already has an active assignment")

// ErrReservationNotFound se devuelve cuando una reserva de visitante por
// id no existe. Mapea a 404.
var ErrReservationNotFound = errors.New("parking: visitor reservation not found")

// ErrReservationSlotConflict se devuelve cuando se intenta reservar un
// slot que ya esta ocupado para el mismo espacio. Mapea a 409.
var ErrReservationSlotConflict = errors.New("parking: visitor reservation slot conflict")

// ErrLotteryNotFound se devuelve cuando un sorteo por id no existe.
// Mapea a 404.
var ErrLotteryNotFound = errors.New("parking: lottery run not found")

// ErrRuleNotFound se devuelve cuando una regla por key no existe.
// Mapea a 404.
var ErrRuleNotFound = errors.New("parking: rule not found")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio. La capa HTTP lo mapea a 409.
var ErrVersionConflict = errors.New("parking: version conflict")

// ErrInvalidTransition se devuelve cuando se intenta transicionar el
// status de una entidad a un estado no permitido. Mapea a 409.
var ErrInvalidTransition = errors.New("parking: invalid status transition")

// ---------------------------------------------------------------------------
// SpaceRepository
// ---------------------------------------------------------------------------

// CreateSpaceInput agrupa los datos para persistir un espacio nuevo.
type CreateSpaceInput struct {
	Code        string
	Type        entities.SpaceType
	StructureID *string
	Level       *string
	Zone        *string
	MonthlyFee  *float64
	IsVisitor   bool
	Notes       *string
}

// UpdateSpaceInput agrupa los datos para actualizar un espacio existente.
type UpdateSpaceInput struct {
	ID              string
	Code            string
	Type            entities.SpaceType
	StructureID     *string
	Level           *string
	Zone            *string
	MonthlyFee      *float64
	IsVisitor       bool
	Notes           *string
	Status          entities.SpaceStatus
	ExpectedVersion int32
	ActorID         string
}

// SpaceRepository es el puerto que persiste espacios de parqueadero.
type SpaceRepository interface {
	// Create inserta un espacio nuevo en estado 'active'.
	Create(ctx context.Context, in CreateSpaceInput) (entities.ParkingSpace, error)
	// GetByID devuelve un espacio por id. Si no existe, devuelve
	// ErrSpaceNotFound.
	GetByID(ctx context.Context, id string) (entities.ParkingSpace, error)
	// GetByCode devuelve un espacio por codigo (activo, no soft-deleted).
	// Si no existe, devuelve ErrSpaceNotFound.
	GetByCode(ctx context.Context, code string) (entities.ParkingSpace, error)
	// List devuelve los espacios activos (no soft-deleted) ordenados por
	// codigo.
	List(ctx context.Context) ([]entities.ParkingSpace, error)
	// Update actualiza un espacio existente con concurrencia optimista.
	// Si la version no coincide, devuelve ErrVersionConflict.
	Update(ctx context.Context, in UpdateSpaceInput) (entities.ParkingSpace, error)
	// SoftDelete marca un espacio como eliminado (soft delete).
	SoftDelete(ctx context.Context, id string, expectedVersion int32, actorID string) error
}

// ---------------------------------------------------------------------------
// AssignmentRepository
// ---------------------------------------------------------------------------

// CreateAssignmentInput agrupa los datos para persistir una asignacion.
type CreateAssignmentInput struct {
	ParkingSpaceID   string
	UnitID           string
	VehicleID        *string
	AssignedByUserID *string
	SinceDate        time.Time
	Notes            *string
}

// AssignmentRepository es el puerto que persiste asignaciones de
// parqueadero.
type AssignmentRepository interface {
	// Create inserta una asignacion nueva en estado 'active'. Si el
	// espacio ya tiene una asignacion activa, devuelve
	// ErrAssignmentAlreadyActive.
	Create(ctx context.Context, in CreateAssignmentInput) (entities.ParkingAssignment, error)
	// GetByID devuelve una asignacion por id. Si no existe, devuelve
	// ErrAssignmentNotFound.
	GetByID(ctx context.Context, id string) (entities.ParkingAssignment, error)
	// GetActiveBySpaceID devuelve la asignacion activa (until_date IS
	// NULL) para un espacio. Si no hay asignacion activa, devuelve
	// ErrAssignmentNotFound.
	GetActiveBySpaceID(ctx context.Context, spaceID string) (entities.ParkingAssignment, error)
	// ListActiveByUnitID devuelve las asignaciones activas de una unidad
	// ordenadas por since_date descendente.
	ListActiveByUnitID(ctx context.Context, unitID string) ([]entities.ParkingAssignment, error)
	// ListBySpaceID devuelve todas las asignaciones (activas y cerradas)
	// de un espacio ordenadas por since_date descendente.
	ListBySpaceID(ctx context.Context, spaceID string) ([]entities.ParkingAssignment, error)
	// CloseAssignment establece until_date y cambia el status a 'closed'
	// con concurrencia optimista. Si la version no coincide, devuelve
	// ErrVersionConflict.
	CloseAssignment(ctx context.Context, id string, untilDate time.Time, expectedVersion int32, actorID string) (entities.ParkingAssignment, error)
	// SoftDelete marca una asignacion como eliminada (soft delete).
	SoftDelete(ctx context.Context, id string, expectedVersion int32, actorID string) error
}

// ---------------------------------------------------------------------------
// AssignmentHistoryRepository
// ---------------------------------------------------------------------------

// RecordHistoryInput agrupa los datos para registrar un snapshot de
// asignacion en el historial.
type RecordHistoryInput struct {
	ParkingSpaceID  string
	UnitID          string
	AssignmentID    *string
	SinceDate       time.Time
	UntilDate       *time.Time
	ClosedReason    *string
	SnapshotPayload []byte
	RecordedBy      *string
}

// AssignmentHistoryRepository es el puerto que persiste el historial
// append-only de asignaciones.
type AssignmentHistoryRepository interface {
	// Record inserta un registro de historial.
	Record(ctx context.Context, in RecordHistoryInput) (entities.AssignmentHistory, error)
	// ListBySpaceID devuelve el historial de un espacio ordenado por
	// recorded_at descendente.
	ListBySpaceID(ctx context.Context, spaceID string) ([]entities.AssignmentHistory, error)
	// ListByUnitID devuelve el historial de una unidad ordenado por
	// recorded_at descendente.
	ListByUnitID(ctx context.Context, unitID string) ([]entities.AssignmentHistory, error)
}

// ---------------------------------------------------------------------------
// VisitorReservationRepository
// ---------------------------------------------------------------------------

// CreateReservationInput agrupa los datos para persistir una reserva de
// visitante.
type CreateReservationInput struct {
	ParkingSpaceID  string
	UnitID          string
	RequestedBy     string
	VisitorName     string
	VisitorDocument *string
	VehiclePlate    *string
	SlotStartAt     time.Time
	SlotEndAt       time.Time
	IdempotencyKey  *string
}

// VisitorReservationRepository es el puerto que persiste reservas de
// visitantes.
type VisitorReservationRepository interface {
	// Create inserta una reserva nueva. Si el slot esta ocupado, devuelve
	// ErrReservationSlotConflict.
	Create(ctx context.Context, in CreateReservationInput) (entities.VisitorReservation, error)
	// GetByID devuelve una reserva por id. Si no existe, devuelve
	// ErrReservationNotFound.
	GetByID(ctx context.Context, id string) (entities.VisitorReservation, error)
	// ListByDate devuelve las reservas de una fecha (entre start y end)
	// ordenadas por slot_start_at.
	ListByDate(ctx context.Context, start, end time.Time) ([]entities.VisitorReservation, error)
	// ListByUnit devuelve las reservas de una unidad ordenadas por
	// slot_start_at descendente.
	ListByUnit(ctx context.Context, unitID string) ([]entities.VisitorReservation, error)
	// UpdateStatus actualiza el status con concurrencia optimista. Si la
	// version no coincide, devuelve ErrVersionConflict.
	UpdateStatus(ctx context.Context, id string, expectedVersion int32, newStatus entities.ReservationStatus, actorID string) (entities.VisitorReservation, error)
	// Cancel cancela una reserva (atajo de UpdateStatus con status
	// 'cancelled'). Retorna ErrVersionConflict si la version no coincide.
	Cancel(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.VisitorReservation, error)
	// GetByIdempotencyKey devuelve una reserva por clave de idempotencia.
	// Si no existe, devuelve ErrReservationNotFound.
	GetByIdempotencyKey(ctx context.Context, key string) (entities.VisitorReservation, error)
}

// ---------------------------------------------------------------------------
// LotteryRunRepository
// ---------------------------------------------------------------------------

// CreateLotteryRunInput agrupa los datos para persistir una ejecucion de
// sorteo.
type CreateLotteryRunInput struct {
	Name       string
	SeedHash   string
	Criteria   []byte
	ExecutedBy string
}

// LotteryRunRepository es el puerto que persiste ejecuciones de sorteo.
type LotteryRunRepository interface {
	// Create inserta un sorteo nuevo en estado 'completed'.
	Create(ctx context.Context, in CreateLotteryRunInput) (entities.LotteryRun, error)
	// GetByID devuelve un sorteo por id. Si no existe, devuelve
	// ErrLotteryNotFound.
	GetByID(ctx context.Context, id string) (entities.LotteryRun, error)
	// List devuelve los sorteos ordenados por executed_at descendente.
	List(ctx context.Context) ([]entities.LotteryRun, error)
}

// ---------------------------------------------------------------------------
// LotteryResultRepository
// ---------------------------------------------------------------------------

// CreateLotteryResultInput agrupa los datos para persistir un resultado
// individual de sorteo.
type CreateLotteryResultInput struct {
	LotteryRunID   string
	UnitID         string
	ParkingSpaceID *string
	Position       int32
	Status         entities.LotteryResultStatus
}

// LotteryResultRepository es el puerto que persiste resultados de sorteo.
type LotteryResultRepository interface {
	// CreateBatch inserta un lote de resultados de sorteo en una sola
	// operacion.
	CreateBatch(ctx context.Context, results []CreateLotteryResultInput) ([]entities.LotteryResult, error)
	// ListByRunID devuelve los resultados de un sorteo ordenados por
	// position ascendente.
	ListByRunID(ctx context.Context, runID string) ([]entities.LotteryResult, error)
}

// ---------------------------------------------------------------------------
// RuleRepository
// ---------------------------------------------------------------------------

// SetRuleInput agrupa los datos para establecer o actualizar una regla.
type SetRuleInput struct {
	RuleKey     string
	RuleValue   []byte
	Description *string
	ActorID     string
}

// RuleRepository es el puerto que persiste reglas de configuracion del
// modulo parking.
type RuleRepository interface {
	// Get devuelve una regla por clave. Si no existe, devuelve
	// ErrRuleNotFound.
	Get(ctx context.Context, key string) (entities.ParkingRule, error)
	// Set establece o actualiza una regla por clave (upsert).
	Set(ctx context.Context, in SetRuleInput) (entities.ParkingRule, error)
	// List devuelve todas las reglas activas ordenadas por rule_key.
	List(ctx context.Context) ([]entities.ParkingRule, error)
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
