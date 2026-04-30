// Package usecases del modulo superadmin orquesta CRUD de tenants
// (delegando al modulo provisioning para la creacion compleja) y la
// busqueda cross-tenant restringida al rol platform_superadmin.
package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas-ph/api/internal/modules/provisioning"
	"github.com/saas-ph/api/internal/modules/superadmin/application/dto"
)

// Errores publicos.
var (
	ErrInvalidInput = errors.New("superadmin: invalid input")
	ErrInternal     = errors.New("superadmin: internal error")
)

// CreateTenantDeps son las dependencias del usecase.
type CreateTenantDeps struct {
	Provisioner *provisioning.Provisioner
}

// CreateTenantUseCase implementa POST /superadmin/tenants.
type CreateTenantUseCase struct {
	deps CreateTenantDeps
}

// NewCreateTenantUseCase construye el usecase.
func NewCreateTenantUseCase(deps CreateTenantDeps) *CreateTenantUseCase {
	return &CreateTenantUseCase{deps: deps}
}

// Execute valida + delega al provisioner.
func (uc *CreateTenantUseCase) Execute(ctx context.Context, req dto.CreateTenantRequest) (dto.CreateTenantResponse, error) {
	if err := basicValidate(req); err != nil {
		return dto.CreateTenantResponse{}, err
	}
	var adminID *uuid.UUID
	if req.AdministratorID != nil && *req.AdministratorID != "" {
		parsed, err := uuid.Parse(*req.AdministratorID)
		if err != nil {
			return dto.CreateTenantResponse{}, ErrInvalidInput
		}
		adminID = &parsed
	}
	out, err := uc.deps.Provisioner.Create(ctx, provisioning.CreateTenantInput{
		Slug:            strings.ToLower(strings.TrimSpace(req.Slug)),
		DisplayName:     strings.TrimSpace(req.DisplayName),
		AdministratorID: adminID,
		Plan:            req.Plan,
		Country:         req.Country,
		Currency:        req.Currency,
		Timezone:        req.Timezone,
		ExpectedUnits:   req.ExpectedUnits,
		Admin: provisioning.AdminInput{
			Email:          req.Admin.Email,
			DocumentType:   strings.ToUpper(req.Admin.DocumentType),
			DocumentNumber: req.Admin.DocumentNumber,
			Names:          req.Admin.Names,
			LastNames:      req.Admin.LastNames,
			Password:       req.Admin.Password,
			Phone:          req.Admin.Phone,
		},
	})
	if err != nil {
		return dto.CreateTenantResponse{}, fmt.Errorf("%w: provision: %w", ErrInternal, err)
	}
	return dto.CreateTenantResponse{
		TenantID:    out.TenantID.String(),
		Slug:        strings.ToLower(strings.TrimSpace(req.Slug)),
		DatabaseURL: out.DatabaseURL,
		AdminUserID: out.AdminUserID.String(),
		AdminReused: out.Reused,
	}, nil
}

func basicValidate(req dto.CreateTenantRequest) error {
	if strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.DisplayName) == "" {
		return ErrInvalidInput
	}
	if req.Admin.Email == "" || req.Admin.Password == "" || req.Admin.DocumentType == "" || req.Admin.DocumentNumber == "" {
		return ErrInvalidInput
	}
	return nil
}

// ListTenantsDeps son las dependencias del usecase.
type ListTenantsDeps struct {
	CentralPool *pgxpool.Pool
}

// ListTenantsUseCase implementa GET /superadmin/tenants.
type ListTenantsUseCase struct {
	deps ListTenantsDeps
}

// NewListTenantsUseCase construye el usecase.
func NewListTenantsUseCase(deps ListTenantsDeps) *ListTenantsUseCase {
	return &ListTenantsUseCase{deps: deps}
}

// Execute lista todos los tenants no-archived.
func (uc *ListTenantsUseCase) Execute(ctx context.Context) (dto.ListTenantsResponse, error) {
	rows, err := uc.deps.CentralPool.Query(ctx, `
		SELECT id, slug, display_name, status, plan
		FROM tenants
		WHERE status != 'archived'
		ORDER BY display_name`)
	if err != nil {
		return dto.ListTenantsResponse{}, fmt.Errorf("%w: query: %w", ErrInternal, err)
	}
	defer rows.Close()
	out := dto.ListTenantsResponse{Items: []dto.TenantSummaryDTO{}}
	for rows.Next() {
		var s dto.TenantSummaryDTO
		if err := rows.Scan(&s.ID, &s.Slug, &s.DisplayName, &s.Status, &s.Plan); err != nil {
			return dto.ListTenantsResponse{}, fmt.Errorf("%w: scan: %w", ErrInternal, err)
		}
		out.Items = append(out.Items, s)
	}
	return out, nil
}
