package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/saas-ph/api/internal/modules/identity/application/dto"
	"github.com/saas-ph/api/internal/modules/identity/domain"
)

// MeDeps agrupa las dependencias del usecase /me.
type MeDeps struct {
	Users domain.UserRepository
}

// MeUseCase implementa GET /me. Recibe el user id ya extraido de la
// claim `sub` del JWT; la verificacion del JWT corre en el handler.
type MeUseCase struct {
	deps MeDeps
}

// NewMeUseCase construye el usecase.
func NewMeUseCase(deps MeDeps) *MeUseCase {
	return &MeUseCase{deps: deps}
}

// Execute carga al usuario autenticado y lo mapea a su DTO publico.
// Devuelve ErrInvalidCredentials si el id no resuelve a un usuario
// activo (no debe ocurrir con un JWT valido, pero protege contra
// usuarios borrados / desactivados despues de la emision del token).
func (uc *MeUseCase) Execute(ctx context.Context, userID string) (dto.MeResponse, error) {
	if userID == "" {
		return dto.MeResponse{}, ErrInvalidInput
	}
	user, err := uc.deps.Users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.MeResponse{}, ErrInvalidCredentials
		}
		return dto.MeResponse{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	if !user.IsActive() {
		return dto.MeResponse{}, ErrAccountInactive
	}
	return dto.MeResponse{
		ID:             user.ID,
		DocumentType:   string(user.DocumentType),
		DocumentNumber: user.DocumentNumber,
		Names:          user.Names,
		LastNames:      user.LastNames,
		Email:          user.Email,
		Phone:          user.Phone,
		MFAEnrolledAt:  user.MFAEnrolledAt,
		LastLoginAt:    user.LastLoginAt,
	}, nil
}
