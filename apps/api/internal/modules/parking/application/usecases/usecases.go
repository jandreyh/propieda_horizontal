// Package usecases orquesta la logica de aplicacion del modulo parking.
// Cada usecase recibe sus dependencias por inyeccion (interfaces) y NO
// conoce HTTP ni la base.
package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/parking/domain"
	"github.com/saas-ph/api/internal/modules/parking/domain/entities"
	"github.com/saas-ph/api/internal/modules/parking/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// CreateSpace
// ---------------------------------------------------------------------------

// CreateSpace crea un espacio de parqueadero nuevo en estado 'active'.
type CreateSpace struct {
	Spaces domain.SpaceRepository
}

// CreateSpaceInput es el input del usecase (sin tags JSON).
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

// Execute valida y delega al repo.
func (u CreateSpace) Execute(ctx context.Context, in CreateSpaceInput) (entities.ParkingSpace, error) {
	if err := policies.ValidateSpaceCode(in.Code); err != nil {
		return entities.ParkingSpace{}, apperrors.BadRequest("code: " + err.Error())
	}
	if !in.Type.IsValid() {
		return entities.ParkingSpace{}, apperrors.BadRequest("type: invalid space type")
	}
	space, err := u.Spaces.Create(ctx, domain.CreateSpaceInput{
		Code:        strings.TrimSpace(in.Code),
		Type:        in.Type,
		StructureID: in.StructureID,
		Level:       in.Level,
		Zone:        in.Zone,
		MonthlyFee:  in.MonthlyFee,
		IsVisitor:   in.IsVisitor,
		Notes:       in.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrSpaceCodeDuplicate) {
			return entities.ParkingSpace{}, apperrors.Conflict("space code already exists")
		}
		return entities.ParkingSpace{}, apperrors.Internal("failed to create space")
	}
	return space, nil
}

// ---------------------------------------------------------------------------
// UpdateSpace
// ---------------------------------------------------------------------------

// UpdateSpace actualiza un espacio existente con concurrencia optimista.
type UpdateSpace struct {
	Spaces domain.SpaceRepository
}

// UpdateSpaceInput es el input del usecase.
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

// Execute valida y delega al repo.
func (u UpdateSpace) Execute(ctx context.Context, in UpdateSpaceInput) (entities.ParkingSpace, error) {
	if err := policies.ValidateUUID(in.ID); err != nil {
		return entities.ParkingSpace{}, apperrors.BadRequest("id: " + err.Error())
	}
	if err := policies.ValidateSpaceCode(in.Code); err != nil {
		return entities.ParkingSpace{}, apperrors.BadRequest("code: " + err.Error())
	}
	if !in.Type.IsValid() {
		return entities.ParkingSpace{}, apperrors.BadRequest("type: invalid space type")
	}
	if !in.Status.IsValid() {
		return entities.ParkingSpace{}, apperrors.BadRequest("status: invalid space status")
	}
	space, err := u.Spaces.Update(ctx, domain.UpdateSpaceInput{
		ID:              in.ID,
		Code:            strings.TrimSpace(in.Code),
		Type:            in.Type,
		StructureID:     in.StructureID,
		Level:           in.Level,
		Zone:            in.Zone,
		MonthlyFee:      in.MonthlyFee,
		IsVisitor:       in.IsVisitor,
		Notes:           in.Notes,
		Status:          in.Status,
		ExpectedVersion: in.ExpectedVersion,
		ActorID:         in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSpaceNotFound):
			return entities.ParkingSpace{}, apperrors.NotFound("space not found")
		case errors.Is(err, domain.ErrVersionConflict):
			return entities.ParkingSpace{}, mapVersionConflict()
		case errors.Is(err, domain.ErrSpaceCodeDuplicate):
			return entities.ParkingSpace{}, apperrors.Conflict("space code already exists")
		default:
			return entities.ParkingSpace{}, apperrors.Internal("failed to update space")
		}
	}
	return space, nil
}

// ---------------------------------------------------------------------------
// ListSpaces
// ---------------------------------------------------------------------------

// ListSpaces lista los espacios activos.
type ListSpaces struct {
	Spaces domain.SpaceRepository
}

