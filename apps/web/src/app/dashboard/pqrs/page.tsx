import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function PqrsPage() {
  return (
    <PostMvpPlaceholder
      title="PQRS"
      description="Peticiones, quejas, reclamos y sugerencias con SLAs legales"
      faseSpec="fase-14-spec.md"
      endpoints={[
        "GET    /pqrs-categories",
        "POST   /pqrs-tickets",
        "POST   /pqrs-tickets/:id/responses",
        "POST   /pqrs-tickets/:id/transition",
      ]}
    />
  );
}
