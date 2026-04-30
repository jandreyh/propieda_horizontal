// Package usecases del modulo tenant_members orquesta los flows de
// vinculacion de personas a un conjunto via codigo unico.
package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/tenant_members/application/dto"
	"github.com/saas-ph/api/internal/modules/tenant_members/domain"
	"github.com/saas-ph/api/internal/modules/tenant_members/domain/entities"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// Errores publicos.
var (
	ErrInvalidInput    = errors.New("tenant_members: invalid input")
	ErrCodeNotFound    = errors.New("tenant_members: public_code not found")
	ErrAlreadyLinked   = errors.New("tenant_members: user already linked to this tenant")
	ErrLinkNotFound    = errors.New("tenant_members: link not found")
	ErrInternal        = errors.New("tenant_members: internal error")
	ErrVersionMismatch = errors.New("tenant_members: version mismatch")
)

// ValidRoles son los roles aceptados al vincular un miembro.
var ValidRoles = map[string]struct{}{
	"tenant_admin":   {},
	"accountant":     {},
	"guard":          {},
	"resident":       {},
	"owner":          {},
	"council_member": {},
}

// AddByCodeDeps son las dependencias del usecase.
type AddByCodeDeps struct {
	Links    domain.LinkRepository
	Enricher domain.EnricherRepository
}

// AddByCodeUseCase implementa POST /tenant-members.
type AddByCodeUseCase struct{ deps AddByCodeDeps }

// NewAddByCodeUseCase construye el usecase.
func NewAddByCodeUseCase(deps AddByCodeDeps) *AddByCodeUseCase {
	return &AddByCodeUseCase{deps: deps}
}

// Execute valida + crea el link.
func (uc *AddByCodeUseCase) Execute(ctx context.Context, req dto.AddMemberRequest) (dto.MemberDTO, error) {
	code := strings.TrimSpace(strings.ToUpper(req.PublicCode))
	role := strings.TrimSpace(req.Role)
	if code == "" || role == "" {
		return dto.MemberDTO{}, ErrInvalidInput
	}
	if _, ok := ValidRoles[role]; !ok {
		return dto.MemberDTO{}, ErrInvalidInput
	}
	var unit *uuid.UUID
	if req.PrimaryUnitID != nil && *req.PrimaryUnitID != "" {
		parsed, err := uuid.Parse(*req.PrimaryUnitID)
		if err != nil {
			return dto.MemberDTO{}, ErrInvalidInput
		}
		unit = &parsed
	}

	puid, names, lastNames, email, err := uc.deps.Enricher.FindPlatformUserIDByCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrPlatformUserNotFound) {
			return dto.MemberDTO{}, ErrCodeNotFound
		}
		return dto.MemberDTO{}, fmt.Errorf("%w: lookup code: %w", ErrInternal, err)
	}

	link, err := uc.deps.Links.Create(ctx, domain.CreateLink{
		PlatformUserID: puid,
		Role:           role,
		PrimaryUnitID:  unit,
	})
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyLinked) {
			return dto.MemberDTO{}, ErrAlreadyLinked
		}
		return dto.MemberDTO{}, fmt.Errorf("%w: create link: %w", ErrInternal, err)
	}

	// Sincronizar la proyeccion central platform_user_memberships para que
	// el JWT del usuario refleje la nueva membresia en su proximo login.
	if tenantID, ok := tenantIDFromCtx(ctx); ok {
		if err := uc.deps.Enricher.UpsertCentralMembership(ctx, puid, tenantID, role, "active"); err != nil {
			// No revertimos el link local: el seeder/admin puede recrear
			// la proyeccion despues. Loguear seria ideal pero el usecase
			// no tiene logger inyectado — devolver error para que el
			// handler 500 sea visible.
			return dto.MemberDTO{}, fmt.Errorf("%w: sync central: %w", ErrInternal, err)
		}
	}

	link.Names = names
	link.LastNames = lastNames
	link.Email = email
	link.PublicCode = code
	return toDTO(*link), nil
}

