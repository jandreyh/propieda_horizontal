// Package usecases orquesta la logica de aplicacion del modulo notifications.
// Cada usecase recibe sus dependencias por inyeccion (interfaces) y NO
// conoce HTTP ni la base.
package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/saas-ph/api/internal/modules/notifications/domain"
	"github.com/saas-ph/api/internal/modules/notifications/domain/entities"
	"github.com/saas-ph/api/internal/modules/notifications/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// ---------------------------------------------------------------------------
// ListTemplates
// ---------------------------------------------------------------------------

// ListTemplates lista las plantillas activas.
type ListTemplates struct {
	Templates domain.TemplateRepository
}

// Execute delega al repo.
func (u ListTemplates) Execute(ctx context.Context) ([]entities.NotificationTemplate, error) {
	out, err := u.Templates.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list templates")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// CreateTemplate
// ---------------------------------------------------------------------------

// CreateTemplate crea una plantilla de notificacion nueva.
type CreateTemplate struct {
	Templates domain.TemplateRepository
}

// CreateTemplateInput es el input del usecase.
type CreateTemplateInput struct {
	EventType           string
	Channel             entities.Channel
	Locale              string
	Subject             *string
	BodyTemplate        string
	ProviderTemplateRef *string
	ActorID             string
}

// Execute valida y delega al repo.
func (u CreateTemplate) Execute(ctx context.Context, in CreateTemplateInput) (entities.NotificationTemplate, error) {
	if err := policies.ValidateEventType(in.EventType); err != nil {
		return entities.NotificationTemplate{}, apperrors.BadRequest("event_type: " + err.Error())
	}
	if err := policies.ValidateChannel(in.Channel); err != nil {
		return entities.NotificationTemplate{}, apperrors.BadRequest("channel: " + err.Error())
	}
	if err := policies.ValidateLocale(in.Locale); err != nil {
		return entities.NotificationTemplate{}, apperrors.BadRequest("locale: " + err.Error())
	}
	if err := policies.ValidateBodyTemplate(in.BodyTemplate); err != nil {
		return entities.NotificationTemplate{}, apperrors.BadRequest("body_template: " + err.Error())
	}

	tmpl, err := u.Templates.Create(ctx, domain.CreateTemplateInput{
		EventType:           in.EventType,
		Channel:             in.Channel,
		Locale:              in.Locale,
		Subject:             in.Subject,
		BodyTemplate:        in.BodyTemplate,
		ProviderTemplateRef: in.ProviderTemplateRef,
		ActorID:             in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTemplateDuplicate) {
			return entities.NotificationTemplate{}, apperrors.Conflict("template already exists for event/channel/locale")
		}
		return entities.NotificationTemplate{}, apperrors.Internal("failed to create template")
	}
	return tmpl, nil
}

// ---------------------------------------------------------------------------
// UpdateTemplate
// ---------------------------------------------------------------------------

// UpdateTemplate actualiza una plantilla existente.
type UpdateTemplate struct {
	Templates domain.TemplateRepository
}

// UpdateTemplateInput es el input del usecase.
type UpdateTemplateInput struct {
	ID                  string
	Subject             *string
	BodyTemplate        string
	ProviderTemplateRef *string
	ExpectedVersion     int32
	ActorID             string
}

// Execute valida y delega al repo.
func (u UpdateTemplate) Execute(ctx context.Context, in UpdateTemplateInput) (entities.NotificationTemplate, error) {
	if err := policies.ValidateUUID(in.ID); err != nil {
		return entities.NotificationTemplate{}, apperrors.BadRequest("id: " + err.Error())
	}
	if err := policies.ValidateBodyTemplate(in.BodyTemplate); err != nil {
		return entities.NotificationTemplate{}, apperrors.BadRequest("body_template: " + err.Error())
	}

	tmpl, err := u.Templates.Update(ctx, domain.UpdateTemplateInput{
		ID:                  in.ID,
		Subject:             in.Subject,
		BodyTemplate:        in.BodyTemplate,
		ProviderTemplateRef: in.ProviderTemplateRef,
		ExpectedVersion:     in.ExpectedVersion,
		ActorID:             in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTemplateNotFound):
			return entities.NotificationTemplate{}, apperrors.NotFound("template not found")
		case errors.Is(err, domain.ErrVersionConflict):
			return entities.NotificationTemplate{}, mapVersionConflict()
		default:
			return entities.NotificationTemplate{}, apperrors.Internal("failed to update template")
		}
	}
	return tmpl, nil
}

// ---------------------------------------------------------------------------
// ListPreferences
// ---------------------------------------------------------------------------

