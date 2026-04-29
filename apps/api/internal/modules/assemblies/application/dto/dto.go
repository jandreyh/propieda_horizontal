// Package dto contiene los Data Transfer Objects del modulo assemblies.
// Aqui (y solo aqui) se aplican tags JSON.
package dto

import "time"

// ---------------------------------------------------------------------------
// Assemblies
// ---------------------------------------------------------------------------

// CreateAssemblyRequest es el body de POST /assemblies.
type CreateAssemblyRequest struct {
	Name              string  `json:"name"`
	AssemblyType      string  `json:"assembly_type"`
	ScheduledAt       string  `json:"scheduled_at"`
	VotingMode        string  `json:"voting_mode"`
	QuorumRequiredPct float64 `json:"quorum_required_pct"`
	Location          *string `json:"location,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

// AssemblyResponse es la representacion HTTP de una Assembly.
type AssemblyResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	AssemblyType      string  `json:"assembly_type"`
	ScheduledAt       string  `json:"scheduled_at"`
	VotingMode        string  `json:"voting_mode"`
	QuorumRequiredPct float64 `json:"quorum_required_pct"`
	Location          *string `json:"location,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	StartedAt         *string `json:"started_at,omitempty"`
	ClosedAt          *string `json:"closed_at,omitempty"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	Version           int32   `json:"version"`
}

// ListAssembliesResponse es el sobre del listado de asambleas.
type ListAssembliesResponse struct {
	Items []AssemblyResponse `json:"items"`
	Total int                `json:"total"`
}

// ---------------------------------------------------------------------------
// Assembly Calls
// ---------------------------------------------------------------------------

// CreateCallRequest es el body de POST /assemblies/{id}/call.
type CreateCallRequest struct {
	Channels []string `json:"channels"`
	Agenda   []string `json:"agenda"`
	BodyMD   *string  `json:"body_md,omitempty"`
}

// CallResponse es la representacion HTTP de una AssemblyCall.
type CallResponse struct {
	ID          string  `json:"id"`
	AssemblyID  string  `json:"assembly_id"`
	PublishedAt string  `json:"published_at"`
	Channels    []byte  `json:"channels"`
	Agenda      []byte  `json:"agenda"`
	BodyMD      *string `json:"body_md,omitempty"`
	PublishedBy *string `json:"published_by,omitempty"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	Version     int32   `json:"version"`
}

// ---------------------------------------------------------------------------
// Attendances
// ---------------------------------------------------------------------------

// CreateAttendanceRequest es el body de POST /assemblies/{id}/attendances.
type CreateAttendanceRequest struct {
	UnitID              string  `json:"unit_id"`
	AttendeeUserID      *string `json:"attendee_user_id,omitempty"`
	RepresentedByUserID *string `json:"represented_by_user_id,omitempty"`
	CoefficientAtEvent  float64 `json:"coefficient_at_event"`
	IsRemote            bool    `json:"is_remote"`
	HasVotingRight      bool    `json:"has_voting_right"`
	Notes               *string `json:"notes,omitempty"`
}

// AttendanceResponse es la representacion HTTP de una AssemblyAttendance.
type AttendanceResponse struct {
	ID                  string  `json:"id"`
	AssemblyID          string  `json:"assembly_id"`
	UnitID              string  `json:"unit_id"`
	AttendeeUserID      *string `json:"attendee_user_id,omitempty"`
	RepresentedByUserID *string `json:"represented_by_user_id,omitempty"`
	CoefficientAtEvent  float64 `json:"coefficient_at_event"`
	ArrivalAt           string  `json:"arrival_at"`
	DepartureAt         *string `json:"departure_at,omitempty"`
	IsRemote            bool    `json:"is_remote"`
	HasVotingRight      bool    `json:"has_voting_right"`
	Notes               *string `json:"notes,omitempty"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	Version             int32   `json:"version"`
}

// ListAttendancesResponse es el sobre del listado de asistencias.
type ListAttendancesResponse struct {
	Items []AttendanceResponse `json:"items"`
	Total int                  `json:"total"`
}

// ---------------------------------------------------------------------------
// Proxies
// ---------------------------------------------------------------------------

// CreateProxyRequest es el body de POST /assemblies/{id}/proxies.
type CreateProxyRequest struct {
	GrantorUserID string  `json:"grantor_user_id"`
	ProxyUserID   string  `json:"proxy_user_id"`
	UnitID        string  `json:"unit_id"`
	DocumentURL   *string `json:"document_url,omitempty"`
	DocumentHash  *string `json:"document_hash,omitempty"`
}

// ProxyResponse es la representacion HTTP de un AssemblyProxy.
type ProxyResponse struct {
	ID            string  `json:"id"`
	AssemblyID    string  `json:"assembly_id"`
	GrantorUserID string  `json:"grantor_user_id"`
	ProxyUserID   string  `json:"proxy_user_id"`
	UnitID        string  `json:"unit_id"`
	DocumentURL   *string `json:"document_url,omitempty"`
	DocumentHash  *string `json:"document_hash,omitempty"`
	ValidatedAt   *string `json:"validated_at,omitempty"`
	ValidatedBy   *string `json:"validated_by,omitempty"`
	RevokedAt     *string `json:"revoked_at,omitempty"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	Version       int32   `json:"version"`
}

