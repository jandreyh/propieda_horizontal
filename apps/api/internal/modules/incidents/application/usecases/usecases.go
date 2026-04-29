// Package usecases orquesta la logica de aplicacion del modulo incidents.
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

	"github.com/saas-ph/api/internal/modules/incidents/domain"
	"github.com/saas-ph/api/internal/modules/incidents/domain/entities"
	"github.com/saas-ph/api/internal/modules/incidents/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// ReportIncident
// ---------------------------------------------------------------------------

// ReportIncident crea un incidente nuevo en estado 'reported'.
type ReportIncident struct {
	Incidents domain.IncidentRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// ReportIncidentInput es el input del usecase (sin tags JSON).
type ReportIncidentInput struct {
	IncidentType   entities.IncidentType
	Severity       entities.Severity
	Title          string
	Description    string
	StructureID    *string
	LocationDetail *string
	ActorID        string
}

// Execute valida y delega al repo.
func (u ReportIncident) Execute(ctx context.Context, in ReportIncidentInput) (entities.Incident, error) {
	if !in.IncidentType.IsValid() {
		return entities.Incident{}, apperrors.BadRequest("incident_type: invalid incident type")
	}
	if !in.Severity.IsValid() {
		return entities.Incident{}, apperrors.BadRequest("severity: invalid severity")
	}
	if strings.TrimSpace(in.Title) == "" {
		return entities.Incident{}, apperrors.BadRequest("title is required")
	}
	if strings.TrimSpace(in.Description) == "" {
		return entities.Incident{}, apperrors.BadRequest("description is required")
	}
	if in.ActorID == "" {
		return entities.Incident{}, apperrors.BadRequest("actor_id is required")
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	assignDue, resolveDue := policies.CalculateSLADueDates(now, in.Severity)

	var created entities.Incident
	run := func(txCtx context.Context) error {
		incident, createErr := u.Incidents.Create(txCtx, domain.CreateIncidentInput{
			IncidentType:     in.IncidentType,
			Severity:         in.Severity,
			Title:            strings.TrimSpace(in.Title),
			Description:      strings.TrimSpace(in.Description),
			ReportedByUserID: in.ActorID,
			ReportedAt:       now,
			StructureID:      in.StructureID,
			LocationDetail:   in.LocationDetail,
			SLAAssignDueAt:   &assignDue,
			SLAResolveDueAt:  &resolveDue,
			ActorID:          in.ActorID,
		})
		if createErr != nil {
			return createErr
		}

		toStatus := string(entities.IncidentStatusReported)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			IncidentID:           incident.ID,
			FromStatus:           nil,
			ToStatus:             toStatus,
			TransitionedByUserID: in.ActorID,
			Notes:                nil,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"incident_id":   incident.ID,
			"incident_type": incident.IncidentType,
			"severity":      incident.Severity,
			"title":         incident.Title,
			"reported_by":   incident.ReportedByUserID,
			"reported_at":   incident.ReportedAt,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			IncidentID: incident.ID,
			EventType:  entities.OutboxEventIncidentReported,
			Payload:    payload,
		}); oErr != nil {
			return oErr
		}

		created = incident
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			return entities.Incident{}, apperrors.Internal("failed to report incident")
		}
	} else {
		if err := run(ctx); err != nil {
			return entities.Incident{}, apperrors.Internal("failed to report incident")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// ListIncidents
// ---------------------------------------------------------------------------

// ListIncidents lista incidentes con filtros opcionales.
type ListIncidents struct {
	Incidents domain.IncidentRepository
}

// ListIncidentsFilter es el filtro del usecase.
type ListIncidentsFilter struct {
	Status           *entities.IncidentStatus
	Severity         *entities.Severity
	ReportedByUserID *string
}

// Execute delega al repo.
func (u ListIncidents) Execute(ctx context.Context, filter ListIncidentsFilter) ([]entities.Incident, error) {
	out, err := u.Incidents.List(ctx, domain.IncidentListFilter{
		Status:           filter.Status,
		Severity:         filter.Severity,
		ReportedByUserID: filter.ReportedByUserID,
	})
	if err != nil {
		return nil, apperrors.Internal("failed to list incidents")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// GetIncident
// ---------------------------------------------------------------------------

// GetIncident devuelve un incidente por ID.
type GetIncident struct {
	Incidents domain.IncidentRepository
}

// Execute valida y delega.
func (u GetIncident) Execute(ctx context.Context, id string) (entities.Incident, error) {
	if err := policies.ValidateUUID(id); err != nil {
		return entities.Incident{}, apperrors.BadRequest("id: " + err.Error())
	}
	incident, err := u.Incidents.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return entities.Incident{}, apperrors.NotFound("incident not found")
		}
		return entities.Incident{}, apperrors.Internal("failed to load incident")
	}
	return incident, nil
}

// ---------------------------------------------------------------------------
// AssignIncident
// ---------------------------------------------------------------------------

// AssignIncident asigna un incidente a un usuario. Si ya hay una
// asignacion activa, la desactiva primero dentro de la misma TX.
type AssignIncident struct {
	Incidents   domain.IncidentRepository
	Assignments domain.IncidentAssignmentRepository
	History     domain.StatusHistoryRepository
	Outbox      domain.OutboxRepository
	TxRunner    TxRunner
	Now         func() time.Time
}

// AssignIncidentInput es el input del usecase.
type AssignIncidentInput struct {
	IncidentID       string
	AssignedToUserID string
	ActorID          string
}

// Execute valida y delega.
func (u AssignIncident) Execute(ctx context.Context, in AssignIncidentInput) (entities.Incident, error) {
	if err := policies.ValidateUUID(in.IncidentID); err != nil {
		return entities.Incident{}, apperrors.BadRequest("incident_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.AssignedToUserID); err != nil {
		return entities.Incident{}, apperrors.BadRequest("assigned_to_user_id: " + err.Error())
	}

	incident, err := u.Incidents.GetByID(ctx, in.IncidentID)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return entities.Incident{}, apperrors.NotFound("incident not found")
		}
		return entities.Incident{}, apperrors.Internal("failed to load incident")
	}

	targetStatus := entities.IncidentStatusAssigned
	if !policies.CanTransitionStatus(incident.Status, targetStatus) {
		return entities.Incident{}, apperrors.Conflict(
			"cannot assign incident in status " + string(incident.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Incident
	run := func(txCtx context.Context) error {
		// Desactivar asignacion previa si existe.
		_ = u.Assignments.UnassignActive(txCtx, in.IncidentID, in.ActorID)

		// Crear nueva asignacion.
		if _, aErr := u.Assignments.Create(txCtx, domain.CreateAssignmentInput{
			IncidentID:       in.IncidentID,
			AssignedToUserID: in.AssignedToUserID,
			AssignedByUserID: in.ActorID,
		}); aErr != nil {
			if errors.Is(aErr, domain.ErrAssignmentAlreadyActive) {
				return domain.ErrAssignmentAlreadyActive
			}
			return aErr
		}

		// Actualizar incidente.
		fromStatus := string(incident.Status)
		result, updateErr := u.Incidents.UpdateStatus(txCtx, domain.UpdateIncidentStatusInput{
			ID:               in.IncidentID,
			NewStatus:        targetStatus,
			ExpectedVersion:  incident.Version,
			ActorID:          in.ActorID,
			AssignedToUserID: &in.AssignedToUserID,
			AssignedAt:       &now,
		})
		if updateErr != nil {
			return updateErr
		}

		toStatus := string(targetStatus)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			IncidentID:           in.IncidentID,
			FromStatus:           &fromStatus,
			ToStatus:             toStatus,
			TransitionedByUserID: in.ActorID,
			Notes:                nil,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"incident_id": result.ID,
			"assigned_to": in.AssignedToUserID,
			"assigned_by": in.ActorID,
			"assigned_at": now,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			IncidentID: result.ID,
			EventType:  entities.OutboxEventIncidentAssigned,
			Payload:    payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			if errors.Is(err, domain.ErrAssignmentAlreadyActive) {
				return entities.Incident{}, apperrors.Conflict("incident already has an active assignment")
			}
			return entities.Incident{}, apperrors.Internal("failed to assign incident")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			if errors.Is(err, domain.ErrAssignmentAlreadyActive) {
				return entities.Incident{}, apperrors.Conflict("incident already has an active assignment")
			}
			return entities.Incident{}, apperrors.Internal("failed to assign incident")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// StartIncident
// ---------------------------------------------------------------------------

// StartIncident marca un incidente como in_progress.
type StartIncident struct {
	Incidents domain.IncidentRepository
	History   domain.StatusHistoryRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// Execute valida y delega.
func (u StartIncident) Execute(ctx context.Context, incidentID, actorID string) (entities.Incident, error) {
	if err := policies.ValidateUUID(incidentID); err != nil {
		return entities.Incident{}, apperrors.BadRequest("id: " + err.Error())
	}

	incident, err := u.Incidents.GetByID(ctx, incidentID)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return entities.Incident{}, apperrors.NotFound("incident not found")
		}
		return entities.Incident{}, apperrors.Internal("failed to load incident")
	}

	targetStatus := entities.IncidentStatusInProgress
	if !policies.CanTransitionStatus(incident.Status, targetStatus) {
		return entities.Incident{}, apperrors.Conflict(
			"cannot start incident in status " + string(incident.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Incident
	run := func(txCtx context.Context) error {
		fromStatus := string(incident.Status)
		result, updateErr := u.Incidents.UpdateStatus(txCtx, domain.UpdateIncidentStatusInput{
			ID:              incidentID,
			NewStatus:       targetStatus,
			ExpectedVersion: incident.Version,
			ActorID:         actorID,
			StartedAt:       &now,
		})
		if updateErr != nil {
			return updateErr
		}

		toStatus := string(targetStatus)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			IncidentID:           incidentID,
			FromStatus:           &fromStatus,
			ToStatus:             toStatus,
			TransitionedByUserID: actorID,
			Notes:                nil,
		}); hErr != nil {
			return hErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to start incident")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to start incident")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// ResolveIncident
// ---------------------------------------------------------------------------

// ResolveIncident marca un incidente como resolved.
type ResolveIncident struct {
	Incidents domain.IncidentRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// ResolveIncidentInput es el input del usecase.
type ResolveIncidentInput struct {
	IncidentID      string
	ResolutionNotes string
	ActorID         string
}

// Execute valida y delega.
func (u ResolveIncident) Execute(ctx context.Context, in ResolveIncidentInput) (entities.Incident, error) {
	if err := policies.ValidateUUID(in.IncidentID); err != nil {
		return entities.Incident{}, apperrors.BadRequest("id: " + err.Error())
	}

	targetStatus := entities.IncidentStatusResolved
	if err := policies.ValidateResolutionNotes(targetStatus, &in.ResolutionNotes); err != nil {
		return entities.Incident{}, apperrors.BadRequest(err.Error())
	}

	incident, err := u.Incidents.GetByID(ctx, in.IncidentID)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return entities.Incident{}, apperrors.NotFound("incident not found")
		}
		return entities.Incident{}, apperrors.Internal("failed to load incident")
	}

	if !policies.CanTransitionStatus(incident.Status, targetStatus) {
		return entities.Incident{}, apperrors.Conflict(
			"cannot resolve incident in status " + string(incident.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Incident
	run := func(txCtx context.Context) error {
		fromStatus := string(incident.Status)
		notes := strings.TrimSpace(in.ResolutionNotes)
		result, updateErr := u.Incidents.UpdateStatus(txCtx, domain.UpdateIncidentStatusInput{
			ID:              in.IncidentID,
			NewStatus:       targetStatus,
			ExpectedVersion: incident.Version,
			ActorID:         in.ActorID,
			ResolutionNotes: &notes,
			ResolvedAt:      &now,
		})
		if updateErr != nil {
			return updateErr
		}

		toStatus := string(targetStatus)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			IncidentID:           in.IncidentID,
			FromStatus:           &fromStatus,
			ToStatus:             toStatus,
			TransitionedByUserID: in.ActorID,
			Notes:                &notes,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"incident_id":      result.ID,
			"resolution_notes": notes,
			"resolved_at":      now,
			"resolved_by":      in.ActorID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			IncidentID: result.ID,
			EventType:  entities.OutboxEventIncidentResolved,
			Payload:    payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to resolve incident")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to resolve incident")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// CloseIncident
// ---------------------------------------------------------------------------

// CloseIncident marca un incidente como closed.
type CloseIncident struct {
	Incidents domain.IncidentRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// CloseIncidentInput es el input del usecase.
type CloseIncidentInput struct {
	IncidentID      string
	ResolutionNotes string
	ActorID         string
}

// Execute valida y delega.
func (u CloseIncident) Execute(ctx context.Context, in CloseIncidentInput) (entities.Incident, error) {
	if err := policies.ValidateUUID(in.IncidentID); err != nil {
		return entities.Incident{}, apperrors.BadRequest("id: " + err.Error())
	}

	targetStatus := entities.IncidentStatusClosed
	if err := policies.ValidateResolutionNotes(targetStatus, &in.ResolutionNotes); err != nil {
		return entities.Incident{}, apperrors.BadRequest(err.Error())
	}

	incident, err := u.Incidents.GetByID(ctx, in.IncidentID)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return entities.Incident{}, apperrors.NotFound("incident not found")
		}
		return entities.Incident{}, apperrors.Internal("failed to load incident")
	}

	if !policies.CanTransitionStatus(incident.Status, targetStatus) {
		return entities.Incident{}, apperrors.Conflict(
			"cannot close incident in status " + string(incident.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Incident
	run := func(txCtx context.Context) error {
		fromStatus := string(incident.Status)
		notes := strings.TrimSpace(in.ResolutionNotes)

		// Use existing resolution notes if incident already has them
		// (e.g., from resolve step).
		resolNotes := &notes
		if incident.ResolutionNotes != nil && *incident.ResolutionNotes != "" && notes == "" {
			resolNotes = incident.ResolutionNotes
		}

		result, updateErr := u.Incidents.UpdateStatus(txCtx, domain.UpdateIncidentStatusInput{
			ID:              in.IncidentID,
			NewStatus:       targetStatus,
			ExpectedVersion: incident.Version,
			ActorID:         in.ActorID,
			ResolutionNotes: resolNotes,
			ClosedAt:        &now,
		})
		if updateErr != nil {
			return updateErr
		}

		toStatus := string(targetStatus)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			IncidentID:           in.IncidentID,
			FromStatus:           &fromStatus,
			ToStatus:             toStatus,
			TransitionedByUserID: in.ActorID,
			Notes:                resolNotes,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"incident_id": result.ID,
			"closed_at":   now,
			"closed_by":   in.ActorID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			IncidentID: result.ID,
			EventType:  entities.OutboxEventIncidentClosed,
			Payload:    payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to close incident")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to close incident")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// CancelIncident
// ---------------------------------------------------------------------------

// CancelIncident marca un incidente como cancelled.
type CancelIncident struct {
	Incidents domain.IncidentRepository
	History   domain.StatusHistoryRepository
	TxRunner  TxRunner
	Now       func() time.Time
}

// Execute valida y delega.
func (u CancelIncident) Execute(ctx context.Context, incidentID, actorID string) (entities.Incident, error) {
	if err := policies.ValidateUUID(incidentID); err != nil {
		return entities.Incident{}, apperrors.BadRequest("id: " + err.Error())
	}

	incident, err := u.Incidents.GetByID(ctx, incidentID)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return entities.Incident{}, apperrors.NotFound("incident not found")
		}
		return entities.Incident{}, apperrors.Internal("failed to load incident")
	}

	targetStatus := entities.IncidentStatusCancelled
	if !policies.CanTransitionStatus(incident.Status, targetStatus) {
		return entities.Incident{}, apperrors.Conflict(
			"cannot cancel incident in status " + string(incident.Status))
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	var updated entities.Incident
	run := func(txCtx context.Context) error {
		fromStatus := string(incident.Status)
		result, updateErr := u.Incidents.UpdateStatus(txCtx, domain.UpdateIncidentStatusInput{
			ID:              incidentID,
			NewStatus:       targetStatus,
			ExpectedVersion: incident.Version,
			ActorID:         actorID,
			CancelledAt:     &now,
		})
		if updateErr != nil {
			return updateErr
		}

		toStatus := string(targetStatus)
		if _, hErr := u.History.Record(txCtx, domain.RecordStatusHistoryInput{
			IncidentID:           incidentID,
			FromStatus:           &fromStatus,
			ToStatus:             toStatus,
			TransitionedByUserID: actorID,
			Notes:                nil,
		}); hErr != nil {
			return hErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to cancel incident")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Incident{}, mapVersionConflict()
			}
			return entities.Incident{}, apperrors.Internal("failed to cancel incident")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// AddAttachment
// ---------------------------------------------------------------------------

// AddAttachment agrega un adjunto a un incidente.
type AddAttachment struct {
	Incidents   domain.IncidentRepository
	Attachments domain.AttachmentRepository
}

// AddAttachmentInput es el input del usecase.
type AddAttachmentInput struct {
	IncidentID string
	URL        string
	MimeType   string
	SizeBytes  int64
	ActorID    string
}

// Execute valida y delega.
func (u AddAttachment) Execute(ctx context.Context, in AddAttachmentInput) (entities.Attachment, error) {
	if err := policies.ValidateUUID(in.IncidentID); err != nil {
		return entities.Attachment{}, apperrors.BadRequest("incident_id: " + err.Error())
	}
	if strings.TrimSpace(in.URL) == "" {
		return entities.Attachment{}, apperrors.BadRequest("url is required")
	}
	if strings.TrimSpace(in.MimeType) == "" {
		return entities.Attachment{}, apperrors.BadRequest("mime_type is required")
	}
	if in.SizeBytes < 0 {
		return entities.Attachment{}, apperrors.BadRequest("size_bytes must be non-negative")
	}

	// Verificar que el incidente existe.
	_, err := u.Incidents.GetByID(ctx, in.IncidentID)
	if err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return entities.Attachment{}, apperrors.NotFound("incident not found")
		}
		return entities.Attachment{}, apperrors.Internal("failed to load incident")
	}

	// Verificar limite de adjuntos.
	count, err := u.Attachments.CountByIncidentID(ctx, in.IncidentID)
	if err != nil {
		return entities.Attachment{}, apperrors.Internal("failed to count attachments")
	}
	if err := policies.ValidateAttachmentCount(count); err != nil {
		return entities.Attachment{}, apperrors.Conflict(err.Error())
	}

	attachment, err := u.Attachments.Create(ctx, domain.CreateAttachmentInput{
		IncidentID: in.IncidentID,
		URL:        strings.TrimSpace(in.URL),
		MimeType:   strings.TrimSpace(in.MimeType),
		SizeBytes:  in.SizeBytes,
		UploadedBy: in.ActorID,
	})
	if err != nil {
		return entities.Attachment{}, apperrors.Internal("failed to create attachment")
	}
	return attachment, nil
}

// ---------------------------------------------------------------------------
// GetStatusHistory
// ---------------------------------------------------------------------------

// GetStatusHistory devuelve el historial de estados de un incidente.
type GetStatusHistory struct {
	Incidents domain.IncidentRepository
	History   domain.StatusHistoryRepository
}

// Execute valida y delega.
func (u GetStatusHistory) Execute(ctx context.Context, incidentID string) ([]entities.StatusHistory, error) {
	if err := policies.ValidateUUID(incidentID); err != nil {
		return nil, apperrors.BadRequest("id: " + err.Error())
	}

	// Verificar que el incidente existe.
	if _, err := u.Incidents.GetByID(ctx, incidentID); err != nil {
		if errors.Is(err, domain.ErrIncidentNotFound) {
			return nil, apperrors.NotFound("incident not found")
		}
		return nil, apperrors.Internal("failed to load incident")
	}

	history, err := u.History.ListByIncidentID(ctx, incidentID)
	if err != nil {
		return nil, apperrors.Internal("failed to list status history")
	}
	return history, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"resource was modified by another request; reload and retry")
}
