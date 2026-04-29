// Package usecases orquesta la logica de aplicacion del modulo pqrs.
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

	"github.com/saas-ph/api/internal/modules/pqrs/domain"
	"github.com/saas-ph/api/internal/modules/pqrs/domain/entities"
	"github.com/saas-ph/api/internal/modules/pqrs/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// CreateCategory
// ---------------------------------------------------------------------------

// CreateCategory crea una categoria de PQRS nueva en estado 'active'.
type CreateCategory struct {
	Categories domain.CategoryRepository
}

// CreateCategoryInput es el input del usecase (sin tags JSON).
type CreateCategoryInput struct {
	Code                  string
	Name                  string
	DefaultAssigneeRoleID *string
	ActorID               string
}

// Execute valida y delega al repo.
func (u CreateCategory) Execute(ctx context.Context, in CreateCategoryInput) (entities.Category, error) {
	if strings.TrimSpace(in.Code) == "" {
		return entities.Category{}, apperrors.BadRequest("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.Category{}, apperrors.BadRequest("name is required")
	}
	cat, err := u.Categories.Create(ctx, domain.CreateCategoryInput{
		Code:                  strings.TrimSpace(in.Code),
		Name:                  strings.TrimSpace(in.Name),
		DefaultAssigneeRoleID: in.DefaultAssigneeRoleID,
		ActorID:               in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCategoryCodeDuplicate) {
			return entities.Category{}, apperrors.Conflict("category code already exists")
		}
		return entities.Category{}, apperrors.Internal("failed to create category")
	}
	return cat, nil
}

// ---------------------------------------------------------------------------
// ListCategories
// ---------------------------------------------------------------------------

// ListCategories lista las categorias activas.
type ListCategories struct {
	Categories domain.CategoryRepository
}

// Execute delega al repo.
func (u ListCategories) Execute(ctx context.Context) ([]entities.Category, error) {
	out, err := u.Categories.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list categories")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// UpdateCategory
// ---------------------------------------------------------------------------

// UpdateCategory actualiza una categoria existente con concurrencia
// optimista.
type UpdateCategory struct {
	Categories domain.CategoryRepository
}

// UpdateCategoryInput es el input del usecase.
type UpdateCategoryInput struct {
	ID                    string
	Code                  string
	Name                  string
	DefaultAssigneeRoleID *string
	Status                entities.CategoryStatus
	ExpectedVersion       int32
	ActorID               string
}

// Execute valida y delega al repo.
func (u UpdateCategory) Execute(ctx context.Context, in UpdateCategoryInput) (entities.Category, error) {
	if err := policies.ValidateUUID(in.ID); err != nil {
		return entities.Category{}, apperrors.BadRequest("id: " + err.Error())
	}
	if strings.TrimSpace(in.Code) == "" {
		return entities.Category{}, apperrors.BadRequest("code is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return entities.Category{}, apperrors.BadRequest("name is required")
	}
	if !in.Status.IsValid() {
		return entities.Category{}, apperrors.BadRequest("status: invalid category status")
	}
	cat, err := u.Categories.Update(ctx, domain.UpdateCategoryInput{
		ID:                    in.ID,
		Code:                  strings.TrimSpace(in.Code),
		Name:                  strings.TrimSpace(in.Name),
		DefaultAssigneeRoleID: in.DefaultAssigneeRoleID,
		Status:                in.Status,
		ExpectedVersion:       in.ExpectedVersion,
		ActorID:               in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCategoryNotFound):
			return entities.Category{}, apperrors.NotFound("category not found")
		case errors.Is(err, domain.ErrVersionConflict):
			return entities.Category{}, mapVersionConflict()
		case errors.Is(err, domain.ErrCategoryCodeDuplicate):
			return entities.Category{}, apperrors.Conflict("category code already exists")
		default:
			return entities.Category{}, apperrors.Internal("failed to update category")
		}
	}
	return cat, nil
}

// ---------------------------------------------------------------------------
// FileTicket
// ---------------------------------------------------------------------------

// FileTicket crea un ticket PQRS nuevo. Genera el serial_number
// atomicamente y registra la transicion inicial y el evento outbox
// dentro de una transaccion.
type FileTicket struct {
	Tickets  domain.TicketRepository
	History  domain.StatusHistoryRepository
	Outbox   domain.OutboxRepository
	TxRunner TxRunner
	Now      func() time.Time
}

// FileTicketInput es el input del usecase.
type FileTicketInput struct {
	PQRType     entities.PQRType
	CategoryID  *string
	Subject     string
	Body        string
	IsAnonymous bool
	ActorID     string
}

// Execute valida, genera serial y persiste en TX.
func (u FileTicket) Execute(ctx context.Context, in FileTicketInput) (entities.Ticket, error) {
	if !in.PQRType.IsValid() {
		return entities.Ticket{}, apperrors.BadRequest("pqr_type: invalid type")
	}
	if strings.TrimSpace(in.Subject) == "" {
		return entities.Ticket{}, apperrors.BadRequest("subject is required")
	}
	if strings.TrimSpace(in.Body) == "" {
		return entities.Ticket{}, apperrors.BadRequest("body is required")
	}
	if err := policies.ValidateAnonymous(in.IsAnonymous); err != nil {
		return entities.Ticket{}, apperrors.New(422, "unprocessable", "Unprocessable Entity", err.Error())
	}
	if in.CategoryID != nil {
		if err := policies.ValidateUUID(*in.CategoryID); err != nil {
			return entities.Ticket{}, apperrors.BadRequest("category_id: " + err.Error())
		}
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	year := int32(now.Year()) //nolint:gosec // year value fits in int32

	// Default SLA: 15 business days ~ 21 calendar days.
	slaDue := now.AddDate(0, 0, 21)

	var created entities.Ticket
	run := func(txCtx context.Context) error {
		serial, serialErr := u.Tickets.NextSerialNumber(txCtx, year)
		if serialErr != nil {
			return serialErr
		}

		ticket, createErr := u.Tickets.Create(txCtx, domain.CreateTicketInput{
			TicketYear:      year,
			SerialNumber:    serial,
			PQRType:         in.PQRType,
			CategoryID:      in.CategoryID,
			Subject:         strings.TrimSpace(in.Subject),
			Body:            strings.TrimSpace(in.Body),
			RequesterUserID: in.ActorID,
			IsAnonymous:     in.IsAnonymous,
			SLADueAt:        &slaDue,
			ActorID:         in.ActorID,
		})
		if createErr != nil {
			return createErr
		}

		toStatus := string(entities.TicketStatusRadicado)
		if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
			TicketID:             ticket.ID,
			FromStatus:           nil,
			ToStatus:             toStatus,
			TransitionedByUserID: in.ActorID,
			Notes:                nil,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"ticket_id":     ticket.ID,
			"ticket_year":   ticket.TicketYear,
			"serial_number": ticket.SerialNumber,
			"pqr_type":      ticket.PQRType,
			"subject":       ticket.Subject,
			"requester":     ticket.RequesterUserID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			TicketID:  ticket.ID,
			EventType: entities.OutboxEventPQRSCreated,
			Payload:   payload,
		}); oErr != nil {
			return oErr
		}

		created = ticket
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			return entities.Ticket{}, apperrors.Internal("failed to file ticket")
		}
	} else {
		if err := run(ctx); err != nil {
			return entities.Ticket{}, apperrors.Internal("failed to file ticket")
		}
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// ListTickets
// ---------------------------------------------------------------------------

// ListTickets lista tickets con filtros opcionales.
type ListTickets struct {
	Tickets domain.TicketRepository
}

// ListTicketsInput es el input del usecase.
type ListTicketsInput struct {
	Status           *entities.TicketStatus
	PQRType          *entities.PQRType
	RequesterUserID  *string
	AssignedToUserID *string
}

// Execute delega al repo.
func (u ListTickets) Execute(ctx context.Context, in ListTicketsInput) ([]entities.Ticket, error) {
	out, err := u.Tickets.List(ctx, domain.TicketListFilter{
		Status:           in.Status,
		PQRType:          in.PQRType,
		RequesterUserID:  in.RequesterUserID,
		AssignedToUserID: in.AssignedToUserID,
	})
	if err != nil {
		return nil, apperrors.Internal("failed to list tickets")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// GetTicket
// ---------------------------------------------------------------------------

// GetTicket devuelve un ticket por id.
type GetTicket struct {
	Tickets domain.TicketRepository
}

// Execute valida y delega.
func (u GetTicket) Execute(ctx context.Context, id string) (entities.Ticket, error) {
	if err := policies.ValidateUUID(id); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("id: " + err.Error())
	}
	ticket, err := u.Tickets.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Ticket{}, apperrors.NotFound("ticket not found")
		}
		return entities.Ticket{}, apperrors.Internal("failed to load ticket")
	}
	return ticket, nil
}

// ---------------------------------------------------------------------------
// AssignTicket
// ---------------------------------------------------------------------------

// AssignTicket asigna un ticket a un usuario.
type AssignTicket struct {
	Tickets  domain.TicketRepository
	History  domain.StatusHistoryRepository
	Outbox   domain.OutboxRepository
	TxRunner TxRunner
}

// AssignTicketInput es el input del usecase.
type AssignTicketInput struct {
	TicketID         string
	AssignedToUserID string
	ActorID          string
}

// Execute valida y delega.
func (u AssignTicket) Execute(ctx context.Context, in AssignTicketInput) (entities.Ticket, error) {
	if err := policies.ValidateUUID(in.TicketID); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("ticket_id: " + err.Error())
	}
	if err := policies.ValidateUUID(in.AssignedToUserID); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("assigned_to_user_id: " + err.Error())
	}

	ticket, err := u.Tickets.GetByID(ctx, in.TicketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Ticket{}, apperrors.NotFound("ticket not found")
		}
		return entities.Ticket{}, apperrors.Internal("failed to load ticket")
	}

	if !ticket.IsOpen() {
		return entities.Ticket{}, apperrors.Conflict("ticket is not open")
	}

	var assigned entities.Ticket
	run := func(txCtx context.Context) error {
		result, assignErr := u.Tickets.Assign(txCtx, ticket.ID, in.AssignedToUserID, ticket.Version, in.ActorID)
		if assignErr != nil {
			return assignErr
		}

		payload, _ := json.Marshal(map[string]any{
			"ticket_id":           result.ID,
			"assigned_to_user_id": in.AssignedToUserID,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			TicketID:  result.ID,
			EventType: entities.OutboxEventPQRSAssigned,
			Payload:   payload,
		}); oErr != nil {
			return oErr
		}

		assigned = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to assign ticket")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to assign ticket")
		}
	}
	return assigned, nil
}

