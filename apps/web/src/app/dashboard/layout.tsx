"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

import { TenantSwitcher } from "@/components/TenantSwitcher";
import {
  clearSession,
  getCurrentTenant,
  isAuthenticated,
} from "@/lib/auth";

interface NavItem {
  label: string;
  href: string;
  icon: string;
}

const navItems: NavItem[] = [
  { label: "Inicio", href: "/dashboard", icon: "[Inicio]" },
  { label: "Finanzas", href: "/dashboard/finance", icon: "[Finanzas]" },
  { label: "Reservas", href: "/dashboard/reservations", icon: "[Reservas]" },
  { label: "Asambleas", href: "/dashboard/assemblies", icon: "[Asambleas]" },
  { label: "Incidentes", href: "/dashboard/incidents", icon: "[Incidentes]" },
  { label: "Sanciones", href: "/dashboard/penalties", icon: "[Sanciones]" },
  { label: "PQRS", href: "/dashboard/pqrs", icon: "[PQRS]" },
  {
    label: "Notificaciones",
    href: "/dashboard/notifications",
    icon: "[Notif.]",
  },
  { label: "Parqueaderos", href: "/dashboard/parking", icon: "[Parking]" },
  { label: "Paquetes", href: "/dashboard/packages", icon: "[Paquetes]" },
  {
    label: "Control de acceso",
    href: "/dashboard/access-control",
    icon: "[Acceso]",
  },
  { label: "Anuncios", href: "/dashboard/announcements", icon: "[Anuncios]" },
  { label: "Unidades", href: "/dashboard/units", icon: "[Unidades]" },
  { label: "Usuarios", href: "/dashboard/users", icon: "[Usuarios]" },
];

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();

  // Guard cliente: si no hay token o no hay current_tenant, redirigir.
  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    if (!getCurrentTenant()) {
      router.replace("/select-tenant");
    }
  }, [router]);

  function handleLogout(e: React.MouseEvent) {
    e.preventDefault();
    clearSession();
    router.push("/login");
  }

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <aside className="flex w-64 shrink-0 flex-col border-r border-gray-200 bg-white">
        <div className="border-b border-gray-200 px-6 py-4">
          <Link href="/dashboard" className="text-lg font-bold text-gray-900">
            Propiedad Horizontal
          </Link>
        </div>

        <div className="border-b border-gray-200 px-3 py-3">
          <TenantSwitcher />
        </div>

        <nav className="flex-1 overflow-y-auto px-3 py-4">
          <ul className="space-y-1">
            {navItems.map((item) => {
              const isActive =
                item.href === "/dashboard"
                  ? pathname === "/dashboard"
                  : pathname.startsWith(item.href);

              return (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                      isActive
                        ? "bg-blue-50 text-blue-700"
                        : "text-gray-700 hover:bg-gray-100 hover:text-gray-900"
                    }`}
                  >
                    <span className="text-xs font-mono text-gray-400 w-20 shrink-0">
                      {item.icon}
                    </span>
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        <div className="border-t border-gray-200 px-6 py-3">
          <a
            href="/login"
            onClick={handleLogout}
            className="block text-sm text-gray-500 hover:text-gray-700"
          >
            Cerrar sesion
          </a>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto bg-gray-50 p-6">{children}</main>
    </div>
  );
}
