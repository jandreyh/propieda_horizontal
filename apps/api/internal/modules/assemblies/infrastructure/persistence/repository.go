// Package persistence implementa los puertos del modulo assemblies usando
// el codigo generado por sqlc.
//
// Reglas:
//   - El pool del Tenant DB se obtiene del contexto via tenantctx.FromCtx.
//   - NO se usa database/sql ni SQL inline.
//   - Las usecases que requieren atomicidad multi-tabla pasan un pgx.Tx
//     en el contexto via WithTx(ctx, tx). Si esta presente, los repos lo
//     usan; si no, usan el pool del tenant.
package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saas-ph/api/internal/modules/assemblies/domain"
	"github.com/saas-ph/api/internal/modules/assemblies/domain/entities"
	assembliesdb "github.com/saas-ph/api/internal/modules/assemblies/infrastructure/persistence/sqlcgen"
	"github.com/saas-ph/api/internal/platform/tenantctx"
)

// --- ctx helper para transaccion ---

type txCtxKey struct{}

// WithTx inyecta una transaccion pgx en el contexto.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

func txFromCtx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx)
	return tx, ok
}

func querier(ctx context.Context) (*assembliesdb.Queries, error) {
	if tx, ok := txFromCtx(ctx); ok && tx != nil {
		return assembliesdb.New(tx), nil
	}
	t, err := tenantctx.FromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if t.Pool == nil {
		return nil, errors.New("assemblies: tenant pool is nil")
	}
	return assembliesdb.New(t.Pool), nil
}

// ===========================================================================
// AssemblyRepository
// ===========================================================================

// AssemblyRepository implementa domain.AssemblyRepository.
type AssemblyRepository struct{}

// NewAssemblyRepository construye una instancia stateless.
func NewAssemblyRepository() *AssemblyRepository { return &AssemblyRepository{} }

// Create implementa domain.AssemblyRepository.
func (r *AssemblyRepository) Create(ctx context.Context, in domain.CreateAssemblyInput) (entities.Assembly, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Assembly{}, err
	}
	row, err := q.CreateAssembly(ctx, assembliesdb.CreateAssemblyParams{
		Name:              in.Name,
		AssemblyType:      string(in.AssemblyType),
		ScheduledAt:       timeToPgTimestamptz(in.ScheduledAt),
		VotingMode:        string(in.VotingMode),
		QuorumRequiredPct: float64ToNumeric(in.QuorumRequiredPct),
		Location:          in.Location,
		Notes:             in.Notes,
		CreatedBy:         uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.Assembly{}, err
	}
	return mapAssembly(row), nil
}

// GetByID implementa domain.AssemblyRepository.
func (r *AssemblyRepository) GetByID(ctx context.Context, id string) (entities.Assembly, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Assembly{}, err
	}
	row, err := q.GetAssemblyByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Assembly{}, domain.ErrAssemblyNotFound
		}
		return entities.Assembly{}, err
	}
	return mapAssembly(row), nil
}

// List implementa domain.AssemblyRepository.
func (r *AssemblyRepository) List(ctx context.Context) ([]entities.Assembly, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssemblies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entities.Assembly, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAssembly(row))
	}
	return out, nil
}

// UpdateStatus implementa domain.AssemblyRepository.
func (r *AssemblyRepository) UpdateStatus(ctx context.Context, in domain.UpdateAssemblyInput) (entities.Assembly, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Assembly{}, err
	}
	row, err := q.UpdateAssemblyStatus(ctx, assembliesdb.UpdateAssemblyStatusParams{
		NewStatus:       string(in.Status),
		NewStartedAt:    timePtrToPgTimestamptz(in.StartedAt),
		NewClosedAt:     timePtrToPgTimestamptz(in.ClosedAt),
		UpdatedBy:       uuidToPgtype(in.ActorID),
		ID:              uuidToPgtype(in.ID),
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Assembly{}, domain.ErrVersionConflict
		}
		return entities.Assembly{}, err
	}
	return mapAssembly(row), nil
}

// ===========================================================================
// CallRepository
// ===========================================================================

// CallRepository implementa domain.CallRepository.
type CallRepository struct{}

// NewCallRepository construye una instancia stateless.
func NewCallRepository() *CallRepository { return &CallRepository{} }

