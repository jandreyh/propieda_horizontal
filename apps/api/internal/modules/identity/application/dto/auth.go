// Package dto agrupa los Data Transfer Objects HTTP del modulo identity.
//
// Las entidades del dominio NO llevan tags JSON. Aqui SI viven los tags
// para serializar y deserializar requests/responses HTTP.
package dto

import "time"

// LoginRequest es el body de POST /auth/login.
//
// `Identifier` puede ser un email (`luis@example.com`) o el par
// documento codificado como `<doc_type>:<doc_number>` (ej. `CC:12345`).
// El usecase resuelve cual es y consulta el repositorio adecuado.
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// LoginResponse es la respuesta de POST /auth/login.
//
// Cuando MFA esta enrolado, el servidor devuelve solo `pre_auth_token`
// y `mfa_required=true`. El cliente debe completar /auth/mfa/verify para
// obtener el par access/refresh real.
//
// Cuando MFA no esta enrolado, el servidor entrega `access_token`,
// `refresh_token` y `expires_in`.
type LoginResponse struct {
	MFARequired  bool   `json:"mfa_required"`
	PreAuthToken string `json:"pre_auth_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

// MFAVerifyRequest es el body de POST /auth/mfa/verify.
type MFAVerifyRequest struct {
	PreAuthToken string `json:"pre_auth_token"`
	Code         string `json:"code"`
}

// MFAVerifyResponse es la respuesta a /auth/mfa/verify exitosa. Reusa la
// forma de TokenPair (no incluye `mfa_required` porque ya fue resuelto).
type MFAVerifyResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// RefreshRequest es el body de POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse es la respuesta a /auth/refresh exitosa.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// MeResponse es la respuesta a GET /me. Solo expone campos seguros: ni
// password_hash ni mfa_secret cruzan la frontera HTTP.
type MeResponse struct {
	ID             string     `json:"id"`
	DocumentType   string     `json:"document_type"`
	DocumentNumber string     `json:"document_number"`
	Names          string     `json:"names"`
	LastNames      string     `json:"last_names"`
	Email          *string    `json:"email,omitempty"`
	Phone          *string    `json:"phone,omitempty"`
	MFAEnrolledAt  *time.Time `json:"mfa_enrolled_at,omitempty"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
}
