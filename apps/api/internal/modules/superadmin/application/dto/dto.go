// Package dto agrupa los DTOs HTTP del modulo superadmin.
package dto

// CreateTenantRequest es el body de POST /superadmin/tenants.
type CreateTenantRequest struct {
	Slug            string  `json:"slug"`
	DisplayName     string  `json:"display_name"`
	AdministratorID *string `json:"administrator_id,omitempty"`
	Plan            string  `json:"plan,omitempty"`
	Country         string  `json:"country,omitempty"`
	Currency        string  `json:"currency,omitempty"`
	Timezone        string  `json:"timezone,omitempty"`
	ExpectedUnits   *int32  `json:"expected_units,omitempty"`
	Admin           AdminInputDTO `json:"admin"`
}

// AdminInputDTO describe al admin inicial del conjunto.
type AdminInputDTO struct {
	Email          string  `json:"email"`
	DocumentType   string  `json:"document_type"`
	DocumentNumber string  `json:"document_number"`
	Names          string  `json:"names"`
	LastNames      string  `json:"last_names"`
	Password       string  `json:"password"`
	Phone          *string `json:"phone,omitempty"`
}

// CreateTenantResponse confirma el tenant creado.
type CreateTenantResponse struct {
	TenantID    string `json:"tenant_id"`
	Slug        string `json:"slug"`
	DatabaseURL string `json:"database_url"`
	AdminUserID string `json:"admin_user_id"`
	AdminReused bool   `json:"admin_reused"`
}

// TenantSummaryDTO es la representacion HTTP de un tenant en listados.
type TenantSummaryDTO struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	Plan        string `json:"plan"`
}

// ListTenantsResponse es el JSON de GET /superadmin/tenants.
type ListTenantsResponse struct {
	Items []TenantSummaryDTO `json:"items"`
}
