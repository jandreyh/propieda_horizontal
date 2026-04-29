import PageHeader from "@/components/PageHeader";
import { Card, CardHeader, StatCard } from "@/components/Card";
import Badge, { statusVariant } from "@/components/Badge";
import EmptyState from "@/components/EmptyState";
import {
  listUnits,
  listPackages,
  listActiveVisits,
  listAnnouncementsFeed,
} from "@/lib/api/modules";
import { ApiError } from "@/lib/api/server";
import { formatDateTime } from "@/lib/format";

async function safe<T>(fn: () => Promise<T>, fallback: T): Promise<T> {
  try {
    return await fn();
  } catch (e) {
    if (e instanceof ApiError) return fallback;
    throw e;
  }
}

export default async function DashboardPage() {
  const [units, packages, visits, feed] = await Promise.all([
    safe(() => listUnits(), { items: [] }),
    safe(() => listPackages(), { items: [], total: 0 }),
    safe(() => listActiveVisits(), { items: [], total: 0 }),
    safe(() => listAnnouncementsFeed(), { items: [], total: 0 }),
  ]);

  const pendingPackages = packages.items.filter((p) => p.status === "RECEIVED");

  return (
    <div>
      <PageHeader
        title="Resumen"
        subtitle="Vista rapida de la operacion del conjunto"
      />

      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Unidades" value={units.items.length} />
        <StatCard
          label="Paquetes en porteria"
          value={pendingPackages.length}
          hint={`Total ${packages.total}`}
        />
        <StatCard
          label="Visitas activas"
          value={visits.total}
          hint="No checkout aun"
        />
        <StatCard
          label="Anuncios vigentes"
          value={feed.total}
          hint="Publicados"
        />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader title="Paquetes pendientes" subtitle="Por entregar al residente" />
          <div className="p-5">
            {pendingPackages.length === 0 ? (
              <EmptyState title="Sin paquetes pendientes" hint="Todo entregado" />
            ) : (
              <ul className="divide-y divide-slate-100">
                {pendingPackages.slice(0, 5).map((p) => (
                  <li key={p.id} className="flex items-center justify-between py-3">
                    <div>
                      <div className="text-sm font-medium text-slate-900">
                        {p.recipient_name}
                      </div>
                      <div className="text-xs text-slate-500">
                        Recibido {formatDateTime(p.received_at)}
                      </div>
                    </div>
                    <Badge variant={statusVariant(p.status)}>{p.status}</Badge>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Card>

        <Card>
          <CardHeader title="Anuncios recientes" />
          <div className="p-5">
            {feed.items.length === 0 ? (
              <EmptyState title="Sin anuncios" />
            ) : (
              <ul className="divide-y divide-slate-100">
                {feed.items.slice(0, 5).map((a) => (
                  <li key={a.id} className="py-3">
                    <div className="flex items-center gap-2">
                      {a.pinned && <Badge variant="warning">Fijado</Badge>}
                      <div className="text-sm font-medium text-slate-900">
                        {a.title}
                      </div>
                    </div>
                    <p className="mt-1 line-clamp-2 text-xs text-slate-500">
                      {a.body}
                    </p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}
