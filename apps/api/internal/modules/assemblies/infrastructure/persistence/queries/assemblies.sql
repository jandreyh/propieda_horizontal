-- Queries del modulo assemblies (sqlc).
--
-- Convenciones:
--   * Listados excluyen filas con deleted_at IS NOT NULL.
--   * Concurrencia optimista con WHERE version = expected.
--   * Outbox modulo-local: worker bloquea con FOR UPDATE SKIP LOCKED.

-- ----------------------------------------------------------------------------
-- assemblies
-- ----------------------------------------------------------------------------

-- name: CreateAssembly :one
-- Crea una asamblea nueva en estado 'draft'.
INSERT INTO assemblies (
    name, assembly_type, scheduled_at, voting_mode,
    quorum_required_pct, location, notes,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'draft', $8, $8
)
RETURNING id, name, assembly_type, scheduled_at, voting_mode,
          quorum_required_pct, location, notes,
          started_at, closed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetAssemblyByID :one
-- Devuelve una asamblea por id (no soft-deleted).
SELECT id, name, assembly_type, scheduled_at, voting_mode,
       quorum_required_pct, location, notes,
       started_at, closed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assemblies
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListAssemblies :many
-- Lista asambleas activas ordenadas por scheduled_at desc.
SELECT id, name, assembly_type, scheduled_at, voting_mode,
       quorum_required_pct, location, notes,
       started_at, closed_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assemblies
 WHERE deleted_at IS NULL
 ORDER BY scheduled_at DESC;

-- name: UpdateAssemblyStatus :one
-- Actualiza el status de una asamblea con concurrencia optimista.
UPDATE assemblies
   SET status     = sqlc.arg('new_status'),
       started_at = COALESCE(sqlc.narg('new_started_at'), started_at),
       closed_at  = COALESCE(sqlc.narg('new_closed_at'), closed_at),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, name, assembly_type, scheduled_at, voting_mode,
          quorum_required_pct, location, notes,
          started_at, closed_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- assembly_calls
-- ----------------------------------------------------------------------------

-- name: CreateAssemblyCall :one
-- Crea una convocatoria en estado 'published'.
INSERT INTO assembly_calls (
    assembly_id, channels, agenda, body_md,
    published_by, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, 'published', $5, $5
)
RETURNING id, assembly_id, published_at, channels, agenda,
          body_md, published_by, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetAssemblyCallByID :one
-- Devuelve una convocatoria por id.
SELECT id, assembly_id, published_at, channels, agenda,
       body_md, published_by, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_calls
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListAssemblyCallsByAssemblyID :many
-- Lista convocatorias de una asamblea.
SELECT id, assembly_id, published_at, channels, agenda,
       body_md, published_by, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_calls
 WHERE assembly_id = $1
   AND deleted_at IS NULL
 ORDER BY published_at DESC;

-- ----------------------------------------------------------------------------
-- assembly_attendances
-- ----------------------------------------------------------------------------

-- name: CreateAssemblyAttendance :one
-- Registra la asistencia de una unidad.
INSERT INTO assembly_attendances (
    assembly_id, unit_id, attendee_user_id, represented_by_user_id,
    coefficient_at_event, is_remote, has_voting_right, notes,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, 'present', $9, $9
)
RETURNING id, assembly_id, unit_id, attendee_user_id,
          represented_by_user_id, coefficient_at_event,
          arrival_at, departure_at, is_remote, has_voting_right,
          notes, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetAssemblyAttendanceByID :one
-- Devuelve una asistencia por id.
SELECT id, assembly_id, unit_id, attendee_user_id,
       represented_by_user_id, coefficient_at_event,
       arrival_at, departure_at, is_remote, has_voting_right,
       notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_attendances
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListAssemblyAttendancesByAssemblyID :many
-- Lista asistencias de una asamblea.
SELECT id, assembly_id, unit_id, attendee_user_id,
       represented_by_user_id, coefficient_at_event,
       arrival_at, departure_at, is_remote, has_voting_right,
       notes, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_attendances
 WHERE assembly_id = $1
   AND deleted_at IS NULL
 ORDER BY arrival_at ASC;

-- name: SumCoefficientByAssemblyID :one
-- Suma los coeficientes de las unidades presentes con derecho a voto.
SELECT COALESCE(SUM(coefficient_at_event), 0)::NUMERIC(7,6) AS total_coefficient
  FROM assembly_attendances
 WHERE assembly_id = $1
   AND has_voting_right = true
   AND status = 'present'
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- assembly_proxies
-- ----------------------------------------------------------------------------

-- name: CreateAssemblyProxy :one
-- Registra un poder (proxy) en estado 'pending'.
INSERT INTO assembly_proxies (
    assembly_id, grantor_user_id, proxy_user_id, unit_id,
    document_url, document_hash,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, 'pending', $7, $7
)
RETURNING id, assembly_id, grantor_user_id, proxy_user_id, unit_id,
          document_url, document_hash,
          validated_at, validated_by, revoked_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetAssemblyProxyByID :one
