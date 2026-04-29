// Package usecases orquesta la logica de aplicacion del modulo reservations.
// Cada usecase recibe sus dependencias por inyeccion (interfaces) y NO
// conoce HTTP ni la base.
package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/reservations/domain"
	"github.com/saas-ph/api/internal/modules/reservations/domain/entities"
	"github.com/saas-ph/api/internal/modules/reservations/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// CreateCommonArea
// ---------------------------------------------------------------------------

// CreateCommonArea crea una zona comun nueva en estado 'active'.
type CreateCommonArea struct {
	CommonAreas domain.CommonAreaRepository
}

// CreateCommonAreaInput es el input del usecase (sin tags JSON).
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

// Execute valida y delega al repo.
func (u CreateCommonArea) Execute(ctx context.Context, in CreateCommonAreaInput) (entities.CommonArea, error) {
	if strings.TrimSpace(in.Code) == "" {
		return entities.CommonArea{}, apperrors.BadRequest("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.CommonArea{}, apperrors.BadRequest("name is required")
	}
	if !in.Kind.IsValid() {
		return entities.CommonArea{}, apperrors.BadRequest("kind: invalid common area kind")
	}
	if in.SlotDurationMinutes <= 0 {
		return entities.CommonArea{}, apperrors.BadRequest("slot_duration_minutes must be positive")
	}

	area, err := u.CommonAreas.Create(ctx, domain.CreateCommonAreaInput{
		Code:                strings.TrimSpace(in.Code),
		Name:                strings.TrimSpace(in.Name),
		Kind:                in.Kind,
		MaxCapacity:         in.MaxCapacity,
		OpeningTime:         in.OpeningTime,
		ClosingTime:         in.ClosingTime,
		SlotDurationMinutes: in.SlotDurationMinutes,
		CostPerUse:          in.CostPerUse,
		SecurityDeposit:     in.SecurityDeposit,
		RequiresApproval:    in.RequiresApproval,
		IsActive:            in.IsActive,
		Description:         in.Description,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCommonAreaCodeDuplicate) {
			return entities.CommonArea{}, apperrors.Conflict("common area code already exists")
		}
		return entities.CommonArea{}, apperrors.Internal("failed to create common area")
	}
	return area, nil
}

// ---------------------------------------------------------------------------
// UpdateCommonArea
// ---------------------------------------------------------------------------

// UpdateCommonArea actualiza una zona comun existente con concurrencia
// optimista.
type UpdateCommonArea struct {
	CommonAreas domain.CommonAreaRepository
}

// UpdateCommonAreaInput es el input del usecase.
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

// Execute valida y delega al repo.
func (u UpdateCommonArea) Execute(ctx context.Context, in UpdateCommonAreaInput) (entities.CommonArea, error) {
	if err := policies.ValidateUUID(in.ID); err != nil {
		return entities.CommonArea{}, apperrors.BadRequest("id: " + err.Error())
	}
	if strings.TrimSpace(in.Code) == "" {
		return entities.CommonArea{}, apperrors.BadRequest("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.CommonArea{}, apperrors.BadRequest("name is required")
	}
	if !in.Kind.IsValid() {
		return entities.CommonArea{}, apperrors.BadRequest("kind: invalid common area kind")
	}
	if !in.Status.IsValid() {
		return entities.CommonArea{}, apperrors.BadRequest("status: invalid common area status")
	}

	area, err := u.CommonAreas.Update(ctx, domain.UpdateCommonAreaInput{
		ID:                  in.ID,
		Code:                strings.TrimSpace(in.Code),
		Name:                strings.TrimSpace(in.Name),
		Kind:                in.Kind,
		MaxCapacity:         in.MaxCapacity,
		OpeningTime:         in.OpeningTime,
		ClosingTime:         in.ClosingTime,
		SlotDurationMinutes: in.SlotDurationMinutes,
		CostPerUse:          in.CostPerUse,
		SecurityDeposit:     in.SecurityDeposit,
		RequiresApproval:    in.RequiresApproval,
		IsActive:            in.IsActive,
		Description:         in.Description,
		Status:              in.Status,
		ExpectedVersion:     in.ExpectedVersion,
		ActorID:             in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCommonAreaNotFound):
			return entities.CommonArea{}, apperrors.NotFound("common area not found")
		case errors.Is(err, domain.ErrVersionConflict):
			return entities.CommonArea{}, mapVersionConflict()
		case errors.Is(err, domain.ErrCommonAreaCodeDuplicate):
			return entities.CommonArea{}, apperrors.Conflict("common area code already exists")
		default:
			return entities.CommonArea{}, apperrors.Internal("failed to update common area")
		}
	}
	return area, nil
}

// ---------------------------------------------------------------------------
// ListCommonAreas
// ---------------------------------------------------------------------------

// ListCommonAreas lista las zonas comunes activas.
type ListCommonAreas struct {
	CommonAreas domain.CommonAreaRepository
}

// Execute delega al repo.
func (u ListCommonAreas) Execute(ctx context.Context) ([]entities.CommonArea, error) {
	out, err := u.CommonAreas.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list common areas")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// CreateBlackout
// ---------------------------------------------------------------------------

// CreateBlackout crea un bloqueo temporal en una zona comun.
type CreateBlackout struct {
	CommonAreas domain.CommonAreaRepository
	Blackouts   domain.BlackoutRepository
}

// CreateBlackoutInput es el input del usecase.
type CreateBlackoutInput struct {
	CommonAreaID string
	FromAt       time.Time
	ToAt         time.Time
	Reason       string
	ActorID      string
}

// Execute valida y delega.
func (u CreateBlackout) Execute(ctx context.Context, in CreateBlackoutInput) (entities.ReservationBlackout, error) {
	if err := policies.ValidateUUID(in.CommonAreaID); err != nil {
		return entities.ReservationBlackout{}, apperrors.BadRequest("common_area_id: " + err.Error())
	}
	if strings.TrimSpace(in.Reason) == "" {
		return entities.ReservationBlackout{}, apperrors.BadRequest("reason is required")
	}
	if err := policies.ValidateBlackoutWindow(in.FromAt, in.ToAt); err != nil {
		return entities.ReservationBlackout{}, apperrors.BadRequest(err.Error())
	}

	// Verify common area exists.
	_, err := u.CommonAreas.GetByID(ctx, in.CommonAreaID)
	if err != nil {
		if errors.Is(err, domain.ErrCommonAreaNotFound) {
			return entities.ReservationBlackout{}, apperrors.NotFound("common area not found")
		}
		return entities.ReservationBlackout{}, apperrors.Internal("failed to load common area")
	}

	blackout, err := u.Blackouts.Create(ctx, domain.CreateBlackoutInput{
		CommonAreaID: in.CommonAreaID,
		FromAt:       in.FromAt,
		ToAt:         in.ToAt,
		Reason:       strings.TrimSpace(in.Reason),
		ActorID:      in.ActorID,
	})
	if err != nil {
		return entities.ReservationBlackout{}, apperrors.Internal("failed to create blackout")
	}
	return blackout, nil
}

// ---------------------------------------------------------------------------
// GetAvailability
// ---------------------------------------------------------------------------

// GetAvailability devuelve los slots disponibles de una zona comun para
// una fecha dada.
type GetAvailability struct {
	CommonAreas  domain.CommonAreaRepository
	Reservations domain.ReservationRepository
	Blackouts    domain.BlackoutRepository
}

// AvailabilitySlot es un slot con su estado de disponibilidad.
type AvailabilitySlot struct {
	SlotStart   time.Time
	SlotEnd     time.Time
	IsAvailable bool
}

// Execute genera los slots del dia y los marca como disponibles o no.
func (u GetAvailability) Execute(ctx context.Context, commonAreaID string, date time.Time) ([]AvailabilitySlot, error) {
	if err := policies.ValidateUUID(commonAreaID); err != nil {
		return nil, apperrors.BadRequest("common_area_id: " + err.Error())
	}

	area, err := u.CommonAreas.GetByID(ctx, commonAreaID)
	if err != nil {
		if errors.Is(err, domain.ErrCommonAreaNotFound) {
			return nil, apperrors.NotFound("common area not found")
		}
		return nil, apperrors.Internal("failed to load common area")
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Get existing reservations for the day.
	reservations, err := u.Reservations.ListByCommonAreaAndDate(ctx, commonAreaID, startOfDay, endOfDay)
	if err != nil {
		return nil, apperrors.Internal("failed to list reservations")
	}

	// Get active blackouts for the day.
	blackouts, err := u.Blackouts.ListByCommonAreaAndWindow(ctx, commonAreaID, startOfDay, endOfDay)
	if err != nil {
		return nil, apperrors.Internal("failed to list blackouts")
	}

	// Build occupied set (confirmed or pending slot_start_at).
	occupiedSlots := make(map[int64]bool)
	for _, r := range reservations {
		if r.Status == entities.ReservationStatusConfirmed || r.Status == entities.ReservationStatusPending {
			occupiedSlots[r.SlotStartAt.Unix()] = true
		}
	}

	// Generate slots for the day.
	slotDuration := time.Duration(area.SlotDurationMinutes) * time.Minute
	var slots []AvailabilitySlot
	cursor := startOfDay
	for cursor.Add(slotDuration).Before(endOfDay) || cursor.Add(slotDuration).Equal(endOfDay) {
		slotEnd := cursor.Add(slotDuration)
		available := true

		// Check if slot is occupied.
		if occupiedSlots[cursor.Unix()] {
			available = false
		}

		// Check if slot falls in a blackout.
		for _, bo := range blackouts {
			if bo.IsActive() && policies.IsSlotInBlackout(cursor, slotEnd, bo.FromAt, bo.ToAt) {
				available = false
				break
			}
		}

		slots = append(slots, AvailabilitySlot{
			SlotStart:   cursor,
			SlotEnd:     slotEnd,
			IsAvailable: available,
		})
		cursor = slotEnd
	}

	return slots, nil
}

// ---------------------------------------------------------------------------
// CreateReservation
// ---------------------------------------------------------------------------

// CreateReservation crea una reserva de zona comun con idempotencia y QR.
type CreateReservation struct {
	CommonAreas  domain.CommonAreaRepository
	Reservations domain.ReservationRepository
	Blackouts    domain.BlackoutRepository
	History      domain.StatusHistoryRepository
	Outbox       domain.OutboxRepository
	TxRunner     TxRunner
	Now          func() time.Time
}

// CreateReservationInput es el input del usecase.
type CreateReservationInput struct {
	CommonAreaID   string
	UnitID         string
	RequestedBy    string
	SlotStartAt    time.Time
	SlotEndAt      time.Time
	AttendeesCount *int32
	Notes          *string
	IdempotencyKey *string
}

// Execute valida y delega.
func (u CreateReservation) Execute(ctx context.Context, in CreateReservationInput) (entities.Reservation, error) {
	if err := policies.ValidateUUID(in.CommonAreaID); err != nil {
		return entities.Reservation{}, apperrors.BadRequest("common_area_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.Reservation{}, apperrors.BadRequest("unit_id: " + err.Error())
	}

	// Idempotency check.
	if in.IdempotencyKey != nil && *in.IdempotencyKey != "" {
		existing, err := u.Reservations.GetByIdempotencyKey(ctx, *in.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, domain.ErrReservationNotFound) {
			return entities.Reservation{}, apperrors.Internal("failed to check idempotency")
		}
	}

	area, err := u.CommonAreas.GetByID(ctx, in.CommonAreaID)
	if err != nil {
		if errors.Is(err, domain.ErrCommonAreaNotFound) {
			return entities.Reservation{}, apperrors.NotFound("common area not found")
		}
		return entities.Reservation{}, apperrors.Internal("failed to load common area")
	}
	if err := policies.CanCreateReservation(area); err != nil {
		return entities.Reservation{}, apperrors.BadRequest(err.Error())
	}

	if err := policies.ValidateSlotDuration(in.SlotStartAt, in.SlotEndAt, 24); err != nil {
		return entities.Reservation{}, apperrors.BadRequest(err.Error())
	}
	if err := policies.ValidateAttendeesCount(in.AttendeesCount, area.MaxCapacity); err != nil {
		return entities.Reservation{}, apperrors.BadRequest(err.Error())
	}

	// Check for blackouts.
	blackouts, err := u.Blackouts.ListByCommonAreaAndWindow(ctx, in.CommonAreaID, in.SlotStartAt, in.SlotEndAt)
	if err != nil {
		return entities.Reservation{}, apperrors.Internal("failed to check blackouts")
	}
	for _, bo := range blackouts {
		if bo.IsActive() && policies.IsSlotInBlackout(in.SlotStartAt, in.SlotEndAt, bo.FromAt, bo.ToAt) {
			return entities.Reservation{}, apperrors.Conflict("slot falls within a blackout period")
		}
	}

	// Determine initial status.
	initialStatus := entities.ReservationStatusConfirmed
	if area.RequiresApproval {
		initialStatus = entities.ReservationStatusPending
	}

	var created entities.Reservation
	run := func(txCtx context.Context) error {
		// Generate QR hash only for confirmed reservations.
		var qrHash *string
		if initialStatus == entities.ReservationStatusConfirmed {
			h := policies.GenerateQRCodeHash("placeholder", in.CommonAreaID, in.SlotStartAt)
			qrHash = &h
		}

		reservation, createErr := u.Reservations.Create(txCtx, domain.CreateReservationInput{
			CommonAreaID:      in.CommonAreaID,
			UnitID:            in.UnitID,
			RequestedByUserID: in.RequestedBy,
			SlotStartAt:       in.SlotStartAt,
			SlotEndAt:         in.SlotEndAt,
			AttendeesCount:    in.AttendeesCount,
			Cost:              area.CostPerUse,
			SecurityDeposit:   area.SecurityDeposit,
			QRCodeHash:        qrHash,
			IdempotencyKey:    in.IdempotencyKey,
			Notes:             in.Notes,
			Status:            initialStatus,
		})
		if createErr != nil {
			return createErr
		}

		// Update QR hash with actual reservation ID if confirmed.
		if initialStatus == entities.ReservationStatusConfirmed {
			realHash := policies.GenerateQRCodeHash(reservation.ID, in.CommonAreaID, in.SlotStartAt)
			reservation.QRCodeHash = &realHash
		}

		// Record initial status in history.
		toStatus := string(initialStatus)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			ReservationID: reservation.ID,
			FromStatus:    nil,
			ToStatus:      toStatus,
			ChangedBy:     &in.RequestedBy,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"reservation_id": reservation.ID,
			"common_area_id": reservation.CommonAreaID,
			"unit_id":        reservation.UnitID,
			"slot_start_at":  reservation.SlotStartAt,
			"slot_end_at":    reservation.SlotEndAt,
			"status":         string(reservation.Status),
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: reservation.ID,
			EventType:   entities.OutboxEventReservationCreated,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		created = reservation
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrReservationSlotConflict) {
				return entities.Reservation{}, apperrors.Conflict("reservation slot conflict")
			}
			return entities.Reservation{}, apperrors.Internal("failed to create reservation")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrReservationSlotConflict) {
				return entities.Reservation{}, apperrors.Conflict("reservation slot conflict")
			}
			return entities.Reservation{}, apperrors.Internal("failed to create reservation")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// CancelReservation
// ---------------------------------------------------------------------------

// CancelReservation cancela una reserva.
type CancelReservation struct {
	Reservations domain.ReservationRepository
	History      domain.StatusHistoryRepository
	Outbox       domain.OutboxRepository
	TxRunner     TxRunner
}

// CancelReservationInput es el input del usecase.
type CancelReservationInput struct {
	ReservationID string
	ActorID       string
}

// Execute valida y delega.
func (u CancelReservation) Execute(ctx context.Context, in CancelReservationInput) (entities.Reservation, error) {
	if err := policies.ValidateUUID(in.ReservationID); err != nil {
		return entities.Reservation{}, apperrors.BadRequest("reservation_id: " + err.Error())
	}

	reservation, err := u.Reservations.GetByID(ctx, in.ReservationID)
	if err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return entities.Reservation{}, apperrors.NotFound("reservation not found")
		}
		return entities.Reservation{}, apperrors.Internal("failed to load reservation")
	}
	if !policies.CanTransitionReservation(reservation.Status, entities.ReservationStatusCancelled) {
		return entities.Reservation{}, apperrors.Conflict(
			"cannot cancel reservation in status " + string(reservation.Status))
	}

	var cancelled entities.Reservation
	run := func(txCtx context.Context) error {
		result, cancelErr := u.Reservations.Cancel(txCtx, reservation.ID, reservation.Version, in.ActorID)
		if cancelErr != nil {
			return cancelErr
		}

		fromStatus := string(reservation.Status)
		toStatus := string(entities.ReservationStatusCancelled)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			ReservationID: result.ID,
			FromStatus:    &fromStatus,
			ToStatus:      toStatus,
			ChangedBy:     &in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"reservation_id": result.ID,
			"common_area_id": result.CommonAreaID,
			"unit_id":        result.UnitID,
			"cancelled_by":   in.ActorID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: result.ID,
			EventType:   entities.OutboxEventReservationCancelled,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		cancelled = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			return entities.Reservation{}, apperrors.Internal("failed to cancel reservation")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			return entities.Reservation{}, apperrors.Internal("failed to cancel reservation")
		}
	}
	return cancelled, nil
}

// ---------------------------------------------------------------------------
// ApproveReservation
// ---------------------------------------------------------------------------

// ApproveReservation aprueba (confirma) una reserva pendiente.
type ApproveReservation struct {
	Reservations domain.ReservationRepository
	History      domain.StatusHistoryRepository
	Outbox       domain.OutboxRepository
	TxRunner     TxRunner
}

// ApproveReservationInput es el input del usecase.
type ApproveReservationInput struct {
	ReservationID string
	ActorID       string
}

// Execute valida y delega.
func (u ApproveReservation) Execute(ctx context.Context, in ApproveReservationInput) (entities.Reservation, error) {
	if err := policies.ValidateUUID(in.ReservationID); err != nil {
		return entities.Reservation{}, apperrors.BadRequest("reservation_id: " + err.Error())
	}

	reservation, err := u.Reservations.GetByID(ctx, in.ReservationID)
	if err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return entities.Reservation{}, apperrors.NotFound("reservation not found")
		}
		return entities.Reservation{}, apperrors.Internal("failed to load reservation")
	}
	if !policies.CanTransitionReservation(reservation.Status, entities.ReservationStatusConfirmed) {
		return entities.Reservation{}, apperrors.Conflict(
			"cannot approve reservation in status " + string(reservation.Status))
	}

	var approved entities.Reservation
	run := func(txCtx context.Context) error {
		result, approveErr := u.Reservations.Approve(txCtx, reservation.ID, reservation.Version, in.ActorID)
		if approveErr != nil {
			return approveErr
		}

		fromStatus := string(reservation.Status)
		toStatus := string(entities.ReservationStatusConfirmed)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			ReservationID: result.ID,
			FromStatus:    &fromStatus,
			ToStatus:      toStatus,
			ChangedBy:     &in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"reservation_id": result.ID,
			"common_area_id": result.CommonAreaID,
			"unit_id":        result.UnitID,
			"approved_by":    in.ActorID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: result.ID,
			EventType:   entities.OutboxEventReservationConfirmed,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		approved = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			if errors.Is(err, domain.ErrReservationSlotConflict) {
				return entities.Reservation{}, apperrors.Conflict("reservation slot conflict")
			}
			return entities.Reservation{}, apperrors.Internal("failed to approve reservation")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			if errors.Is(err, domain.ErrReservationSlotConflict) {
				return entities.Reservation{}, apperrors.Conflict("reservation slot conflict")
			}
			return entities.Reservation{}, apperrors.Internal("failed to approve reservation")
		}
	}
	return approved, nil
}