// ---------------------------------------------------------------------------
// StartStudy
// ---------------------------------------------------------------------------

// StartStudy transitions a ticket from radicado to en_estudio.
type StartStudy struct {
	Tickets  domain.TicketRepository
	History  domain.StatusHistoryRepository
	TxRunner TxRunner
}

// StartStudyInput es el input del usecase.
type StartStudyInput struct {
	TicketID string
	ActorID  string
}

// Execute valida y delega.
func (u StartStudy) Execute(ctx context.Context, in StartStudyInput) (entities.Ticket, error) {
	if err := policies.ValidateUUID(in.TicketID); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("ticket_id: " + err.Error())
	}

	ticket, err := u.Tickets.GetByID(ctx, in.TicketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Ticket{}, apperrors.NotFound("ticket not found")
		}
		return entities.Ticket{}, apperrors.Internal("failed to load ticket")
	}

	if !policies.CanTransitionTicket(ticket.Status, entities.TicketStatusEnEstudio) {
		return entities.Ticket{}, apperrors.Conflict(
			"cannot transition from " + string(ticket.Status) + " to en_estudio")
	}

	var updated entities.Ticket
	run := func(txCtx context.Context) error {
		result, updateErr := u.Tickets.UpdateStatus(txCtx, ticket.ID, entities.TicketStatusEnEstudio, ticket.Version, in.ActorID)
		if updateErr != nil {
			return updateErr
		}

		fromStatus := string(ticket.Status)
		if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
			TicketID:             ticket.ID,
			FromStatus:           &fromStatus,
			ToStatus:             string(entities.TicketStatusEnEstudio),
			TransitionedByUserID: in.ActorID,
		}); hErr != nil {
			return hErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to start study")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to start study")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// RespondTicket
