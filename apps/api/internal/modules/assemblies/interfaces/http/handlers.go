package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/saas-ph/api/internal/modules/assemblies/application/dto"
	"github.com/saas-ph/api/internal/modules/assemblies/application/usecases"
	"github.com/saas-ph/api/internal/modules/assemblies/domain"
	"github.com/saas-ph/api/internal/modules/assemblies/domain/entities"
	apperrors "github.com/saas-ph/api/internal/platform/errors"
)

// Dependencies agrupa las dependencias del modulo HTTP. El orquestador
// (cmd/api) construye repos y los inyecta aqui.
type Dependencies struct {
	Logger      *slog.Logger
	Assemblies  domain.AssemblyRepository
	Calls       domain.CallRepository
	Attendances domain.AttendanceRepository
	Proxies     domain.ProxyRepository
	Motions     domain.MotionRepository
	Votes       domain.VoteRepository
	Evidence    domain.VoteEvidenceRepository
	Acts        domain.ActRepository
	Signatures  domain.ActSignatureRepository
	Outbox      domain.OutboxRepository
	TxRunner    usecases.TxRunner
	Now         func() time.Time
	MaxProxies  int
}

func (d *Dependencies) validate() {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.MaxProxies <= 0 {
		d.MaxProxies = 1
	}
}

type handlers struct {
	deps Dependencies
}

func newHandlers(d Dependencies) *handlers {
	d.validate()
	return &handlers{deps: d}
}

// --- Assemblies ---

