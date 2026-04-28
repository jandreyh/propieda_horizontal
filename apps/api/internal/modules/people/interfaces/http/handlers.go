// Package http contiene los adaptadores HTTP del modulo people.
//
// Los handlers traducen request/response al usecase correspondiente y
// emiten errores RFC 7807 via apperrors. NO contienen logica de negocio.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/people/application/dto"
	"github.com/saas-ph/api/internal/modules/people/application/usecases"
	"github.com/saas-ph/api/internal/modules/people/domain"
	"github.com/saas-ph/api/internal/modules/people/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger         *slog.Logger
	VehicleRepo    domain.VehicleRepository
	AssignmentRepo domain.AssignmentRepository
	Now            func() time.Time
}

// validate completa los defaults razonables (slogger, clock).
func (d *Dependencies) validate() {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
}

// handlers agrupa los handlers HTTP construidos a partir de Dependencies.
type handlers struct {
	deps Dependencies
}

func newHandlers(d Dependencies) *handlers {
	d.validate()
	return &handlers{deps: d}
}

// --- Vehicles ---

// createVehicle POST /vehicles
func (h *handlers) createVehicle(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateVehicleRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.CreateVehicle{Repo: h.deps.VehicleRepo}
	v, err := uc.Execute(r.Context(), usecases.CreateVehicleInput{
		Plate:   body.Plate,
		Type:    body.Type,
		Brand:   body.Brand,
		Model:   body.Model,
		Color:   body.Color,
		Year:    body.Year,
		ActorID: actorIDFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, vehicleToDTO(v))
}

// getVehicle GET /vehicles/{id}
func (h *handlers) getVehicle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetVehicle{Repo: h.deps.VehicleRepo}
	v, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, vehicleToDTO(v))
}

// listVehicles GET /vehicles?plate=
//
// Si `plate` esta presente, devuelve el vehiculo unico con esa placa
// (envuelto en items para preservar la forma del listado).
// Si no, lista todos los vehiculos.
func (h *handlers) listVehicles(w http.ResponseWriter, r *http.Request) {
	plate := r.URL.Query().Get("plate")
	if plate != "" {
		uc := usecases.GetVehicleByPlate{Repo: h.deps.VehicleRepo}
		v, err := uc.Execute(r.Context(), plate)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, dto.ListVehiclesResponse{
			Items: []dto.VehicleResponse{vehicleToDTO(v)},
			Total: 1,
		})
		return
	}

	vehicles, err := h.deps.VehicleRepo.ListAll(r.Context())
	if err != nil {
		h.fail(w, r, apperrors.Internal("failed to list vehicles"))
		return
	}
	resp := dto.ListVehiclesResponse{
		Items: make([]dto.VehicleResponse, 0, len(vehicles)),
		Total: len(vehicles),
	}
	for _, v := range vehicles {
		resp.Items = append(resp.Items, vehicleToDTO(v))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Assignments ---

// assignVehicle POST /units/{unitID}/vehicles
func (h *handlers) assignVehicle(w http.ResponseWriter, r *http.Request) {
	unitID := chi.URLParam(r, "unitID")
	var body dto.AssignVehicleRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.AssignVehicleToUnit{Repo: h.deps.AssignmentRepo}
	a, err := uc.Execute(r.Context(), usecases.AssignVehicleInput{
		UnitID:    unitID,
		VehicleID: body.VehicleID,
		SinceDate: body.SinceDate,
		ActorID:   actorIDFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, assignmentToDTO(a))
}

// listVehiclesForUnit GET /units/{unitID}/vehicles
func (h *handlers) listVehiclesForUnit(w http.ResponseWriter, r *http.Request) {
	unitID := chi.URLParam(r, "unitID")
	uc := usecases.ListActiveVehiclesForUnit{Repo: h.deps.AssignmentRepo}
	out, err := uc.Execute(r.Context(), unitID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListAssignmentsResponse{
		Items: make([]dto.AssignmentResponse, 0, len(out)),
		Total: len(out),
	}
	for _, a := range out {
		resp.Items = append(resp.Items, assignmentToDTO(a))
	}
	writeJSON(w, http.StatusOK, resp)
}

// endAssignment DELETE /units/{unitID}/vehicles/{assignmentID}
//
// Cierra (soft-end) la asignacion fijando until_date = today.
func (h *handlers) endAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "assignmentID")
	uc := usecases.EndAssignment{Repo: h.deps.AssignmentRepo, Now: h.deps.Now}
	if _, err := uc.Execute(r.Context(), usecases.EndAssignmentInput{
		AssignmentID: assignmentID,
		ActorID:      actorIDFromCtx(r),
	}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "people: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "people: unexpected error",
		slog.String("path", r.URL.Path),
		slog.String("err", err.Error()))
	apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return apperrors.BadRequest("invalid JSON body: " + err.Error())
	}
	return nil
}

func vehicleToDTO(v entities.Vehicle) dto.VehicleResponse {
	return dto.VehicleResponse{
		ID:        v.ID,
		Plate:     v.Plate,
		Type:      string(v.Type),
		Brand:     v.Brand,
		Model:     v.Model,
		Color:     v.Color,
		Year:      v.Year,
		Status:    string(v.Status),
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
		Version:   v.Version,
	}
}

func assignmentToDTO(a entities.UnitVehicleAssignment) dto.AssignmentResponse {
	out := dto.AssignmentResponse{
		ID:        a.ID,
		UnitID:    a.UnitID,
		VehicleID: a.VehicleID,
		SinceDate: a.SinceDate,
		UntilDate: a.UntilDate,
		Status:    string(a.Status),
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
		Version:   a.Version,
	}
	if a.Vehicle != nil {
		v := vehicleToDTO(*a.Vehicle)
		out.Vehicle = &v
	}
	return out
}

// actorCtxKey clave de contexto para el actor (user_id) que origina la
// peticion. La inyeccion la hace un middleware externo (auth); el modulo
// no se acopla a un paquete authentication especifico.
type actorCtxKey struct{}

// WithActorID es helper para inyectar el actor desde un middleware
// externo (test o capa auth).
func WithActorID(r *http.Request, actorID string) *http.Request {
	if actorID == "" {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), actorCtxKey{}, actorID))
}

func actorIDFromCtx(r *http.Request) string {
	if v, ok := r.Context().Value(actorCtxKey{}).(string); ok {
		return v
	}
	return ""
}
