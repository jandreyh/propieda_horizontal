// Package domain define los puertos del modulo assemblies.
//
// La capa de aplicacion consume estas interfaces; la infra las implementa
// con sqlc + pgx. No hay SQL inline.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/saas-ph/api/internal/modules/assemblies/domain/entities"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrAssemblyNotFound se devuelve cuando una asamblea por id no existe.
var ErrAssemblyNotFound = errors.New("assemblies: assembly not found")

// ErrCallNotFound se devuelve cuando una convocatoria por id no existe.
var ErrCallNotFound = errors.New("assemblies: call not found")

// ErrAttendanceNotFound se devuelve cuando una asistencia por id no existe.
var ErrAttendanceNotFound = errors.New("assemblies: attendance not found")

// ErrAttendanceDuplicate se devuelve cuando ya existe asistencia para la
// misma unidad en la misma asamblea.
var ErrAttendanceDuplicate = errors.New("assemblies: attendance already registered for this unit")

// ErrProxyNotFound se devuelve cuando un poder por id no existe.
var ErrProxyNotFound = errors.New("assemblies: proxy not found")

// ErrProxyDuplicate se devuelve cuando ya existe un poder activo para la
// misma unidad en la misma asamblea.
var ErrProxyDuplicate = errors.New("assemblies: proxy already exists for this unit")

// ErrMotionNotFound se devuelve cuando una mocion por id no existe.
var ErrMotionNotFound = errors.New("assemblies: motion not found")

// ErrVoteNotFound se devuelve cuando un voto por id no existe.
var ErrVoteNotFound = errors.New("assemblies: vote not found")

// ErrVoteDuplicate se devuelve cuando ya existe un voto activo para la
// misma unidad en la misma mocion.
var ErrVoteDuplicate = errors.New("assemblies: active vote already exists for this unit on this motion")

// ErrActNotFound se devuelve cuando un acta por id no existe.
var ErrActNotFound = errors.New("assemblies: act not found")

// ErrActImmutable se devuelve cuando se intenta modificar un acta firmada.
var ErrActImmutable = errors.New("assemblies: act is signed and immutable")

// ErrSignatureDuplicate se devuelve cuando ya existe una firma valida para
// el mismo rol en el mismo acta.
var ErrSignatureDuplicate = errors.New("assemblies: signature already exists for this role")

// ErrVersionConflict se devuelve cuando un UPDATE optimista no afecto
// filas porque la version cambio.
var ErrVersionConflict = errors.New("assemblies: version conflict")

// ErrInvalidTransition se devuelve cuando se intenta transicionar el
// status de una entidad a un estado no permitido.
var ErrInvalidTransition = errors.New("assemblies: invalid status transition")

// ErrVoteHashDuplicate se devuelve cuando el vote_hash ya existe.
var ErrVoteHashDuplicate = errors.New("assemblies: vote hash already exists")

// ---------------------------------------------------------------------------
// AssemblyRepository
// ---------------------------------------------------------------------------

// CreateAssemblyInput agrupa los datos para persistir una asamblea nueva.
type CreateAssemblyInput struct {
	Name              string
	AssemblyType      entities.AssemblyType
	ScheduledAt       time.Time
	VotingMode        entities.VotingMode
	QuorumRequiredPct float64
	Location          *string
	Notes             *string
	ActorID           string
}

// UpdateAssemblyInput agrupa los datos para actualizar una asamblea.
type UpdateAssemblyInput struct {
	ID              string
	Status          entities.AssemblyStatus
	StartedAt       *time.Time
	ClosedAt        *time.Time
	ExpectedVersion int32
	ActorID         string
}

// AssemblyRepository es el puerto que persiste asambleas.
type AssemblyRepository interface {
	Create(ctx context.Context, in CreateAssemblyInput) (entities.Assembly, error)
	GetByID(ctx context.Context, id string) (entities.Assembly, error)
	List(ctx context.Context) ([]entities.Assembly, error)
	UpdateStatus(ctx context.Context, in UpdateAssemblyInput) (entities.Assembly, error)
}