-- Devuelve un poder por id.
SELECT id, assembly_id, grantor_user_id, proxy_user_id, unit_id,
       document_url, document_hash,
       validated_at, validated_by, revoked_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_proxies
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListAssemblyProxiesByAssemblyID :many
-- Lista poderes de una asamblea.
SELECT id, assembly_id, grantor_user_id, proxy_user_id, unit_id,
       document_url, document_hash,
       validated_at, validated_by, revoked_at, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_proxies
 WHERE assembly_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at DESC;

-- name: CountProxiesByProxyUser :one
-- Cuenta los poderes activos de un apoderado en una asamblea.
SELECT COUNT(*)::INTEGER AS proxy_count
  FROM assembly_proxies
 WHERE assembly_id = $1
   AND proxy_user_id = $2
   AND status IN ('pending', 'validated')
   AND deleted_at IS NULL;

-- name: ValidateAssemblyProxy :one
-- Valida un poder.
UPDATE assembly_proxies
   SET status       = 'validated',
       validated_at = now(),
       validated_by = sqlc.arg('validated_by'),
       updated_at   = now(),
       updated_by   = sqlc.arg('validated_by'),
       version      = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, assembly_id, grantor_user_id, proxy_user_id, unit_id,
          document_url, document_hash,
          validated_at, validated_by, revoked_at, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- assembly_motions
-- ----------------------------------------------------------------------------

-- name: CreateAssemblyMotion :one
-- Crea una mocion en estado 'draft'.
INSERT INTO assembly_motions (
    assembly_id, title, description, decision_type,
    voting_method, options,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, 'draft', $7, $7
)
RETURNING id, assembly_id, title, description,
          decision_type, voting_method, options,
          opens_at, closes_at, results, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetAssemblyMotionByID :one
-- Devuelve una mocion por id.
SELECT id, assembly_id, title, description,
       decision_type, voting_method, options,
       opens_at, closes_at, results, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_motions
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListAssemblyMotionsByAssemblyID :many
-- Lista mociones de una asamblea.
SELECT id, assembly_id, title, description,
       decision_type, voting_method, options,
       opens_at, closes_at, results, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM assembly_motions
 WHERE assembly_id = $1
   AND deleted_at IS NULL
 ORDER BY created_at ASC;

-- name: UpdateAssemblyMotionStatus :one
-- Actualiza el status de una mocion con concurrencia optimista.
UPDATE assembly_motions
   SET status     = sqlc.arg('new_status'),
       opens_at   = COALESCE(sqlc.narg('new_opens_at'), opens_at),
       closes_at  = COALESCE(sqlc.narg('new_closes_at'), closes_at),
       results    = COALESCE(sqlc.narg('new_results'), results),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, assembly_id, title, description,
          decision_type, voting_method, options,
          opens_at, closes_at, results, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- votes
-- ----------------------------------------------------------------------------

-- name: CreateVote :one
-- Crea un voto.
INSERT INTO votes (
    motion_id, voter_user_id, unit_id, coefficient_used,
    option, cast_at, prev_vote_hash, vote_hash, nonce,
    is_proxy_vote, status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    'cast', $11, $11
)
RETURNING id, motion_id, voter_user_id, unit_id, coefficient_used,
          option, cast_at, prev_vote_hash, vote_hash, nonce,
          is_proxy_vote, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetVoteByID :one
-- Devuelve un voto por id.
SELECT id, motion_id, voter_user_id, unit_id, coefficient_used,
       option, cast_at, prev_vote_hash, vote_hash, nonce,
       is_proxy_vote, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM votes
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: ListVotesByMotionID :many
-- Lista votos de una mocion.
SELECT id, motion_id, voter_user_id, unit_id, coefficient_used,
       option, cast_at, prev_vote_hash, vote_hash, nonce,
       is_proxy_vote, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM votes
 WHERE motion_id = $1
   AND deleted_at IS NULL
 ORDER BY cast_at ASC;

-- name: GetActiveVoteByMotionAndUnit :one
-- Devuelve el voto activo para una unidad en una mocion.
SELECT id, motion_id, voter_user_id, unit_id, coefficient_used,
       option, cast_at, prev_vote_hash, vote_hash, nonce,
       is_proxy_vote, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM votes
 WHERE motion_id = $1
   AND unit_id = $2
   AND status = 'cast'
   AND deleted_at IS NULL
 LIMIT 1;

-- name: GetLastVoteHash :one
-- Devuelve el ultimo vote_hash de la cadena para una mocion.
SELECT vote_hash
  FROM votes
 WHERE motion_id = $1
   AND deleted_at IS NULL
 ORDER BY cast_at DESC
 LIMIT 1;

-- name: VoidVote :exec
-- Cambia el status de un voto a 'changed' con concurrencia optimista.
UPDATE votes
   SET status     = 'changed',
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL;

-- ----------------------------------------------------------------------------
-- vote_evidence
-- ----------------------------------------------------------------------------