// Create implementa domain.CallRepository.
func (r *CallRepository) Create(ctx context.Context, in domain.CreateCallInput) (entities.AssemblyCall, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyCall{}, err
	}
	row, err := q.CreateAssemblyCall(ctx, assembliesdb.CreateAssemblyCallParams{
		AssemblyID:  uuidToPgtype(in.AssemblyID),
		Channels:    in.Channels,
		Agenda:      in.Agenda,
		BodyMD:      in.BodyMD,
		PublishedBy: uuidToPgtype(in.PublishedBy),
	})
	if err != nil {
		return entities.AssemblyCall{}, err
	}
	return mapCall(row), nil
}

// GetByID implementa domain.CallRepository.
func (r *CallRepository) GetByID(ctx context.Context, id string) (entities.AssemblyCall, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyCall{}, err
	}
	row, err := q.GetAssemblyCallByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.AssemblyCall{}, domain.ErrCallNotFound
		}
		return entities.AssemblyCall{}, err
	}
	return mapCall(row), nil
}

// ListByAssemblyID implementa domain.CallRepository.
func (r *CallRepository) ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyCall, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssemblyCallsByAssemblyID(ctx, uuidToPgtype(assemblyID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.AssemblyCall, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCall(row))
	}
	return out, nil
}

// ===========================================================================
// AttendanceRepository
// ===========================================================================

// AttendanceRepository implementa domain.AttendanceRepository.
type AttendanceRepository struct{}

// NewAttendanceRepository construye una instancia stateless.
func NewAttendanceRepository() *AttendanceRepository { return &AttendanceRepository{} }

// Create implementa domain.AttendanceRepository.
func (r *AttendanceRepository) Create(ctx context.Context, in domain.CreateAttendanceInput) (entities.AssemblyAttendance, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyAttendance{}, err
	}
	row, err := q.CreateAssemblyAttendance(ctx, assembliesdb.CreateAssemblyAttendanceParams{
		AssemblyID:          uuidToPgtype(in.AssemblyID),
		UnitID:              uuidToPgtype(in.UnitID),
		AttendeeUserID:      uuidToPgtypePtr(in.AttendeeUserID),
		RepresentedByUserID: uuidToPgtypePtr(in.RepresentedByUserID),
		CoefficientAtEvent:  float64ToNumeric(in.CoefficientAtEvent),
		IsRemote:            in.IsRemote,
		HasVotingRight:      in.HasVotingRight,
		Notes:               in.Notes,
		CreatedBy:           uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.AssemblyAttendance{}, domain.ErrAttendanceDuplicate
		}
		return entities.AssemblyAttendance{}, err
	}
	return mapAttendance(row), nil
}

// GetByID implementa domain.AttendanceRepository.
func (r *AttendanceRepository) GetByID(ctx context.Context, id string) (entities.AssemblyAttendance, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyAttendance{}, err
	}
	row, err := q.GetAssemblyAttendanceByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.AssemblyAttendance{}, domain.ErrAttendanceNotFound
		}
		return entities.AssemblyAttendance{}, err
	}
	return mapAttendance(row), nil
}

// ListByAssemblyID implementa domain.AttendanceRepository.
func (r *AttendanceRepository) ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyAttendance, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssemblyAttendancesByAssemblyID(ctx, uuidToPgtype(assemblyID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.AssemblyAttendance, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAttendance(row))
	}
	return out, nil
}

// SumCoefficientByAssemblyID implementa domain.AttendanceRepository.
func (r *AttendanceRepository) SumCoefficientByAssemblyID(ctx context.Context, assemblyID string) (float64, error) {
	q, err := querier(ctx)
	if err != nil {
		return 0, err
	}
	n, err := q.SumCoefficientByAssemblyID(ctx, uuidToPgtype(assemblyID))
	if err != nil {
		return 0, err
	}
	f, fErr := numericToFloat64(n)
	if fErr != nil {
		return 0, fErr
	}
	return f, nil
}

// ===========================================================================
// ProxyRepository
// ===========================================================================

// ProxyRepository implementa domain.ProxyRepository.
type ProxyRepository struct{}

// NewProxyRepository construye una instancia stateless.
func NewProxyRepository() *ProxyRepository { return &ProxyRepository{} }

