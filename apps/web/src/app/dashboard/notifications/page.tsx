import PostMvpPlaceholder from "@/components/PostMvpPlaceholder";

export default function NotificationsPage() {
  return (
    <PostMvpPlaceholder
      title="Notificaciones"
      description="Plantillas, preferencias, consentimientos y outbox por canal"
      faseSpec="fase-15-spec.md"
      endpoints={[
        "GET    /notification-templates",
        "PUT    /notification-preferences",
        "POST   /notification-consents",
        "POST   /push-tokens",
      ]}
    />
  );
}