// ---------------------------------------------------------------------------

// RespondTicket adds an official response to a ticket and transitions
// to respondido.
type RespondTicket struct {
	Tickets   domain.TicketRepository
	Responses domain.ResponseRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
}

// RespondTicketInput es el input del usecase.
type RespondTicketInput struct {
	TicketID string
	Body     string
	ActorID  string
}

// Execute valida y delega.
func (u RespondTicket) Execute(ctx context.Context, in RespondTicketInput) (entities.Ticket, error) {
	if err := policies.ValidateUUID(in.TicketID); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("ticket_id: " + err.Error())
	}
	if strings.TrimSpace(in.Body) == "" {
		return entities.Ticket{}, apperrors.BadRequest("body is required")
	}

	ticket, err := u.Tickets.GetByID(ctx, in.TicketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Ticket{}, apperrors.NotFound("ticket not found")
		}
		return entities.Ticket{}, apperrors.Internal("failed to load ticket")
	}

	if !policies.CanTransitionTicket(ticket.Status, entities.TicketStatusRespondido) {
		return entities.Ticket{}, apperrors.Conflict(
			"cannot respond to ticket in status " + string(ticket.Status))
	}

	var updated entities.Ticket
	run := func(txCtx context.Context) error {
		_, createErr := u.Responses.Create(txCtx, domain.CreateResponseInput{
			TicketID:          ticket.ID,
			ResponseType:      entities.ResponseTypeOfficialResponse,
			Body:              strings.TrimSpace(in.Body),
			RespondedByUserID: in.ActorID,
		})
		if createErr != nil {
			if errors.Is(createErr, domain.ErrOfficialResponseExists) {
				return domain.ErrOfficialResponseExists
			}
			return createErr
		}

		result, setErr := u.Tickets.SetResponded(txCtx, ticket.ID, ticket.Version, in.ActorID)
		if setErr != nil {
			return setErr
		}

		fromStatus := string(ticket.Status)
		if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
			TicketID:             ticket.ID,
			FromStatus:           &fromStatus,
			ToStatus:             string(entities.TicketStatusRespondido),
			TransitionedByUserID: in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"ticket_id":     result.ID,
			"responded_by":  in.ActorID,
			"serial_number": result.SerialNumber,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			TicketID:  result.ID,
			EventType: entities.OutboxEventPQRSResponded,
			Payload:   payload,
		}); oErr != nil {
			return oErr
		}

		updated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrOfficialResponseExists) {
				return entities.Ticket{}, apperrors.Conflict("official response already exists for this ticket")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to respond to ticket")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrOfficialResponseExists) {
				return entities.Ticket{}, apperrors.Conflict("official response already exists for this ticket")
			}
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to respond to ticket")
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// AddNote
// ---------------------------------------------------------------------------

