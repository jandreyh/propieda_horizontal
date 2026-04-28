// Package domain expone las interfaces y errores de dominio del modulo
// identity (usuarios, sesiones, MFA). La inversion de dependencias es
// estricta: nada en este paquete importa infraestructura.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/identity/domain/entities"
)

// Errores de dominio del modulo identity. Los usecases mapean estos a
// Problem+JSON de la capa HTTP.
var (
	// ErrUserNotFound se devuelve cuando una busqueda no encuentra al
	// usuario indicado (por id, email o documento). El usecase de login
	// trata esto como credencial invalida sin filtrar al cliente cual
	// fue el motivo.
	ErrUserNotFound = errors.New("identity: user not found")

	// ErrSessionNotFound se devuelve cuando un refresh token o session id
	// no corresponde a ninguna fila viva.
	ErrSessionNotFound = errors.New("identity: session not found")

	// ErrSessionRevoked indica que la sesion existe pero ya esta revocada.
	// Para refresh, el caller la usa para diferenciar reuso vs perdida.
	ErrSessionRevoked = errors.New("identity: session revoked")
)

// UserRepository abstrae el acceso a la tabla users. La interfaz vive
// en domain para respetar la inversion de dependencias: los usecases
// importan esto, las implementaciones (sqlc) estan en infrastructure.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByDocument(ctx context.Context, docType entities.DocumentType, docNumber string) (*entities.User, error)
	IncrementFailedAttempts(ctx context.Context, userID string) (int, error)
	ResetFailedAttempts(ctx context.Context, userID string) error
	LockUser(ctx context.Context, userID string, lockedUntil time.Time) error
	UnlockUser(ctx context.Context, userID string) error
	UpdateLastLoginAt(ctx context.Context, userID string, at time.Time) error
}

// SessionRepository abstrae el acceso a user_sessions. Los metodos
// modelan el ciclo de vida del refresh token: crear, buscar por hash,
// revocar individual o en cadena.
type SessionRepository interface {
	Create(ctx context.Context, s *entities.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*entities.Session, error)
	GetByID(ctx context.Context, id string) (*entities.Session, error)
	Revoke(ctx context.Context, sessionID string, reason string, at time.Time) error
	// RevokeChain revoca toda la cadena de sesiones que comparten linaje
	// con sessionID (siguiendo parent_session_id hacia atras y todas las
	// hijas hacia adelante). Usado en deteccion de reuso de refresh token.
	RevokeChain(ctx context.Context, sessionID string, reason string, at time.Time) error
}
