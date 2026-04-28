package usecases

import (
	"context"
	"errors"

	"github.com/saas-ph/api/internal/modules/packages/domain"
	"github.com/saas-ph/api/internal/modules/packages/domain/entities"
	"github.com/saas-ph/api/internal/modules/packages/domain/policies"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// GetPackage devuelve un paquete por id.
type GetPackage struct {
	Packages domain.PackageRepository
}

// Execute valida y delega.
func (u GetPackage) Execute(ctx context.Context, id string) (entities.Package, error) {
	if err := policies.ValidateUUID(id); err != nil {
		return entities.Package{}, apperrors.BadRequest("id: " + err.Error())
	}
	pkg, err := u.Packages.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrPackageNotFound) {
			return entities.Package{}, apperrors.NotFound("package not found")
		}
		return entities.Package{}, apperrors.Internal("failed to load package")
	}
	return pkg, nil
}

// ListPackagesByUnit lista los paquetes de una unidad (cualquier status).
type ListPackagesByUnit struct {
	Packages domain.PackageRepository
}

// Execute valida y delega.
func (u ListPackagesByUnit) Execute(ctx context.Context, unitID string) ([]entities.Package, error) {
	if err := policies.ValidateUUID(unitID); err != nil {
		return nil, apperrors.BadRequest("unit_id: " + err.Error())
	}
	out, err := u.Packages.ListByUnit(ctx, unitID)
	if err != nil {
		return nil, apperrors.Internal("failed to list packages")
	}
	return out, nil
}

// ListPackagesByStatus lista paquetes con un status dado.
type ListPackagesByStatus struct {
	Packages domain.PackageRepository
}

// Execute valida y delega.
func (u ListPackagesByStatus) Execute(ctx context.Context, status entities.PackageStatus) ([]entities.Package, error) {
	if !status.IsValid() {
		return nil, apperrors.BadRequest("invalid status")
	}
	out, err := u.Packages.ListByStatus(ctx, status)
	if err != nil {
		return nil, apperrors.Internal("failed to list packages")
	}
	return out, nil
}

// ListCategories lista las categorias activas.
type ListCategories struct {
	Categories domain.CategoryRepository
}

// Execute delega al repo.
func (u ListCategories) Execute(ctx context.Context) ([]entities.PackageCategory, error) {
	out, err := u.Categories.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list categories")
	}
	return out, nil
}
