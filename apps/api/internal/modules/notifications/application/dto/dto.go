// Package dto contiene los Data Transfer Objects del modulo notifications.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Notification Templates
// ---------------------------------------------------------------------------

// CreateTemplateRequest es el body de POST /notifications/templates.
type CreateTemplateRequest struct {
	EventType           string  `json:"event_type"`
	Channel             string  `json:"channel"`
	Locale              string  `json:"locale"`
	Subject             *string `json:"subject,omitempty"`
	BodyTemplate        string  `json:"body_template"`
	ProviderTemplateRef *string `json:"provider_template_ref,omitempty"`
}

// UpdateTemplateRequest es el body de PATCH /notifications/templates/{id}.
type UpdateTemplateRequest struct {
	Subject             *string `json:"subject,omitempty"`
	BodyTemplate        string  `json:"body_template"`
	ProviderTemplateRef *string `json:"provider_template_ref,omitempty"`
	Version             int32   `json:"version"`
}

// TemplateResponse es la representacion HTTP de una NotificationTemplate.
type TemplateResponse struct {
	ID                  string  `json:"id"`
	EventType           string  `json:"event_type"`
	Channel             string  `json:"channel"`
	Locale              string  `json:"locale"`
	Subject             *string `json:"subject,omitempty"`
	BodyTemplate        string  `json:"body_template"`
	ProviderTemplateRef *string `json:"provider_template_ref,omitempty"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	Version             int32   `json:"version"`
}

// ListTemplatesResponse es el sobre del listado de plantillas.
type ListTemplatesResponse struct {
	Items []TemplateResponse `json:"items"`
	Total int                `json:"total"`
}

// ---------------------------------------------------------------------------
// Notification Preferences
// ---------------------------------------------------------------------------

// PatchPreferenceRequest es un item del body de PATCH /notifications/preferences.
type PatchPreferenceRequest struct {
	EventType string `json:"event_type"`
	Channel   string `json:"channel"`
	Enabled   bool   `json:"enabled"`
}

// PatchPreferencesRequest es el body de PATCH /notifications/preferences.
type PatchPreferencesRequest struct {
	Preferences []PatchPreferenceRequest `json:"preferences"`
}

// PreferenceResponse es la representacion HTTP de una NotificationPreference.
type PreferenceResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	EventType string `json:"event_type"`
	Channel   string `json:"channel"`
	Enabled   bool   `json:"enabled"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Version   int32  `json:"version"`
}

// ListPreferencesResponse es el sobre del listado de preferencias.
type ListPreferencesResponse struct {
	Items []PreferenceResponse `json:"items"`
	Total int                  `json:"total"`
}

// ---------------------------------------------------------------------------
// Notification Consents
// ---------------------------------------------------------------------------

// CreateConsentRequest es el body de POST /notifications/consents.
type CreateConsentRequest struct {
	Channel         string  `json:"channel"`
	ConsentProofURL *string `json:"consent_proof_url,omitempty"`
	LegalBasis      *string `json:"legal_basis,omitempty"`
}

// ConsentResponse es la representacion HTTP de un NotificationConsent.
type ConsentResponse struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	Channel         string  `json:"channel"`
	ConsentedAt     string  `json:"consented_at"`
	RevokedAt       *string `json:"revoked_at,omitempty"`
	ConsentProofURL *string `json:"consent_proof_url,omitempty"`
	LegalBasis      *string `json:"legal_basis,omitempty"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	Version         int32   `json:"version"`
}

// ---------------------------------------------------------------------------
// Push Tokens
// ---------------------------------------------------------------------------

// CreatePushTokenRequest es el body de POST /notifications/push-tokens.
type CreatePushTokenRequest struct {
	Platform string `json:"platform"`
	Token    string `json:"token"`
}

// PushTokenResponse es la representacion HTTP de un NotificationPushToken.
type PushTokenResponse struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Platform   string `json:"platform"`
	Token      string `json:"token"`
	LastSeenAt string `json:"last_seen_at"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Version    int32  `json:"version"`
}

// ListPushTokensResponse es el sobre del listado de push tokens.
type ListPushTokensResponse struct {
	Items []PushTokenResponse `json:"items"`
	Total int                 `json:"total"`
}

// ---------------------------------------------------------------------------
// Provider Configs
// ---------------------------------------------------------------------------

// PatchProviderConfigRequest es el body de PATCH /notifications/provider-configs.
type PatchProviderConfigRequest struct {
	ID       string `json:"id"`
	Config   []byte `json:"config"`
	IsActive bool   `json:"is_active"`
	Version  int32  `json:"version"`
}

// ProviderConfigResponse es la representacion HTTP de un
// NotificationProviderConfig.
type ProviderConfigResponse struct {
	ID           string `json:"id"`
	Channel      string `json:"channel"`
	ProviderName string `json:"provider_name"`
	IsActive     bool   `json:"is_active"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Version      int32  `json:"version"`
}

// ListProviderConfigsResponse es el sobre del listado de provider configs.
type ListProviderConfigsResponse struct {
	Items []ProviderConfigResponse `json:"items"`
	Total int                      `json:"total"`
}

// ---------------------------------------------------------------------------
// Broadcast
// ---------------------------------------------------------------------------

// BroadcastRequest es el body de POST /notifications/broadcast.
type BroadcastRequest struct {
	EventType      string   `json:"event_type"`
	Channels       []string `json:"channels"`
	RecipientIDs   []string `json:"recipient_ids"`
	Payload        []byte   `json:"payload"`
	IdempotencyKey string   `json:"idempotency_key"`
}

// BroadcastResponse es la respuesta de POST /notifications/broadcast.
type BroadcastResponse struct {
	Queued  int `json:"queued"`
	Skipped int `json:"skipped"`
	Blocked int `json:"blocked"`
}

// ---------------------------------------------------------------------------
// Time formatting helpers
// ---------------------------------------------------------------------------

// FormatTime formatea un time.Time como RFC3339 para JSON.
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatTimePtr formatea un *time.Time como RFC3339 string pointer.
func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
