package usecases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
)

// ErrInvalidDevice lo emite RegisterPushDevice cuando el body es invalido
// (token vacio o platform fuera del set permitido).
var ErrInvalidDevice = errors.New("platform_identity: invalid push device")

// validPlatforms son los tres valores aceptados por la columna platform.
var validPlatforms = map[string]struct{}{"ios": {}, "android": {}, "web": {}}

// RegisterPushDeviceDeps son las dependencias del usecase.
type RegisterPushDeviceDeps struct {
	Devices domain.PushDeviceRepository
}

// RegisterPushDeviceUseCase implementa POST /me/push-devices.
type RegisterPushDeviceUseCase struct {
	deps RegisterPushDeviceDeps
}

// NewRegisterPushDeviceUseCase construye el usecase.
func NewRegisterPushDeviceUseCase(deps RegisterPushDeviceDeps) *RegisterPushDeviceUseCase {
	return &RegisterPushDeviceUseCase{deps: deps}
}

// Execute valida el body y hace upsert del device para el usuario.
func (uc *RegisterPushDeviceUseCase) Execute(ctx context.Context, subject string, req dto.RegisterPushDeviceRequest) (dto.RegisterPushDeviceResponse, error) {
	id, err := uuid.Parse(subject)
	if err != nil {
		return dto.RegisterPushDeviceResponse{}, ErrInvalidInput
	}
	token := strings.TrimSpace(req.DeviceToken)
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if token == "" {
		return dto.RegisterPushDeviceResponse{}, ErrInvalidDevice
	}
	if _, ok := validPlatforms[platform]; !ok {
		return dto.RegisterPushDeviceResponse{}, ErrInvalidDevice
	}

	dev, err := uc.deps.Devices.Register(ctx, id, token, platform, req.DeviceLabel)
	if err != nil {
		return dto.RegisterPushDeviceResponse{}, fmt.Errorf("%w: register: %w", ErrInternal, err)
	}
	return dto.RegisterPushDeviceResponse{
		ID:          dev.ID.String(),
		DeviceToken: dev.DeviceToken,
		Platform:    dev.Platform,
		DeviceLabel: dev.DeviceLabel,
	}, nil
}

// RemovePushDeviceDeps son las dependencias del usecase.
type RemovePushDeviceDeps struct {
	Devices domain.PushDeviceRepository
}

// RemovePushDeviceUseCase implementa DELETE /me/push-devices/{id}.
type RemovePushDeviceUseCase struct {
	deps RemovePushDeviceDeps
}

// NewRemovePushDeviceUseCase construye el usecase.
func NewRemovePushDeviceUseCase(deps RemovePushDeviceDeps) *RemovePushDeviceUseCase {
	return &RemovePushDeviceUseCase{deps: deps}
}

// Execute marca el device como revocado, validando que pertenece al
// usuario autenticado.
func (uc *RemovePushDeviceUseCase) Execute(ctx context.Context, subject, deviceID string) error {
	uid, err := uuid.Parse(subject)
	if err != nil {
		return ErrInvalidInput
	}
	did, err := uuid.Parse(deviceID)
	if err != nil {
		return ErrInvalidInput
	}
	if err := uc.deps.Devices.Revoke(ctx, did, uid); err != nil {
		return fmt.Errorf("%w: revoke: %w", ErrInternal, err)
	}
	return nil
}
