-- Reversa de 001_identity.up.sql.
-- Drop en orden inverso para respetar las FK.

DROP TABLE IF EXISTS user_mfa_recovery_codes;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS users;
