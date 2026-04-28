package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/access_control/application/dto"
	"github.com/saas-ph/api/internal/modules/access_control/application/usecases"
	"github.com/saas-ph/api/internal/modules/access_control/domain"
	"github.com/saas-ph/api/internal/modules/access_control/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger        *slog.Logger
	BlacklistRepo domain.BlacklistRepository
	PreRegRepo    domain.PreRegistrationRepository
	EntryRepo     domain.VisitorEntryRepository
	Now           func() time.Time
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

// --- Pre-registrations ---

// createPreRegistration POST /visitor-preregistrations
func (h *handlers) createPreRegistration(w http.ResponseWriter, r *http.Request) {
	var body dto.CreatePreRegistrationRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.CreatePreRegistration{Repo: h.deps.PreRegRepo, Now: h.deps.Now}
	out, err := uc.Execute(r.Context(), usecases.CreatePreRegistrationInput{
		UnitID:                body.UnitID,
		CreatedByUserID:       actorIDFromCtx(r),
		VisitorFullName:       body.VisitorFullName,
		VisitorDocumentType:   body.VisitorDocumentType,
		VisitorDocumentNumber: body.VisitorDocumentNumber,
		ExpectedAt:            body.ExpectedAt,
		ExpiresAt:             body.ExpiresAt,
		MaxUses:               body.MaxUses,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.CreatePreRegistrationResponse{
		ID:        out.Entity.ID,
		QRCode:    out.QRCode,
		ExpiresAt: out.Entity.ExpiresAt,
		MaxUses:   out.Entity.MaxUses,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// --- Visits ---

// checkinByQR POST /visits/checkin-by-qr
func (h *handlers) checkinByQR(w http.ResponseWriter, r *http.Request) {
	var body dto.CheckinByQRRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	guardID := body.GuardID
	if guardID == "" {
		guardID = actorIDFromCtx(r)
	}
	uc := usecases.CheckinByQR{
		PreRegRepo:    h.deps.PreRegRepo,
		BlacklistRepo: h.deps.BlacklistRepo,
		EntryRepo:     h.deps.EntryRepo,
	}
	entry, err := uc.Execute(r.Context(), usecases.CheckinByQRInput{
		QRCode:   body.QRCode,
		GuardID:  guardID,
		PhotoURL: body.PhotoURL,
		Notes:    body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, entryToDTO(entry))
}

// checkinManual POST /visits/checkin-manual
func (h *handlers) checkinManual(w http.ResponseWriter, r *http.Request) {
	var body dto.CheckinManualRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	guardID := body.GuardID
	if guardID == "" {
		guardID = actorIDFromCtx(r)
	}
	uc := usecases.CheckinManual{
		BlacklistRepo: h.deps.BlacklistRepo,
		EntryRepo:     h.deps.EntryRepo,
	}
	entry, err := uc.Execute(r.Context(), usecases.CheckinManualInput{
		UnitID:                body.UnitID,
		VisitorFullName:       body.VisitorFullName,
		VisitorDocumentType:   body.VisitorDocumentType,
		VisitorDocumentNumber: body.VisitorDocumentNumber,
		PhotoURL:              body.PhotoURL,
		GuardID:               guardID,
		Notes:                 body.Notes,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, entryToDTO(entry))
}

// checkout POST /visits/{id}/checkout
func (h *handlers) checkout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.Checkout{EntryRepo: h.deps.EntryRepo}
	entry, err := uc.Execute(r.Context(), usecases.CheckoutInput{
		EntryID: id,
		ActorID: actorIDFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, entryToDTO(entry))
}

// listActive GET /visits/active
func (h *handlers) listActive(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListActiveVisits{EntryRepo: h.deps.EntryRepo}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListVisitorEntriesResponse{
		Items: make([]dto.VisitorEntryResponse, 0, len(out)),
		Total: len(out),
	}
	for _, e := range out {
		resp.Items = append(resp.Items, entryToDTO(e))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Blacklist ---

// createBlacklist POST /blacklist
func (h *handlers) createBlacklist(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateBlacklistRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.CreateBlacklistEntry{Repo: h.deps.BlacklistRepo}
	out, err := uc.Execute(r.Context(), usecases.CreateBlacklistInput{
		DocumentType:     entities.DocumentType(body.DocumentType),
		DocumentNumber:   body.DocumentNumber,
		FullName:         body.FullName,
		Reason:           body.Reason,
		ReportedByUnitID: body.ReportedByUnitID,
		ExpiresAt:        body.ExpiresAt,
		ActorID:          actorIDFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, blacklistToDTO(out))
}

// listBlacklist GET /blacklist
func (h *handlers) listBlacklist(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListBlacklist{Repo: h.deps.BlacklistRepo}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListBlacklistResponse{
		Items: make([]dto.BlacklistResponse, 0, len(out)),
		Total: len(out),
	}
	for _, b := range out {
		resp.Items = append(resp.Items, blacklistToDTO(b))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		// Log especifico para 403 blacklist (auditoria del intento).
		if p.Status == http.StatusForbidden {
			h.deps.Logger.WarnContext(r.Context(), "access_control: blacklisted attempt",
				slog.String("path", r.URL.Path),
				slog.String("detail", p.Detail))
		}
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "access_control: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "access_control: unexpected error",
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

func entryToDTO(e entities.VisitorEntry) dto.VisitorEntryResponse {
	return dto.VisitorEntryResponse{
		ID:                    e.ID,
		UnitID:                e.UnitID,
		PreRegistrationID:     e.PreRegistrationID,
		VisitorFullName:       e.VisitorFullName,
		VisitorDocumentType:   e.VisitorDocumentType,
		VisitorDocumentNumber: e.VisitorDocumentNumber,
		PhotoURL:              e.PhotoURL,
		GuardID:               e.GuardID,
		EntryTime:             e.EntryTime,
		ExitTime:              e.ExitTime,
		Source:                string(e.Source),
		Notes:                 e.Notes,
		Status:                string(e.Status),
		CreatedAt:             e.CreatedAt,
		UpdatedAt:             e.UpdatedAt,
		Version:               e.Version,
	}
}

func blacklistToDTO(b entities.BlacklistEntry) dto.BlacklistResponse {
	return dto.BlacklistResponse{
		ID:               b.ID,
		DocumentType:     string(b.DocumentType),
		DocumentNumber:   b.DocumentNumber,
		FullName:         b.FullName,
		Reason:           b.Reason,
		ReportedByUnitID: b.ReportedByUnitID,
		ReportedByUserID: b.ReportedByUserID,
		ExpiresAt:        b.ExpiresAt,
		Status:           string(b.Status),
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
		Version:          b.Version,
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