// AddNote adds an internal note to a ticket.
type AddNote struct {
	Tickets   domain.TicketRepository
	Responses domain.ResponseRepository
}

// AddNoteInput es el input del usecase.
type AddNoteInput struct {
	TicketID string
	Body     string
	ActorID  string
}

// Execute valida y delega.
func (u AddNote) Execute(ctx context.Context, in AddNoteInput) (entities.Response, error) {
	if err := policies.ValidateUUID(in.TicketID); err != nil {
		return entities.Response{}, apperrors.BadRequest("ticket_id: " + err.Error())
	}
	if strings.TrimSpace(in.Body) == "" {
		return entities.Response{}, apperrors.BadRequest("body is required")
	}

	ticket, err := u.Tickets.GetByID(ctx, in.TicketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Response{}, apperrors.NotFound("ticket not found")
		}
		return entities.Response{}, apperrors.Internal("failed to load ticket")
	}

	if !ticket.IsOpen() {
		return entities.Response{}, apperrors.Conflict("ticket is not open")
	}

	resp, err := u.Responses.Create(ctx, domain.CreateResponseInput{
		TicketID:          ticket.ID,
		ResponseType:      entities.ResponseTypeInternalNote,
		Body:              strings.TrimSpace(in.Body),
		RespondedByUserID: in.ActorID,
	})
	if err != nil {
		return entities.Response{}, apperrors.Internal("failed to add note")
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// CloseTicket
// ---------------------------------------------------------------------------

// CloseTicket cierra un ticket con rating y feedback opcionales.
type CloseTicket struct {
	Tickets   domain.TicketRepository
	Responses domain.ResponseRepository
	History   domain.StatusHistoryRepository
	Outbox    domain.OutboxRepository
	TxRunner  TxRunner
}

// CloseTicketInput es el input del usecase.
type CloseTicketInput struct {
	TicketID string
	Rating   *int32
	Feedback *string
	ActorID  string
}

// Execute valida y delega.
func (u CloseTicket) Execute(ctx context.Context, in CloseTicketInput) (entities.Ticket, error) {
	if err := policies.ValidateUUID(in.TicketID); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("ticket_id: " + err.Error())
	}
	if err := policies.ValidateRating(in.Rating); err != nil {
		return entities.Ticket{}, apperrors.BadRequest(err.Error())
	}

	ticket, err := u.Tickets.GetByID(ctx, in.TicketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Ticket{}, apperrors.NotFound("ticket not found")
		}
		return entities.Ticket{}, apperrors.Internal("failed to load ticket")
	}

	if !policies.CanTransitionTicket(ticket.Status, entities.TicketStatusCerrado) {
		return entities.Ticket{}, apperrors.Conflict(
			"cannot close ticket in status " + string(ticket.Status))
	}

	hasOfficial, err := u.Responses.HasOfficialResponse(ctx, in.TicketID)
	if err != nil {
		return entities.Ticket{}, apperrors.Internal("failed to check official response")
	}
	if err := policies.ValidateCloseBeforeResponse(ticket, hasOfficial); err != nil {
		return entities.Ticket{}, apperrors.New(422, "unprocessable", "Unprocessable Entity", err.Error())
	}

	var closed entities.Ticket
	run := func(txCtx context.Context) error {
		result, closeErr := u.Tickets.Close(txCtx, ticket.ID, in.Rating, in.Feedback, ticket.Version, in.ActorID)
		if closeErr != nil {
			return closeErr
		}

		fromStatus := string(ticket.Status)
		if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
			TicketID:             ticket.ID,
			FromStatus:           &fromStatus,
			ToStatus:             string(entities.TicketStatusCerrado),
			TransitionedByUserID: in.ActorID,
		}); hErr != nil {
			return hErr
		}

		payload, _ := json.Marshal(map[string]any{
			"ticket_id":     result.ID,
			"serial_number": result.SerialNumber,
			"rating":        in.Rating,
		})
		if _, oErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
			TicketID:  result.ID,
			EventType: entities.OutboxEventPQRSClosed,
			Payload:   payload,
		}); oErr != nil {
			return oErr
		}

		closed = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to close ticket")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to close ticket")
		}
	}
	return closed, nil
}

