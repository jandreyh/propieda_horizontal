-- Rollback del modulo authorization. Drop en orden inverso a las
-- dependencias FK.

DROP INDEX IF EXISTS user_role_assignments_scope_idx;
DROP INDEX IF EXISTS user_role_assignments_role_idx;
DROP INDEX IF EXISTS user_role_assignments_user_active_idx;
DROP TABLE IF EXISTS user_role_assignments;

DROP INDEX IF EXISTS role_permissions_permission_idx;
DROP TABLE IF EXISTS role_permissions;

DROP INDEX IF EXISTS permissions_namespace_idx;
DROP TABLE IF EXISTS permissions;

DROP INDEX IF EXISTS roles_status_idx;
DROP TABLE IF EXISTS roles;
