package usecases_test

import (
	"context"
	"sync"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/domain"
	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
)

// userRepoMock implementa domain.UserRepository en memoria. Cada metodo
// que el repo expone graba la ultima invocacion para que el test pueda
// hacer aserciones simples sin frameworks externos.
type userRepoMock struct {
	mu sync.Mutex

	byID       map[string]*entities.User
	byEmail    map[string]*entities.User
	byDocument map[string]*entities.User // key = docType:docNumber

	getByIDErr       error
	getByEmailErr    error
	getByDocumentErr error
	updateErr        error

	failedAttemptsCalls int
	resetCalls          int
	lockCalls           int
	lastLockUntil       time.Time
	lastLoginAt         time.Time
}

func newUserRepoMock() *userRepoMock {
	return &userRepoMock{
		byID:       map[string]*entities.User{},
		byEmail:    map[string]*entities.User{},
		byDocument: map[string]*entities.User{},
	}
}

func (m *userRepoMock) put(u *entities.User) {
	m.byID[u.ID] = u
	if u.Email != nil {
		m.byEmail[*u.Email] = u
	}
	m.byDocument[string(u.DocumentType)+":"+u.DocumentNumber] = u
}

func (m *userRepoMock) GetByID(_ context.Context, id string) (*entities.User, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	u, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *userRepoMock) GetByEmail(_ context.Context, email string) (*entities.User, error) {
	if m.getByEmailErr != nil {
		return nil, m.getByEmailErr
	}
	u, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *userRepoMock) GetByDocument(_ context.Context, dt entities.DocumentType, dn string) (*entities.User, error) {
	if m.getByDocumentErr != nil {
		return nil, m.getByDocumentErr
	}
	u, ok := m.byDocument[string(dt)+":"+dn]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *userRepoMock) IncrementFailedAttempts(_ context.Context, userID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedAttemptsCalls++
	if u, ok := m.byID[userID]; ok {
		u.FailedLoginAttempts++
		return u.FailedLoginAttempts, nil
	}
	return 0, domain.ErrUserNotFound
}

func (m *userRepoMock) ResetFailedAttempts(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetCalls++
	if u, ok := m.byID[userID]; ok {
		u.FailedLoginAttempts = 0
	}
	return nil
}

func (m *userRepoMock) LockUser(_ context.Context, userID string, until time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lockCalls++
	m.lastLockUntil = until
	if u, ok := m.byID[userID]; ok {
		u.LockedUntil = &until
		u.FailedLoginAttempts = 0
	}
	return nil
}

func (m *userRepoMock) UnlockUser(_ context.Context, userID string) error {
	if u, ok := m.byID[userID]; ok {
		u.LockedUntil = nil
	}
	return nil
}

func (m *userRepoMock) UpdateLastLoginAt(_ context.Context, userID string, at time.Time) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.lastLoginAt = at
	if u, ok := m.byID[userID]; ok {
		u.LastLoginAt = &at
	}
	return nil
}

// sessionRepoMock implementa domain.SessionRepository en memoria.
type sessionRepoMock struct {
	mu sync.Mutex

	byID        map[string]*entities.Session
	byTokenHash map[string]*entities.Session

	createErr   error
	getErr      error
	revokeErr   error
	revokeCalls []string
	chainCalls  []string
}

func newSessionRepoMock() *sessionRepoMock {
	return &sessionRepoMock{
		byID:        map[string]*entities.Session{},
		byTokenHash: map[string]*entities.Session{},
	}
}

func (m *sessionRepoMock) Create(_ context.Context, s *entities.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	if s.ID == "" {
		s.ID = "sess-" + s.TokenHash[:8]
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = s.IssuedAt
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = s.IssuedAt
	}
	m.byID[s.ID] = s
	m.byTokenHash[s.TokenHash] = s
	return nil
}

func (m *sessionRepoMock) GetByTokenHash(_ context.Context, hash string) (*entities.Session, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.byTokenHash[hash]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return s, nil
}

func (m *sessionRepoMock) GetByID(_ context.Context, id string) (*entities.Session, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return s, nil
}

func (m *sessionRepoMock) Revoke(_ context.Context, sessionID, reason string, at time.Time) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	m.revokeCalls = append(m.revokeCalls, sessionID)
	if s, ok := m.byID[sessionID]; ok {
		s.RevokedAt = &at
		r := reason
		s.RevocationReason = &r
		s.Status = entities.SessionStatusRevoked
		return nil
	}
	return domain.ErrSessionNotFound
}

func (m *sessionRepoMock) RevokeChain(_ context.Context, sessionID, reason string, at time.Time) error {
	m.chainCalls = append(m.chainCalls, sessionID)
	// recorrer hacia arriba (parents)
	cur, ok := m.byID[sessionID]
	visited := map[string]struct{}{}
	for ok {
		if _, seen := visited[cur.ID]; seen {
			break
		}
		visited[cur.ID] = struct{}{}
		atCopy := at
		reasonCopy := reason
		cur.RevokedAt = &atCopy
		cur.RevocationReason = &reasonCopy
		cur.Status = entities.SessionStatusRevoked
		if cur.ParentSessionID == nil {
			break
		}
		cur, ok = m.byID[*cur.ParentSessionID]
	}
	// recorrer hacia abajo (children)
	for _, s := range m.byID {
		if s.ParentSessionID != nil {
			if _, marked := visited[*s.ParentSessionID]; marked {
				atCopy := at
				reasonCopy := reason
				s.RevokedAt = &atCopy
				s.RevocationReason = &reasonCopy
				s.Status = entities.SessionStatusRevoked
				visited[s.ID] = struct{}{}
			}
		}
	}
	return nil
}