// ---------------------------------------------------------------------------
// EscalateTicket
// ---------------------------------------------------------------------------

// EscalateTicket escala un ticket.
type EscalateTicket struct {
	Tickets  domain.TicketRepository
	History  domain.StatusHistoryRepository
	TxRunner TxRunner
}

// EscalateTicketInput es el input del usecase.
type EscalateTicketInput struct {
	TicketID string
	Notes    *string
	ActorID  string
}

// Execute valida y delega.
func (u EscalateTicket) Execute(ctx context.Context, in EscalateTicketInput) (entities.Ticket, error) {
	if err := policies.ValidateUUID(in.TicketID); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("ticket_id: " + err.Error())
	}

	ticket, err := u.Tickets.GetByID(ctx, in.TicketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Ticket{}, apperrors.NotFound("ticket not found")
		}
		return entities.Ticket{}, apperrors.Internal("failed to load ticket")
	}

	if !policies.CanTransitionTicket(ticket.Status, entities.TicketStatusEscalado) {
		return entities.Ticket{}, apperrors.Conflict(
			"cannot escalate ticket in status " + string(ticket.Status))
	}

	var escalated entities.Ticket
	run := func(txCtx context.Context) error {
		result, escErr := u.Tickets.Escalate(txCtx, ticket.ID, ticket.Version, in.ActorID)
		if escErr != nil {
			return escErr
		}

		fromStatus := string(ticket.Status)
		if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
			TicketID:             ticket.ID,
			FromStatus:           &fromStatus,
			ToStatus:             string(entities.TicketStatusEscalado),
			TransitionedByUserID: in.ActorID,
			Notes:                in.Notes,
		}); hErr != nil {
			return hErr
		}

		escalated = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to escalate ticket")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to escalate ticket")
		}
	}
	return escalated, nil
}

