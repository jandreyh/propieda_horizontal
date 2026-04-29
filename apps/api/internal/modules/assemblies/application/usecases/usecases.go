// Package usecases orquesta la logica de aplicacion del modulo assemblies.
// Cada usecase recibe sus dependencias por inyeccion (interfaces) y NO
// conoce HTTP ni la base.
package usecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/assemblies/domain"
	"github.com/saas-ph/api/internal/modules/assemblies/domain/entities"
	"github.com/saas-ph/api/internal/modules/assemblies/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// CreateAssembly
// ---------------------------------------------------------------------------

// CreateAssembly crea una asamblea nueva en estado 'draft'.
type CreateAssembly struct {
	Assemblies domain.AssemblyRepository
	Outbox     domain.OutboxRepository
}

// CreateAssemblyInput es el input del usecase (sin tags JSON).
type CreateAssemblyInput struct {
	Name              string
	AssemblyType      entities.AssemblyType
	ScheduledAt       time.Time
	VotingMode        entities.VotingMode
	QuorumRequiredPct float64
	Location          *string
	Notes             *string
	ActorID           string
}

// Execute valida y delega al repo.
func (u CreateAssembly) Execute(ctx context.Context, in CreateAssemblyInput) (entities.Assembly, error) {
	if strings.TrimSpace(in.Name) == "" {
		return entities.Assembly{}, apperrors.BadRequest("name is required")
	}
	if !in.AssemblyType.IsValid() {
		return entities.Assembly{}, apperrors.BadRequest("assembly_type: invalid type")
	}
	if !in.VotingMode.IsValid() {
		return entities.Assembly{}, apperrors.BadRequest("voting_mode: invalid mode")
	}
	if in.QuorumRequiredPct <= 0 || in.QuorumRequiredPct > 1 {
		return entities.Assembly{}, apperrors.BadRequest("quorum_required_pct must be between 0 (exclusive) and 1 (inclusive)")
	}

	assembly, err := u.Assemblies.Create(ctx, domain.CreateAssemblyInput{
		Name:              strings.TrimSpace(in.Name),
		AssemblyType:      in.AssemblyType,
		ScheduledAt:       in.ScheduledAt,
		VotingMode:        in.VotingMode,
		QuorumRequiredPct: in.QuorumRequiredPct,
		Location:          in.Location,
		Notes:             in.Notes,
		ActorID:           in.ActorID,
	})
	if err != nil {
		return entities.Assembly{}, apperrors.Internal("failed to create assembly")
	}
	return assembly, nil
}

// ---------------------------------------------------------------------------
// GetAssembly
// ---------------------------------------------------------------------------

// GetAssembly devuelve una asamblea por ID.
type GetAssembly struct {
	Assemblies domain.AssemblyRepository
}

// Execute valida y delega.
func (u GetAssembly) Execute(ctx context.Context, id string) (entities.Assembly, error) {
	if err := policies.ValidateUUID(id); err != nil {
		return entities.Assembly{}, apperrors.BadRequest("id: " + err.Error())
	}
	assembly, err := u.Assemblies.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.Assembly{}, apperrors.NotFound("assembly not found")
		}
		return entities.Assembly{}, apperrors.Internal("failed to load assembly")
	}
	return assembly, nil
}

// ---------------------------------------------------------------------------
// ListAssemblies
// ---------------------------------------------------------------------------

// ListAssemblies lista las asambleas.
type ListAssemblies struct {
	Assemblies domain.AssemblyRepository
}

