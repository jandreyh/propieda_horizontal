// Package http (modulo units) implementa los handlers chi del modulo.
//
// Los handlers son delgados: parsean el request, llaman al usecase y
// traducen los errores del usecase a Problem+JSON con el codigo HTTP
// adecuado. La logica de negocio vive en application/usecases.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/units/application/dto"
	"github.com/saas-ph/api/internal/modules/units/application/usecases"
	"github.com/saas-ph/api/internal/modules/units/domain"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

type handlers struct {
	logger *slog.Logger

	createUC    *usecases.CreateUnitUseCase
	getUC       *usecases.GetUnitUseCase
	listUC      *usecases.ListUnitsUseCase
	addOwnerUC  *usecases.AddOwnerToUnitUseCase
	termOwnerUC *usecases.TerminateOwnershipUseCase
	addOccUC    *usecases.AddOccupantToUnitUseCase
	moveOutUC   *usecases.MoveOutOccupantUseCase
	peopleUC    *usecases.GetPeopleInUnitUseCase
}

func (h *handlers) createUnit(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUnitRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	out, err := h.createUC.Execute(r.Context(), usecases.CreateUnitInput{
		StructureID: req.StructureID,
		Code:        req.Code,
		Type:        req.Type,
		AreaM2:      req.AreaM2,
		Bedrooms:    req.Bedrooms,
		Coefficient: req.Coefficient,
		ActorUserID: actorFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *handlers) getUnit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := h.getUC.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) listUnits(w http.ResponseWriter, r *http.Request) {
	in := usecases.ListUnitsInput{}
	if v := r.URL.Query().Get("structure_id"); v != "" {
		in.StructureID = &v
	}
	out, err := h.listUC.Execute(r.Context(), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) addOwner(w http.ResponseWriter, r *http.Request) {
	unitID := chi.URLParam(r, "id")
	var req dto.AddOwnerRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	out, err := h.addOwnerUC.Execute(r.Context(), usecases.AddOwnerToUnitInput{
		UnitID:      unitID,
		UserID:      req.UserID,
		Percentage:  req.Percentage,
		SinceDate:   req.SinceDate,
		ActorUserID: actorFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *handlers) terminateOwner(w http.ResponseWriter, r *http.Request) {
	ownerID := chi.URLParam(r, "ownerID")
	if err := h.termOwnerUC.Execute(r.Context(), usecases.TerminateOwnershipInput{
		OwnerID:     ownerID,
		ActorUserID: actorFromCtx(r),
	}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) addOccupant(w http.ResponseWriter, r *http.Request) {
	unitID := chi.URLParam(r, "id")
	var req dto.AddOccupantRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	out, err := h.addOccUC.Execute(r.Context(), usecases.AddOccupantToUnitInput{
		UnitID:      unitID,
		UserID:      req.UserID,
		Role:        req.Role,
		IsPrimary:   req.IsPrimary,
		MoveInDate:  req.MoveInDate,
		ActorUserID: actorFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *handlers) moveOutOccupant(w http.ResponseWriter, r *http.Request) {
	occID := chi.URLParam(r, "occupancyID")
	if err := h.moveOutUC.Execute(r.Context(), usecases.MoveOutOccupantInput{
		OccupancyID: occID,
		ActorUserID: actorFromCtx(r),
	}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) peopleInUnit(w http.ResponseWriter, r *http.Request) {
	unitID := chi.URLParam(r, "id")
	out, err := h.peopleUC.Execute(r.Context(), unitID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// fail mapea errores tipados de los usecases / dominio a Problem+JSON.
func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecases.ErrInvalidInput):
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	case errors.Is(err, usecases.ErrNotFound),
		errors.Is(err, domain.ErrUnitNotFound),
		errors.Is(err, domain.ErrOwnerNotFound),
		errors.Is(err, domain.ErrOccupancyNotFound):
		apperrors.Write(w, apperrors.NotFound(err.Error()).WithInstance(r.URL.Path))
		return
	case errors.Is(err, usecases.ErrConflict),
		errors.Is(err, domain.ErrUnitCodeTaken),
		errors.Is(err, domain.ErrOwnerDuplicateActive),
		errors.Is(err, domain.ErrPrimaryOccupantConflict):
		apperrors.Write(w, apperrors.Conflict(err.Error()).WithInstance(r.URL.Path))
		return
	case errors.Is(err, usecases.ErrPolicyRejected),
		errors.Is(err, domain.ErrInvalidPercentage),
		errors.Is(err, domain.ErrInvalidUnitType),
		errors.Is(err, domain.ErrInvalidOccupancyRole):
		apperrors.Write(w, apperrors.BadRequest(err.Error()).WithInstance(r.URL.Path))
		return
	}
	if h.logger != nil {
		h.logger.ErrorContext(r.Context(), "units handler error",
			slog.String("error", err.Error()),
			slog.String("path", r.URL.Path),
		)
	}
	apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.New("invalid json body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// actorFromCtx extrae el user_id del actor del contexto del request.
// La fuente concreta es responsabilidad del middleware de auth montado
// aguas arriba. El modulo units no requiere acoplamiento directo, asi
// que reusa la misma key string que otros modulos del proyecto. Si no
// hay actor, devuelve nil (creacion via tareas internas / sistema).
func actorFromCtx(r *http.Request) *string {
	if v, ok := r.Context().Value(actorIDKey{}).(string); ok && v != "" {
		return &v
	}
	return nil
}

// actorIDKey es la clave de contexto donde el middleware aguas arriba
// publica el user_id del actor. Definida vacia aqui para no acoplar con
// otros modulos; main.go monta un middleware que la llena.
type actorIDKey struct{}