// ---------------------------------------------------------------------------
// RejectReservation
// ---------------------------------------------------------------------------

// RejectReservation rechaza una reserva pendiente.
type RejectReservation struct {
	Reservations domain.ReservationRepository
	History      domain.StatusHistoryRepository
	Outbox       domain.OutboxRepository
	TxRunner     TxRunner
}

// RejectReservationInput es el input del usecase.
type RejectReservationInput struct {
	ReservationID string
	ActorID       string
}

// Execute valida y delega.
func (u RejectReservation) Execute(ctx context.Context, in RejectReservationInput) (entities.Reservation, error) {
	if err := policies.ValidateUUID(in.ReservationID); err != nil {
		return entities.Reservation{}, apperrors.BadRequest("reservation_id: " + err.Error())
	}

	reservation, err := u.Reservations.GetByID(ctx, in.ReservationID)
	if err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return entities.Reservation{}, apperrors.NotFound("reservation not found")
		}
		return entities.Reservation{}, apperrors.Internal("failed to load reservation")
	}
	if !policies.CanTransitionReservation(reservation.Status, entities.ReservationStatusRejected) {
		return entities.Reservation{}, apperrors.Conflict(
			"cannot reject reservation in status " + string(reservation.Status))
	}

	var rejected entities.Reservation
	run := func(txCtx context.Context) error {
		result, rejectErr := u.Reservations.Reject(txCtx, reservation.ID, reservation.Version, in.ActorID)
		if rejectErr != nil {
			return rejectErr
		}

		fromStatus := string(reservation.Status)
		toStatus := string(entities.ReservationStatusRejected)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			ReservationID: result.ID,
			FromStatus:    &fromStatus,
			ToStatus:      toStatus,
			ChangedBy:     &in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"reservation_id": result.ID,
			"common_area_id": result.CommonAreaID,
			"unit_id":        result.UnitID,
			"rejected_by":    in.ActorID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: result.ID,
			EventType:   entities.OutboxEventReservationRejected,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		rejected = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			return entities.Reservation{}, apperrors.Internal("failed to reject reservation")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			return entities.Reservation{}, apperrors.Internal("failed to reject reservation")
		}
	}
	return rejected, nil
}

