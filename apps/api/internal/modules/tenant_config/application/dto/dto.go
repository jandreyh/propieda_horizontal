// Package dto contiene los Data Transfer Objects del modulo tenant_config.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import (
	"encoding/json"
	"time"
)

// SettingResponse es la representacion HTTP de una entries.Setting.
type SettingResponse struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Version     int32           `json:"version"`
}

// ListSettingsResponse es el sobre paginado del listado.
type ListSettingsResponse struct {
	Items  []SettingResponse `json:"items"`
	Total  int64             `json:"total"`
	Limit  int32             `json:"limit"`
	Offset int32             `json:"offset"`
}

// UpsertSettingRequest es el body de PUT /settings/:key.
type UpsertSettingRequest struct {
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
}

// BrandingResponse es la representacion HTTP del singleton.
type BrandingResponse struct {
	ID             string    `json:"id"`
	DisplayName    string    `json:"display_name"`
	LogoURL        *string   `json:"logo_url,omitempty"`
	PrimaryColor   *string   `json:"primary_color,omitempty"`
	SecondaryColor *string   `json:"secondary_color,omitempty"`
	Timezone       string    `json:"timezone"`
	Locale         string    `json:"locale"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Version        int32     `json:"version"`
}

// UpdateBrandingRequest es el body de PUT /branding.
type UpdateBrandingRequest struct {
	DisplayName     string  `json:"display_name"`
	LogoURL         *string `json:"logo_url,omitempty"`
	PrimaryColor    *string `json:"primary_color,omitempty"`
	SecondaryColor  *string `json:"secondary_color,omitempty"`
	Timezone        string  `json:"timezone"`
	Locale          string  `json:"locale"`
	ExpectedVersion int32   `json:"expected_version"`
}
