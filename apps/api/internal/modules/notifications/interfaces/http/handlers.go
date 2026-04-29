package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/notifications/application/dto"
	"github.com/saas-ph/api/internal/modules/notifications/application/usecases"
	"github.com/saas-ph/api/internal/modules/notifications/domain"
	"github.com/saas-ph/api/internal/modules/notifications/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger          *slog.Logger
	Templates       domain.TemplateRepository
	Preferences     domain.PreferenceRepository
	Consents        domain.ConsentRepository
	PushTokens      domain.PushTokenRepository
	ProviderConfigs domain.ProviderConfigRepository
	Outbox          domain.OutboxRepository
	Deliveries      domain.DeliveryRepository
	TxRunner        usecases.TxRunner
	Now             func() time.Time
}

// validate completa los defaults razonables (logger, clock).
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

// --- Templates ---

// listTemplates GET /notifications/templates
func (h *handlers) listTemplates(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListTemplates{Templates: h.deps.Templates}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListTemplatesResponse{
		Items: make([]dto.TemplateResponse, 0, len(out)),
		Total: len(out),
	}
	for _, t := range out {
		resp.Items = append(resp.Items, templateToDTO(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

// createTemplate POST /notifications/templates
func (h *handlers) createTemplate(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateTemplateRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateTemplate{Templates: h.deps.Templates}
	out, err := uc.Execute(r.Context(), usecases.CreateTemplateInput{
		EventType:           body.EventType,
		Channel:             entities.Channel(body.Channel),
		Locale:              body.Locale,
		Subject:             body.Subject,
		BodyTemplate:        body.BodyTemplate,
		ProviderTemplateRef: body.ProviderTemplateRef,
		ActorID:             actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, templateToDTO(out))
}

// updateTemplate PATCH /notifications/templates/{id}
func (h *handlers) updateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body dto.UpdateTemplateRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.UpdateTemplate{Templates: h.deps.Templates}
	out, err := uc.Execute(r.Context(), usecases.UpdateTemplateInput{
		ID:                  id,
		Subject:             body.Subject,
		BodyTemplate:        body.BodyTemplate,
		ProviderTemplateRef: body.ProviderTemplateRef,
		ExpectedVersion:     body.Version,
		ActorID:             actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, templateToDTO(out))
}

// --- Preferences ---

// listPreferences GET /notifications/preferences
func (h *handlers) listPreferences(w http.ResponseWriter, r *http.Request) {
	actorID := actorIDFromCtx(r)
	uc := usecases.ListPreferences{Preferences: h.deps.Preferences}
	out, err := uc.Execute(r.Context(), actorID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListPreferencesResponse{
		Items: make([]dto.PreferenceResponse, 0, len(out)),
		Total: len(out),
	}
	for _, p := range out {
		resp.Items = append(resp.Items, preferenceToDTO(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

// patchPreferences PATCH /notifications/preferences
func (h *handlers) patchPreferences(w http.ResponseWriter, r *http.Request) {
	var body dto.PatchPreferencesRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	items := make([]usecases.PatchPreferenceItem, 0, len(body.Preferences))
	for _, p := range body.Preferences {
		items = append(items, usecases.PatchPreferenceItem{
			EventType: p.EventType,
			Channel:   entities.Channel(p.Channel),
			Enabled:   p.Enabled,
		})
	}
	uc := usecases.PatchPreferences{Preferences: h.deps.Preferences}
	out, err := uc.Execute(r.Context(), usecases.PatchPreferencesInput{
		UserID:      actorID,
		Preferences: items,
		ActorID:     actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListPreferencesResponse{
		Items: make([]dto.PreferenceResponse, 0, len(out)),
		Total: len(out),
	}
	for _, p := range out {
		resp.Items = append(resp.Items, preferenceToDTO(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Consents ---

// createConsent POST /notifications/consents
func (h *handlers) createConsent(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateConsentRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateConsent{Consents: h.deps.Consents}
	out, err := uc.Execute(r.Context(), usecases.CreateConsentInput{
		UserID:          actorID,
		Channel:         entities.Channel(body.Channel),
		ConsentProofURL: body.ConsentProofURL,
		LegalBasis:      body.LegalBasis,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, consentToDTO(out))
}

// revokeConsent DELETE /notifications/consents/{channel}
func (h *handlers) revokeConsent(w http.ResponseWriter, r *http.Request) {
	channel := chi.URLParam(r, "channel")
	actorID := actorIDFromCtx(r)
	uc := usecases.RevokeConsent{Consents: h.deps.Consents}
	out, err := uc.Execute(r.Context(), usecases.RevokeConsentInput{
		UserID:  actorID,
		Channel: entities.Channel(channel),
		ActorID: actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, consentToDTO(out))
}

// --- Push Tokens ---

// createPushToken POST /notifications/push-tokens
func (h *handlers) createPushToken(w http.ResponseWriter, r *http.Request) {
	var body dto.CreatePushTokenRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreatePushToken{PushTokens: h.deps.PushTokens}
	out, err := uc.Execute(r.Context(), usecases.CreatePushTokenInput{
		UserID:   actorID,
		Platform: entities.Platform(body.Platform),
		Token:    body.Token,
		ActorID:  actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, pushTokenToDTO(out))
}

// deletePushToken DELETE /notifications/push-tokens/{id}
func (h *handlers) deletePushToken(w http.ResponseWriter, r *http.Request) {
	tokenID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.DeletePushToken{PushTokens: h.deps.PushTokens}
	if err := uc.Execute(r.Context(), usecases.DeletePushTokenInput{
		TokenID: tokenID,
		ActorID: actorID,
	}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Provider Configs ---

// listProviderConfigs GET /notifications/provider-configs
func (h *handlers) listProviderConfigs(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListProviderConfigs{ProviderConfigs: h.deps.ProviderConfigs}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListProviderConfigsResponse{
		Items: make([]dto.ProviderConfigResponse, 0, len(out)),
		Total: len(out),
	}
	for _, c := range out {
		resp.Items = append(resp.Items, providerConfigToDTO(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// updateProviderConfig PATCH /notifications/provider-configs
func (h *handlers) updateProviderConfig(w http.ResponseWriter, r *http.Request) {
	var body dto.PatchProviderConfigRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.UpdateProviderConfig{ProviderConfigs: h.deps.ProviderConfigs}
	out, err := uc.Execute(r.Context(), usecases.UpdateProviderConfigInput{
		ID:              body.ID,
		Config:          body.Config,
		IsActive:        body.IsActive,
		ExpectedVersion: body.Version,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, providerConfigToDTO(out))
}

// --- Broadcast ---

// broadcast POST /notifications/broadcast
func (h *handlers) broadcast(w http.ResponseWriter, r *http.Request) {
	var body dto.BroadcastRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)

	channels := make([]entities.Channel, 0, len(body.Channels))
	for _, ch := range body.Channels {
		channels = append(channels, entities.Channel(ch))
	}

	uc := usecases.Broadcast{
		Outbox:      h.deps.Outbox,
		Preferences: h.deps.Preferences,
		Consents:    h.deps.Consents,
		TxRunner:    h.deps.TxRunner,
		Now:         h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.BroadcastInput{
		EventType:      body.EventType,
		Channels:       channels,
		RecipientIDs:   body.RecipientIDs,
		Payload:        body.Payload,
		IdempotencyKey: body.IdempotencyKey,
		ActorID:        actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, dto.BroadcastResponse{
		Queued:  out.Queued,
		Skipped: out.Skipped,
		Blocked: out.Blocked,
	})
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "notifications: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "notifications: unexpected error",
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

// --- Entity-to-DTO mapping functions ---

func templateToDTO(t entities.NotificationTemplate) dto.TemplateResponse {
	return dto.TemplateResponse{
		ID:                  t.ID,
		EventType:           t.EventType,
		Channel:             string(t.Channel),
		Locale:              t.Locale,
		Subject:             t.Subject,
		BodyTemplate:        t.BodyTemplate,
		ProviderTemplateRef: t.ProviderTemplateRef,
		Status:              string(t.Status),
		CreatedAt:           dto.FormatTime(t.CreatedAt),
		UpdatedAt:           dto.FormatTime(t.UpdatedAt),
		Version:             t.Version,
	}
}

func preferenceToDTO(p entities.NotificationPreference) dto.PreferenceResponse {
	return dto.PreferenceResponse{
		ID:        p.ID,
		UserID:    p.UserID,
		EventType: p.EventType,
		Channel:   string(p.Channel),
		Enabled:   p.Enabled,
		Status:    string(p.Status),
		CreatedAt: dto.FormatTime(p.CreatedAt),
		UpdatedAt: dto.FormatTime(p.UpdatedAt),
		Version:   p.Version,
	}
}

func consentToDTO(c entities.NotificationConsent) dto.ConsentResponse {
	return dto.ConsentResponse{
		ID:              c.ID,
		UserID:          c.UserID,
		Channel:         string(c.Channel),
		ConsentedAt:     dto.FormatTime(c.ConsentedAt),
		RevokedAt:       dto.FormatTimePtr(c.RevokedAt),
		ConsentProofURL: c.ConsentProofURL,
		LegalBasis:      c.LegalBasis,
		Status:          string(c.Status),
		CreatedAt:       dto.FormatTime(c.CreatedAt),
		UpdatedAt:       dto.FormatTime(c.UpdatedAt),
		Version:         c.Version,
	}
}

func pushTokenToDTO(t entities.NotificationPushToken) dto.PushTokenResponse {
	return dto.PushTokenResponse{
		ID:         t.ID,
		UserID:     t.UserID,
		Platform:   string(t.Platform),
		Token:      t.Token,
		LastSeenAt: dto.FormatTime(t.LastSeenAt),
		Status:     string(t.Status),
		CreatedAt:  dto.FormatTime(t.CreatedAt),
		UpdatedAt:  dto.FormatTime(t.UpdatedAt),
		Version:    t.Version,
	}
}

func providerConfigToDTO(c entities.NotificationProviderConfig) dto.ProviderConfigResponse {
	return dto.ProviderConfigResponse{
		ID:           c.ID,
		Channel:      string(c.Channel),
		ProviderName: c.ProviderName,
		IsActive:     c.IsActive,
		Status:       c.Status,
		CreatedAt:    dto.FormatTime(c.CreatedAt),
		UpdatedAt:    dto.FormatTime(c.UpdatedAt),
		Version:      c.Version,
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
