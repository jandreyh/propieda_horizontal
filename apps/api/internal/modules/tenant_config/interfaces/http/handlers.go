// Package http contiene los adaptadores HTTP del modulo tenant_config.
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
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/tenant_config/application/dto"
	"github.com/saas-ph/api/internal/modules/tenant_config/application/usecases"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain"
	"github.com/saas-ph/api/internal/modules/tenant_config/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger       *slog.Logger
	SettingsRepo domain.SettingsRepository
	BrandingRepo domain.BrandingRepository
	Now          func() time.Time
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

// --- Settings ---

// listSettings GET /settings
func (h *handlers) listSettings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := parseInt32(q.Get("limit"), 0)
	offset := parseInt32(q.Get("offset"), 0)
	category := q.Get("category")

	uc := usecases.ListSettings{Repo: h.deps.SettingsRepo}
	out, err := uc.Execute(r.Context(), usecases.ListInput{
		Category: category,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	resp := dto.ListSettingsResponse{
		Items:  make([]dto.SettingResponse, 0, len(out.Items)),
		Total:  out.Total,
		Limit:  out.Limit,
		Offset: out.Offset,
	}
	for _, s := range out.Items {
		resp.Items = append(resp.Items, settingToDTO(s))
	}
	writeJSON(w, http.StatusOK, resp)
}

// getSetting GET /settings/{key}
func (h *handlers) getSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	uc := usecases.GetSetting{Repo: h.deps.SettingsRepo}
	s, err := uc.Execute(r.Context(), key)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, settingToDTO(s))
}

// putSetting PUT /settings/{key}
func (h *handlers) putSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var body dto.UpsertSettingRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	if len(body.Value) == 0 {
		h.fail(w, r, apperrors.BadRequest("value is required"))
		return
	}
	if !json.Valid(body.Value) {
		h.fail(w, r, apperrors.BadRequest("value must be valid JSON"))
		return
	}
	uc := usecases.SetSetting{Repo: h.deps.SettingsRepo}
	s, err := uc.Execute(r.Context(), usecases.SetInput{
		Key:         key,
		Value:       body.Value,
		Description: body.Description,
		Category:    body.Category,
		ActorID:     actorIDFromCtx(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, settingToDTO(s))
}

// deleteSetting DELETE /settings/{key}
func (h *handlers) deleteSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	uc := usecases.ArchiveSetting{Repo: h.deps.SettingsRepo}
	if _, err := uc.Execute(r.Context(), usecases.ArchiveInput{
		Key:     key,
		ActorID: actorIDFromCtx(r),
	}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Branding ---

// getBranding GET /branding
func (h *handlers) getBranding(w http.ResponseWriter, r *http.Request) {
	uc := usecases.GetBranding{Repo: h.deps.BrandingRepo}
	b, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, brandingToDTO(b))
}

// putBranding PUT /branding
func (h *handlers) putBranding(w http.ResponseWriter, r *http.Request) {
	var body dto.UpdateBrandingRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	uc := usecases.UpdateBranding{Repo: h.deps.BrandingRepo}
	b, err := uc.Execute(r.Context(), usecases.UpdateBrandingInput{
		DisplayName:     body.DisplayName,
		LogoURL:         body.LogoURL,
		PrimaryColor:    body.PrimaryColor,
		SecondaryColor:  body.SecondaryColor,
		Timezone:        body.Timezone,
		Locale:          body.Locale,
		ActorID:         actorIDFromCtx(r),
		ExpectedVersion: body.ExpectedVersion,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, brandingToDTO(b))
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "tenant_config: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "tenant_config: unexpected error",
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

func parseInt32(s string, def int32) int32 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return def
	}
	return int32(n)
}

func settingToDTO(s entities.Setting) dto.SettingResponse {
	return dto.SettingResponse{
		ID:          s.ID,
		Key:         s.Key,
		Value:       json.RawMessage(s.Value),
		Description: s.Description,
		Category:    s.Category,
		Status:      string(s.Status),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Version:     s.Version,
	}
}

func brandingToDTO(b entities.Branding) dto.BrandingResponse {
	return dto.BrandingResponse{
		ID:             b.ID,
		DisplayName:    b.DisplayName,
		LogoURL:        b.LogoURL,
		PrimaryColor:   b.PrimaryColor,
		SecondaryColor: b.SecondaryColor,
		Timezone:       b.Timezone,
		Locale:         b.Locale,
		Status:         string(b.Status),
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
		Version:        b.Version,
	}
}

// actorIDFromCtx intenta extraer el user_id del contexto. El modulo no
// importa authentication; un middleware externo (auth) puede colocar el
// user_id como string en una clave conocida. Para mantener desacople, en
// MVP devolvemos "" (sistema) si no esta presente.
type actorCtxKey struct{}

// WithActorID es helper para inyectar el actor desde un middleware
// externo (test o capa auth). Vive aqui para no acoplar a un paquete
// authentication especifico.
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