// ---------------------------------------------------------------------------
// CallRepository
// ---------------------------------------------------------------------------

// CreateCallInput agrupa los datos para persistir una convocatoria.
type CreateCallInput struct {
	AssemblyID  string
	Channels    []byte
	Agenda      []byte
	BodyMD      *string
	PublishedBy string
}

// CallRepository es el puerto que persiste convocatorias.
type CallRepository interface {
	Create(ctx context.Context, in CreateCallInput) (entities.AssemblyCall, error)
	GetByID(ctx context.Context, id string) (entities.AssemblyCall, error)
	ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyCall, error)
}

// ---------------------------------------------------------------------------
// AttendanceRepository
// ---------------------------------------------------------------------------

// CreateAttendanceInput agrupa los datos para registrar asistencia.
type CreateAttendanceInput struct {
	AssemblyID          string
	UnitID              string
	AttendeeUserID      *string
	RepresentedByUserID *string
	CoefficientAtEvent  float64
	IsRemote            bool
	HasVotingRight      bool
	Notes               *string
	ActorID             string
}

// AttendanceRepository es el puerto que persiste asistencia.
type AttendanceRepository interface {
	Create(ctx context.Context, in CreateAttendanceInput) (entities.AssemblyAttendance, error)
	GetByID(ctx context.Context, id string) (entities.AssemblyAttendance, error)
	ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyAttendance, error)
	SumCoefficientByAssemblyID(ctx context.Context, assemblyID string) (float64, error)
}

// ---------------------------------------------------------------------------
// ProxyRepository
// ---------------------------------------------------------------------------

// CreateProxyInput agrupa los datos para registrar un poder.
type CreateProxyInput struct {
	AssemblyID    string
	GrantorUserID string
	ProxyUserID   string
	UnitID        string
	DocumentURL   *string
	DocumentHash  *string
	ActorID       string
}

// ProxyRepository es el puerto que persiste poderes.
type ProxyRepository interface {
	Create(ctx context.Context, in CreateProxyInput) (entities.AssemblyProxy, error)
	GetByID(ctx context.Context, id string) (entities.AssemblyProxy, error)
	ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyProxy, error)
	CountByProxyUser(ctx context.Context, assemblyID, proxyUserID string) (int, error)
	ValidateProxy(ctx context.Context, id string, validatedBy string, expectedVersion int32) (entities.AssemblyProxy, error)
}

// ---------------------------------------------------------------------------
// MotionRepository
// ---------------------------------------------------------------------------

// CreateMotionInput agrupa los datos para persistir una mocion.
type CreateMotionInput struct {
	AssemblyID   string
	Title        string
	Description  *string
	DecisionType entities.DecisionType
	VotingMethod entities.VotingMethod
	Options      []byte
	ActorID      string
}

// UpdateMotionStatusInput agrupa los datos para actualizar el status
// de una mocion.
type UpdateMotionStatusInput struct {
	ID              string
	Status          entities.MotionStatus
	OpensAt         *time.Time
	ClosesAt        *time.Time
	Results         []byte
	ExpectedVersion int32
	ActorID         string
}

// MotionRepository es el puerto que persiste mociones.
type MotionRepository interface {
	Create(ctx context.Context, in CreateMotionInput) (entities.AssemblyMotion, error)
	GetByID(ctx context.Context, id string) (entities.AssemblyMotion, error)
	ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyMotion, error)
	UpdateStatus(ctx context.Context, in UpdateMotionStatusInput) (entities.AssemblyMotion, error)
}

// ---------------------------------------------------------------------------
// VoteRepository
// ---------------------------------------------------------------------------