// Create implementa domain.ProxyRepository.
func (r *ProxyRepository) Create(ctx context.Context, in domain.CreateProxyInput) (entities.AssemblyProxy, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyProxy{}, err
	}
	row, err := q.CreateAssemblyProxy(ctx, assembliesdb.CreateAssemblyProxyParams{
		AssemblyID:    uuidToPgtype(in.AssemblyID),
		GrantorUserID: uuidToPgtype(in.GrantorUserID),
		ProxyUserID:   uuidToPgtype(in.ProxyUserID),
		UnitID:        uuidToPgtype(in.UnitID),
		DocumentURL:   in.DocumentURL,
		DocumentHash:  in.DocumentHash,
		CreatedBy:     uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.AssemblyProxy{}, domain.ErrProxyDuplicate
		}
		return entities.AssemblyProxy{}, err
	}
	return mapProxy(row), nil
}

// GetByID implementa domain.ProxyRepository.
func (r *ProxyRepository) GetByID(ctx context.Context, id string) (entities.AssemblyProxy, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyProxy{}, err
	}
	row, err := q.GetAssemblyProxyByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.AssemblyProxy{}, domain.ErrProxyNotFound
		}
		return entities.AssemblyProxy{}, err
	}
	return mapProxy(row), nil
}

// ListByAssemblyID implementa domain.ProxyRepository.
func (r *ProxyRepository) ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyProxy, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssemblyProxiesByAssemblyID(ctx, uuidToPgtype(assemblyID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.AssemblyProxy, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapProxy(row))
	}
	return out, nil
}

// CountByProxyUser implementa domain.ProxyRepository.
func (r *ProxyRepository) CountByProxyUser(ctx context.Context, assemblyID, proxyUserID string) (int, error) {
	q, err := querier(ctx)
	if err != nil {
		return 0, err
	}
	count, err := q.CountProxiesByProxyUser(ctx, uuidToPgtype(assemblyID), uuidToPgtype(proxyUserID))
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// ValidateProxy implementa domain.ProxyRepository.
func (r *ProxyRepository) ValidateProxy(ctx context.Context, id string, validatedBy string, expectedVersion int32) (entities.AssemblyProxy, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyProxy{}, err
	}
	row, err := q.ValidateAssemblyProxy(ctx, assembliesdb.ValidateAssemblyProxyParams{
		ValidatedBy:     uuidToPgtype(validatedBy),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.AssemblyProxy{}, domain.ErrVersionConflict
		}
		return entities.AssemblyProxy{}, err
	}
	return mapProxy(row), nil
}

// ===========================================================================
// MotionRepository
// ===========================================================================

// MotionRepository implementa domain.MotionRepository.
type MotionRepository struct{}

// NewMotionRepository construye una instancia stateless.
func NewMotionRepository() *MotionRepository { return &MotionRepository{} }

// Create implementa domain.MotionRepository.
func (r *MotionRepository) Create(ctx context.Context, in domain.CreateMotionInput) (entities.AssemblyMotion, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyMotion{}, err
	}
	row, err := q.CreateAssemblyMotion(ctx, assembliesdb.CreateAssemblyMotionParams{
		AssemblyID:   uuidToPgtype(in.AssemblyID),
		Title:        in.Title,
		Description:  in.Description,
		DecisionType: string(in.DecisionType),
		VotingMethod: string(in.VotingMethod),
		Options:      in.Options,
		CreatedBy:    uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.AssemblyMotion{}, err
	}
	return mapMotion(row), nil
}

// GetByID implementa domain.MotionRepository.
func (r *MotionRepository) GetByID(ctx context.Context, id string) (entities.AssemblyMotion, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyMotion{}, err
	}
	row, err := q.GetAssemblyMotionByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.AssemblyMotion{}, domain.ErrMotionNotFound
		}
		return entities.AssemblyMotion{}, err
	}
	return mapMotion(row), nil
}

// ListByAssemblyID implementa domain.MotionRepository.
func (r *MotionRepository) ListByAssemblyID(ctx context.Context, assemblyID string) ([]entities.AssemblyMotion, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListAssemblyMotionsByAssemblyID(ctx, uuidToPgtype(assemblyID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.AssemblyMotion, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapMotion(row))
	}
	return out, nil
}