// ---------------------------------------------------------------------------
// Motions
// ---------------------------------------------------------------------------

// CreateMotionRequest es el body de POST /assemblies/{id}/motions.
type CreateMotionRequest struct {
	Title        string   `json:"title"`
	Description  *string  `json:"description,omitempty"`
	DecisionType string   `json:"decision_type"`
	VotingMethod string   `json:"voting_method"`
	Options      []string `json:"options,omitempty"`
}

// MotionResponse es la representacion HTTP de una AssemblyMotion.
type MotionResponse struct {
	ID           string  `json:"id"`
	AssemblyID   string  `json:"assembly_id"`
	Title        string  `json:"title"`
	Description  *string `json:"description,omitempty"`
	DecisionType string  `json:"decision_type"`
	VotingMethod string  `json:"voting_method"`
	Options      []byte  `json:"options"`
	OpensAt      *string `json:"opens_at,omitempty"`
	ClosesAt     *string `json:"closes_at,omitempty"`
	Results      []byte  `json:"results,omitempty"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	Version      int32   `json:"version"`
}

// ---------------------------------------------------------------------------
// Votes
// ---------------------------------------------------------------------------

// CastVoteRequest es el body de POST /motions/{id}/votes.
type CastVoteRequest struct {
	VoterUserID     string  `json:"voter_user_id"`
	UnitID          string  `json:"unit_id"`
	CoefficientUsed float64 `json:"coefficient_used"`
	Option          string  `json:"option"`
	IsProxyVote     bool    `json:"is_proxy_vote"`
	ClientIP        *string `json:"client_ip,omitempty"`
	UserAgent       *string `json:"user_agent,omitempty"`
	NTPOffsetMS     *int32  `json:"ntp_offset_ms,omitempty"`
}

// VoteResponse es la representacion HTTP de un Vote.
type VoteResponse struct {
	ID              string  `json:"id"`
	MotionID        string  `json:"motion_id"`
	VoterUserID     string  `json:"voter_user_id"`
	UnitID          string  `json:"unit_id"`
	CoefficientUsed float64 `json:"coefficient_used"`
	Option          string  `json:"option"`
	CastAt          string  `json:"cast_at"`
	VoteHash        string  `json:"vote_hash"`
	IsProxyVote     bool    `json:"is_proxy_vote"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	Version         int32   `json:"version"`
}

// MotionResultsResponse es la respuesta de GET /motions/{id}/results.
type MotionResultsResponse struct {
	Motion MotionResponse `json:"motion"`
	Votes  []VoteResponse `json:"votes"`
	Total  int            `json:"total"`
}

// ---------------------------------------------------------------------------
// Acts
// ---------------------------------------------------------------------------

// CreateActRequest es el body de POST /assemblies/{id}/act.
type CreateActRequest struct {
	BodyMD       string  `json:"body_md"`
	ArchiveUntil *string `json:"archive_until,omitempty"`
}

// ActResponse es la representacion HTTP de un Act.
type ActResponse struct {
	ID           string              `json:"id"`
	AssemblyID   string              `json:"assembly_id"`
	BodyMD       string              `json:"body_md"`
	PDFURL       *string             `json:"pdf_url,omitempty"`
	PDFHash      *string             `json:"pdf_hash,omitempty"`
	SealedAt     *string             `json:"sealed_at,omitempty"`
	ArchiveUntil *string             `json:"archive_until,omitempty"`
	Status       string              `json:"status"`
	Signatures   []SignatureResponse `json:"signatures,omitempty"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at"`
	Version      int32               `json:"version"`
}

// SignActRequest es el body de POST /acts/{id}/sign.
type SignActRequest struct {
	SignerUserID    string  `json:"signer_user_id"`
	Role            string  `json:"role"`
	SignatureMethod string  `json:"signature_method"`
	EvidenceHash    string  `json:"evidence_hash"`
	ClientIP        *string `json:"client_ip,omitempty"`
	UserAgent       *string `json:"user_agent,omitempty"`
}

// SignatureResponse es la representacion HTTP de una ActSignature.
type SignatureResponse struct {
	ID              string `json:"id"`
	ActID           string `json:"act_id"`
	SignerUserID    string `json:"signer_user_id"`
	Role            string `json:"role"`
	SignedAt        string `json:"signed_at"`
	SignatureMethod string `json:"signature_method"`
	EvidenceHash    string `json:"evidence_hash"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	Version         int32  `json:"version"`
}

// ---------------------------------------------------------------------------
// Time formatting helpers
// ---------------------------------------------------------------------------

// FormatTime formatea un time.Time como RFC3339 para JSON.
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatTimePtr formatea un *time.Time como RFC3339 string pointer.
func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// FormatDatePtr formatea un *time.Time como YYYY-MM-DD string pointer.
func FormatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}