// tenantIDFromCtx extrae el tenant.ID resuelto por el middleware
// TenantResolver. Devuelve "" si no esta en el contexto.
func tenantIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	t, err := tenantctx.FromCtx(ctx)
	if err != nil || t == nil {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(t.ID)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// ListDeps son las dependencias.
type ListDeps struct {
	Links    domain.LinkRepository
	Enricher domain.EnricherRepository
}

// ListUseCase implementa GET /tenant-members.
type ListUseCase struct{ deps ListDeps }

// NewListUseCase construye el usecase.
func NewListUseCase(deps ListDeps) *ListUseCase { return &ListUseCase{deps: deps} }

// Execute lista los links + hidrata.
func (uc *ListUseCase) Execute(ctx context.Context) (dto.ListMembersResponse, error) {
	rows, err := uc.deps.Links.List(ctx)
	if err != nil {
		return dto.ListMembersResponse{}, fmt.Errorf("%w: list: %w", ErrInternal, err)
	}
	hydrated, err := uc.deps.Enricher.Hydrate(ctx, rows)
	if err != nil {
		return dto.ListMembersResponse{}, fmt.Errorf("%w: hydrate: %w", ErrInternal, err)
	}
	out := dto.ListMembersResponse{Items: make([]dto.MemberDTO, 0, len(hydrated))}
	for _, m := range hydrated {
		out.Items = append(out.Items, toDTO(m))
	}
	return out, nil
}

// UpdateDeps son las dependencias.
type UpdateDeps struct {
	Links    domain.LinkRepository
	Enricher domain.EnricherRepository
}

// UpdateUseCase implementa PUT /tenant-members/{id}.
type UpdateUseCase struct{ deps UpdateDeps }

// NewUpdateUseCase construye el usecase.
func NewUpdateUseCase(deps UpdateDeps) *UpdateUseCase { return &UpdateUseCase{deps: deps} }

// Execute valida + actualiza role/unit con optimistic lock.
func (uc *UpdateUseCase) Execute(ctx context.Context, idStr string, req dto.UpdateMemberRequest) (dto.MemberDTO, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return dto.MemberDTO{}, ErrInvalidInput
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		return dto.MemberDTO{}, ErrInvalidInput
	}
	if _, ok := ValidRoles[role]; !ok {
		return dto.MemberDTO{}, ErrInvalidInput
	}
	var unit *uuid.UUID
	if req.PrimaryUnitID != nil && *req.PrimaryUnitID != "" {
		parsed, err := uuid.Parse(*req.PrimaryUnitID)
		if err != nil {
			return dto.MemberDTO{}, ErrInvalidInput
		}
		unit = &parsed
	}
	link, err := uc.deps.Links.Update(ctx, domain.UpdateLink{
		ID: id, Role: role, PrimaryUnitID: unit, Version: req.Version,
	})
	if err != nil {
		if errors.Is(err, domain.ErrLinkNotFound) {
			return dto.MemberDTO{}, ErrLinkNotFound
		}
		if strings.Contains(err.Error(), "version") {
			return dto.MemberDTO{}, ErrVersionMismatch
		}
		return dto.MemberDTO{}, fmt.Errorf("%w: update: %w", ErrInternal, err)
	}
	hydrated, err := uc.deps.Enricher.Hydrate(ctx, []entities.TenantMember{*link})
	if err != nil {
		return dto.MemberDTO{}, fmt.Errorf("%w: hydrate: %w", ErrInternal, err)
	}
	if len(hydrated) > 0 {
		return toDTO(hydrated[0]), nil
	}
	return toDTO(*link), nil
}

// BlockDeps son las dependencias.
type BlockDeps struct {
	Links    domain.LinkRepository
	Enricher domain.EnricherRepository
}

// BlockUseCase implementa POST /tenant-members/{id}/block.
type BlockUseCase struct{ deps BlockDeps }

// NewBlockUseCase construye el usecase.
func NewBlockUseCase(deps BlockDeps) *BlockUseCase { return &BlockUseCase{deps: deps} }

// Execute marca status=blocked en el tenant + actualiza la proyeccion
// central platform_user_memberships.
func (uc *BlockUseCase) Execute(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ErrInvalidInput
	}
	// Antes de bloquear, leer el link para conocer el platform_user_id.
	link, err := uc.deps.Links.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrLinkNotFound) {
			return ErrLinkNotFound
		}
		return fmt.Errorf("%w: lookup: %w", ErrInternal, err)
	}
	if err := uc.deps.Links.Block(ctx, id); err != nil {
		if errors.Is(err, domain.ErrLinkNotFound) {
			return ErrLinkNotFound
		}
		return fmt.Errorf("%w: block: %w", ErrInternal, err)
	}
	if uc.deps.Enricher != nil {
		if tenantID, ok := tenantIDFromCtx(ctx); ok {
			if err := uc.deps.Enricher.BlockCentralMembership(ctx, link.PlatformUserID, tenantID); err != nil {
				return fmt.Errorf("%w: sync central block: %w", ErrInternal, err)
			}
		}
	}
	return nil
}

func toDTO(m entities.TenantMember) dto.MemberDTO {
	d := dto.MemberDTO{
		ID:             m.ID.String(),
		PlatformUserID: m.PlatformUserID.String(),
		Names:          m.Names,
		LastNames:      m.LastNames,
		Email:          m.Email,
		PublicCode:     m.PublicCode,
		Role:           m.Role,
		Status:         m.Status,
		Version:        m.Version,
	}
	if m.PrimaryUnitID != nil {
		s := m.PrimaryUnitID.String()
		d.PrimaryUnitID = &s
	}
	return d
}