// UpdateStatus implementa domain.MotionRepository.
func (r *MotionRepository) UpdateStatus(ctx context.Context, in domain.UpdateMotionStatusInput) (entities.AssemblyMotion, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.AssemblyMotion{}, err
	}
	row, err := q.UpdateAssemblyMotionStatus(ctx, assembliesdb.UpdateAssemblyMotionStatusParams{
		NewStatus:       string(in.Status),
		NewOpensAt:      timePtrToPgTimestamptz(in.OpensAt),
		NewClosesAt:     timePtrToPgTimestamptz(in.ClosesAt),
		NewResults:      in.Results,
		UpdatedBy:       uuidToPgtype(in.ActorID),
		ID:              uuidToPgtype(in.ID),
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.AssemblyMotion{}, domain.ErrVersionConflict
		}
		return entities.AssemblyMotion{}, err
	}
	return mapMotion(row), nil
}

// ===========================================================================
// VoteRepository
// ===========================================================================

// VoteRepository implementa domain.VoteRepository.
type VoteRepository struct{}

// NewVoteRepository construye una instancia stateless.
func NewVoteRepository() *VoteRepository { return &VoteRepository{} }

// Create implementa domain.VoteRepository.
func (r *VoteRepository) Create(ctx context.Context, in domain.CreateVoteInput) (entities.Vote, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Vote{}, err
	}
	row, err := q.CreateVote(ctx, assembliesdb.CreateVoteParams{
		MotionID:        uuidToPgtype(in.MotionID),
		VoterUserID:     uuidToPgtype(in.VoterUserID),
		UnitID:          uuidToPgtype(in.UnitID),
		CoefficientUsed: float64ToNumeric(in.CoefficientUsed),
		Option:          in.Option,
		CastAt:          timeToPgTimestamptz(in.CastAt),
		PrevVoteHash:    in.PrevVoteHash,
		VoteHash:        in.VoteHash,
		Nonce:           in.Nonce,
		IsProxyVote:     in.IsProxyVote,
		CreatedBy:       uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.Vote{}, domain.ErrVoteDuplicate
		}
		return entities.Vote{}, err
	}
	return mapVote(row), nil
}

// GetByID implementa domain.VoteRepository.
func (r *VoteRepository) GetByID(ctx context.Context, id string) (entities.Vote, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Vote{}, err
	}
	row, err := q.GetVoteByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Vote{}, domain.ErrVoteNotFound
		}
		return entities.Vote{}, err
	}
	return mapVote(row), nil
}

// ListByMotionID implementa domain.VoteRepository.
func (r *VoteRepository) ListByMotionID(ctx context.Context, motionID string) ([]entities.Vote, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListVotesByMotionID(ctx, uuidToPgtype(motionID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.Vote, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapVote(row))
	}
	return out, nil
}

// GetActiveByMotionAndUnit implementa domain.VoteRepository.
func (r *VoteRepository) GetActiveByMotionAndUnit(ctx context.Context, motionID, unitID string) (entities.Vote, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Vote{}, err
	}
	row, err := q.GetActiveVoteByMotionAndUnit(ctx, uuidToPgtype(motionID), uuidToPgtype(unitID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Vote{}, domain.ErrVoteNotFound
		}
		return entities.Vote{}, err
	}
	return mapVote(row), nil
}

