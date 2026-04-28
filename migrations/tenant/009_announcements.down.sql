-- Tenant DB: rollback del modulo announcements.

DROP INDEX IF EXISTS announcement_acknowledgments_user_idx;
DROP TABLE IF EXISTS announcement_acknowledgments;

DROP INDEX IF EXISTS announcement_audiences_target_idx;
DROP INDEX IF EXISTS announcement_audiences_announcement_idx;
DROP TABLE IF EXISTS announcement_audiences;

DROP INDEX IF EXISTS announcements_pinned_idx;
DROP INDEX IF EXISTS announcements_expires_at_idx;
DROP INDEX IF EXISTS announcements_published_at_idx;
DROP TABLE IF EXISTS announcements;
