package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
)

type fakePushRepo struct {
	registered  []entities.PushDevice
	revoked     []uuid.UUID
	registerErr error
	revokeErr   error
}

func (f *fakePushRepo) Register(_ context.Context, userID uuid.UUID, token, platform string, label *string) (*entities.PushDevice, error) {
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	d := entities.PushDevice{
		ID:             uuid.New(),
		PlatformUserID: userID,
		DeviceToken:    token,
		Platform:       platform,
		DeviceLabel:    label,
	}
	f.registered = append(f.registered, d)
	return &d, nil
}

func (f *fakePushRepo) Revoke(_ context.Context, deviceID, _ uuid.UUID) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	f.revoked = append(f.revoked, deviceID)
	return nil
}

func (f *fakePushRepo) List(context.Context, uuid.UUID) ([]entities.PushDevice, error) {
	return nil, nil
}

func TestRegisterPushDevice_Success(t *testing.T) {
	repo := &fakePushRepo{}
	uc := NewRegisterPushDeviceUseCase(RegisterPushDeviceDeps{Devices: repo})
	label := "iPhone Ana"

	res, err := uc.Execute(context.Background(), uuid.New().String(), dto.RegisterPushDeviceRequest{
		DeviceToken: "ExponentPushToken[abc]",
		Platform:    "IOS",
		DeviceLabel: &label,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.Platform != "ios" {
		t.Errorf("platform = %q (expected lowercase ios)", res.Platform)
	}
	if len(repo.registered) != 1 {
		t.Errorf("expected 1 register, got %d", len(repo.registered))
	}
}

func TestRegisterPushDevice_InvalidPlatform(t *testing.T) {
	repo := &fakePushRepo{}
	uc := NewRegisterPushDeviceUseCase(RegisterPushDeviceDeps{Devices: repo})

	_, err := uc.Execute(context.Background(), uuid.New().String(), dto.RegisterPushDeviceRequest{
		DeviceToken: "tok", Platform: "blackberry",
	})
	if !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("expected ErrInvalidDevice, got %v", err)
	}
}

func TestRegisterPushDevice_EmptyToken(t *testing.T) {
	repo := &fakePushRepo{}
	uc := NewRegisterPushDeviceUseCase(RegisterPushDeviceDeps{Devices: repo})

	_, err := uc.Execute(context.Background(), uuid.New().String(), dto.RegisterPushDeviceRequest{
		DeviceToken: "  ", Platform: "ios",
	})
	if !errors.Is(err, ErrInvalidDevice) {
		t.Fatalf("expected ErrInvalidDevice, got %v", err)
	}
}

func TestRegisterPushDevice_BadSubject(t *testing.T) {
	repo := &fakePushRepo{}
	uc := NewRegisterPushDeviceUseCase(RegisterPushDeviceDeps{Devices: repo})

	_, err := uc.Execute(context.Background(), "not-a-uuid", dto.RegisterPushDeviceRequest{
		DeviceToken: "tok", Platform: "ios",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRemovePushDevice_Success(t *testing.T) {
	repo := &fakePushRepo{}
	uc := NewRemovePushDeviceUseCase(RemovePushDeviceDeps{Devices: repo})
	deviceID := uuid.New()

	err := uc.Execute(context.Background(), uuid.New().String(), deviceID.String())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(repo.revoked) != 1 || repo.revoked[0] != deviceID {
		t.Errorf("expected revoke for %s, got %v", deviceID, repo.revoked)
	}
}

func TestRemovePushDevice_BadID(t *testing.T) {
	repo := &fakePushRepo{}
	uc := NewRemovePushDeviceUseCase(RemovePushDeviceDeps{Devices: repo})

	err := uc.Execute(context.Background(), uuid.New().String(), "bogus")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