// GetLastVoteHash implementa domain.VoteRepository.
func (r *VoteRepository) GetLastVoteHash(ctx context.Context, motionID string) (*string, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	hash, err := q.GetLastVoteHash(ctx, uuidToPgtype(motionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return hash, nil
}

// VoidVote implementa domain.VoteRepository.
func (r *VoteRepository) VoidVote(ctx context.Context, id string, expectedVersion int32, actorID string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.VoidVote(ctx, assembliesdb.VoidVoteParams{
		UpdatedBy:       uuidToPgtype(actorID),
		ID:              uuidToPgtype(id),
		ExpectedVersion: expectedVersion,
	})
}

// ===========================================================================
// VoteEvidenceRepository
// ===========================================================================

// VoteEvidenceRepository implementa domain.VoteEvidenceRepository.
type VoteEvidenceRepository struct{}

// NewVoteEvidenceRepository construye una instancia stateless.
func NewVoteEvidenceRepository() *VoteEvidenceRepository { return &VoteEvidenceRepository{} }

// Create implementa domain.VoteEvidenceRepository.
func (r *VoteEvidenceRepository) Create(ctx context.Context, in domain.CreateVoteEvidenceInput) (entities.VoteEvidence, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.VoteEvidence{}, err
	}
	row, err := q.CreateVoteEvidence(ctx, assembliesdb.CreateVoteEvidenceParams{
		VoteID:       uuidToPgtype(in.VoteID),
		MotionID:     uuidToPgtype(in.MotionID),
		PrevVoteHash: in.PrevVoteHash,
		VoteHash:     in.VoteHash,
		PayloadJSON:  in.PayloadJSON,
		ClientIP:     in.ClientIP,
		UserAgent:    in.UserAgent,
		NTPOffsetMS:  in.NTPOffsetMS,
	})
	if err != nil {
		return entities.VoteEvidence{}, err
	}
	return mapVoteEvidence(row), nil
}

// ListByMotionID implementa domain.VoteEvidenceRepository.
func (r *VoteEvidenceRepository) ListByMotionID(ctx context.Context, motionID string) ([]entities.VoteEvidence, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListVoteEvidenceByMotionID(ctx, uuidToPgtype(motionID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.VoteEvidence, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapVoteEvidence(row))
	}
	return out, nil
}

// ===========================================================================
// ActRepository
// ===========================================================================

// ActRepository implementa domain.ActRepository.
type ActRepository struct{}

// NewActRepository construye una instancia stateless.
func NewActRepository() *ActRepository { return &ActRepository{} }

// Create implementa domain.ActRepository.
func (r *ActRepository) Create(ctx context.Context, in domain.CreateActInput) (entities.Act, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Act{}, err
	}
	row, err := q.CreateAct(ctx, assembliesdb.CreateActParams{
		AssemblyID:   uuidToPgtype(in.AssemblyID),
		BodyMD:       in.BodyMD,
		ArchiveUntil: timePtrToPgDate(in.ArchiveUntil),
		CreatedBy:    uuidToPgtype(in.ActorID),
	})
	if err != nil {
		return entities.Act{}, err
	}
	return mapAct(row), nil
}

// GetByID implementa domain.ActRepository.
func (r *ActRepository) GetByID(ctx context.Context, id string) (entities.Act, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Act{}, err
	}
	row, err := q.GetActByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Act{}, domain.ErrActNotFound
		}
		return entities.Act{}, err
	}
	return mapAct(row), nil
}

// GetByAssemblyID implementa domain.ActRepository.
func (r *ActRepository) GetByAssemblyID(ctx context.Context, assemblyID string) (entities.Act, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Act{}, err
	}
	row, err := q.GetActByAssemblyID(ctx, uuidToPgtype(assemblyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Act{}, domain.ErrActNotFound
		}
		return entities.Act{}, err
	}
	return mapAct(row), nil
}

// UpdateStatus implementa domain.ActRepository.
func (r *ActRepository) UpdateStatus(ctx context.Context, in domain.UpdateActInput) (entities.Act, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.Act{}, err
	}
	row, err := q.UpdateActStatus(ctx, assembliesdb.UpdateActStatusParams{
		NewStatus:       string(in.Status),
		NewSealedAt:     timePtrToPgTimestamptz(in.SealedAt),
		UpdatedBy:       uuidToPgtype(in.ActorID),
		ID:              uuidToPgtype(in.ID),
		ExpectedVersion: in.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Act{}, domain.ErrVersionConflict
		}
		if isCheckViolation(err) {
			return entities.Act{}, domain.ErrActImmutable
		}
		return entities.Act{}, err
	}
	return mapAct(row), nil
}

// ===========================================================================
// ActSignatureRepository
// ===========================================================================

// ActSignatureRepository implementa domain.ActSignatureRepository.
type ActSignatureRepository struct{}

// NewActSignatureRepository construye una instancia stateless.
func NewActSignatureRepository() *ActSignatureRepository { return &ActSignatureRepository{} }

// Create implementa domain.ActSignatureRepository.
func (r *ActSignatureRepository) Create(ctx context.Context, in domain.CreateSignatureInput) (entities.ActSignature, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.ActSignature{}, err
	}
	row, err := q.CreateActSignature(ctx, assembliesdb.CreateActSignatureParams{
		ActID:           uuidToPgtype(in.ActID),
		SignerUserID:    uuidToPgtype(in.SignerUserID),
		Role:            string(in.Role),
		SignatureMethod: string(in.SignatureMethod),
		EvidenceHash:    in.EvidenceHash,
		ClientIP:        in.ClientIP,
		UserAgent:       in.UserAgent,
		CreatedBy:       uuidToPgtype(in.ActorID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return entities.ActSignature{}, domain.ErrSignatureDuplicate
		}
		return entities.ActSignature{}, err
	}
	return mapSignature(row), nil
}

