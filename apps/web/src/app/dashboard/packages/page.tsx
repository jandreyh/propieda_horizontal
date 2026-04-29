import PageHeader from "@/components/PageHeader";
import { Card } from "@/components/Card";
import Badge, { statusVariant } from "@/components/Badge";
import EmptyState from "@/components/EmptyState";
import { listPackages } from "@/lib/api/modules";
import { ApiError } from "@/lib/api/server";
import { formatDateTime, shortId } from "@/lib/format";

async function safe<T>(fn: () => Promise<T>, fallback: T): Promise<{ data: T; error: string | null }> {
  try {
    return { data: await fn(), error: null };
  } catch (e) {
    if (e instanceof ApiError) return { data: fallback, error: e.problem.detail || e.problem.title };
    throw e;
  }
}

export default async function PackagesPage() {
  const { data, error } = await safe(() => listPackages(), { items: [], total: 0 });
  const { items, total } = data;

  return (
    <div>
      <PageHeader
        title="Paquetes"
        subtitle={`Correspondencia y paqueteria · Total: ${total}`}
      />

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <Card>
        {items.length === 0 ? (
          <EmptyState
            title="Sin paquetes registrados"
            hint="El guarda los registra desde porteria"
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <Th>Destinatario</Th>
                  <Th>Unidad</Th>
                  <Th>Operador</Th>
                  <Th>Tracking</Th>
                  <Th>Recibido</Th>
                  <Th>Entregado</Th>
                  <Th>Estado</Th>
                  <Th>v</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {items.map((p) => (
                  <tr key={p.id} className="hover:bg-slate-50">
                    <Td className="font-medium text-slate-900">
                      {p.recipient_name}
                    </Td>
                    <Td className="font-mono text-xs text-slate-500">
                      {shortId(p.unit_id)}
                    </Td>
                    <Td>{p.carrier ?? "—"}</Td>
                    <Td className="font-mono text-xs">
                      {p.tracking_number ?? "—"}
                    </Td>
                    <Td className="text-xs text-slate-600">
                      {formatDateTime(p.received_at)}
                    </Td>
                    <Td className="text-xs text-slate-600">
                      {formatDateTime(p.delivered_at)}
                    </Td>
                    <Td>
                      <Badge variant={statusVariant(p.status)}>{p.status}</Badge>
                    </Td>
                    <Td className="text-xs text-slate-400">{p.version}</Td>
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
