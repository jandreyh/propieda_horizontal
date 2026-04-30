package usecases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/saas-ph/api/internal/modules/platform_identity/application/dto"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain"
	"github.com/saas-ph/api/internal/modules/platform_identity/domain/entities"
)

type fakeSessionRepo struct {
	mu        sync.Mutex
	byHash    map[string]*entities.PlatformSession
	revoked   map[uuid.UUID]string
	createErr error
	findErr   error
}

func newFakeSessions() *fakeSessionRepo {
	return &fakeSessionRepo{byHash: map[string]*entities.PlatformSession{}, revoked: map[uuid.UUID]string{}}
}

func (f *fakeSessionRepo) Create(_ context.Context, userID uuid.UUID, tokenHash string, _ *string, expiresAt time.Time) (*entities.PlatformSession, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	s := &entities.PlatformSession{
		ID:             uuid.New(),
		PlatformUserID: userID,
		TokenHash:      tokenHash,
		ExpiresAt:      expiresAt,
		Status:         "active",
	}
	f.mu.Lock()
	f.byHash[tokenHash] = s
	f.mu.Unlock()
	return s, nil
}

func (f *fakeSessionRepo) FindByTokenHash(_ context.Context, tokenHash string) (*entities.PlatformSession, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.byHash[tokenHash]; ok {
		return s, nil
	}
	return nil, domain.ErrSessionNotFound
}

func (f *fakeSessionRepo) Revoke(_ context.Context, sessionID uuid.UUID, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked[sessionID] = reason
	for h, s := range f.byHash {
		if s.ID == sessionID {
			now := time.Now()
			s.RevokedAt = &now
			s.Status = "revoked"
			delete(f.byHash, h)
		}
	}
	return nil
}

func (f *fakeSessionRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for h, s := range f.byHash {
		if s.PlatformUserID == userID {
			f.revoked[s.ID] = reason
			delete(f.byHash, h)
		}
	}
	return nil
}

// fakeRepoForRefresh combina lookup por id (necesario para refresh).
type fakeRepoForRefresh struct {
	*fakeRepo
	users map[uuid.UUID]*entities.PlatformUser
}

func (f *fakeRepoForRefresh) FindByID(_ context.Context, id uuid.UUID) (*entities.PlatformUser, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func TestLogin_WithSessions_ReturnsRefresh(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	tenantID := uuid.New()
	repo := &fakeRepo{
		byEmail: map[string]*entities.PlatformUser{user.Email: user},
		memberships: []entities.Membership{{
			TenantID: tenantID, TenantSlug: "demo", TenantName: "Demo", Role: "tenant_admin", Status: "active",
		}},
	}
	sessions := newFakeSessions()
	uc := NewLoginUseCase(LoginDeps{Users: repo, Sessions: sessions, Signer: newTestSigner(t)})

	res, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:          user.Email,
		DocumentType:   "CC",
		DocumentNumber: "1000000001",
		Password:       "secret123",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.RefreshToken == "" {
		t.Error("expected refresh_token when Sessions repo configured")
	}
	if len(sessions.byHash) != 1 {
		t.Errorf("expected 1 session created, got %d", len(sessions.byHash))
	}
}

func TestRefresh_Success_RotatesToken(t *testing.T) {
	user := newActiveUser(t, "ana@demo.test", "secret123", false)
	users := &fakeRepoForRefresh{
		fakeRepo: &fakeRepo{
			memberships: []entities.Membership{{TenantID: uuid.New(), TenantSlug: "demo", TenantName: "Demo", Role: "tenant_admin", Status: "active"}},
		},
		users: map[uuid.UUID]*entities.PlatformUser{user.ID: user},
	}
	sessions := newFakeSessions()
	plain, hash, _ := generateRefreshToken()
	_, _ = sessions.Create(context.Background(), user.ID, hash, nil, time.Now().Add(refreshTokenTTL))
	_ = plain // unused; we reuse the one we just generated
	uc := NewRefreshUseCase(RefreshDeps{Users: users, Sessions: sessions, Signer: newTestSigner(t)})

	res, err := uc.Execute(context.Background(), dto.RefreshRequest{RefreshToken: plain})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Error("expected new access + refresh tokens")
	}
	if res.RefreshToken == plain {
		t.Error("refresh must rotate")
	}
	if _, exists := sessions.byHash[hash]; exists {
		t.Error("old session should have been revoked")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	users := &fakeRepoForRefresh{fakeRepo: &fakeRepo{}, users: map[uuid.UUID]*entities.PlatformUser{}}
	sessions := newFakeSessions()
	uc := NewRefreshUseCase(RefreshDeps{Users: users, Sessions: sessions, Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.RefreshRequest{RefreshToken: "not-real"})
	if !errors.Is(err, ErrInvalidRefresh) {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
}

func TestRefresh_EmptyToken(t *testing.T) {
	uc := NewRefreshUseCase(RefreshDeps{Sessions: newFakeSessions(), Signer: newTestSigner(t)})

	_, err := uc.Execute(context.Background(), dto.RefreshRequest{RefreshToken: ""})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLogout_Success(t *testing.T) {
	sessions := newFakeSessions()
	uid := uuid.New()
	plain, hash, _ := generateRefreshToken()
	_, _ = sessions.Create(context.Background(), uid, hash, nil, time.Now().Add(time.Hour))
	uc := NewLogoutUseCase(LogoutDeps{Sessions: sessions})

	if err := uc.Execute(context.Background(), plain); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if _, exists := sessions.byHash[hash]; exists {
		t.Error("session should be revoked")
	}
}

func TestLogout_UnknownTokenIsIdempotent(t *testing.T) {
	sessions := newFakeSessions()
	uc := NewLogoutUseCase(LogoutDeps{Sessions: sessions})

	if err := uc.Execute(context.Background(), "ghost"); err != nil {
		t.Fatalf("expected nil (idempotent), got %v", err)
	}
}

func TestLogout_EmptyToken(t *testing.T) {
	sessions := newFakeSessions()
	uc := NewLogoutUseCase(LogoutDeps{Sessions: sessions})

	if err := uc.Execute(context.Background(), ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
