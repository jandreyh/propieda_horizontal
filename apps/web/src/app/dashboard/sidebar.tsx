"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";

interface NavItem {
  label: string;
  href: string;
  group: "Operaciones" | "Comunidad" | "Administracion" | "Post-MVP";
}

const navItems: NavItem[] = [
  { label: "Resumen", href: "/dashboard", group: "Operaciones" },
  { label: "Paquetes", href: "/dashboard/packages", group: "Operaciones" },
  { label: "Control de acceso", href: "/dashboard/access-control", group: "Operaciones" },
  { label: "Anuncios", href: "/dashboard/announcements", group: "Comunidad" },
  { label: "Unidades", href: "/dashboard/units", group: "Comunidad" },
  { label: "Vehiculos", href: "/dashboard/people", group: "Comunidad" },
  { label: "Usuarios y roles", href: "/dashboard/users", group: "Administracion" },
  { label: "Parqueaderos", href: "/dashboard/parking", group: "Post-MVP" },
  { label: "Finanzas", href: "/dashboard/finance", group: "Post-MVP" },
  { label: "Reservas", href: "/dashboard/reservations", group: "Post-MVP" },
  { label: "Asambleas", href: "/dashboard/assemblies", group: "Post-MVP" },
  { label: "Incidentes", href: "/dashboard/incidents", group: "Post-MVP" },
  { label: "Sanciones", href: "/dashboard/penalties", group: "Post-MVP" },
  { label: "PQRS", href: "/dashboard/pqrs", group: "Post-MVP" },
  { label: "Notificaciones", href: "/dashboard/notifications", group: "Post-MVP" },
];

const groups = ["Operaciones", "Comunidad", "Administracion", "Post-MVP"] as const;

interface SidebarProps {
  userName: string;
  userEmail: string;
}

export default function Sidebar({ userName, userEmail }: SidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const [loggingOut, setLoggingOut] = useState(false);

  async function handleLogout() {
    setLoggingOut(true);
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/login");
    router.refresh();
  }

  return (
    <aside className="flex w-72 shrink-0 flex-col border-r border-slate-200 bg-white">
      <div className="flex items-center gap-3 border-b border-slate-200 px-5 py-4">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-600 text-white">
          <span className="text-sm font-bold">PH</span>
        </div>
        <div className="leading-tight">
          <div className="text-sm font-semibold text-slate-900">
            Propiedad Horizontal
          </div>
          <div className="text-xs text-slate-500">Conjunto demo</div>
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {groups.map((group) => (
          <div key={group} className="mb-5">
            <div className="mb-1 px-3 text-xs font-semibold uppercase tracking-wider text-slate-400">
              {group}
            </div>
            <ul className="space-y-0.5">
              {navItems
                .filter((i) => i.group === group)
                .map((item) => {
                  const isActive =
                    item.href === "/dashboard"
                      ? pathname === "/dashboard"
                      : pathname.startsWith(item.href);
                  return (
                    <li key={item.href}>
                      <Link
                        href={item.href}
                        className={`block rounded-md px-3 py-2 text-sm transition-colors ${
                          isActive
                            ? "bg-indigo-50 font-medium text-indigo-700"
                            : "text-slate-700 hover:bg-slate-100"
                        }`}
                      >
                        {item.label}
                      </Link>
                    </li>
                  );
                })}
          </ul>
          </div>
        ))}
      </nav>

      <div className="border-t border-slate-200 px-4 py-3">
        <div className="mb-2">
          <div className="truncate text-sm font-medium text-slate-900">
            {userName}
          </div>
          <div className="truncate text-xs text-slate-500">{userEmail}</div>
        </div>
        <button
          onClick={handleLogout}
          disabled={loggingOut}
          className="w-full rounded-md border border-slate-200 px-3 py-1.5 text-xs text-slate-600 hover:bg-slate-50 disabled:opacity-50"
        >
          {loggingOut ? "Saliendo..." : "Cerrar sesion"}
        </button>
      </div>
    </aside>
  );
}