// ListPreferences lista las preferencias de un usuario.
type ListPreferences struct {
	Preferences domain.PreferenceRepository
}

// Execute delega al repo.
func (u ListPreferences) Execute(ctx context.Context, userID string) ([]entities.NotificationPreference, error) {
	if err := policies.ValidateUUID(userID); err != nil {
		return nil, apperrors.BadRequest("user_id: " + err.Error())
	}
	out, err := u.Preferences.ListByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to list preferences")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// PatchPreferences
// ---------------------------------------------------------------------------

// PatchPreferences actualiza multiples preferencias de un usuario.
type PatchPreferences struct {
	Preferences domain.PreferenceRepository
}

// PatchPreferenceItem es un item individual de la peticion.
type PatchPreferenceItem struct {
	EventType string
	Channel   entities.Channel
	Enabled   bool
}

// PatchPreferencesInput es el input del usecase.
type PatchPreferencesInput struct {
	UserID      string
	Preferences []PatchPreferenceItem
	ActorID     string
}

// Execute valida y delega.
func (u PatchPreferences) Execute(ctx context.Context, in PatchPreferencesInput) ([]entities.NotificationPreference, error) {
	if err := policies.ValidateUUID(in.UserID); err != nil {
		return nil, apperrors.BadRequest("user_id: " + err.Error())
	}
	if len(in.Preferences) == 0 {
		return nil, apperrors.BadRequest("preferences must not be empty")
	}

	for _, p := range in.Preferences {
		if err := policies.ValidateEventType(p.EventType); err != nil {
			return nil, apperrors.BadRequest("event_type: " + err.Error())
		}
		if err := policies.ValidateChannel(p.Channel); err != nil {
			return nil, apperrors.BadRequest("channel: " + err.Error())
		}
	}

	results := make([]entities.NotificationPreference, 0, len(in.Preferences))
	for _, p := range in.Preferences {
		pref, err := u.Preferences.Upsert(ctx, domain.UpsertPreferenceInput{
			UserID:    in.UserID,
			EventType: p.EventType,
			Channel:   p.Channel,
			Enabled:   p.Enabled,
			ActorID:   in.ActorID,
		})
		if err != nil {
			return nil, apperrors.Internal("failed to upsert preference")
		}
		results = append(results, pref)
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// CreateConsent
// ---------------------------------------------------------------------------

// CreateConsent registra un consentimiento de canal.
type CreateConsent struct {
	Consents domain.ConsentRepository
}

// CreateConsentInput es el input del usecase.
type CreateConsentInput struct {
	UserID          string
	Channel         entities.Channel
	ConsentProofURL *string
	LegalBasis      *string
	ActorID         string
}

// Execute valida y delega.
func (u CreateConsent) Execute(ctx context.Context, in CreateConsentInput) (entities.NotificationConsent, error) {
	if err := policies.ValidateUUID(in.UserID); err != nil {
		return entities.NotificationConsent{}, apperrors.BadRequest("user_id: " + err.Error())
	}
	if err := policies.ValidateChannel(in.Channel); err != nil {
		return entities.NotificationConsent{}, apperrors.BadRequest("channel: " + err.Error())
	}

	consent, err := u.Consents.Create(ctx, domain.CreateConsentInput{
		UserID:          in.UserID,
		Channel:         in.Channel,
		ConsentProofURL: in.ConsentProofURL,
		LegalBasis:      in.LegalBasis,
		ActorID:         in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrConsentDuplicate) {
			return entities.NotificationConsent{}, apperrors.Conflict("consent already exists for this channel")
		}
		return entities.NotificationConsent{}, apperrors.Internal("failed to create consent")
	}
	return consent, nil
}

// ---------------------------------------------------------------------------
// RevokeConsent
// ---------------------------------------------------------------------------

// RevokeConsent revoca un consentimiento por canal.
type RevokeConsent struct {
	Consents domain.ConsentRepository
}

// RevokeConsentInput es el input del usecase.
type RevokeConsentInput struct {
	UserID  string
	Channel entities.Channel
	ActorID string
}

// Execute valida y delega.
func (u RevokeConsent) Execute(ctx context.Context, in RevokeConsentInput) (entities.NotificationConsent, error) {
	if err := policies.ValidateUUID(in.UserID); err != nil {
		return entities.NotificationConsent{}, apperrors.BadRequest("user_id: " + err.Error())
	}
	if err := policies.ValidateChannel(in.Channel); err != nil {
		return entities.NotificationConsent{}, apperrors.BadRequest("channel: " + err.Error())
	}

	consent, err := u.Consents.Revoke(ctx, in.UserID, in.Channel, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrConsentNotFound) {
			return entities.NotificationConsent{}, apperrors.NotFound("consent not found for channel")
		}
		return entities.NotificationConsent{}, apperrors.Internal("failed to revoke consent")
	}
	return consent, nil
}

// ---------------------------------------------------------------------------
// CreatePushToken
// ---------------------------------------------------------------------------

// CreatePushToken registra un push token.
type CreatePushToken struct {
	PushTokens domain.PushTokenRepository
}

// CreatePushTokenInput es el input del usecase.
type CreatePushTokenInput struct {
	UserID   string
	Platform entities.Platform
	Token    string
	ActorID  string
}

// Execute valida y delega.
func (u CreatePushToken) Execute(ctx context.Context, in CreatePushTokenInput) (entities.NotificationPushToken, error) {
	if err := policies.ValidateUUID(in.UserID); err != nil {
		return entities.NotificationPushToken{}, apperrors.BadRequest("user_id: " + err.Error())
	}
	if err := policies.ValidatePlatform(in.Platform); err != nil {
		return entities.NotificationPushToken{}, apperrors.BadRequest("platform: " + err.Error())
	}
	if err := policies.ValidateToken(in.Token); err != nil {
		return entities.NotificationPushToken{}, apperrors.BadRequest("token: " + err.Error())
	}

	token, err := u.PushTokens.Create(ctx, domain.CreatePushTokenInput{
		UserID:   in.UserID,
		Platform: in.Platform,
		Token:    in.Token,
		ActorID:  in.ActorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrPushTokenDuplicate) {
			return entities.NotificationPushToken{}, apperrors.Conflict("push token already registered")
		}
		return entities.NotificationPushToken{}, apperrors.Internal("failed to create push token")
	}
	return token, nil
}

// ---------------------------------------------------------------------------
// DeletePushToken
// ---------------------------------------------------------------------------

// DeletePushToken elimina (soft delete) un push token.
type DeletePushToken struct {
	PushTokens domain.PushTokenRepository
}

// DeletePushTokenInput es el input del usecase.
type DeletePushTokenInput struct {
	TokenID string
	ActorID string
}

// Execute valida y delega.
func (u DeletePushToken) Execute(ctx context.Context, in DeletePushTokenInput) error {
	if err := policies.ValidateUUID(in.TokenID); err != nil {
		return apperrors.BadRequest("id: " + err.Error())
	}

	// Verify token exists.
	if _, err := u.PushTokens.GetByID(ctx, in.TokenID); err != nil {
		if errors.Is(err, domain.ErrPushTokenNotFound) {
			return apperrors.NotFound("push token not found")
		}
		return apperrors.Internal("failed to load push token")
	}

	if err := u.PushTokens.SoftDelete(ctx, in.TokenID, in.ActorID); err != nil {
		return apperrors.Internal("failed to delete push token")
	}
	return nil
}

// ---------------------------------------------------------------------------
// ListProviderConfigs
// ---------------------------------------------------------------------------

// ListProviderConfigs lista las configuraciones de proveedores.
type ListProviderConfigs struct {
	ProviderConfigs domain.ProviderConfigRepository
}

// Execute delega al repo.
func (u ListProviderConfigs) Execute(ctx context.Context) ([]entities.NotificationProviderConfig, error) {
	out, err := u.ProviderConfigs.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list provider configs")
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// UpdateProviderConfig
// ---------------------------------------------------------------------------

// UpdateProviderConfig actualiza una config de proveedor.
type UpdateProviderConfig struct {
	ProviderConfigs domain.ProviderConfigRepository
}

// UpdateProviderConfigInput es el input del usecase.
type UpdateProviderConfigInput struct {
	ID              string
	Config          []byte
	IsActive        bool
	ExpectedVersion int32
	ActorID         string
}

// Execute valida y delega.
func (u UpdateProviderConfig) Execute(ctx context.Context, in UpdateProviderConfigInput) (entities.NotificationProviderConfig, error) {
	if err := policies.ValidateUUID(in.ID); err != nil {
		return entities.NotificationProviderConfig{}, apperrors.BadRequest("id: " + err.Error())
	}

	cfg, err := u.ProviderConfigs.Update(ctx, domain.UpdateProviderConfigInput{
		ID:              in.ID,
		Config:          in.Config,
		IsActive:        in.IsActive,
		ExpectedVersion: in.ExpectedVersion,
		ActorID:         in.ActorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProviderConfigNotFound):
			return entities.NotificationProviderConfig{}, apperrors.NotFound("provider config not found")
		case errors.Is(err, domain.ErrVersionConflict):
			return entities.NotificationProviderConfig{}, mapVersionConflict()
		default:
			return entities.NotificationProviderConfig{}, apperrors.Internal("failed to update provider config")
		}
	}
	return cfg, nil
}

// ---------------------------------------------------------------------------
// Broadcast
// ---------------------------------------------------------------------------

// Broadcast encola mensajes de notificacion para multiples destinatarios
// y canales. Verifica preferencias y consentimiento por destinatario.
type Broadcast struct {
	Outbox      domain.OutboxRepository
	Preferences domain.PreferenceRepository
	Consents    domain.ConsentRepository
	TxRunner    TxRunner
	Now         func() time.Time
}

// BroadcastInput es el input del usecase.
type BroadcastInput struct {
	EventType      string
	Channels       []entities.Channel
	RecipientIDs   []string
	Payload        []byte
	IdempotencyKey string
	ActorID        string
}

// BroadcastResult es el output del usecase.
type BroadcastResult struct {
	Queued  int
	Skipped int
	Blocked int
}

// Execute valida, verifica preferencias/consent, y encola.
func (u Broadcast) Execute(ctx context.Context, in BroadcastInput) (BroadcastResult, error) {
	if err := policies.ValidateEventType(in.EventType); err != nil {
		return BroadcastResult{}, apperrors.BadRequest("event_type: " + err.Error())
	}
	if len(in.Channels) == 0 {
		return BroadcastResult{}, apperrors.BadRequest("channels must not be empty")
	}
	for _, ch := range in.Channels {
		if err := policies.ValidateChannel(ch); err != nil {
			return BroadcastResult{}, apperrors.BadRequest("channels: " + err.Error())
		}
	}
	if len(in.RecipientIDs) == 0 {
		return BroadcastResult{}, apperrors.BadRequest("recipient_ids must not be empty")
	}
	for _, rid := range in.RecipientIDs {
		if err := policies.ValidateUUID(rid); err != nil {
			return BroadcastResult{}, apperrors.BadRequest("recipient_ids: " + err.Error())
		}
	}
	if err := policies.ValidateIdempotencyKey(in.IdempotencyKey); err != nil {
		return BroadcastResult{}, apperrors.BadRequest("idempotency_key: " + err.Error())
	}

	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}

	isCritical := policies.IsCriticalEvent(in.EventType)
	var result BroadcastResult

	run := func(txCtx context.Context) error {
		for _, recipientID := range in.RecipientIDs {
			for _, ch := range in.Channels {
				// Check consent for channels that require it.
				if policies.RequiresConsent(ch) {
					_, consentErr := u.Consents.GetActiveByUserChannel(txCtx, recipientID, ch)
					if consentErr != nil {
						if errors.Is(consentErr, domain.ErrConsentNotFound) {
							result.Blocked++
							continue
						}
						return consentErr
					}
				}

				// Check preferences (unless critical event).
				if !isCritical {
					pref, prefErr := u.Preferences.GetByUserEventChannel(txCtx, recipientID, in.EventType, ch)
					if prefErr == nil && !pref.Enabled {
						result.Skipped++
						continue
					}
					// If preference not found, default is enabled.
				}

				// Enqueue.
				idempKey := in.IdempotencyKey + ":" + recipientID + ":" + string(ch)
				_, enqErr := u.Outbox.Enqueue(txCtx, domain.EnqueueOutboxInput{
					EventType:       in.EventType,
					RecipientUserID: recipientID,
					Channel:         ch,
					Payload:         in.Payload,
					IdempotencyKey:  idempKey,
					ScheduledAt:     now,
					ActorID:         in.ActorID,
				})
				if enqErr != nil {
					if errors.Is(enqErr, domain.ErrOutboxDuplicate) {
						result.Skipped++
						continue
					}
					return enqErr
				}
				result.Queued++
			}
		}
		return nil
	}

	if u.TxRunner != nil {
		if err := u.TxRunner.RunInTx(ctx, pgx.ReadCommitted, run); err != nil {
			return BroadcastResult{}, apperrors.Internal("failed to broadcast notifications")
		}
	} else {
		if err := run(ctx); err != nil {
			return BroadcastResult{}, apperrors.Internal("failed to broadcast notifications")
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mapVersionConflict construye un Problem 409 estable.
func mapVersionConflict() error {
	return apperrors.New(409, "version-conflict", "Conflict",
		"resource was modified by another request; reload and retry")
}
