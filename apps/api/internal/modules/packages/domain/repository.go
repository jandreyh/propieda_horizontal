// Package domain define los puertos del modulo packages.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"

	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
)

// --- Sentinels ---

// ErrPackageNotFound se devuelve cuando un paquete por id no existe.
// La capa HTTP lo mapea a 404.
var ErrPackageNotFound = errors.New("packages: package not found")

// ErrCategoryNotFound se devuelve cuando una categoria por nombre/id no
// existe. Mapea a 404 / 400 segun el contexto.
var ErrCategoryNotFound = errors.New("packages: category not found")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio. La capa HTTP lo mapea a 409.
var ErrVersionConflict = errors.New("packages: version conflict")

// ErrInvalidTransition se devuelve cuando se intenta transicionar el
// status de un paquete a un estado no permitido (p. ej. entregar uno ya
// entregado). Mapea a 409.
var ErrInvalidTransition = errors.New("packages: invalid status transition")

// ErrEvidenceRequired se devuelve cuando una entrega manual no incluye
// firma ni foto. Mapea a 400.
var ErrEvidenceRequired = errors.New("packages: signature_url or photo_evidence_url is required for manual delivery")

// ErrIdempotencyConflict se reserva para colisiones de idempotency_key
// con cuerpos distintos. Mapea a 409.
var ErrIdempotencyConflict = errors.New("packages: idempotency key conflict")

// --- CategoryRepository ---

// CategoryRepository es el puerto que persiste categorias de paquete.
type CategoryRepository interface {
	// List devuelve las categorias activas (no soft-deleted) ordenadas por
	// nombre.
	List(ctx context.Context) ([]entities.PackageCategory, error)
	// GetByName devuelve la categoria activa por nombre exacto. Si no
	// existe, devuelve ErrCategoryNotFound.
	GetByName(ctx context.Context, name string) (entities.PackageCategory, error)
	// GetByID devuelve la categoria por id. Si no existe, devuelve
	// ErrCategoryNotFound.
	GetByID(ctx context.Context, id string) (entities.PackageCategory, error)
}

// --- PackageRepository ---

// CreatePackageInput agrupa los datos para persistir un paquete nuevo.
type CreatePackageInput struct {
	UnitID              string
	RecipientName       string
	CategoryID          *string
	ReceivedEvidenceURL *string
	Carrier             *string
	TrackingNumber      *string
	ReceivedByUserID    string
}

// PackageRepository es el puerto que persiste paquetes.
type PackageRepository interface {
	// Create inserta un paquete nuevo en estado 'received'.
	Create(ctx context.Context, in CreatePackageInput) (entities.Package, error)
	// GetByID devuelve un paquete por id. Si no existe, devuelve
	// ErrPackageNotFound.
	GetByID(ctx context.Context, id string) (entities.Package, error)
	// ListByUnit devuelve los paquetes de una unidad ordenados por fecha
	// descendente.
	ListByUnit(ctx context.Context, unitID string) ([]entities.Package, error)
	// ListByStatus devuelve los paquetes con un status dado ordenados por
	// fecha descendente.
	ListByStatus(ctx context.Context, status entities.PackageStatus) ([]entities.Package, error)
	// UpdateStatusOptimistic ejecuta un UPDATE atomico con WHERE version =
	// expectedVersion. Si afecta 0 filas (otra goroutine ya cambio el
	// estado), devuelve ErrVersionConflict.
	UpdateStatusOptimistic(ctx context.Context, id string, expectedVersion int32, newStatus entities.PackageStatus, actorID string) (entities.Package, error)
	// Return marca el paquete como devuelto al transportador. Tambien usa
	// version optimista (mismo contrato que UpdateStatusOptimistic).
	Return(ctx context.Context, id string, expectedVersion int32, actorID string) (entities.Package, error)
	// ListPendingReminder devuelve paquetes en estado 'received' con mas
	// de 3 dias en porteria y SIN reminder en las ultimas 24h.
	ListPendingReminder(ctx context.Context) ([]entities.Package, error)
}

// --- DeliveryRepository ---

// RecordDeliveryInput agrupa los datos para registrar una entrega.
type RecordDeliveryInput struct {
	PackageID           string
	DeliveredToUserID   *string
	RecipientNameManual *string
	DeliveryMethod      entities.DeliveryMethod
	SignatureURL        *string
	PhotoEvidenceURL    *string
	DeliveredByUserID   string
	Notes               *string
}

// DeliveryRepository es el puerto que persiste eventos de entrega.
type DeliveryRepository interface {
	// Record inserta el evento de entrega.
	Record(ctx context.Context, in RecordDeliveryInput) (entities.DeliveryEvent, error)
}

// --- OutboxRepository ---

// EnqueueOutboxInput agrupa los datos para encolar un evento.
type EnqueueOutboxInput struct {
	PackageID string
	EventType entities.OutboxEventType
	Payload   []byte
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
