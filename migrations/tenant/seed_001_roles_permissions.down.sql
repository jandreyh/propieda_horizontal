-- Rollback de la seed inicial del modulo authorization.
-- Borra mappings rol->permiso, luego permisos seed, luego roles seed.

DELETE FROM role_permissions
WHERE role_id IN (
    SELECT id FROM roles
    WHERE name IN (
        'tenant_admin',
        'accountant',
        'guard',
        'owner',
        'tenant_resident',
        'authorized_resident',
        'board_member',
        'auditor_or_revisor',
        'support_l1'
    )
);

DELETE FROM permissions
WHERE namespace IN (
    'identity.read',
    'identity.write',
    'role.create',
    'role.read',
    'role.update',
    'role.delete',
    'permission.read',
    'user.assign_role',
    'user.unassign_role',
    'settings.read',
    'settings.write',
    'branding.read',
    'branding.write',
    'unit.read',
    'unit.write',
    'person.read',
    'person.write',
    'visit.create',
    'visit.read',
    'visit.approve',
    'package.create',
    'package.read',
    'package.deliver',
    'announcement.read',
    'announcement.publish',
    'audit.read'
);

DELETE FROM roles
WHERE is_system = true
  AND name IN (
      'tenant_admin',
      'accountant',
      'guard',
      'owner',
      'tenant_resident',
      'authorized_resident',
      'board_member',
      'auditor_or_revisor',
      'support_l1'
  );