// Execute delega al repo.
func (u ListSpaces) Execute(ctx context.Context) ([]entities.ParkingSpace, error) {
	out, err := u.Spaces.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list spaces")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// AssignSpace
// ---------------------------------------------------------------------------

// AssignSpace asigna un espacio a una unidad. Si el espacio ya tiene
// asignacion activa, cierra la anterior y abre una nueva dentro de la
// misma TX.
type AssignSpace struct {
	Spaces      domain.SpaceRepository
	Assignments domain.AssignmentRepository
	History     domain.AssignmentHistoryRepository
	Outbox      domain.OutboxRepository
	TxRunner    TxRunner
	Now         func() time.Time
}

// AssignSpaceInput es el input del usecase.
type AssignSpaceInput struct {
	SpaceID   string
	UnitID    string
	VehicleID *string
	ActorID   string
	Notes     *string
}

// Execute valida y delega.
func (u AssignSpace) Execute(ctx context.Context, in AssignSpaceInput) (entities.ParkingAssignment, error) {
	if err := policies.ValidateUUID(in.SpaceID); err != nil {
		return entities.ParkingAssignment{}, apperrors.BadRequest("space_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.ParkingAssignment{}, apperrors.BadRequest("unit_id: " + err.Error())
	}

	space, err := u.Spaces.GetByID(ctx, in.SpaceID)
	if err != nil {
		if errors.Is(err, domain.ErrSpaceNotFound) {
			return entities.ParkingAssignment{}, apperrors.NotFound("space not found")
		}
		return entities.ParkingAssignment{}, apperrors.Internal("failed to load space")
	}
	if err := policies.CanAssignSpace(space); err != nil {
		return entities.ParkingAssignment{}, apperrors.BadRequest(err.Error())
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	actorID := in.ActorID

	var created entities.ParkingAssignment
	run := func(txCtx context.Context) error {
		// Close existing active assignment if any.
		existing, existErr := u.Assignments.GetActiveBySpaceID(txCtx, in.SpaceID)
		if existErr == nil {
			reason := "reassigned"
			closed, closeErr := u.Assignments.CloseAssignment(txCtx, existing.ID, now, existing.Version, actorID)
			if closeErr != nil {
				return closeErr
			}
			snapshot, _ := json.Marshal(map[string]any{
				"assignment_id": closed.ID,
				"unit_id":       closed.UnitID,
				"since_date":    closed.SinceDate,
				"until_date":    closed.UntilDate,
			})
			if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
				ParkingSpaceID:  closed.ParkingSpaceID,
				UnitID:          closed.UnitID,
				AssignmentID:    &closed.ID,
				SinceDate:       closed.SinceDate,
				UntilDate:       closed.UntilDate,
				ClosedReason:    &reason,
				SnapshotPayload: snapshot,
				RecordedBy:      &actorID,
			}); hErr != nil {
				return hErr
			}
		} else if !errors.Is(existErr, domain.ErrAssignmentNotFound) {
			return existErr
		}

		assignment, createErr := u.Assignments.Create(txCtx, domain.CreateAssignmentInput{
			ParkingSpaceID:   in.SpaceID,
			UnitID:           in.UnitID,
			VehicleID:        in.VehicleID,
			AssignedByUserID: &actorID,
			SinceDate:        now,
			Notes:            in.Notes,
		})
		if createErr != nil {
			if errors.Is(createErr, domain.ErrAssignmentAlreadyActive) {
				return domain.ErrAssignmentAlreadyActive
			}
			return createErr
		}

		payload, _ := json.Marshal(map[string]any{
			"assignment_id":    assignment.ID,
			"parking_space_id": assignment.ParkingSpaceID,
			"unit_id":          assignment.UnitID,
			"since_date":       assignment.SinceDate,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: assignment.ID,
			EventType:   entities.OutboxEventParkingAssigned,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		created = assignment
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrAssignmentAlreadyActive) {
				return entities.ParkingAssignment{}, apperrors.Conflict("space already has an active assignment")
			}
			return entities.ParkingAssignment{}, apperrors.Internal("failed to assign space")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrAssignmentAlreadyActive) {
				return entities.ParkingAssignment{}, apperrors.Conflict("space already has an active assignment")
			}
			return entities.ParkingAssignment{}, apperrors.Internal("failed to assign space")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// ReleaseAssignment
// ---------------------------------------------------------------------------

// ReleaseAssignment cierra una asignacion activa (establece until_date).
type ReleaseAssignment struct {
	Assignments domain.AssignmentRepository
	History     domain.AssignmentHistoryRepository
	Outbox      domain.OutboxRepository
	TxRunner    TxRunner
	Now         func() time.Time
}

// ReleaseAssignmentInput es el input del usecase.
type ReleaseAssignmentInput struct {
	AssignmentID string
	ActorID      string
	Reason       *string
}

// Execute valida y delega.
func (u ReleaseAssignment) Execute(ctx context.Context, in ReleaseAssignmentInput) (entities.ParkingAssignment, error) {
	if err := policies.ValidateUUID(in.AssignmentID); err != nil {
		return entities.ParkingAssignment{}, apperrors.BadRequest("assignment_id: " + err.Error())
	}

	assignment, err := u.Assignments.GetByID(ctx, in.AssignmentID)
	if err != nil {
		if errors.Is(err, domain.ErrAssignmentNotFound) {
			return entities.ParkingAssignment{}, apperrors.NotFound("assignment not found")
		}
		return entities.ParkingAssignment{}, apperrors.Internal("failed to load assignment")
	}
	if assignment.Status != entities.AssignmentStatusActive {
		return entities.ParkingAssignment{}, apperrors.Conflict("assignment is not active")
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var closed entities.ParkingAssignment
	run := func(txCtx context.Context) error {
		result, closeErr := u.Assignments.CloseAssignment(txCtx, assignment.ID, now, assignment.Version, in.ActorID)
		if closeErr != nil {
			return closeErr
		}

		reason := "released"
		if in.Reason != nil {
			reason = *in.Reason
		}
		snapshot, _ := json.Marshal(map[string]any{
			"assignment_id": result.ID,
			"unit_id":       result.UnitID,
			"since_date":    result.SinceDate,
			"until_date":    result.UntilDate,
		})
		if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
			ParkingSpaceID:  result.ParkingSpaceID,
			UnitID:          result.UnitID,
			AssignmentID:    &result.ID,
			SinceDate:       result.SinceDate,
			UntilDate:       result.UntilDate,
			ClosedReason:    &reason,
			SnapshotPayload: snapshot,
			RecordedBy:      &in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"assignment_id":    result.ID,
			"parking_space_id": result.ParkingSpaceID,
			"unit_id":          result.UnitID,
			"until_date":       result.UntilDate,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: result.ID,
			EventType:   entities.OutboxEventParkingReleased,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		closed = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.ParkingAssignment{}, mapVersionConflict()
			}
			return entities.ParkingAssignment{}, apperrors.Internal("failed to release assignment")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.ParkingAssignment{}, mapVersionConflict()
			}
			return entities.ParkingAssignment{}, apperrors.Internal("failed to release assignment")
		}
	}
	return closed, nil
}

// ---------------------------------------------------------------------------
// GetUnitParking
// ---------------------------------------------------------------------------

// GetUnitParking devuelve asignaciones activas y reservas de una unidad.
type GetUnitParking struct {
	Assignments  domain.AssignmentRepository
	Reservations domain.VisitorReservationRepository
}

// UnitParkingResult es el output del usecase.
type UnitParkingResult struct {
	Assignments  []entities.ParkingAssignment
	Reservations []entities.VisitorReservation
}

// Execute valida y delega.
func (u GetUnitParking) Execute(ctx context.Context, unitID string) (UnitParkingResult, error) {
	if err := policies.ValidateUUID(unitID); err != nil {
		return UnitParkingResult{}, apperrors.BadRequest("unit_id: " + err.Error())
	}

	assignments, err := u.Assignments.ListActiveByUnitID(ctx, unitID)
	if err != nil {
		return UnitParkingResult{}, apperrors.Internal("failed to list assignments")
	}
	reservations, err := u.Reservations.ListByUnit(ctx, unitID)
	if err != nil {
		return UnitParkingResult{}, apperrors.Internal("failed to list reservations")
	}

	return UnitParkingResult{
		Assignments:  assignments,
		Reservations: reservations,
	}, nil
}

// ---------------------------------------------------------------------------
// CreateVisitorReservation
// ---------------------------------------------------------------------------

// CreateVisitorReservation crea una reserva de visitante.
type CreateVisitorReservation struct {
	Spaces       domain.SpaceRepository
	Reservations domain.VisitorReservationRepository
	Outbox       domain.OutboxRepository
	TxRunner     TxRunner
	Now          func() time.Time
}

// CreateVisitorReservationInput es el input del usecase.
type CreateVisitorReservationInput struct {
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

// Execute valida y delega.
func (u CreateVisitorReservation) Execute(ctx context.Context, in CreateVisitorReservationInput) (entities.VisitorReservation, error) {
	if err := policies.ValidateUUID(in.ParkingSpaceID); err != nil {
		return entities.VisitorReservation{}, apperrors.BadRequest("parking_space_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.VisitorReservation{}, apperrors.BadRequest("unit_id: " + err.Error())
	}
	if strings.TrimSpace(in.VisitorName) == "" {
		return entities.VisitorReservation{}, apperrors.BadRequest("visitor_name is required")
	}

	space, err := u.Spaces.GetByID(ctx, in.ParkingSpaceID)
	if err != nil {
		if errors.Is(err, domain.ErrSpaceNotFound) {
			return entities.VisitorReservation{}, apperrors.NotFound("space not found")
		}
		return entities.VisitorReservation{}, apperrors.Internal("failed to load space")
	}
	if err := policies.CanReserveForVisitor(space); err != nil {
		return entities.VisitorReservation{}, apperrors.BadRequest(err.Error())
	}

	if err := policies.ValidateSlotDuration(in.SlotStartAt, in.SlotEndAt, 12); err != nil {
		return entities.VisitorReservation{}, apperrors.BadRequest(err.Error())
	}

	var created entities.VisitorReservation
	run := func(txCtx context.Context) error {
		reservation, createErr := u.Reservations.Create(txCtx, domain.CreateReservationInput{
			ParkingSpaceID:  in.ParkingSpaceID,
			UnitID:          in.UnitID,
			RequestedBy:     in.RequestedBy,
			VisitorName:     strings.TrimSpace(in.VisitorName),
			VisitorDocument: in.VisitorDocument,
			VehiclePlate:    in.VehiclePlate,
			SlotStartAt:     in.SlotStartAt,
			SlotEndAt:       in.SlotEndAt,
			IdempotencyKey:  in.IdempotencyKey,
		})
		if createErr != nil {
			return createErr
		}

		payload, _ := json.Marshal(map[string]any{
			"reservation_id":   reservation.ID,
			"parking_space_id": reservation.ParkingSpaceID,
			"unit_id":          reservation.UnitID,
			"visitor_name":     reservation.VisitorName,
			"slot_start_at":    reservation.SlotStartAt,
			"slot_end_at":      reservation.SlotEndAt,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: reservation.ID,
			EventType:   entities.OutboxEventVisitorReservationCreated,
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
				return entities.VisitorReservation{}, apperrors.Conflict("visitor reservation slot conflict")
			}
			return entities.VisitorReservation{}, apperrors.Internal("failed to create reservation")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrReservationSlotConflict) {
				return entities.VisitorReservation{}, apperrors.Conflict("visitor reservation slot conflict")
			}
			return entities.VisitorReservation{}, apperrors.Internal("failed to create reservation")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// CancelVisitorReservation
// ---------------------------------------------------------------------------

// CancelVisitorReservation cancela una reserva de visitante.
type CancelVisitorReservation struct {
	Reservations domain.VisitorReservationRepository
}

// CancelVisitorReservationInput es el input del usecase.
type CancelVisitorReservationInput struct {
	ReservationID string
	ActorID       string
}

// Execute valida y delega.
func (u CancelVisitorReservation) Execute(ctx context.Context, in CancelVisitorReservationInput) (entities.VisitorReservation, error) {
	if err := policies.ValidateUUID(in.ReservationID); err != nil {
		return entities.VisitorReservation{}, apperrors.BadRequest("reservation_id: " + err.Error())
	}

	reservation, err := u.Reservations.GetByID(ctx, in.ReservationID)
	if err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return entities.VisitorReservation{}, apperrors.NotFound("reservation not found")
		}
		return entities.VisitorReservation{}, apperrors.Internal("failed to load reservation")
	}
	if !policies.CanTransitionReservation(reservation.Status, entities.ReservationStatusCancelled) {
		return entities.VisitorReservation{}, apperrors.Conflict(
			"cannot cancel reservation in status " + string(reservation.Status))
	}

	cancelled, err := u.Reservations.Cancel(ctx, reservation.ID, reservation.Version, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrVersionConflict) {
			return entities.VisitorReservation{}, mapVersionConflict()
		}
		return entities.VisitorReservation{}, apperrors.Internal("failed to cancel reservation")
	}
	return cancelled, nil
}

// ---------------------------------------------------------------------------
// ListVisitorReservations
// ---------------------------------------------------------------------------

// ListVisitorReservations lista reservas de visitante para una fecha.
type ListVisitorReservations struct {
	Reservations domain.VisitorReservationRepository
}

// Execute valida y delega.
func (u ListVisitorReservations) Execute(ctx context.Context, start, end time.Time) ([]entities.VisitorReservation, error) {
	out, err := u.Reservations.ListByDate(ctx, start, end)
	if err != nil {
		return nil, apperrors.Internal("failed to list reservations")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// RunLottery
// ---------------------------------------------------------------------------

// RunLottery ejecuta un sorteo determinista de parqueaderos.
type RunLottery struct {
	Spaces    domain.SpaceRepository
	Lotteries domain.LotteryRunRepository
	Results   domain.LotteryResultRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// RunLotteryInput es el input del usecase.
type RunLotteryInput struct {
	Name          string
	Seed          string
	SpaceIDs      []string
	EligibleUnits []string
	ActorID       string
}

// RunLotteryResult es el output del usecase.
type RunLotteryResult struct {
	Run     entities.LotteryRun
	Results []entities.LotteryResult
}

// Execute valida, ejecuta el sorteo determinista, y persiste resultados.
func (u RunLottery) Execute(ctx context.Context, in RunLotteryInput) (RunLotteryResult, error) {
	if strings.TrimSpace(in.Name) == "" {
		return RunLotteryResult{}, apperrors.BadRequest("name is required")
	}
	if strings.TrimSpace(in.Seed) == "" {
		return RunLotteryResult{}, apperrors.BadRequest("seed is required")
	}
	if len(in.SpaceIDs) == 0 {
		return RunLotteryResult{}, apperrors.BadRequest("space_ids must not be empty")
	}
	if len(in.EligibleUnits) == 0 {
		return RunLotteryResult{}, apperrors.BadRequest("eligible_units must not be empty")
	}

	for _, sid := range in.SpaceIDs {
		if err := policies.ValidateUUID(sid); err != nil {
			return RunLotteryResult{}, apperrors.BadRequest("space_ids: " + err.Error())
		}
	}
	for _, uid := range in.EligibleUnits {
		if err := policies.ValidateUUID(uid); err != nil {
			return RunLotteryResult{}, apperrors.BadRequest("eligible_units: " + err.Error())
		}
	}

	// Shuffle determinista.
	shuffled := policies.ShuffleDeterministic(in.Seed, in.EligibleUnits)

	seedHash := sha256.Sum256([]byte(in.Seed))
	seedHashHex := hex.EncodeToString(seedHash[:])

	criteria, _ := json.Marshal(map[string]any{
		"space_ids":      in.SpaceIDs,
		"eligible_units": in.EligibleUnits,
	})

	var result RunLotteryResult
	run := func(txCtx context.Context) error {
		lotteryRun, createErr := u.Lotteries.Create(txCtx, domain.CreateLotteryRunInput{
			Name:       strings.TrimSpace(in.Name),
			SeedHash:   seedHashHex,
			Criteria:   criteria,
			ExecutedBy: in.ActorID,
		})
		if createErr != nil {
			return createErr
		}

		inputs := make([]domain.CreateLotteryResultInput, 0, len(shuffled))
		for i, unitID := range shuffled {
			var spaceID *string
			if i < len(in.SpaceIDs) {
				sid := in.SpaceIDs[i]
				spaceID = &sid
			}
			status := entities.LotteryResultStatusAllocated
			if spaceID == nil {
				status = entities.LotteryResultStatusWaitlist
			}
			inputs = append(inputs, domain.CreateLotteryResultInput{
				LotteryRunID:   lotteryRun.ID,
				UnitID:         unitID,
				ParkingSpaceID: spaceID,
				Position:       int32(i + 1),
				Status:         status,
			})
		}

		results, batchErr := u.Results.CreateBatch(txCtx, inputs)
		if batchErr != nil {
			return batchErr
		}

		payload, _ := json.Marshal(map[string]any{
			"lottery_run_id": lotteryRun.ID,
			"name":           lotteryRun.Name,
			"total_results":  len(results),
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: lotteryRun.ID,
			EventType:   entities.OutboxEventLotteryPublished,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		result = RunLotteryResult{Run: lotteryRun, Results: results}
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			return RunLotteryResult{}, apperrors.Internal("failed to run lottery")
		}
	} else {
		if err := run(ctx); err != nil {
			return RunLotteryResult{}, apperrors.Internal("failed to run lottery")
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// GetLotteryResults
// ---------------------------------------------------------------------------

// GetLotteryResults devuelve un sorteo y sus resultados.
type GetLotteryResults struct {
	Lotteries domain.LotteryRunRepository
	Results   domain.LotteryResultRepository
}

// Execute valida y delega.
func (u GetLotteryResults) Execute(ctx context.Context, runID string) (RunLotteryResult, error) {
	if err := policies.ValidateUUID(runID); err != nil {
		return RunLotteryResult{}, apperrors.BadRequest("lottery_run_id: " + err.Error())
	}

	run, err := u.Lotteries.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, domain.ErrLotteryNotFound) {
			return RunLotteryResult{}, apperrors.NotFound("lottery run not found")
		}
		return RunLotteryResult{}, apperrors.Internal("failed to load lottery run")
	}

	results, err := u.Results.ListByRunID(ctx, runID)
	if err != nil {
		return RunLotteryResult{}, apperrors.Internal("failed to load lottery results")
	}

	return RunLotteryResult{Run: run, Results: results}, nil
}

// ---------------------------------------------------------------------------
// GuardParkingToday
// ---------------------------------------------------------------------------

// GuardParkingEntry es una fila de la vista del guarda.
type GuardParkingEntry struct {
	SpaceCode    string
	SpaceType    entities.SpaceType
	UnitID       *string
	VehiclePlate *string
	VisitorName  *string
	SlotStartAt  *time.Time
	SlotEndAt    *time.Time
	EntryType    string // "assignment" o "visitor"
}

// GuardParkingToday construye la vista del guarda para el dia.
type GuardParkingToday struct {
	Spaces       domain.SpaceRepository
	Assignments  domain.AssignmentRepository
	Reservations domain.VisitorReservationRepository
	Now          func() time.Time
}

// Execute construye la vista.
func (u GuardParkingToday) Execute(ctx context.Context) ([]GuardParkingEntry, error) {
	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	spaces, err := u.Spaces.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list spaces")
	}

	// Build space map by ID.
	spaceMap := make(map[string]entities.ParkingSpace, len(spaces))
	for _, s := range spaces {
		spaceMap[s.ID] = s
	}

	var entries []GuardParkingEntry

	// Add assignment entries.
	for _, space := range spaces {
		if space.IsVisitor {
			continue
		}
		assignment, aErr := u.Assignments.GetActiveBySpaceID(ctx, space.ID)
		if aErr != nil {
			if errors.Is(aErr, domain.ErrAssignmentNotFound) {
				continue
			}
			return nil, apperrors.Internal("failed to load assignment")
		}
		entries = append(entries, GuardParkingEntry{
			SpaceCode: space.Code,
			SpaceType: space.Type,
			UnitID:    &assignment.UnitID,
			EntryType: "assignment",
		})
	}

	// Add visitor reservation entries for today.
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	reservations, rErr := u.Reservations.ListByDate(ctx, startOfDay, endOfDay)
	if rErr != nil {
		return nil, apperrors.Internal("failed to list reservations")
	}

	for _, r := range reservations {
		if r.Status != entities.ReservationStatusConfirmed && r.Status != entities.ReservationStatusPending {
			continue
		}
		space, ok := spaceMap[r.ParkingSpaceID]
		if !ok {
			continue
		}
		slotStart := r.SlotStartAt
		slotEnd := r.SlotEndAt
		entries = append(entries, GuardParkingEntry{
			SpaceCode:    space.Code,
			SpaceType:    space.Type,
			UnitID:       &r.UnitID,
			VehiclePlate: r.VehiclePlate,
			VisitorName:  &r.VisitorName,
			SlotStartAt:  &slotStart,
			SlotEndAt:    &slotEnd,
			EntryType:    "visitor",
		})
	}

	return entries, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"resource was modified by another request; reload and retry")
}