func (h *handlers) createAssembly(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateAssemblyRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	scheduledAt, err := time.Parse(time.RFC3339, body.ScheduledAt)
	if err != nil {
		h.fail(w, r, apperrors.BadRequest("scheduled_at: invalid RFC3339 format"))
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CreateAssembly{
		Assemblies: h.deps.Assemblies,
		Outbox:     h.deps.Outbox,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateAssemblyInput{
		Name:              body.Name,
		AssemblyType:      entities.AssemblyType(body.AssemblyType),
		ScheduledAt:       scheduledAt,
		VotingMode:        entities.VotingMode(body.VotingMode),
		QuorumRequiredPct: body.QuorumRequiredPct,
		Location:          body.Location,
		Notes:             body.Notes,
		ActorID:           actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, assemblyToDTO(out))
}

func (h *handlers) listAssemblies(w http.ResponseWriter, r *http.Request) {
	uc := usecases.ListAssemblies{Assemblies: h.deps.Assemblies}
	out, err := uc.Execute(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	resp := dto.ListAssembliesResponse{
		Items: make([]dto.AssemblyResponse, 0, len(out)),
		Total: len(out),
	}
	for _, a := range out {
		resp.Items = append(resp.Items, assemblyToDTO(a))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) getAssembly(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := usecases.GetAssembly{Assemblies: h.deps.Assemblies}
	out, err := uc.Execute(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, assemblyToDTO(out))
}

// --- Calls ---

func (h *handlers) publishCall(w http.ResponseWriter, r *http.Request) {
	assemblyID := chi.URLParam(r, "id")
	var body dto.CreateCallRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	channelsJSON, _ := json.Marshal(body.Channels)
	agendaJSON, _ := json.Marshal(body.Agenda)
	uc := usecases.PublishCall{
		Assemblies: h.deps.Assemblies,
		Calls:      h.deps.Calls,
		Outbox:     h.deps.Outbox,
		TxRunner:   h.deps.TxRunner,
	}
	out, err := uc.Execute(r.Context(), usecases.PublishCallInput{
		AssemblyID: assemblyID,
		Channels:   channelsJSON,
		Agenda:     agendaJSON,
		BodyMD:     body.BodyMD,
		ActorID:    actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, callToDTO(out))
}

// --- Start / Close ---

func (h *handlers) startAssembly(w http.ResponseWriter, r *http.Request) {
	assemblyID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.StartAssembly{
		Assemblies: h.deps.Assemblies,
		Outbox:     h.deps.Outbox,
		TxRunner:   h.deps.TxRunner,
		Now:        h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), assemblyID, actorID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, assemblyToDTO(out))
}

func (h *handlers) closeAssembly(w http.ResponseWriter, r *http.Request) {
	assemblyID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CloseAssembly{
		Assemblies: h.deps.Assemblies,
		Outbox:     h.deps.Outbox,
		TxRunner:   h.deps.TxRunner,
		Now:        h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), assemblyID, actorID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, assemblyToDTO(out))
}

// --- Attendances ---

func (h *handlers) registerAttendance(w http.ResponseWriter, r *http.Request) {
	assemblyID := chi.URLParam(r, "id")
	var body dto.CreateAttendanceRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.RegisterAttendance{
		Assemblies:  h.deps.Assemblies,
		Attendances: h.deps.Attendances,
	}
	out, err := uc.Execute(r.Context(), usecases.RegisterAttendanceInput{
		AssemblyID:          assemblyID,
		UnitID:              body.UnitID,
		AttendeeUserID:      body.AttendeeUserID,
		RepresentedByUserID: body.RepresentedByUserID,
		CoefficientAtEvent:  body.CoefficientAtEvent,
		IsRemote:            body.IsRemote,
		HasVotingRight:      body.HasVotingRight,
		Notes:               body.Notes,
		ActorID:             actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, attendanceToDTO(out))
}

// --- Proxies ---

func (h *handlers) registerProxy(w http.ResponseWriter, r *http.Request) {
	assemblyID := chi.URLParam(r, "id")
	var body dto.CreateProxyRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.RegisterProxy{
		Assemblies: h.deps.Assemblies,
		Proxies:    h.deps.Proxies,
	}
	out, err := uc.Execute(r.Context(), usecases.RegisterProxyInput{
		AssemblyID:    assemblyID,
		GrantorUserID: body.GrantorUserID,
		ProxyUserID:   body.ProxyUserID,
		UnitID:        body.UnitID,
		DocumentURL:   body.DocumentURL,
		DocumentHash:  body.DocumentHash,
		MaxProxies:    h.deps.MaxProxies,
		ActorID:       actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, proxyToDTO(out))
}

// --- Motions ---

func (h *handlers) createMotion(w http.ResponseWriter, r *http.Request) {
	assemblyID := chi.URLParam(r, "id")
	var body dto.CreateMotionRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	optionsJSON, _ := json.Marshal(body.Options)
	uc := usecases.CreateMotion{
		Assemblies: h.deps.Assemblies,
		Motions:    h.deps.Motions,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateMotionInput{
		AssemblyID:   assemblyID,
		Title:        body.Title,
		Description:  body.Description,
		DecisionType: entities.DecisionType(body.DecisionType),
		VotingMethod: entities.VotingMethod(body.VotingMethod),
		Options:      optionsJSON,
		ActorID:      actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, motionToDTO(out))
}

func (h *handlers) openVoting(w http.ResponseWriter, r *http.Request) {
	motionID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.OpenVoting{
		Motions:  h.deps.Motions,
		Outbox:   h.deps.Outbox,
		TxRunner: h.deps.TxRunner,
		Now:      h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), motionID, actorID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, motionToDTO(out))
}

func (h *handlers) closeVoting(w http.ResponseWriter, r *http.Request) {
	motionID := chi.URLParam(r, "id")
	actorID := actorIDFromCtx(r)
	uc := usecases.CloseVoting{
		Motions:  h.deps.Motions,
		Votes:    h.deps.Votes,
		Outbox:   h.deps.Outbox,
		TxRunner: h.deps.TxRunner,
		Now:      h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), motionID, actorID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, motionToDTO(out))
}

// --- Votes ---

func (h *handlers) castVote(w http.ResponseWriter, r *http.Request) {
	motionID := chi.URLParam(r, "id")
	var body dto.CastVoteRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.CastVote{
		Motions:  h.deps.Motions,
		Votes:    h.deps.Votes,
		Evidence: h.deps.Evidence,
		Outbox:   h.deps.Outbox,
		TxRunner: h.deps.TxRunner,
		Now:      h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.CastVoteInput{
		MotionID:        motionID,
		VoterUserID:     body.VoterUserID,
		UnitID:          body.UnitID,
		CoefficientUsed: body.CoefficientUsed,
		Option:          body.Option,
		IsProxyVote:     body.IsProxyVote,
		ClientIP:        body.ClientIP,
		UserAgent:       body.UserAgent,
		NTPOffsetMS:     body.NTPOffsetMS,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, voteToDTO(out))
}

func (h *handlers) getMotionResults(w http.ResponseWriter, r *http.Request) {
	motionID := chi.URLParam(r, "id")
	uc := usecases.GetMotionResults{
		Motions: h.deps.Motions,
		Votes:   h.deps.Votes,
	}
	out, err := uc.Execute(r.Context(), motionID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	votes := make([]dto.VoteResponse, 0, len(out.Votes))
	for _, v := range out.Votes {
		votes = append(votes, voteToDTO(v))
	}
	writeJSON(w, http.StatusOK, dto.MotionResultsResponse{
		Motion: motionToDTO(out.Motion),
		Votes:  votes,
		Total:  len(votes),
	})
}

// --- Acts ---

func (h *handlers) createAct(w http.ResponseWriter, r *http.Request) {
	assemblyID := chi.URLParam(r, "id")
	var body dto.CreateActRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	var archiveUntil *time.Time
	if body.ArchiveUntil != nil {
		t, err := time.Parse("2006-01-02", *body.ArchiveUntil)
		if err != nil {
			h.fail(w, r, apperrors.BadRequest("archive_until: invalid date format (YYYY-MM-DD)"))
			return
		}
		archiveUntil = &t
	}
	uc := usecases.CreateAct{
		Assemblies: h.deps.Assemblies,
		Acts:       h.deps.Acts,
	}
	out, err := uc.Execute(r.Context(), usecases.CreateActInput{
		AssemblyID:   assemblyID,
		BodyMD:       body.BodyMD,
		ArchiveUntil: archiveUntil,
		ActorID:      actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, actToDTO(out, nil))
}

func (h *handlers) signAct(w http.ResponseWriter, r *http.Request) {
	actID := chi.URLParam(r, "id")
	var body dto.SignActRequest
	if err := decodeJSON(r, &body); err != nil {
		h.fail(w, r, err)
		return
	}
	actorID := actorIDFromCtx(r)
	uc := usecases.SignAct{
		Acts:       h.deps.Acts,
		Signatures: h.deps.Signatures,
		Outbox:     h.deps.Outbox,
		TxRunner:   h.deps.TxRunner,
		Now:        h.deps.Now,
	}
	out, err := uc.Execute(r.Context(), usecases.SignActInput{
		ActID:           actID,
		SignerUserID:    body.SignerUserID,
		Role:            entities.SignatureRole(body.Role),
		SignatureMethod: entities.SignatureMethod(body.SignatureMethod),
		EvidenceHash:    body.EvidenceHash,
		ClientIP:        body.ClientIP,
		UserAgent:       body.UserAgent,
		ActorID:         actorID,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, signatureToDTO(out))
}

func (h *handlers) getAct(w http.ResponseWriter, r *http.Request) {
	actID := chi.URLParam(r, "id")
	uc := usecases.GetAct{
		Acts:       h.deps.Acts,
		Signatures: h.deps.Signatures,
	}
	out, err := uc.Execute(r.Context(), actID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	sigs := make([]dto.SignatureResponse, 0, len(out.Signatures))
	for _, s := range out.Signatures {
		sigs = append(sigs, signatureToDTO(s))
	}
	writeJSON(w, http.StatusOK, actToDTO(out.Act, sigs))
}

// --- helpers ---

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p apperrors.Problem
	if errors.As(err, &p) {
		p = p.WithInstance(r.URL.Path)
		if p.Status >= 500 {
			h.deps.Logger.ErrorContext(r.Context(), "assemblies: server error",
				slog.String("path", r.URL.Path),
				slog.String("err", err.Error()))
		}
		apperrors.Write(w, p)
		return
	}
	h.deps.Logger.ErrorContext(r.Context(), "assemblies: unexpected error",
		slog.String("path", r.URL.Path),
		slog.String("err", err.Error()))
	apperrors.Write(w, apperrors.Internal("").WithInstance(r.URL.Path))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return apperrors.BadRequest("invalid JSON body: " + err.Error())
	}
	return nil
}

// --- Entity-to-DTO mapping ---

func assemblyToDTO(a entities.Assembly) dto.AssemblyResponse {
	return dto.AssemblyResponse{
		ID:                a.ID,
		Name:              a.Name,
		AssemblyType:      string(a.AssemblyType),
		ScheduledAt:       dto.FormatTime(a.ScheduledAt),
		VotingMode:        string(a.VotingMode),
		QuorumRequiredPct: a.QuorumRequiredPct,
		Location:          a.Location,
		Notes:             a.Notes,
		StartedAt:         dto.FormatTimePtr(a.StartedAt),
		ClosedAt:          dto.FormatTimePtr(a.ClosedAt),
		Status:            string(a.Status),
		CreatedAt:         dto.FormatTime(a.CreatedAt),
		UpdatedAt:         dto.FormatTime(a.UpdatedAt),
		Version:           a.Version,
	}
}

func callToDTO(c entities.AssemblyCall) dto.CallResponse {
	return dto.CallResponse{
		ID:          c.ID,
		AssemblyID:  c.AssemblyID,
		PublishedAt: dto.FormatTime(c.PublishedAt),
		Channels:    c.Channels,
		Agenda:      c.Agenda,
		BodyMD:      c.BodyMD,
		PublishedBy: c.PublishedBy,
		Status:      string(c.Status),
		CreatedAt:   dto.FormatTime(c.CreatedAt),
		UpdatedAt:   dto.FormatTime(c.UpdatedAt),
		Version:     c.Version,
	}
}

func attendanceToDTO(a entities.AssemblyAttendance) dto.AttendanceResponse {
	return dto.AttendanceResponse{
		ID:                  a.ID,
		AssemblyID:          a.AssemblyID,
		UnitID:              a.UnitID,
		AttendeeUserID:      a.AttendeeUserID,
		RepresentedByUserID: a.RepresentedByUserID,
		CoefficientAtEvent:  a.CoefficientAtEvent,
		ArrivalAt:           dto.FormatTime(a.ArrivalAt),
		DepartureAt:         dto.FormatTimePtr(a.DepartureAt),
		IsRemote:            a.IsRemote,
		HasVotingRight:      a.HasVotingRight,
		Notes:               a.Notes,
		Status:              string(a.Status),
		CreatedAt:           dto.FormatTime(a.CreatedAt),
		UpdatedAt:           dto.FormatTime(a.UpdatedAt),
		Version:             a.Version,
	}
}

func proxyToDTO(p entities.AssemblyProxy) dto.ProxyResponse {
	return dto.ProxyResponse{
		ID:            p.ID,
		AssemblyID:    p.AssemblyID,
		GrantorUserID: p.GrantorUserID,
		ProxyUserID:   p.ProxyUserID,
		UnitID:        p.UnitID,
		DocumentURL:   p.DocumentURL,
		DocumentHash:  p.DocumentHash,
		ValidatedAt:   dto.FormatTimePtr(p.ValidatedAt),
		ValidatedBy:   p.ValidatedBy,
		RevokedAt:     dto.FormatTimePtr(p.RevokedAt),
		Status:        string(p.Status),
		CreatedAt:     dto.FormatTime(p.CreatedAt),
		UpdatedAt:     dto.FormatTime(p.UpdatedAt),
		Version:       p.Version,
	}
}

func motionToDTO(m entities.AssemblyMotion) dto.MotionResponse {
	return dto.MotionResponse{
		ID:           m.ID,
		AssemblyID:   m.AssemblyID,
		Title:        m.Title,
		Description:  m.Description,
		DecisionType: string(m.DecisionType),
		VotingMethod: string(m.VotingMethod),
		Options:      m.Options,
		OpensAt:      dto.FormatTimePtr(m.OpensAt),
		ClosesAt:     dto.FormatTimePtr(m.ClosesAt),
		Results:      m.Results,
		Status:       string(m.Status),
		CreatedAt:    dto.FormatTime(m.CreatedAt),
		UpdatedAt:    dto.FormatTime(m.UpdatedAt),
		Version:      m.Version,
	}
}

func voteToDTO(v entities.Vote) dto.VoteResponse {
	return dto.VoteResponse{
		ID:              v.ID,
		MotionID:        v.MotionID,
		VoterUserID:     v.VoterUserID,
		UnitID:          v.UnitID,
		CoefficientUsed: v.CoefficientUsed,
		Option:          v.Option,
		CastAt:          dto.FormatTime(v.CastAt),
		VoteHash:        v.VoteHash,
		IsProxyVote:     v.IsProxyVote,
		Status:          string(v.Status),
		CreatedAt:       dto.FormatTime(v.CreatedAt),
		UpdatedAt:       dto.FormatTime(v.UpdatedAt),
		Version:         v.Version,
	}
}

func actToDTO(a entities.Act, sigs []dto.SignatureResponse) dto.ActResponse {
	return dto.ActResponse{
		ID:           a.ID,
		AssemblyID:   a.AssemblyID,
		BodyMD:       a.BodyMD,
		PDFURL:       a.PDFURL,
		PDFHash:      a.PDFHash,
		SealedAt:     dto.FormatTimePtr(a.SealedAt),
		ArchiveUntil: dto.FormatDatePtr(a.ArchiveUntil),
		Status:       string(a.Status),
		Signatures:   sigs,
		CreatedAt:    dto.FormatTime(a.CreatedAt),
		UpdatedAt:    dto.FormatTime(a.UpdatedAt),
		Version:      a.Version,
	}
}

func signatureToDTO(s entities.ActSignature) dto.SignatureResponse {
	return dto.SignatureResponse{
		ID:              s.ID,
		ActID:           s.ActID,
		SignerUserID:    s.SignerUserID,
		Role:            string(s.Role),
		SignedAt:        dto.FormatTime(s.SignedAt),
		SignatureMethod: string(s.SignatureMethod),
		EvidenceHash:    s.EvidenceHash,
		Status:          string(s.Status),
		CreatedAt:       dto.FormatTime(s.CreatedAt),
		UpdatedAt:       dto.FormatTime(s.UpdatedAt),
		Version:         s.Version,
	}
}

// actorCtxKey clave de contexto para el actor (user_id).
type actorCtxKey struct{}

// WithActorID es helper para inyectar el actor desde un middleware
// externo (test o capa auth).
func WithActorID(r *http.Request, actorID string) *http.Request {
	if actorID == "" {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), actorCtxKey{}, actorID))
}

func actorIDFromCtx(r *http.Request) string {
	if v, ok := r.Context().Value(actorCtxKey{}).(string); ok {
		return v
	}
	return ""
}
