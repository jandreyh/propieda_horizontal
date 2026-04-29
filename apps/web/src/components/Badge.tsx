type Variant = "neutral" | "success" | "warning" | "danger" | "info";

const styles: Record<Variant, string> = {
  neutral: "bg-slate-100 text-slate-700",
  success: "bg-emerald-50 text-emerald-700",
  warning: "bg-amber-50 text-amber-700",
  danger: "bg-red-50 text-red-700",
  info: "bg-indigo-50 text-indigo-700",
};

export default function Badge({
  children,
  variant = "neutral",
}: {
  children: React.ReactNode;
  variant?: Variant;
}) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${styles[variant]}`}
    >
      {children}
    </span>
  );
}

export function statusVariant(status: string): Variant {
  const s = status.toUpperCase();
  if (
    ["DELIVERED", "ACTIVE", "PAID", "CLOSED", "RESOLVED", "ACKNOWLEDGED"].includes(
      s,
    )
  )
    return "success";
  if (["RECEIVED", "PENDING", "OPEN", "IN_PROGRESS"].includes(s)) return "warning";
  if (["RETURNED", "ARCHIVED", "CANCELLED", "BLOCKED"].includes(s)) return "danger";
  if (["DRAFT", "SCHEDULED"].includes(s)) return "info";
  return "neutral";
}
