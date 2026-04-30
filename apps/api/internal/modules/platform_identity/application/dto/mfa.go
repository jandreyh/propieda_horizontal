package dto

// MFAVerifyRequest es el body de POST /auth/mfa/verify.
type MFAVerifyRequest struct {
	PreAuthToken string `json:"pre_auth_token"`
	Code         string `json:"code"`
}