// ListByActID implementa domain.ActSignatureRepository.
func (r *ActSignatureRepository) ListByActID(ctx context.Context, actID string) ([]entities.ActSignature, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActSignaturesByActID(ctx, uuidToPgtype(actID))
	if err != nil {
		return nil, err
	}
	out := make([]entities.ActSignature, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapSignature(row))
	}
	return out, nil
}

// ===========================================================================
// OutboxRepository
// ===========================================================================

// OutboxRepository implementa domain.OutboxRepository.
type OutboxRepository struct{}

// NewOutboxRepository construye una instancia stateless.
func NewOutboxRepository() *OutboxRepository { return &OutboxRepository{} }

// Enqueue implementa domain.OutboxRepository.
func (r *OutboxRepository) Enqueue(ctx context.Context, in domain.EnqueueOutboxInput) (entities.OutboxEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return entities.OutboxEvent{}, err
	}
	row, err := q.EnqueueAssembliesOutboxEvent(ctx, assembliesdb.EnqueueAssembliesOutboxEventParams{
		AggregateID: uuidToPgtype(in.AggregateID),
		EventType:   string(in.EventType),
		Payload:     in.Payload,
	})
	if err != nil {
		return entities.OutboxEvent{}, err
	}
	return mapOutbox(row), nil
}

// LockPending implementa domain.OutboxRepository.
func (r *OutboxRepository) LockPending(ctx context.Context, limit int32) ([]entities.OutboxEvent, error) {
	q, err := querier(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.LockPendingAssembliesOutboxEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]entities.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOutbox(row))
	}
	return out, nil
}

// MarkDelivered implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkDelivered(ctx context.Context, id string) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	return q.MarkAssembliesOutboxEventDelivered(ctx, uuidToPgtype(id))
}

// MarkFailed implementa domain.OutboxRepository.
func (r *OutboxRepository) MarkFailed(ctx context.Context, id, lastError string, nextAttemptDeltaSeconds int) error {
	q, err := querier(ctx)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(nextAttemptDeltaSeconds) * time.Second)
	le := lastError
	return q.MarkAssembliesOutboxEventFailed(ctx, assembliesdb.MarkAssembliesOutboxEventFailedParams{
		LastError:     &le,
		NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: true},
		ID:            uuidToPgtype(id),
	})
}

// ===========================================================================
// Mapping helpers
// ===========================================================================

