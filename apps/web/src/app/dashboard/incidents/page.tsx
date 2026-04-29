import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function IncidentsPage() {
  return (
    <PostMvpPlaceholder
      title="Incidentes"
      description="Reportes con adjuntos, asignaciones y SLA por severidad"
      faseSpec="fase-12-spec.md"
      endpoints={[
        "POST   /incidents",
        "POST   /incidents/:id/attachments",
        "POST   /incidents/:id/assignments",
        "POST   /incidents/:id/transition",
      ]}
    />
  );
}
