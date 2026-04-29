import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function FinancePage() {
  return (
    <PostMvpPlaceholder
      title="Finanzas"
      description="Plan de cuentas, cargos, pagos, asientos contables y cierres"
      faseSpec="fase-9-spec.md"
      endpoints={[
        "GET    /chart-of-accounts",
        "POST   /charges",
        "POST   /payments",
        "POST   /payments/:id/reverse",
        "POST   /period-closures",
        "POST   /webhooks/payment-gateway (idempotente)",
      ]}
    />
  );
}
