import PageHeader from "@/components/PageHeader";
import { Card, CardHeader } from "@/components/Card";
import Badge, { statusVariant } from "@/components/Badge";
import EmptyState from "@/components/EmptyState";
import { listActiveVisits, listBlacklist } from "@/lib/api/modules";
import { ApiError } from "@/lib/api/server";
import { formatDateTime, shortId } from "@/lib/format";

async function safe<T>(fn: () => Promise<T>, fallback: T) {
  try {
    return await fn();
  } catch (e) {
    if (e instanceof ApiError) return fallback;
    throw e;
  }
}

export default async function AccessControlPage() {
  const [visits, blacklist] = await Promise.all([
    safe(() => listActiveVisits(), { items: [], total: 0 }),
    safe(() => listBlacklist(), { items: [], total: 0 }),
  ]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Control de acceso"
        subtitle="Visitas activas y blacklist del conjunto"
      />

      <Card>
        <CardHeader
          title="Visitas activas"
          subtitle={`${visits.total} ingresos sin checkout`}
        />
        {visits.items.length === 0 ? (
          <div className="p-5">
            <EmptyState title="Sin visitas activas" hint="Toda visita registrada salio" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <Th>Visitante</Th>
                  <Th>Documento</Th>
                  <Th>Unidad</Th>
                  <Th>Origen</Th>
                  <Th>Ingreso</Th>
                  <Th>Estado</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {visits.items.map((v) => (
                  <tr key={v.id} className="hover:bg-slate-50">
                    <Td className="font-medium text-slate-900">{v.visitor_full_name}</Td>
                    <Td className="font-mono text-xs">
                      {v.visitor_document_type ?? ""} {v.visitor_document_number}
                    </Td>
                    <Td className="font-mono text-xs text-slate-500">
                      {v.unit_id ? shortId(v.unit_id) : "—"}
                    </Td>
                    <Td>{v.source}</Td>
                    <Td className="text-xs text-slate-600">
                      {formatDateTime(v.entry_time)}
                    </Td>
                    <Td>
                      <Badge variant={statusVariant(v.status)}>{v.status}</Badge>
                    </Td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card>
        <CardHeader
          title="Blacklist"
          subtitle={`${blacklist.total} personas bloqueadas`}
        />
        {blacklist.items.length === 0 ? (
          <div className="p-5">
            <EmptyState title="Blacklist vacia" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <Th>Documento</Th>
                  <Th>Nombre</Th>
                  <Th>Razon</Th>
                  <Th>Vence</Th>
                  <Th>Estado</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {blacklist.items.map((b) => (
                  <tr key={b.id} className="hover:bg-slate-50">
                    <Td className="font-mono text-xs">
                      {b.document_type} {b.document_number}
                    </Td>
                    <Td>{b.full_name ?? "—"}</Td>
                    <Td className="max-w-md truncate">{b.reason}</Td>
                    <Td className="text-xs text-slate-600">
                      {formatDateTime(b.expires_at)}
                    </Td>
                    <Td>
                      <Badge variant={statusVariant(b.status)}>{b.status}</Badge>
                    </Td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

function Th({ children }: { children: React.ReactNode }) {
  return (
    <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-500">
      {children}
    </th>
  );
}
function Td({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return (
    <td className={`whitespace-nowrap px-4 py-3 text-sm text-slate-700 ${className}`}>
      {children}
    </td>
  );
}