// CreateVoteInput agrupa los datos para persistir un voto.
type CreateVoteInput struct {
	MotionID        string
	VoterUserID     string
	UnitID          string
	CoefficientUsed float64
	Option          string
	CastAt          time.Time
	PrevVoteHash    *string
	VoteHash        string
	Nonce           string
	IsProxyVote     bool
	ActorID         string
}

// VoteRepository es el puerto que persiste votos.
type VoteRepository interface {
	Create(ctx context.Context, in CreateVoteInput) (entities.Vote, error)
	GetByID(ctx context.Context, id string) (entities.Vote, error)
	ListByMotionID(ctx context.Context, motionID string) ([]entities.Vote, error)
	GetActiveByMotionAndUnit(ctx context.Context, motionID, unitID string) (entities.Vote, error)
	GetLastVoteHash(ctx context.Context, motionID string) (*string, error)
	VoidVote(ctx context.Context, id string, expectedVersion int32, actorID string) error
}

// ---------------------------------------------------------------------------
// VoteEvidenceRepository
// ---------------------------------------------------------------------------

// CreateVoteEvidenceInput agrupa los datos para persistir evidencia de voto.
type CreateVoteEvidenceInput struct {
	VoteID       string
	MotionID     string
	PrevVoteHash *string
	VoteHash     string
	PayloadJSON  []byte
	ClientIP     *string
	UserAgent    *string
	NTPOffsetMS  *int32
}

// VoteEvidenceRepository es el puerto que persiste evidencia de votos.
type VoteEvidenceRepository interface {
	Create(ctx context.Context, in CreateVoteEvidenceInput) (entities.VoteEvidence, error)
	ListByMotionID(ctx context.Context, motionID string) ([]entities.VoteEvidence, error)
}

// ---------------------------------------------------------------------------
// ActRepository
// ---------------------------------------------------------------------------

// CreateActInput agrupa los datos para persistir un acta.
type CreateActInput struct {
	AssemblyID   string
	BodyMD       string
	ArchiveUntil *time.Time
	ActorID      string
}

// UpdateActInput agrupa los datos para actualizar un acta.
type UpdateActInput struct {
	ID              string
	Status          entities.ActStatus
	SealedAt        *time.Time
	ExpectedVersion int32
	ActorID         string
}

// ActRepository es el puerto que persiste actas.
type ActRepository interface {
	Create(ctx context.Context, in CreateActInput) (entities.Act, error)
	GetByID(ctx context.Context, id string) (entities.Act, error)
	GetByAssemblyID(ctx context.Context, assemblyID string) (entities.Act, error)
	UpdateStatus(ctx context.Context, in UpdateActInput) (entities.Act, error)
}

// ---------------------------------------------------------------------------
// ActSignatureRepository
// ---------------------------------------------------------------------------

// CreateSignatureInput agrupa los datos para persistir una firma.
type CreateSignatureInput struct {
	ActID           string
	SignerUserID    string
	Role            entities.SignatureRole
	SignatureMethod entities.SignatureMethod
	EvidenceHash    string
	ClientIP        *string
	UserAgent       *string
	ActorID         string
}

// ActSignatureRepository es el puerto que persiste firmas de actas.
type ActSignatureRepository interface {
	Create(ctx context.Context, in CreateSignatureInput) (entities.ActSignature, error)
	ListByActID(ctx context.Context, actID string) ([]entities.ActSignature, error)
}

// ---------------------------------------------------------------------------
// OutboxRepository
// ---------------------------------------------------------------------------

// EnqueueOutboxInput agrupa los datos para encolar un evento.
type EnqueueOutboxInput struct {
	AggregateID string
	EventType   entities.OutboxEventType
	Payload     []byte
}

// OutboxRepository es el puerto que persiste eventos del outbox.
type OutboxRepository interface {
	Enqueue(ctx context.Context, in EnqueueOutboxInput) (entities.OutboxEvent, error)
	LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error)
	MarkDelivered(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error
}
