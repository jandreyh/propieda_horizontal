import PageHeader from "@/components/PageHeader";
import { Card } from "@/components/Card";
import Badge, { statusVariant } from "@/components/Badge";
import EmptyState from "@/components/EmptyState";
import { listUnits } from "@/lib/api/modules";
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

export default async function UnitsPage() {
  const { data, error } = await safe(() => listUnits(), { items: [] });
  const { items } = data;

  return (
    <div>
      <PageHeader
        title="Unidades"
        subtitle="Apartamentos, casas, locales y oficinas del conjunto"
      />

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <Card>
        {items.length === 0 ? (
          <EmptyState
            title="Sin unidades registradas"
            hint="Agrega unidades via POST /units"
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <Th>Codigo</Th>
                  <Th>Tipo</Th>
                  <Th>Area m²</Th>
                  <Th>Habitaciones</Th>
                  <Th>Coeficiente</Th>
                  <Th>Estado</Th>
                  <Th>Creado</Th>
                  <Th>ID</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {items.map((u) => (
                  <tr key={u.id} className="hover:bg-slate-50">
                    <Td className="font-medium text-slate-900">{u.code}</Td>
                    <Td>{u.type}</Td>
                    <Td>{u.area_m2 ?? "—"}</Td>
                    <Td>{u.bedrooms ?? "—"}</Td>
                    <Td>{u.coefficient ?? "—"}</Td>
                    <Td>
                      <Badge variant={statusVariant(u.status)}>{u.status}</Badge>
                    </Td>
                    <Td className="text-xs text-slate-500">
                      {formatDateTime(u.created_at)}
                    </Td>
                    <Td className="font-mono text-xs text-slate-400">
                      {shortId(u.id)}
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
