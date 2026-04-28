-- Seed inicial del modulo authorization.
--
-- Inserta:
--   * 9 roles semilla del producto (is_system = true). El rol
--     `platform_superadmin` NO se materializa aqui: vive en el Control
--     Plane (ADR 0002, ADR 0003).
--   * Catalogo MVP de permisos (~26 namespaces).
--   * Mapeo default rol -> permisos siguiendo ADR 0003 ("Mapeo inicial
--     rol -> permisos") con extensiones razonables para los namespaces
--     MVP solicitados (identity.*, role.*, person.*, etc.).
--
-- Idempotencia: ON CONFLICT DO NOTHING en cada insert para que la seed
-- pueda rerunearse sin romper.

INSERT INTO roles (name, description, is_system) VALUES
    ('tenant_admin',         'Administrador del conjunto',                 true),
    ('accountant',           'Contador del conjunto',                      true),
    ('guard',                'Vigilante / porteria',                       true),
    ('owner',                'Propietario de unidad',                      true),
    ('tenant_resident',      'Residente arrendatario',                     true),
    ('authorized_resident',  'Residente autorizado por propietario',       true),
    ('board_member',         'Miembro del consejo de administracion',      true),
    ('auditor_or_revisor',   'Revisor fiscal / auditor',                   true),
    ('support_l1',           'Soporte nivel 1 (lectura amplia)',           true)
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (namespace, description) VALUES
    ('identity.read',          'Leer identidades de usuarios'),
    ('identity.write',         'Crear/actualizar identidades'),
    ('role.create',            'Crear roles custom'),
    ('role.read',              'Leer catalogo de roles'),
    ('role.update',            'Modificar roles custom'),
    ('role.delete',            'Eliminar roles custom'),
    ('permission.read',        'Leer catalogo de permisos'),
    ('user.assign_role',       'Asignar rol a usuario'),
    ('user.unassign_role',     'Revocar rol de usuario'),
    ('settings.read',          'Leer configuracion del tenant'),
    ('settings.write',         'Modificar configuracion del tenant'),
    ('branding.read',          'Leer branding del tenant'),
    ('branding.write',         'Modificar branding del tenant'),
    ('unit.read',              'Leer unidades / torres'),
    ('unit.write',             'Modificar unidades / torres'),
    ('person.read',            'Leer personas / residentes'),
    ('person.write',           'Modificar personas / residentes'),
    ('visit.create',           'Registrar visita'),
    ('visit.read',             'Leer visitas'),
    ('visit.approve',          'Aprobar / rechazar visitas'),
    ('package.create',         'Registrar paquete recibido'),
    ('package.read',           'Leer paquetes'),
    ('package.deliver',        'Entregar paquete a residente'),
    ('announcement.read',      'Leer comunicados'),
    ('announcement.publish',   'Publicar comunicados'),
    ('audit.read',             'Leer audit log')
ON CONFLICT (namespace) DO NOTHING;

-- Mapeo default rol -> permisos.
--
-- tenant_admin: todo excepto audit.read (que se asigna al revisor).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'tenant_admin'
  AND p.namespace <> 'audit.read'
ON CONFLICT DO NOTHING;

-- accountant: contabilidad + lectura de unidades/personas + settings.read.
-- (Los namespaces accounting.* aun no existen en MVP; se mapea lo
-- disponible.)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'accountant'
  AND p.namespace IN (
      'unit.read',
      'person.read',
      'settings.read'
  )
ON CONFLICT DO NOTHING;

-- guard: paquetes y visitas en porteria.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'guard'
  AND p.namespace IN (
      'package.create',
      'package.read',
      'package.deliver',
      'visit.create',
      'visit.read',
      'visit.approve',
      'person.read',
      'unit.read'
  )
ON CONFLICT DO NOTHING;

-- owner: lectura de su unidad, paquetes propios, registrar visitas y leer
-- comunicados.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'owner'
  AND p.namespace IN (
      'unit.read',
      'person.read',
      'package.read',
      'visit.create',
      'visit.read',
      'announcement.read'
  )
ON CONFLICT DO NOTHING;

-- tenant_resident: como owner pero sin lectura amplia de unit.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'tenant_resident'
  AND p.namespace IN (
      'package.read',
      'visit.create',
      'visit.read',
      'announcement.read'
  )
ON CONFLICT DO NOTHING;

-- authorized_resident: scope tipico unit; minimo de visitas y paquetes.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'authorized_resident'
  AND p.namespace IN (
      'package.read',
      'visit.create',
      'announcement.read'
  )
ON CONFLICT DO NOTHING;

-- board_member: lectura amplia + publicar comunicados.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'board_member'
  AND p.namespace IN (
      'identity.read',
      'role.read',
      'permission.read',
      'unit.read',
      'person.read',
      'package.read',
      'visit.read',
      'announcement.read',
      'announcement.publish',
      'settings.read',
      'branding.read'
  )
ON CONFLICT DO NOTHING;

-- auditor_or_revisor: audit.read + lecturas contables/settings.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'auditor_or_revisor'
  AND p.namespace IN (
      'audit.read',
      'settings.read',
      'unit.read',
      'person.read'
  )
ON CONFLICT DO NOTHING;

-- support_l1: lecturas amplias para soporte (sin escrituras).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'support_l1'
  AND p.namespace IN (
      'identity.read',
      'role.read',
      'permission.read',
      'unit.read',
      'person.read',
      'package.read',
      'visit.read',
      'announcement.read',
      'settings.read',
      'branding.read'
  )
ON CONFLICT DO NOTHING;
