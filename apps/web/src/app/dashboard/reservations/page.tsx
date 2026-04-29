import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function ReservationsPage() {
  return (
    <PostMvpPlaceholder
      title="Reservas de zonas comunes"
      description="Salon, BBQ, piscina, gimnasio: reglas, blackouts y reservas"
      faseSpec="fase-10-spec.md"
      endpoints={[
        "GET    /common-areas",
        "POST   /common-areas/:id/blackouts",
        "POST   /reservations",
        "POST   /reservations/:id/cancel",
      ]}
    />
  );
}