-- name: CreateVoteEvidence :one
-- Inserta evidencia de voto (append-only).
INSERT INTO vote_evidence (
    vote_id, motion_id, prev_vote_hash, vote_hash,
    payload_json, client_ip, user_agent, ntp_offset_ms
) VALUES (
    $1, $2, $3, $4, $5, $6::INET, $7, $8
)
RETURNING id, vote_id, motion_id, prev_vote_hash, vote_hash,
          payload_json, client_ip, user_agent, ntp_offset_ms,
          sealed_at, created_at;

-- name: ListVoteEvidenceByMotionID :many
-- Lista evidencia de votos para una mocion ordenada por sealed_at.
SELECT id, vote_id, motion_id, prev_vote_hash, vote_hash,
       payload_json, client_ip, user_agent, ntp_offset_ms,
       sealed_at, created_at
  FROM vote_evidence
 WHERE motion_id = $1
 ORDER BY sealed_at ASC;

-- ----------------------------------------------------------------------------
-- acts
-- ----------------------------------------------------------------------------

-- name: CreateAct :one
-- Crea un acta en estado 'draft'.
INSERT INTO acts (
    assembly_id, body_md, archive_until,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, 'draft', $4, $4
)
RETURNING id, assembly_id, body_md, pdf_url, pdf_hash,
          sealed_at, archive_until, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: GetActByID :one
-- Devuelve un acta por id.
SELECT id, assembly_id, body_md, pdf_url, pdf_hash,
       sealed_at, archive_until, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM acts
 WHERE id = $1
   AND deleted_at IS NULL;

-- name: GetActByAssemblyID :one
-- Devuelve el acta de una asamblea.
SELECT id, assembly_id, body_md, pdf_url, pdf_hash,
       sealed_at, archive_until, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM acts
 WHERE assembly_id = $1
   AND deleted_at IS NULL
 LIMIT 1;

-- name: UpdateActStatus :one
-- Actualiza el status de un acta con concurrencia optimista.
UPDATE acts
   SET status     = sqlc.arg('new_status'),
       sealed_at  = COALESCE(sqlc.narg('new_sealed_at'), sealed_at),
       updated_at = now(),
       updated_by = sqlc.arg('updated_by'),
       version    = version + 1
 WHERE id = sqlc.arg('id')
   AND version = sqlc.arg('expected_version')
   AND deleted_at IS NULL
RETURNING id, assembly_id, body_md, pdf_url, pdf_hash,
          sealed_at, archive_until, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- ----------------------------------------------------------------------------
-- act_signatures
-- ----------------------------------------------------------------------------

-- name: CreateActSignature :one
-- Crea una firma de acta.
INSERT INTO act_signatures (
    act_id, signer_user_id, role, signature_method,
    evidence_hash, client_ip, user_agent,
    status, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6::INET, $7, 'valid', $8, $8
)
RETURNING id, act_id, signer_user_id, role, signed_at,
          signature_method, evidence_hash,
          client_ip, user_agent, status,
          created_at, updated_at, deleted_at,
          created_by, updated_by, deleted_by, version;

-- name: ListActSignaturesByActID :many
-- Lista firmas de un acta.
SELECT id, act_id, signer_user_id, role, signed_at,
       signature_method, evidence_hash,
       client_ip, user_agent, status,
       created_at, updated_at, deleted_at,
       created_by, updated_by, deleted_by, version
  FROM act_signatures
 WHERE act_id = $1
   AND deleted_at IS NULL
 ORDER BY signed_at ASC;

-- ----------------------------------------------------------------------------
-- assemblies_outbox_events
-- ----------------------------------------------------------------------------

-- name: EnqueueAssembliesOutboxEvent :one
-- Inserta un evento en el outbox modulo-local.
INSERT INTO assemblies_outbox_events (
    aggregate_id, event_type, payload, next_attempt_at, attempts
) VALUES (
    $1, $2, $3, now(), 0
)
RETURNING id, aggregate_id, event_type, payload, created_at,
          next_attempt_at, attempts, delivered_at, last_error;

-- name: LockPendingAssembliesOutboxEvents :many
-- Bloquea eventos pendientes con FOR UPDATE SKIP LOCKED.
SELECT id, aggregate_id, event_type, payload, created_at,
       next_attempt_at, attempts, delivered_at, last_error
  FROM assemblies_outbox_events
 WHERE delivered_at IS NULL
   AND next_attempt_at <= now()
 ORDER BY next_attempt_at ASC
 LIMIT $1
 FOR UPDATE SKIP LOCKED;

-- name: MarkAssembliesOutboxEventDelivered :exec
-- Marca un evento como entregado.
UPDATE assemblies_outbox_events
   SET delivered_at = now(),
       attempts     = attempts + 1,
       last_error   = NULL
 WHERE id = $1;

-- name: MarkAssembliesOutboxEventFailed :exec
-- Marca un fallo con backoff.
UPDATE assemblies_outbox_events
   SET attempts        = attempts + 1,
       last_error      = sqlc.arg('last_error'),
       next_attempt_at = sqlc.arg('next_attempt_at')
 WHERE id = sqlc.arg('id');
