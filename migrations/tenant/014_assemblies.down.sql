-- Tenant DB: rollback del modulo assemblies (Fase 11).

DROP INDEX IF EXISTS assemblies_outbox_events_pending_idx;
DROP TABLE IF EXISTS assemblies_outbox_events;

DROP INDEX IF EXISTS act_signatures_act_idx;
DROP INDEX IF EXISTS act_signatures_act_role_unique;
DROP TABLE IF EXISTS act_signatures;

DROP TRIGGER IF EXISTS tg_acts_immutable_when_signed ON acts;
DROP FUNCTION IF EXISTS fn_acts_immutable_when_signed();
DROP INDEX IF EXISTS acts_assembly_idx;
DROP TABLE IF EXISTS acts;

DROP INDEX IF EXISTS vote_evidence_prev_hash_idx;
DROP INDEX IF EXISTS vote_evidence_motion_chain_idx;
DROP INDEX IF EXISTS vote_evidence_hash_unique;
DROP INDEX IF EXISTS vote_evidence_vote_unique;
DROP TABLE IF EXISTS vote_evidence;

DROP INDEX IF EXISTS votes_hash_unique;
DROP INDEX IF EXISTS votes_motion_idx;
DROP INDEX IF EXISTS votes_motion_unit_active_unique;
DROP TABLE IF EXISTS votes;

DROP INDEX IF EXISTS assembly_motions_open_idx;
DROP INDEX IF EXISTS assembly_motions_assembly_idx;
DROP TABLE IF EXISTS assembly_motions;

DROP INDEX IF EXISTS assembly_proxies_proxy_idx;
DROP INDEX IF EXISTS assembly_proxies_unit_unique;
DROP TABLE IF EXISTS assembly_proxies;

DROP INDEX IF EXISTS assembly_attendances_assembly_idx;
DROP INDEX IF EXISTS assembly_attendances_assembly_unit_unique;
DROP TABLE IF EXISTS assembly_attendances;

DROP INDEX IF EXISTS assembly_calls_assembly_idx;
DROP TABLE IF EXISTS assembly_calls;

DROP INDEX IF EXISTS assemblies_status_idx;
DROP INDEX IF EXISTS assemblies_scheduled_idx;
DROP TABLE IF EXISTS assemblies;