// ---------------------------------------------------------------------------
// CheckinReservation
// ---------------------------------------------------------------------------

// CheckinReservation registra el checkin de una reserva (guarda).
type CheckinReservation struct {
	Reservations domain.ReservationRepository
	History      domain.StatusHistoryRepository
	Outbox       domain.OutboxRepository
	TxRunner     TxRunner
}

// CheckinReservationInput es el input del usecase.
type CheckinReservationInput struct {
	ReservationID string
	ActorID       string
}

// Execute valida y delega.
func (u CheckinReservation) Execute(ctx context.Context, in CheckinReservationInput) (entities.Reservation, error) {
	if err := policies.ValidateUUID(in.ReservationID); err != nil {
		return entities.Reservation{}, apperrors.BadRequest("reservation_id: " + err.Error())
	}

	reservation, err := u.Reservations.GetByID(ctx, in.ReservationID)
	if err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return entities.Reservation{}, apperrors.NotFound("reservation not found")
		}
		return entities.Reservation{}, apperrors.Internal("failed to load reservation")
	}
	if !policies.CanTransitionReservation(reservation.Status, entities.ReservationStatusConsumed) {
		return entities.Reservation{}, apperrors.Conflict(
			"cannot checkin reservation in status " + string(reservation.Status))
	}

	var consumed entities.Reservation
	run := func(txCtx context.Context) error {
		result, checkinErr := u.Reservations.Checkin(txCtx, reservation.ID, reservation.Version, in.ActorID)
		if checkinErr != nil {
			return checkinErr
		}

		fromStatus := string(reservation.Status)
		toStatus := string(entities.ReservationStatusConsumed)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			ReservationID: result.ID,
			FromStatus:    &fromStatus,
			ToStatus:      toStatus,
			ChangedBy:     &in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"reservation_id": result.ID,
			"common_area_id": result.CommonAreaID,
			"unit_id":        result.UnitID,
			"consumed_by":    in.ActorID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: result.ID,
			EventType:   entities.OutboxEventReservationConsumed,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		consumed = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			return entities.Reservation{}, apperrors.Internal("failed to checkin reservation")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Reservation{}, mapVersionConflict()
			}
			return entities.Reservation{}, apperrors.Internal("failed to checkin reservation")
		}
	}
	return consumed, nil
}

// ---------------------------------------------------------------------------
// ListReservations
// ---------------------------------------------------------------------------

// ListReservations lista todas las reservas.
type ListReservations struct {
	Reservations domain.ReservationRepository
}

// Execute delega al repo.
func (u ListReservations) Execute(ctx context.Context) ([]entities.Reservation, error) {
	out, err := u.Reservations.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list reservations")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// ListMyReservations
// ---------------------------------------------------------------------------

// ListMyReservations lista las reservas de la unidad del usuario.
type ListMyReservations struct {
	Reservations domain.ReservationRepository
}

// Execute delega al repo.
func (u ListMyReservations) Execute(ctx context.Context, unitID string) ([]entities.Reservation, error) {
	if err := policies.ValidateUUID(unitID); err != nil {
		return nil, apperrors.BadRequest("unit_id: " + err.Error())
	}
	out, err := u.Reservations.ListByUnit(ctx, unitID)
	if err != nil {
		return nil, apperrors.Internal("failed to list reservations")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"resource was modified by another request; reload and retry")
}
