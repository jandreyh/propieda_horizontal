// Package dto define los DTOs HTTP del modulo tenant_members.
package dto

// AddMemberRequest es el body de POST /tenant-members.
type AddMemberRequest struct {
	PublicCode    string  `json:"public_code"`
	Role          string  `json:"role"`
	PrimaryUnitID *string `json:"primary_unit_id,omitempty"`
}

// MemberDTO es la representacion HTTP de un tenant_user_link enriquecido.
type MemberDTO struct {
	ID             string  `json:"id"`
	PlatformUserID string  `json:"platform_user_id"`
	Names          string  `json:"names"`
	LastNames      string  `json:"last_names"`
	Email          string  `json:"email"`
	PublicCode     string  `json:"public_code"`
	Role           string  `json:"role"`
	PrimaryUnitID  *string `json:"primary_unit_id,omitempty"`
	Status         string  `json:"status"`
	Version        int32   `json:"version"`
}

// ListMembersResponse es el body de GET /tenant-members.
type ListMembersResponse struct {
	Items []MemberDTO `json:"items"`
}

// UpdateMemberRequest es el body de PUT /tenant-members/{id}.
type UpdateMemberRequest struct {
	Role          string  `json:"role"`
	PrimaryUnitID *string `json:"primary_unit_id,omitempty"`
	Version       int32   `json:"version"`
}
