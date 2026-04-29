import { redirect } from "next/navigation";
import Sidebar from "./sidebar";
import { me } from "@/lib/api/modules";
import { ApiError } from "@/lib/api/server";

export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  let user;
  try {
    user = await me();
  } catch (e) {
    if (e instanceof ApiError && (e.problem.status === 401 || e.problem.status === 403)) {
      redirect("/login");
    }
    throw e;
  }

  const fullName = `${user.names} ${user.last_names}`.trim();
  const subtitle = user.email ?? `${user.document_type}:${user.document_number}`;

  return (
    <div className="flex min-h-screen bg-slate-50">
      <Sidebar userName={fullName} userEmail={subtitle} />
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-7xl px-8 py-8">{children}</div>
      </main>
    </div>
  );
}
