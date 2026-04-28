package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/packages/application/dto"
	"github.com/saas-ph/api/internal/modules/packages/application/usecases"
	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger      *slog.Logger
	Packages    domain.PackageRepository
	Categories  domain.CategoryRepository
	Deliveries  domain.DeliveryRepository
	Outbox      domain.OutboxRepository
	TxRunner    usecases.TxRunner
	Idempotency *usecases.IdempotencyCache
	Now         func() time.Time
}

// validate completa los defaults razonables (logger, clock,
// idempotency).
func (d *Dependencies) validate() {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Idempotency == nil {
		d.Idempotency = usecases.NewIdempotencyCache(24*time.Hour, d.Now)
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

// --- Packages ---

// createPackage POST /packages
func (h *handlers) createPackage(w http.ResponseWriter, r *http.Request) {
	var body dto.CreatePackageRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	receivedBy := body.ReceivedByUserID
	if receivedBy == "" {
		receivedBy = actorIDFromCtx(r)
	}
	uc := usecases.CreatePackage{
		Packages:   h.deps.Packages,
		Categories: h.deps.Categories,
		Outbox:     h.deps.Outbox,
		TxRunner:   h.deps.TxRunner,
		Now:        h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CreatePackageInput{
		UnitID:              body.UnitID,
		RecipientName:       body.RecipientName,
		CategoryID:          body.CategoryID,
		CategoryName:        body.CategoryName,
		ReceivedEvidenceURL: body.ReceivedEvidenceURL,
		Carrier:             body.Carrier,
		TrackingNumber:      body.TrackingNumber,
		ReceivedByUserID:    receivedBy,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, packageToDTO(out))
}

// listPackages GET /packages?unit_id=&status=
func (h *handlers) listPackages(w http.ResponseWriter, r *http.Request) {
	unitID := r.URL.Query().Get("unit_id")
	status := r.URL.Query().Get("status")
	if unitID == "" && status == "" {
		h.fail(w, r, apperrors.BadRequest("provide unit_id or status query parameter"))
		return
	}

	var pkgs []entities.Package
	var err error
	if unitID != "" {
		uc := usecases.ListPackagesByUnit{Packages: h.deps.Packages}
		pkgs, err = uc.Execute(r.Context(), unitID)
	} else {
		uc := usecases.ListPackagesByStatus{Packages: h.deps.Packages}
		pkgs, err = uc.Execute(r.Context(), entities.PackageStatus(status))
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListPackagesResponse{
		Items: make([]dto.PackageResponse, 0, len(pkgs)),
		Total: len(pkgs),
	}
	for _, p := range pkgs {
		resp.Items = append(resp.Items, packageToDTO(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

// getPackage GET /packages/{id}
func (h *handlers) getPackage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetPackage{Packages: h.deps.Packages}
	out, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, packageToDTO(out))
}

// deliverByQR POST /packages/{id}/deliver-by-qr
func (h *handlers) deliverByQR(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.DeliverByQRRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	guardID := body.GuardID
	if guardID == "" {
		guardID = actorIDFromCtx(r)
	}
	idem := ""
	if body.IdempotencyKey != nil {
		idem = *body.IdempotencyKey
	}
	uc := usecases.DeliverByQR{
		Packages:    h.deps.Packages,
		Deliveries:  h.deps.Deliveries,
		Outbox:      h.deps.Outbox,
		TxRunner:    h.deps.TxRunner,
		Idempotency: h.deps.Idempotency,
	}
	out, err := uc.Execute(r.Context(), usecases.DeliverByQRInput{
		PackageID:         id,
		DeliveredToUserID: body.DeliveredToUserID,
		GuardID:           guardID,
		IdempotencyKey:    idem,
		Notes:             body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.DeliverResponse{
		Package: packageToDTO(out.Package),
		Event:   eventToDTO(out.Event),
	})
}

// deliverManual POST /packages/{id}/deliver-manual
func (h *handlers) deliverManual(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.DeliverManualRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	guardID := body.GuardID
	if guardID == "" {
		guardID = actorIDFromCtx(r)
	}
	idem := ""
	if body.IdempotencyKey != nil {
		idem = *body.IdempotencyKey
	}
	uc := usecases.DeliverManual{
		Packages:    h.deps.Packages,
		Deliveries:  h.deps.Deliveries,
		Outbox:      h.deps.Outbox,
		TxRunner:    h.deps.TxRunner,
		Idempotency: h.deps.Idempotency,
	}
	out, err := uc.Execute(r.Context(), usecases.DeliverManualInput{
		PackageID:           id,
		RecipientNameManual: body.RecipientNameManual,
		SignatureURL:        body.SignatureURL,
		PhotoEvidenceURL:    body.PhotoEvidenceURL,
		GuardID:             guardID,
		IdempotencyKey:      idem,
		Notes:               body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.DeliverResponse{
		Package: packageToDTO(out.Package),
		Event:   eventToDTO(out.Event),
	})
}

// returnPackage POST /packages/{id}/return
func (h *handlers) returnPackage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.ReturnPackageRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	guardID := body.GuardID
	if guardID == "" {
		guardID = actorIDFromCtx(r)
	}
	uc := usecases.ReturnPackage{
		Packages: h.deps.Packages,
		Outbox:   h.deps.Outbox,
		TxRunner: h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.ReturnPackageInput{
		PackageID: id,
		GuardID:   guardID,
		Notes:     body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, packageToDTO(out))
}

// listCategories GET /package-categories
func (h *handlers) listCategories(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListCategories{Categories: h.deps.Categories}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListCategoriesResponse{
		Items: make([]dto.CategoryResponse, 0, len(out)),
		Total: len(out),
	}
	for _, c := range out {
		resp.Items = append(resp.Items, dto.CategoryResponse{
			ID:               c.ID,
			Name:             c.Name,
			RequiresEvidence: c.RequiresEvidence,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "packages: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "packages: unexpected error",
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

func packageToDTO(p entities.Package) dto.PackageResponse {
	return dto.PackageResponse{
		ID:                  p.ID,
		UnitID:              p.UnitID,
		RecipientName:       p.RecipientName,
		CategoryID:          p.CategoryID,
		ReceivedEvidenceURL: p.ReceivedEvidenceURL,
		Carrier:             p.Carrier,
		TrackingNumber:      p.TrackingNumber,
		ReceivedByUserID:    p.ReceivedByUserID,
		ReceivedAt:          p.ReceivedAt,
		DeliveredAt:         p.DeliveredAt,
		ReturnedAt:          p.ReturnedAt,
		Status:              string(p.Status),
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
		Version:             p.Version,
	}
}

func eventToDTO(e entities.DeliveryEvent) dto.DeliveryEventResponse {
	return dto.DeliveryEventResponse{
		ID:                  e.ID,
		PackageID:           e.PackageID,
		DeliveredToUserID:   e.DeliveredToUserID,
		RecipientNameManual: e.RecipientNameManual,
		DeliveryMethod:      string(e.DeliveryMethod),
		SignatureURL:        e.SignatureURL,
		PhotoEvidenceURL:    e.PhotoEvidenceURL,
		DeliveredByUserID:   e.DeliveredByUserID,
		DeliveredAt:         e.DeliveredAt,
		Notes:               e.Notes,
		Status:              string(e.Status),
	}
}

// actorCtxKey clave de contexto para el actor (user_id) que origina la
// peticion.
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