// ---------------------------------------------------------------------------
// CancelTicket
// ---------------------------------------------------------------------------

// CancelTicket cancela un ticket.
type CancelTicket struct {
	Tickets  domain.TicketRepository
	History  domain.StatusHistoryRepository
	TxRunner TxRunner
}

// CancelTicketInput es el input del usecase.
type CancelTicketInput struct {
	TicketID string
	Notes    *string
	ActorID  string
}

// Execute valida y delega.
func (u CancelTicket) Execute(ctx context.Context, in CancelTicketInput) (entities.Ticket, error) {
	if err := policies.ValidateUUID(in.TicketID); err != nil {
		return entities.Ticket{}, apperrors.BadRequest("ticket_id: " + err.Error())
	}

	ticket, err := u.Tickets.GetByID(ctx, in.TicketID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return entities.Ticket{}, apperrors.NotFound("ticket not found")
		}
		return entities.Ticket{}, apperrors.Internal("failed to load ticket")
	}

	if !policies.CanTransitionTicket(ticket.Status, entities.TicketStatusCancelado) {
		return entities.Ticket{}, apperrors.Conflict(
			"cannot cancel ticket in status " + string(ticket.Status))
	}

	var cancelled entities.Ticket
	run := func(txCtx context.Context) error {
		result, cancelErr := u.Tickets.Cancel(txCtx, ticket.ID, ticket.Version, in.ActorID)
		if cancelErr != nil {
			return cancelErr
		}

		fromStatus := string(ticket.Status)
		if _, hErr := u.History.Record(txCtx, domain.RecordHistoryInput{
			TicketID:             ticket.ID,
			FromStatus:           &fromStatus,
			ToStatus:             string(entities.TicketStatusCancelado),
			TransitionedByUserID: in.ActorID,
			Notes:                in.Notes,
		}); hErr != nil {
			return hErr
		}

		cancelled = result
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to cancel ticket")
		}
	} else {
		if err := run(ctx); err != nil {
			if errors.Is(err, domain.ErrVersionConflict) {
				return entities.Ticket{}, mapVersionConflict()
			}
			return entities.Ticket{}, apperrors.Internal("failed to cancel ticket")
		}
	}
	return cancelled, nil
}

// ---------------------------------------------------------------------------
// GetTicketHistory
// ---------------------------------------------------------------------------

// GetTicketHistory devuelve el historial de transiciones de un ticket.
type GetTicketHistory struct {
	Tickets domain.TicketRepository
	History domain.StatusHistoryRepository
}

// Execute valida y delega.
func (u GetTicketHistory) Execute(ctx context.Context, ticketID string) ([]entities.StatusHistory, error) {
	if err := policies.ValidateUUID(ticketID); err != nil {
		return nil, apperrors.BadRequest("ticket_id: " + err.Error())
	}

	if _, err := u.Tickets.GetByID(ctx, ticketID); err != nil {
		if errors.Is(err, domain.ErrTicketNotFound) {
			return nil, apperrors.NotFound("ticket not found")
		}
		return nil, apperrors.Internal("failed to load ticket")
	}

	history, err := u.History.ListByTicketID(ctx, ticketID)
	if err != nil {
		return nil, apperrors.Internal("failed to load ticket history")
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
