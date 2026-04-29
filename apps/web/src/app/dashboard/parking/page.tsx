import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function ParkingPage() {
  return (
    <PostMvpPlaceholder
      title="Parqueaderos"
      description="Asignaciones permanentes, reservas de visitantes y sorteo determinista"
      faseSpec="fase-8-spec.md"
      endpoints={[
        "GET    /parking-spaces",
        "POST   /parking-spaces",
        "POST   /parking-spaces/:id/assign",
        "POST   /parking-visitor-reservations",
        "POST   /parking-lotteries/run",
        "GET    /parking-lotteries/:id/results",
      ]}
    />
  );
}
