import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function PenaltiesPage() {
  return (
    <PostMvpPlaceholder
      title="Sanciones"
      description="Catalogo, imposicion, apelaciones y vinculacion con cartera"
      faseSpec="fase-13-spec.md"
      endpoints={[
        "GET    /penalty-catalog",
        "POST   /penalties",
        "POST   /penalties/:id/appeal",
        "POST   /penalties/:id/transition",
      ]}
    />
  );
}
