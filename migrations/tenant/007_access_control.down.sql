-- Reversa de 007_access_control.up.sql.
-- Drop en orden inverso: primero entries (FK a pre_registrations), luego
-- pre_registrations, finalmente blacklisted_persons.

DROP TABLE IF EXISTS visitor_entries;
DROP TABLE IF EXISTS visitor_pre_registrations;
DROP TABLE IF EXISTS blacklisted_persons;
