import PageHeader from "@/components/PageHeader";
import { Card, CardHeader } from "@/components/Card";
import Badge from "@/components/Badge";
import EmptyState from "@/components/EmptyState";
import { listRoles, listPermissions } from "@/lib/api/modules";
import { ApiError } from "@/lib/api/server";

async function safe<T>(fn: () => Promise<T>, fallback: T) {
  try {
    return await fn();
  } catch (e) {
    if (e instanceof ApiError) return fallback;
    throw e;
  }
}

export default async function UsersPage() {
  const [roles, perms] = await Promise.all([
    safe(() => listRoles(), { items: [] }),
    safe(() => listPermissions(), { items: [] }),
  ]);

  const grouped: Record<string, typeof perms.items> = {};
  for (const p of perms.items) {
    const key = p.namespace.split(".")[0] || "general";
    grouped[key] = grouped[key] ?? [];
    grouped[key].push(p);
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Usuarios y roles"
        subtitle="Catalogo de roles del conjunto y namespaces de permisos"
      />

      <Card>
        <CardHeader title="Roles" subtitle={`${roles.items.length} roles definidos`} />
        {roles.items.length === 0 ? (
          <div className="p-5">
            <EmptyState title="Sin roles" />
          </div>
        ) : (
          <div className="divide-y divide-slate-100">
            {roles.items.map((r) => (
              <div key={r.id} className="flex items-center justify-between px-5 py-3">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-slate-900">{r.name}</span>
                    {r.is_system && <Badge variant="info">sistema</Badge>}
                  </div>
                  <p className="mt-0.5 text-xs text-slate-500">{r.description}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card>
        <CardHeader
          title="Catalogo de permisos"
          subtitle={`${perms.items.length} namespaces`}
        />
        <div className="space-y-4 p-5">
          {Object.entries(grouped).map(([group, items]) => (
            <div key={group}>
              <div className="mb-1 text-xs font-semibold uppercase tracking-wider text-slate-500">
                {group}
              </div>
              <div className="flex flex-wrap gap-1.5">
                {items.map((p) => (
                  <Badge key={p.id} variant="neutral">
                    {p.namespace}
                  </Badge>
                ))}
              </div>
            </div>
          ))}
          {perms.items.length === 0 && <EmptyState title="Sin permisos cargados" />}
        </div>
      </Card>
    </div>
  );
}