func mapAssembly(r assembliesdb.Assembly) entities.Assembly {
	out := entities.Assembly{
		ID:           uuidString(r.ID),
		Name:         r.Name,
		AssemblyType: entities.AssemblyType(r.AssemblyType),
		ScheduledAt:  tsToTime(r.ScheduledAt),
		VotingMode:   entities.VotingMode(r.VotingMode),
		Location:     r.Location,
		Notes:        r.Notes,
		Status:       entities.AssemblyStatus(r.Status),
		CreatedAt:    tsToTime(r.CreatedAt),
		UpdatedAt:    tsToTime(r.UpdatedAt),
		Version:      r.Version,
	}
	if r.QuorumRequiredPct.Valid {
		f, err := numericToFloat64(r.QuorumRequiredPct)
		if err == nil {
			out.QuorumRequiredPct = f
		}
	}
	if r.StartedAt.Valid {
		t := r.StartedAt.Time
		out.StartedAt = &t
	}
	if r.ClosedAt.Valid {
		t := r.ClosedAt.Time
		out.ClosedAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapCall(r assembliesdb.AssemblyCall) entities.AssemblyCall {
	out := entities.AssemblyCall{
		ID:          uuidString(r.ID),
		AssemblyID:  uuidString(r.AssemblyID),
		PublishedAt: tsToTime(r.PublishedAt),
		Channels:    r.Channels,
		Agenda:      r.Agenda,
		BodyMD:      r.BodyMD,
		Status:      entities.CallStatus(r.Status),
		CreatedAt:   tsToTime(r.CreatedAt),
		UpdatedAt:   tsToTime(r.UpdatedAt),
		Version:     r.Version,
	}
	out.PublishedBy = uuidStringPtr(r.PublishedBy)
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapAttendance(r assembliesdb.AssemblyAttendance) entities.AssemblyAttendance {
	out := entities.AssemblyAttendance{
		ID:             uuidString(r.ID),
		AssemblyID:     uuidString(r.AssemblyID),
		UnitID:         uuidString(r.UnitID),
		ArrivalAt:      tsToTime(r.ArrivalAt),
		IsRemote:       r.IsRemote,
		HasVotingRight: r.HasVotingRight,
		Notes:          r.Notes,
		Status:         entities.AttendanceStatus(r.Status),
		CreatedAt:      tsToTime(r.CreatedAt),
		UpdatedAt:      tsToTime(r.UpdatedAt),
		Version:        r.Version,
	}
	out.AttendeeUserID = uuidStringPtr(r.AttendeeUserID)
	out.RepresentedByUserID = uuidStringPtr(r.RepresentedByUserID)
	if r.CoefficientAtEvent.Valid {
		f, err := numericToFloat64(r.CoefficientAtEvent)
		if err == nil {
			out.CoefficientAtEvent = f
		}
	}
	if r.DepartureAt.Valid {
		t := r.DepartureAt.Time
		out.DepartureAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapProxy(r assembliesdb.AssemblyProxy) entities.AssemblyProxy {
	out := entities.AssemblyProxy{
		ID:            uuidString(r.ID),
		AssemblyID:    uuidString(r.AssemblyID),
		GrantorUserID: uuidString(r.GrantorUserID),
		ProxyUserID:   uuidString(r.ProxyUserID),
		UnitID:        uuidString(r.UnitID),
		DocumentURL:   r.DocumentURL,
		DocumentHash:  r.DocumentHash,
		Status:        entities.ProxyStatus(r.Status),
		CreatedAt:     tsToTime(r.CreatedAt),
		UpdatedAt:     tsToTime(r.UpdatedAt),
		Version:       r.Version,
	}
	if r.ValidatedAt.Valid {
		t := r.ValidatedAt.Time
		out.ValidatedAt = &t
	}
	out.ValidatedBy = uuidStringPtr(r.ValidatedBy)
	if r.RevokedAt.Valid {
		t := r.RevokedAt.Time
		out.RevokedAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapMotion(r assembliesdb.AssemblyMotion) entities.AssemblyMotion {
	out := entities.AssemblyMotion{
		ID:           uuidString(r.ID),
		AssemblyID:   uuidString(r.AssemblyID),
		Title:        r.Title,
		Description:  r.Description,
		DecisionType: entities.DecisionType(r.DecisionType),
		VotingMethod: entities.VotingMethod(r.VotingMethod),
		Options:      r.Options,
		Results:      r.Results,
		Status:       entities.MotionStatus(r.Status),
		CreatedAt:    tsToTime(r.CreatedAt),
		UpdatedAt:    tsToTime(r.UpdatedAt),
		Version:      r.Version,
	}
	if r.OpensAt.Valid {
		t := r.OpensAt.Time
		out.OpensAt = &t
	}
	if r.ClosesAt.Valid {
		t := r.ClosesAt.Time
		out.ClosesAt = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapVote(r assembliesdb.Vote) entities.Vote {
	out := entities.Vote{
		ID:           uuidString(r.ID),
		MotionID:     uuidString(r.MotionID),
		VoterUserID:  uuidString(r.VoterUserID),
		UnitID:       uuidString(r.UnitID),
		Option:       r.Option,
		CastAt:       tsToTime(r.CastAt),
		PrevVoteHash: r.PrevVoteHash,
		VoteHash:     r.VoteHash,
		Nonce:        r.Nonce,
		IsProxyVote:  r.IsProxyVote,
		Status:       entities.VoteStatus(r.Status),
		CreatedAt:    tsToTime(r.CreatedAt),
		UpdatedAt:    tsToTime(r.UpdatedAt),
		Version:      r.Version,
	}
	if r.CoefficientUsed.Valid {
		f, err := numericToFloat64(r.CoefficientUsed)
		if err == nil {
			out.CoefficientUsed = f
		}
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapVoteEvidence(r assembliesdb.VoteEvidence) entities.VoteEvidence {
	return entities.VoteEvidence{
		ID:           uuidString(r.ID),
		VoteID:       uuidString(r.VoteID),
		MotionID:     uuidString(r.MotionID),
		PrevVoteHash: r.PrevVoteHash,
		VoteHash:     r.VoteHash,
		PayloadJSON:  r.PayloadJSON,
		ClientIP:     r.ClientIP,
		UserAgent:    r.UserAgent,
		NTPOffsetMS:  r.NTPOffsetMS,
		SealedAt:     tsToTime(r.SealedAt),
		CreatedAt:    tsToTime(r.CreatedAt),
	}
}

func mapAct(r assembliesdb.Act) entities.Act {
	out := entities.Act{
		ID:         uuidString(r.ID),
		AssemblyID: uuidString(r.AssemblyID),
		BodyMD:     r.BodyMD,
		PDFURL:     r.PDFURL,
		PDFHash:    r.PDFHash,
		Status:     entities.ActStatus(r.Status),
		CreatedAt:  tsToTime(r.CreatedAt),
		UpdatedAt:  tsToTime(r.UpdatedAt),
		Version:    r.Version,
	}
	if r.SealedAt.Valid {
		t := r.SealedAt.Time
		out.SealedAt = &t
	}
	if r.ArchiveUntil.Valid {
		t := r.ArchiveUntil.Time
		out.ArchiveUntil = &t
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapSignature(r assembliesdb.ActSignature) entities.ActSignature {
	out := entities.ActSignature{
		ID:              uuidString(r.ID),
		ActID:           uuidString(r.ActID),
		SignerUserID:    uuidString(r.SignerUserID),
		Role:            entities.SignatureRole(r.Role),
		SignedAt:        tsToTime(r.SignedAt),
		SignatureMethod: entities.SignatureMethod(r.SignatureMethod),
		EvidenceHash:    r.EvidenceHash,
		ClientIP:        r.ClientIP,
		UserAgent:       r.UserAgent,
		Status:          entities.SignatureStatus(r.Status),
		CreatedAt:       tsToTime(r.CreatedAt),
		UpdatedAt:       tsToTime(r.UpdatedAt),
		Version:         r.Version,
	}
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		out.DeletedAt = &t
	}
	out.CreatedBy = uuidStringPtr(r.CreatedBy)
	out.UpdatedBy = uuidStringPtr(r.UpdatedBy)
	out.DeletedBy = uuidStringPtr(r.DeletedBy)
	return out
}

func mapOutbox(r assembliesdb.AssembliesOutboxEvent) entities.OutboxEvent {
	out := entities.OutboxEvent{
		ID:            uuidString(r.ID),
		AggregateID:   uuidString(r.AggregateID),
		EventType:     entities.OutboxEventType(r.EventType),
		Payload:       r.Payload,
		CreatedAt:     tsToTime(r.CreatedAt),
		NextAttemptAt: tsToTime(r.NextAttemptAt),
		Attempts:      r.Attempts,
		LastError:     r.LastError,
	}
	if r.DeliveredAt.Valid {
		t := r.DeliveredAt.Time
		out.DeliveredAt = &t
	}
	return out
}

// ===========================================================================
// pgtype helpers
// ===========================================================================

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func timeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func timePtrToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timePtrToPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	v, err := u.Value()
	if err != nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidString(u)
	if s == "" {
		return nil
	}
	return &s
}

func uuidToPgtype(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{Valid: false}
	}
	return u
}

func uuidToPgtypePtr(s *string) pgtype.UUID {
	if s == nil {
		return pgtype.UUID{Valid: false}
	}
	return uuidToPgtype(*s)
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(f); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return n
}

func numericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, errors.New("numeric is null")
	}
	f64, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	return f64.Float64, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23514"
	}
	return false
}

// Compile-time checks: each repo implements the domain port.
var (
	_ domain.AssemblyRepository     = (*AssemblyRepository)(nil)
	_ domain.CallRepository         = (*CallRepository)(nil)
	_ domain.AttendanceRepository   = (*AttendanceRepository)(nil)
	_ domain.ProxyRepository        = (*ProxyRepository)(nil)
	_ domain.MotionRepository       = (*MotionRepository)(nil)
	_ domain.VoteRepository         = (*VoteRepository)(nil)
	_ domain.VoteEvidenceRepository = (*VoteEvidenceRepository)(nil)
	_ domain.ActRepository          = (*ActRepository)(nil)
	_ domain.ActSignatureRepository = (*ActSignatureRepository)(nil)
	_ domain.OutboxRepository       = (*OutboxRepository)(nil)
)
