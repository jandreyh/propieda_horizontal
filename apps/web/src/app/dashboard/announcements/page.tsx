import PageHeader from "@/components/PageHeader";
import { Card } from "@/components/Card";
import Badge, { statusVariant } from "@/components/Badge";
import EmptyState from "@/components/EmptyState";
import { listAnnouncementsFeed } from "@/lib/api/modules";
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

export default async function AnnouncementsPage() {
  const { data, error } = await safe(() => listAnnouncementsFeed(), { items: [], total: 0 });
  const { items, total } = data;

  return (
    <div>
      <PageHeader
        title="Anuncios"
        subtitle={`Tablero de comunicados · ${total} publicados`}
      />

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState title="Sin anuncios" hint="Solo admin con permiso announcement.publish" />
      ) : (
        <div className="space-y-3">
          {items.map((a) => (
            <Card key={a.id} className="px-5 py-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    {a.pinned && <Badge variant="warning">Fijado</Badge>}
                    <Badge variant={statusVariant(a.status)}>{a.status}</Badge>
                    <span className="text-xs text-slate-400">
                      {formatDateTime(a.published_at)}
                    </span>
                  </div>
                  <h3 className="mt-2 text-sm font-semibold text-slate-900">
                    {a.title}
                  </h3>
                  <p className="mt-1 whitespace-pre-line text-sm text-slate-700">
                    {a.body}
                  </p>
                  {a.audiences.length > 0 && (
                    <div className="mt-3 flex flex-wrap gap-1">
                      {a.audiences.map((aud, i) => (
                        <Badge key={i} variant="info">
                          {aud.target_type}
                          {aud.target_id ? `:${aud.target_id.slice(0, 6)}` : ""}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
