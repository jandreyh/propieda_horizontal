import PageHeader from "@/components/PageHeader";
import { Card } from "@/components/Card";
import Badge, { statusVariant } from "@/components/Badge";
import EmptyState from "@/components/EmptyState";
import { listVehicles } from "@/lib/api/modules";
import { ApiError } from "@/lib/api/server";
import { formatDateTime } from "@/lib/format";

async function safe<T>(fn: () => Promise<T>, fallback: T): Promise<{ data: T; error: string | null }> {
  try {
    return { data: await fn(), error: null };
  } catch (e) {
    if (e instanceof ApiError) return { data: fallback, error: e.problem.detail || e.problem.title };
    throw e;
  }
}

export default async function VehiclesPage() {
  const { data, error } = await safe(() => listVehicles(), { items: [], total: 0 });
  const { items, total } = data;

  return (
    <div>
      <PageHeader
        title="Vehiculos"
        subtitle={`Vehiculos registrados en el conjunto · Total: ${total}`}
      />

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <Card>
        {items.length === 0 ? (
          <EmptyState title="Sin vehiculos" hint="Registralos via POST /vehicles" />
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <Th>Placa</Th>
                  <Th>Tipo</Th>
                  <Th>Marca</Th>
                  <Th>Modelo</Th>
                  <Th>Color</Th>
                  <Th>Año</Th>
                  <Th>Estado</Th>
                  <Th>Creado</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {items.map((v) => (
                  <tr key={v.id} className="hover:bg-slate-50">
                    <Td className="font-mono font-medium text-slate-900">{v.plate}</Td>
                    <Td>{v.type}</Td>
                    <Td>{v.brand ?? "—"}</Td>
                    <Td>{v.model ?? "—"}</Td>
                    <Td>{v.color ?? "—"}</Td>
                    <Td>{v.year ?? "—"}</Td>
                    <Td>
                      <Badge variant={statusVariant(v.status)}>{v.status}</Badge>
                    </Td>
                    <Td className="text-xs text-slate-500">{formatDateTime(v.created_at)}</Td>
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
