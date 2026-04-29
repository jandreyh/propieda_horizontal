import PageHeader from "./PageHeader";

interface PostMvpPlaceholderProps {
  title: string;
  description: string;
  endpoints: string[];
  faseSpec?: string;
}

export default function PostMvpPlaceholder({
  title,
  description,
  endpoints,
  faseSpec,
}: PostMvpPlaceholderProps) {
  return (
    <div>
      <PageHeader title={title} subtitle={description} />

      <div className="rounded-xl border border-amber-200 bg-amber-50 p-5 text-sm text-amber-900">
        <div className="mb-2 font-semibold">Modulo backend implementado</div>
        <p className="mb-3 text-amber-800">
          Las migraciones, endpoints y tests del modulo estan en main. La UI
          completa de este modulo es parte del siguiente bloque de trabajo (no
          incluido en este MVP demo).
        </p>
        {faseSpec && (
          <p className="text-xs text-amber-700">
            Spec frozen:{" "}
            <code className="rounded bg-amber-100 px-1 py-0.5">
              docs/specs/{faseSpec}
            </code>
          </p>
        )}
        <div className="mt-3">
          <div className="text-xs font-medium uppercase tracking-wider text-amber-700">
            Endpoints disponibles
          </div>
          <ul className="mt-1 space-y-0.5 font-mono text-xs text-amber-900">
            {endpoints.map((e) => (
              <li key={e}>{e}</li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}
