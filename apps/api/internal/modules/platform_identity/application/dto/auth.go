// Package dto agrupa los DTOs HTTP del modulo platform_identity.
// Aqui (y solo aqui) viven los tags JSON.
package dto

import "time"

// LoginRequest es el body de POST /auth/login.
//
// El usuario debe entregar tres factores de identificacion ademas del
// password: email, document_type, document_number. Esto es por decision
// del usuario en Discovery (ADR 0007).
type LoginRequest struct {
	Email          string `json:"email"`
	DocumentType   string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
	Password       string `json:"password"`
}

// MembershipDTO es la representacion HTTP de una pertenencia a tenant.
type MembershipDTO struct {
	TenantID     string  `json:"tenant_id"`
	TenantSlug   string  `json:"tenant_slug"`
	TenantName   string  `json:"tenant_name"`
	LogoURL      *string `json:"logo_url,omitempty"`
	PrimaryColor *string `json:"primary_color,omitempty"`
	Role         string  `json:"role"`
	Status       string  `json:"status"`
}

// LoginResponse es el JSON de POST /auth/login exitoso.
//
// `access_token` lleva memberships[] y current_tenant=null. El cliente
// muestra el selector (o entra directo si memberships.length == 1).
//
// Si la persona tiene MFA enrolado, en lugar de access_token se devuelve
// pre_auth_token con TTL corto y el cliente debe llamar
// POST /auth/mfa/verify para completar el login.
type LoginResponse struct {
	AccessToken  string          `json:"access_token,omitempty"`
	TokenType    string          `json:"token_type,omitempty"`
	ExpiresIn    int             `json:"expires_in,omitempty"`
	Memberships  []MembershipDTO `json:"memberships,omitempty"`
	NeedsTenant  bool            `json:"needs_tenant,omitempty"`
	MFARequired  bool            `json:"mfa_required,omitempty"`
	PreAuthToken string          `json:"pre_auth_token,omitempty"`
}

// SwitchTenantRequest es el body de POST /auth/switch-tenant.
type SwitchTenantRequest struct {
	TenantSlug string `json:"tenant_slug"`
}

// SwitchTenantResponse re-firma el JWT con current_tenant fijado.
type SwitchTenantResponse struct {
	AccessToken   string        `json:"access_token"`
	TokenType     string        `json:"token_type"`
	ExpiresIn     int           `json:"expires_in"`
	CurrentTenant MembershipDTO `json:"current_tenant"`
}

// MeResponse es el JSON de GET /me.
type MeResponse struct {
	ID             string     `json:"id"`
	DocumentType   string     `json:"document_type"`
	DocumentNumber string     `json:"document_number"`
	Names          string     `json:"names"`
	LastNames      string     `json:"last_names"`
	Email          string     `json:"email"`
	Phone          *string    `json:"phone,omitempty"`
	PhotoURL       *string    `json:"photo_url,omitempty"`
	PublicCode     string     `json:"public_code"`
	MFAEnrolledAt  *time.Time `json:"mfa_enrolled_at,omitempty"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
}

// MembershipsResponse es el JSON de GET /me/memberships.
type MembershipsResponse struct {
	Items []MembershipDTO `json:"items"`
}
