import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function AssembliesPage() {
  return (
    <PostMvpPlaceholder
      title="Asambleas"
      description="Convocatorias, asistencia, poderes, mociones, votaciones y actas"
      faseSpec="fase-11-spec.md"
      endpoints={[
        "POST   /assemblies",
        "POST   /assemblies/:id/calls",
        "POST   /assemblies/:id/attendances",
        "POST   /assemblies/:id/proxies",
        "POST   /motions/:id/votes",
        "POST   /assemblies/:id/acts/finalize",
      ]}
    />
  );
}