// Execute delega al repo.
func (u ListAssemblies) Execute(ctx context.Context) ([]entities.Assembly, error) {
	out, err := u.Assemblies.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list assemblies")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// PublishCall
// ---------------------------------------------------------------------------

// PublishCall publica una convocatoria formal y transiciona la asamblea
// a estado 'called'.
type PublishCall struct {
	Assemblies domain.AssemblyRepository
	Calls      domain.CallRepository
	Outbox     domain.OutboxRepository
	TxRunner   TxRunner
}

// PublishCallInput es el input del usecase.
type PublishCallInput struct {
	AssemblyID string
	Channels   []byte
	Agenda     []byte
	BodyMD     *string
	ActorID    string
}

// Execute valida y delega.
func (u PublishCall) Execute(ctx context.Context, in PublishCallInput) (entities.AssemblyCall, error) {
	if err := policies.ValidateUUID(in.AssemblyID); err != nil {
		return entities.AssemblyCall{}, apperrors.BadRequest("assembly_id: " + err.Error())
	}

	assembly, err := u.Assemblies.GetByID(ctx, in.AssemblyID)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.AssemblyCall{}, apperrors.NotFound("assembly not found")
		}
		return entities.AssemblyCall{}, apperrors.Internal("failed to load assembly")
	}

	if !policies.CanTransitionAssembly(assembly.Status, entities.AssemblyStatusCalled) {
		return entities.AssemblyCall{}, apperrors.Conflict(
			"cannot transition assembly from " + string(assembly.Status) + " to called")
	}

	var created entities.AssemblyCall
	run := func(txCtx context.Context) error {
		call, createErr := u.Calls.Create(txCtx, domain.CreateCallInput{
			AssemblyID:  in.AssemblyID,
			Channels:    in.Channels,
			Agenda:      in.Agenda,
			BodyMD:      in.BodyMD,
			PublishedBy: in.ActorID,
		})
		if createErr != nil {
			return createErr
		}

		_, updateErr := u.Assemblies.UpdateStatus(txCtx, domain.UpdateAssemblyInput{
			ID:              in.AssemblyID,
			Status:          entities.AssemblyStatusCalled,
			ExpectedVersion: assembly.Version,
			ActorID:         in.ActorID,
		})
		if updateErr != nil {
			return updateErr
		}

		payload, _ := json.Marshal(map[string]any{
			"assembly_id": in.AssemblyID,
			"call_id":     call.ID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: in.AssemblyID,
			EventType:   entities.OutboxEventAssemblyCalled,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		created = call
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.AssemblyCall{}, mapVersionConflict()
			}
			return entities.AssemblyCall{}, apperrors.Internal("failed to publish call")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.AssemblyCall{}, mapVersionConflict()
			}
			return entities.AssemblyCall{}, apperrors.Internal("failed to publish call")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// StartAssembly
// ---------------------------------------------------------------------------

// StartAssembly inicia una asamblea (in_progress).
type StartAssembly struct {
	Assemblies domain.AssemblyRepository
	Outbox     domain.OutboxRepository
	TxRunner   TxRunner
	Now        func() time.Time
}

// Execute valida y delega.
func (u StartAssembly) Execute(ctx context.Context, assemblyID, actorID string) (entities.Assembly, error) {
	if err := policies.ValidateUUID(assemblyID); err != nil {
		return entities.Assembly{}, apperrors.BadRequest("assembly_id: " + err.Error())
	}

	assembly, err := u.Assemblies.GetByID(ctx, assemblyID)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.Assembly{}, apperrors.NotFound("assembly not found")
		}
		return entities.Assembly{}, apperrors.Internal("failed to load assembly")
	}
	if err := policies.CanAssemblyBeStarted(assembly); err != nil {
		return entities.Assembly{}, apperrors.Conflict(err.Error())
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Assembly
	run := func(txCtx context.Context) error {
		result, updateErr := u.Assemblies.UpdateStatus(txCtx, domain.UpdateAssemblyInput{
			ID:              assemblyID,
			Status:          entities.AssemblyStatusInProgress,
			StartedAt:       &now,
			ExpectedVersion: assembly.Version,
			ActorID:         actorID,
		})
		if updateErr != nil {
			return updateErr
		}

		payload, _ := json.Marshal(map[string]any{
			"assembly_id": assemblyID,
			"started_at":  now,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: assemblyID,
			EventType:   entities.OutboxEventAssemblyStarted,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Assembly{}, mapVersionConflict()
			}
			return entities.Assembly{}, apperrors.Internal("failed to start assembly")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Assembly{}, mapVersionConflict()
			}
			return entities.Assembly{}, apperrors.Internal("failed to start assembly")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// CloseAssembly
// ---------------------------------------------------------------------------

// CloseAssembly cierra una asamblea.
type CloseAssembly struct {
	Assemblies domain.AssemblyRepository
	Outbox     domain.OutboxRepository
	TxRunner   TxRunner
	Now        func() time.Time
}

// Execute valida y delega.
func (u CloseAssembly) Execute(ctx context.Context, assemblyID, actorID string) (entities.Assembly, error) {
	if err := policies.ValidateUUID(assemblyID); err != nil {
		return entities.Assembly{}, apperrors.BadRequest("assembly_id: " + err.Error())
	}

	assembly, err := u.Assemblies.GetByID(ctx, assemblyID)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.Assembly{}, apperrors.NotFound("assembly not found")
		}
		return entities.Assembly{}, apperrors.Internal("failed to load assembly")
	}
	if err := policies.CanAssemblyBeClosed(assembly); err != nil {
		return entities.Assembly{}, apperrors.Conflict(err.Error())
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Assembly
	run := func(txCtx context.Context) error {
		result, updateErr := u.Assemblies.UpdateStatus(txCtx, domain.UpdateAssemblyInput{
			ID:              assemblyID,
			Status:          entities.AssemblyStatusClosed,
			ClosedAt:        &now,
			ExpectedVersion: assembly.Version,
			ActorID:         actorID,
		})
		if updateErr != nil {
			return updateErr
		}

		payload, _ := json.Marshal(map[string]any{
			"assembly_id": assemblyID,
			"closed_at":   now,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: assemblyID,
			EventType:   entities.OutboxEventAssemblyClosed,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Assembly{}, mapVersionConflict()
			}
			return entities.Assembly{}, apperrors.Internal("failed to close assembly")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Assembly{}, mapVersionConflict()
			}
			return entities.Assembly{}, apperrors.Internal("failed to close assembly")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// RegisterAttendance
// ---------------------------------------------------------------------------

// RegisterAttendance registra la asistencia de una unidad.
type RegisterAttendance struct {
	Assemblies  domain.AssemblyRepository
	Attendances domain.AttendanceRepository
}

// RegisterAttendanceInput es el input del usecase.
type RegisterAttendanceInput struct {
	AssemblyID          string
	UnitID              string
	AttendeeUserID      *string
	RepresentedByUserID *string
	CoefficientAtEvent  float64
	IsRemote            bool
	HasVotingRight      bool
	Notes               *string
	ActorID             string
}

// Execute valida y delega.
func (u RegisterAttendance) Execute(ctx context.Context, in RegisterAttendanceInput) (entities.AssemblyAttendance, error) {
	if err := policies.ValidateUUID(in.AssemblyID); err != nil {
		return entities.AssemblyAttendance{}, apperrors.BadRequest("assembly_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.AssemblyAttendance{}, apperrors.BadRequest("unit_id: " + err.Error())
	}

	assembly, err := u.Assemblies.GetByID(ctx, in.AssemblyID)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.AssemblyAttendance{}, apperrors.NotFound("assembly not found")
		}
		return entities.AssemblyAttendance{}, apperrors.Internal("failed to load assembly")
	}
	if assembly.Status != entities.AssemblyStatusInProgress && assembly.Status != entities.AssemblyStatusCalled {
		return entities.AssemblyAttendance{}, apperrors.Conflict(
			"assembly must be called or in_progress to register attendance")
	}

	attendance, err := u.Attendances.Create(ctx, domain.CreateAttendanceInput{
		AssemblyID:          in.AssemblyID,
		UnitID:              in.UnitID,
		AttendeeUserID:      in.AttendeeUserID,
		RepresentedByUserID: in.RepresentedByUserID,
		CoefficientAtEvent:  in.CoefficientAtEvent,
		IsRemote:            in.IsRemote,
		HasVotingRight:      in.HasVotingRight,
		Notes:               in.Notes,
		ActorID:             in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrAttendanceDuplicate) {
			return entities.AssemblyAttendance{}, apperrors.Conflict("attendance already registered for this unit")
		}
		return entities.AssemblyAttendance{}, apperrors.Internal("failed to register attendance")
	}
	return attendance, nil
}

// ---------------------------------------------------------------------------
// RegisterProxy
// ---------------------------------------------------------------------------

// RegisterProxy registra un poder (proxy) para una asamblea.
type RegisterProxy struct {
	Assemblies domain.AssemblyRepository
	Proxies    domain.ProxyRepository
}

// RegisterProxyInput es el input del usecase.
type RegisterProxyInput struct {
	AssemblyID    string
	GrantorUserID string
	ProxyUserID   string
	UnitID        string
	DocumentURL   *string
	DocumentHash  *string
	MaxProxies    int
	ActorID       string
}

// Execute valida y delega.
func (u RegisterProxy) Execute(ctx context.Context, in RegisterProxyInput) (entities.AssemblyProxy, error) {
	if err := policies.ValidateUUID(in.AssemblyID); err != nil {
		return entities.AssemblyProxy{}, apperrors.BadRequest("assembly_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.GrantorUserID); err != nil {
		return entities.AssemblyProxy{}, apperrors.BadRequest("grantor_user_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.ProxyUserID); err != nil {
		return entities.AssemblyProxy{}, apperrors.BadRequest("proxy_user_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.AssemblyProxy{}, apperrors.BadRequest("unit_id: " + err.Error())
	}

	assembly, err := u.Assemblies.GetByID(ctx, in.AssemblyID)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.AssemblyProxy{}, apperrors.NotFound("assembly not found")
		}
		return entities.AssemblyProxy{}, apperrors.Internal("failed to load assembly")
	}
	if assembly.Status == entities.AssemblyStatusClosed ||
		assembly.Status == entities.AssemblyStatusArchived ||
		assembly.Status == entities.AssemblyStatusQuorumFailed {
		return entities.AssemblyProxy{}, apperrors.Conflict("assembly is not accepting proxies")
	}

	// Check max proxies per attendee.
	maxProxies := in.MaxProxies
	if maxProxies <= 0 {
		maxProxies = 1 // default
	}
	count, err := u.Proxies.CountByProxyUser(ctx, in.AssemblyID, in.ProxyUserID)
	if err != nil {
		return entities.AssemblyProxy{}, apperrors.Internal("failed to count proxies")
	}
	if err := policies.ValidateMaxProxies(count, maxProxies); err != nil {
		return entities.AssemblyProxy{}, apperrors.Conflict(err.Error())
	}

	proxy, err := u.Proxies.Create(ctx, domain.CreateProxyInput{
		AssemblyID:    in.AssemblyID,
		GrantorUserID: in.GrantorUserID,
		ProxyUserID:   in.ProxyUserID,
		UnitID:        in.UnitID,
		DocumentURL:   in.DocumentURL,
		DocumentHash:  in.DocumentHash,
		ActorID:       in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrProxyDuplicate) {
			return entities.AssemblyProxy{}, apperrors.Conflict("proxy already exists for this unit")
		}
		return entities.AssemblyProxy{}, apperrors.Internal("failed to register proxy")
	}
	return proxy, nil
}

// ---------------------------------------------------------------------------
// CreateMotion
// ---------------------------------------------------------------------------

// CreateMotion crea una mocion dentro de una asamblea.
type CreateMotion struct {
	Assemblies domain.AssemblyRepository
	Motions    domain.MotionRepository
}

// CreateMotionInput es el input del usecase.
type CreateMotionInput struct {
	AssemblyID   string
	Title        string
	Description  *string
	DecisionType entities.DecisionType
	VotingMethod entities.VotingMethod
	Options      []byte
	ActorID      string
}

// Execute valida y delega.
func (u CreateMotion) Execute(ctx context.Context, in CreateMotionInput) (entities.AssemblyMotion, error) {
	if err := policies.ValidateUUID(in.AssemblyID); err != nil {
		return entities.AssemblyMotion{}, apperrors.BadRequest("assembly_id: " + err.Error())
	}
	if strings.TrimSpace(in.Title) == "" {
		return entities.AssemblyMotion{}, apperrors.BadRequest("title is required")
	}
	if !in.DecisionType.IsValid() {
		return entities.AssemblyMotion{}, apperrors.BadRequest("decision_type: invalid type")
	}
	if !in.VotingMethod.IsValid() {
		return entities.AssemblyMotion{}, apperrors.BadRequest("voting_method: invalid method")
	}

	assembly, err := u.Assemblies.GetByID(ctx, in.AssemblyID)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.AssemblyMotion{}, apperrors.NotFound("assembly not found")
		}
		return entities.AssemblyMotion{}, apperrors.Internal("failed to load assembly")
	}
	if assembly.Status != entities.AssemblyStatusInProgress &&
		assembly.Status != entities.AssemblyStatusCalled &&
		assembly.Status != entities.AssemblyStatusDraft {
		return entities.AssemblyMotion{}, apperrors.Conflict(
			"assembly is not accepting new motions")
	}

	motion, err := u.Motions.Create(ctx, domain.CreateMotionInput{
		AssemblyID:   in.AssemblyID,
		Title:        strings.TrimSpace(in.Title),
		Description:  in.Description,
		DecisionType: in.DecisionType,
		VotingMethod: in.VotingMethod,
		Options:      in.Options,
		ActorID:      in.ActorID,
	})
	if err != nil {
		return entities.AssemblyMotion{}, apperrors.Internal("failed to create motion")
	}
	return motion, nil
}

// ---------------------------------------------------------------------------
// OpenVoting
// ---------------------------------------------------------------------------

// OpenVoting abre la votacion de una mocion.
type OpenVoting struct {
	Motions  domain.MotionRepository
	Outbox   domain.OutboxRepository
	TxRunner TxRunner
	Now      func() time.Time
}

// Execute valida y delega.
func (u OpenVoting) Execute(ctx context.Context, motionID, actorID string) (entities.AssemblyMotion, error) {
	if err := policies.ValidateUUID(motionID); err != nil {
		return entities.AssemblyMotion{}, apperrors.BadRequest("motion_id: " + err.Error())
	}

	motion, err := u.Motions.GetByID(ctx, motionID)
	if err != nil {
		if errors.Is(err, domain.ErrMotionNotFound) {
			return entities.AssemblyMotion{}, apperrors.NotFound("motion not found")
		}
		return entities.AssemblyMotion{}, apperrors.Internal("failed to load motion")
	}
	if !policies.CanTransitionMotion(motion.Status, entities.MotionStatusOpen) {
		return entities.AssemblyMotion{}, apperrors.Conflict(
			"cannot open voting on motion in status " + string(motion.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.AssemblyMotion
	run := func(txCtx context.Context) error {
		result, updateErr := u.Motions.UpdateStatus(txCtx, domain.UpdateMotionStatusInput{
			ID:              motionID,
			Status:          entities.MotionStatusOpen,
			OpensAt:         &now,
			ExpectedVersion: motion.Version,
			ActorID:         actorID,
		})
		if updateErr != nil {
			return updateErr
		}

		payload, _ := json.Marshal(map[string]any{
			"motion_id":   motionID,
			"assembly_id": motion.AssemblyID,
			"opens_at":    now,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: motionID,
			EventType:   entities.OutboxEventMotionOpened,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.AssemblyMotion{}, mapVersionConflict()
			}
			return entities.AssemblyMotion{}, apperrors.Internal("failed to open voting")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.AssemblyMotion{}, mapVersionConflict()
			}
			return entities.AssemblyMotion{}, apperrors.Internal("failed to open voting")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// CloseVoting
// ---------------------------------------------------------------------------

// CloseVoting cierra la votacion de una mocion.
type CloseVoting struct {
	Motions  domain.MotionRepository
	Votes    domain.VoteRepository
	Outbox   domain.OutboxRepository
	TxRunner TxRunner
	Now      func() time.Time
}

// Execute valida y delega.
func (u CloseVoting) Execute(ctx context.Context, motionID, actorID string) (entities.AssemblyMotion, error) {
	if err := policies.ValidateUUID(motionID); err != nil {
		return entities.AssemblyMotion{}, apperrors.BadRequest("motion_id: " + err.Error())
	}

	motion, err := u.Motions.GetByID(ctx, motionID)
	if err != nil {
		if errors.Is(err, domain.ErrMotionNotFound) {
			return entities.AssemblyMotion{}, apperrors.NotFound("motion not found")
		}
		return entities.AssemblyMotion{}, apperrors.Internal("failed to load motion")
	}
	if !policies.CanTransitionMotion(motion.Status, entities.MotionStatusClosed) {
		return entities.AssemblyMotion{}, apperrors.Conflict(
			"cannot close voting on motion in status " + string(motion.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	// Tally results.
	votes, err := u.Votes.ListByMotionID(ctx, motionID)
	if err != nil {
		return entities.AssemblyMotion{}, apperrors.Internal("failed to load votes")
	}
	results := tallyVotes(votes)
	resultsJSON, _ := json.Marshal(results)

	var updated entities.AssemblyMotion
	run := func(txCtx context.Context) error {
		result, updateErr := u.Motions.UpdateStatus(txCtx, domain.UpdateMotionStatusInput{
			ID:              motionID,
			Status:          entities.MotionStatusClosed,
			ClosesAt:        &now,
			Results:         resultsJSON,
			ExpectedVersion: motion.Version,
			ActorID:         actorID,
		})
		if updateErr != nil {
			return updateErr
		}

		payload, _ := json.Marshal(map[string]any{
			"motion_id":   motionID,
			"assembly_id": motion.AssemblyID,
			"closes_at":   now,
			"results":     results,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: motionID,
			EventType:   entities.OutboxEventMotionClosed,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.AssemblyMotion{}, mapVersionConflict()
			}
			return entities.AssemblyMotion{}, apperrors.Internal("failed to close voting")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.AssemblyMotion{}, mapVersionConflict()
			}
			return entities.AssemblyMotion{}, apperrors.Internal("failed to close voting")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// CastVote
// ---------------------------------------------------------------------------

// CastVote emite un voto en una mocion. Utiliza TxRunner para
// atomicamente crear el voto, la evidencia y el evento outbox, y
// mantener la cadena de hashes.
type CastVote struct {
	Motions  domain.MotionRepository
	Votes    domain.VoteRepository
	Evidence domain.VoteEvidenceRepository
	Outbox   domain.OutboxRepository
	TxRunner TxRunner
	Now      func() time.Time
}

// CastVoteInput es el input del usecase.
type CastVoteInput struct {
	MotionID        string
	VoterUserID     string
	UnitID          string
	CoefficientUsed float64
	Option          string
	IsProxyVote     bool
	ClientIP        *string
	UserAgent       *string
	NTPOffsetMS     *int32
	ActorID         string
}

// Execute valida y delega.
func (u CastVote) Execute(ctx context.Context, in CastVoteInput) (entities.Vote, error) {
	if err := policies.ValidateUUID(in.MotionID); err != nil {
		return entities.Vote{}, apperrors.BadRequest("motion_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.VoterUserID); err != nil {
		return entities.Vote{}, apperrors.BadRequest("voter_user_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.UnitID); err != nil {
		return entities.Vote{}, apperrors.BadRequest("unit_id: " + err.Error())
	}
	if strings.TrimSpace(in.Option) == "" {
		return entities.Vote{}, apperrors.BadRequest("option is required")
	}

	motion, err := u.Motions.GetByID(ctx, in.MotionID)
	if err != nil {
		if errors.Is(err, domain.ErrMotionNotFound) {
			return entities.Vote{}, apperrors.NotFound("motion not found")
		}
		return entities.Vote{}, apperrors.Internal("failed to load motion")
	}
	if err := policies.CanVoteOnMotion(motion); err != nil {
		return entities.Vote{}, apperrors.Conflict(err.Error())
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	// Generate nonce.
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return entities.Vote{}, apperrors.Internal("failed to generate nonce")
	}
	nonce := hex.EncodeToString(nonceBytes)

	var created entities.Vote
	run := func(txCtx context.Context) error {
		// Get the last vote hash for chain continuity.
		prevHash, hashErr := u.Votes.GetLastVoteHash(txCtx, in.MotionID)
		if hashErr != nil {
			return hashErr
		}
		prevHashStr := ""
		if prevHash != nil {
			prevHashStr = *prevHash
		}

		// Void existing active vote for this unit if any.
		existing, existErr := u.Votes.GetActiveByMotionAndUnit(txCtx, in.MotionID, in.UnitID)
		if existErr == nil {
			if voidErr := u.Votes.VoidVote(txCtx, existing.ID, existing.Version, in.ActorID); voidErr != nil {
				return voidErr
			}
		} else if !errors.Is(existErr, domain.ErrVoteNotFound) {
			return existErr
		}

		// Compute hash chain.
		voteHash := policies.ComputeVoteHash(
			prevHashStr, in.MotionID, in.VoterUserID,
			in.Option, now, nonce,
		)

		vote, createErr := u.Votes.Create(txCtx, domain.CreateVoteInput{
			MotionID:        in.MotionID,
			VoterUserID:     in.VoterUserID,
			UnitID:          in.UnitID,
			CoefficientUsed: in.CoefficientUsed,
			Option:          in.Option,
			CastAt:          now,
			PrevVoteHash:    prevHash,
			VoteHash:        voteHash,
			Nonce:           nonce,
			IsProxyVote:     in.IsProxyVote,
			ActorID:         in.ActorID,
		})
		if createErr != nil {
			if errors.Is(createErr, domain.ErrVoteDuplicate) {
				return domain.ErrVoteDuplicate
			}
			if errors.Is(createErr, domain.ErrVoteHashDuplicate) {
				return domain.ErrVoteHashDuplicate
			}
			return createErr
		}

		// Create evidence.
		payloadJSON, _ := json.Marshal(map[string]any{
			"motion_id":        in.MotionID,
			"voter_user_id":    in.VoterUserID,
			"unit_id":          in.UnitID,
			"option":           in.Option,
			"coefficient_used": in.CoefficientUsed,
			"cast_at":          now,
			"is_proxy_vote":    in.IsProxyVote,
		})
		if _, evErr := u.Evidence.Create(txCtx, domain.CreateVoteEvidenceInput{
			VoteID:       vote.ID,
			MotionID:     in.MotionID,
			PrevVoteHash: prevHash,
			VoteHash:     voteHash,
			PayloadJSON:  payloadJSON,
			ClientIP:     in.ClientIP,
			UserAgent:    in.UserAgent,
			NTPOffsetMS:  in.NTPOffsetMS,
		}); evErr != nil {
			return evErr
		}

		// Outbox event.
		outboxPayload, _ := json.Marshal(map[string]any{
			"vote_id":   vote.ID,
			"motion_id": in.MotionID,
			"unit_id":   in.UnitID,
			"vote_hash": voteHash,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: vote.ID,
			EventType:   entities.OutboxEventVoteCast,
			Payload:     outboxPayload,
		}); oErr != nil {
			return oErr
		}

		created = vote
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.Serializable, run); err != nil {
			if errors.Is(err, domain.ErrVoteDuplicate) {
				return entities.Vote{}, apperrors.Conflict("active vote already exists for this unit on this motion")
			}
			if errors.Is(err, domain.ErrVoteHashDuplicate) {
				return entities.Vote{}, apperrors.Conflict("vote hash collision, retry")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Vote{}, mapVersionConflict()
			}
			return entities.Vote{}, apperrors.Internal("failed to cast vote")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVoteDuplicate) {
				return entities.Vote{}, apperrors.Conflict("active vote already exists for this unit on this motion")
			}
			if errors.Is(err, domain.ErrVoteHashDuplicate) {
				return entities.Vote{}, apperrors.Conflict("vote hash collision, retry")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Vote{}, mapVersionConflict()
			}
			return entities.Vote{}, apperrors.Internal("failed to cast vote")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// GetMotionResults
// ---------------------------------------------------------------------------

// GetMotionResults devuelve una mocion y sus votos.
type GetMotionResults struct {
	Motions domain.MotionRepository
	Votes   domain.VoteRepository
}

// MotionResultsOutput es el output del usecase.
type MotionResultsOutput struct {
	Motion entities.AssemblyMotion
	Votes  []entities.Vote
}

// Execute valida y delega.
func (u GetMotionResults) Execute(ctx context.Context, motionID string) (MotionResultsOutput, error) {
	if err := policies.ValidateUUID(motionID); err != nil {
		return MotionResultsOutput{}, apperrors.BadRequest("motion_id: " + err.Error())
	}

	motion, err := u.Motions.GetByID(ctx, motionID)
	if err != nil {
		if errors.Is(err, domain.ErrMotionNotFound) {
			return MotionResultsOutput{}, apperrors.NotFound("motion not found")
		}
		return MotionResultsOutput{}, apperrors.Internal("failed to load motion")
	}

	votes, err := u.Votes.ListByMotionID(ctx, motionID)
	if err != nil {
		return MotionResultsOutput{}, apperrors.Internal("failed to load votes")
	}

	return MotionResultsOutput{Motion: motion, Votes: votes}, nil
}

// ---------------------------------------------------------------------------
// CreateAct
// ---------------------------------------------------------------------------

// CreateAct crea un acta para una asamblea cerrada.
type CreateAct struct {
	Assemblies domain.AssemblyRepository
	Acts       domain.ActRepository
}

// CreateActInput es el input del usecase.
type CreateActInput struct {
	AssemblyID   string
	BodyMD       string
	ArchiveUntil *time.Time
	ActorID      string
}

// Execute valida y delega.
func (u CreateAct) Execute(ctx context.Context, in CreateActInput) (entities.Act, error) {
	if err := policies.ValidateUUID(in.AssemblyID); err != nil {
		return entities.Act{}, apperrors.BadRequest("assembly_id: " + err.Error())
	}
	if strings.TrimSpace(in.BodyMD) == "" {
		return entities.Act{}, apperrors.BadRequest("body_md is required")
	}

	assembly, err := u.Assemblies.GetByID(ctx, in.AssemblyID)
	if err != nil {
		if errors.Is(err, domain.ErrAssemblyNotFound) {
			return entities.Act{}, apperrors.NotFound("assembly not found")
		}
		return entities.Act{}, apperrors.Internal("failed to load assembly")
	}
	if assembly.Status != entities.AssemblyStatusClosed {
		return entities.Act{}, apperrors.Conflict(
			"assembly must be closed to create an act")
	}

	act, err := u.Acts.Create(ctx, domain.CreateActInput{
		AssemblyID:   in.AssemblyID,
		BodyMD:       in.BodyMD,
		ArchiveUntil: in.ArchiveUntil,
		ActorID:      in.ActorID,
	})
	if err != nil {
		return entities.Act{}, apperrors.Internal("failed to create act")
	}
	return act, nil
}

// ---------------------------------------------------------------------------
// SignAct
// ---------------------------------------------------------------------------

// SignAct firma un acta.
type SignAct struct {
	Acts       domain.ActRepository
	Signatures domain.ActSignatureRepository
	Outbox     domain.OutboxRepository
	TxRunner   TxRunner
	Now        func() time.Time
}

// SignActInput es el input del usecase.
type SignActInput struct {
	ActID           string
	SignerUserID    string
	Role            entities.SignatureRole
	SignatureMethod entities.SignatureMethod
	EvidenceHash    string
	ClientIP        *string
	UserAgent       *string
	ActorID         string
}

// Execute valida y delega.
func (u SignAct) Execute(ctx context.Context, in SignActInput) (entities.ActSignature, error) {
	if err := policies.ValidateUUID(in.ActID); err != nil {
		return entities.ActSignature{}, apperrors.BadRequest("act_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.SignerUserID); err != nil {
		return entities.ActSignature{}, apperrors.BadRequest("signer_user_id: " + err.Error())
	}
	if !in.Role.IsValid() {
		return entities.ActSignature{}, apperrors.BadRequest("role: invalid role")
	}
	if !in.SignatureMethod.IsValid() {
		return entities.ActSignature{}, apperrors.BadRequest("signature_method: invalid method")
	}
	if strings.TrimSpace(in.EvidenceHash) == "" {
		return entities.ActSignature{}, apperrors.BadRequest("evidence_hash is required")
	}

	act, err := u.Acts.GetByID(ctx, in.ActID)
	if err != nil {
		if errors.Is(err, domain.ErrActNotFound) {
			return entities.ActSignature{}, apperrors.NotFound("act not found")
		}
		return entities.ActSignature{}, apperrors.Internal("failed to load act")
	}
	if err := policies.CanSignAct(act); err != nil {
		return entities.ActSignature{}, apperrors.Conflict(err.Error())
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var sig entities.ActSignature
	run := func(txCtx context.Context) error {
		signature, createErr := u.Signatures.Create(txCtx, domain.CreateSignatureInput{
			ActID:           in.ActID,
			SignerUserID:    in.SignerUserID,
			Role:            in.Role,
			SignatureMethod: in.SignatureMethod,
			EvidenceHash:    in.EvidenceHash,
			ClientIP:        in.ClientIP,
			UserAgent:       in.UserAgent,
			ActorID:         in.ActorID,
		})
		if createErr != nil {
			if errors.Is(createErr, domain.ErrSignatureDuplicate) {
				return domain.ErrSignatureDuplicate
			}
			return createErr
		}

		// Seal the act (transition to signed).
		if _, updateErr := u.Acts.UpdateStatus(txCtx, domain.UpdateActInput{
			ID:              in.ActID,
			Status:          entities.ActStatusSigned,
			SealedAt:        &now,
			ExpectedVersion: act.Version,
			ActorID:         in.ActorID,
		}); updateErr != nil {
			return updateErr
		}

		payload, _ := json.Marshal(map[string]any{
			"act_id":         in.ActID,
			"signer_user_id": in.SignerUserID,
			"role":           string(in.Role),
			"sealed_at":      now,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			AggregateID: in.ActID,
			EventType:   entities.OutboxEventActSigned,
			Payload:     payload,
		}); oErr != nil {
			return oErr
		}

		sig = signature
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrSignatureDuplicate) {
				return entities.ActSignature{}, apperrors.Conflict("signature already exists for this role")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.ActSignature{}, mapVersionConflict()
			}
			return entities.ActSignature{}, apperrors.Internal("failed to sign act")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrSignatureDuplicate) {
				return entities.ActSignature{}, apperrors.Conflict("signature already exists for this role")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.ActSignature{}, mapVersionConflict()
			}
			return entities.ActSignature{}, apperrors.Internal("failed to sign act")
		}
	}
	return sig, nil
}

// ---------------------------------------------------------------------------
// GetAct
// ---------------------------------------------------------------------------

// GetAct devuelve un acta con sus firmas.
type GetAct struct {
	Acts       domain.ActRepository
	Signatures domain.ActSignatureRepository
}

// GetActOutput es el output del usecase.
type GetActOutput struct {
	Act        entities.Act
	Signatures []entities.ActSignature
}

// Execute valida y delega.
func (u GetAct) Execute(ctx context.Context, actID string) (GetActOutput, error) {
	if err := policies.ValidateUUID(actID); err != nil {
		return GetActOutput{}, apperrors.BadRequest("act_id: " + err.Error())
	}

	act, err := u.Acts.GetByID(ctx, actID)
	if err != nil {
		if errors.Is(err, domain.ErrActNotFound) {
			return GetActOutput{}, apperrors.NotFound("act not found")
		}
		return GetActOutput{}, apperrors.Internal("failed to load act")
	}

	sigs, err := u.Signatures.ListByActID(ctx, actID)
	if err != nil {
		return GetActOutput{}, apperrors.Internal("failed to load signatures")
	}

	return GetActOutput{Act: act, Signatures: sigs}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"resource was modified by another request; reload and retry")
}

// tallyVotes agrupa los votos activos por opcion y totaliza los
// coeficientes.
func tallyVotes(votes []entities.Vote) map[string]any {
	optionTotals := make(map[string]float64)
	optionCounts := make(map[string]int)
	for _, v := range votes {
		if v.Status != entities.VoteStatusCast {
			continue
		}
		optionTotals[v.Option] += v.CoefficientUsed
		optionCounts[v.Option]++
	}
	return map[string]any{
		"totals_by_coefficient": optionTotals,
		"counts_by_option":      optionCounts,
	}
}
