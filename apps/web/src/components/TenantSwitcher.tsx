"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiPost } from "@/lib/api";
import {
  getCurrentTenant,
  getMemberships,
  setCurrentTenant,
  setSession,
  type MembershipDTO,
} from "@/lib/auth";

interface SwitchTenantResponse {
  access_token: string;
  current_tenant: MembershipDTO;
}

export function TenantSwitcher() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [memberships, setMemberships] = useState<MembershipDTO[]>([]);
  const [current, setCurrent] = useState<string | null>(null);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- syncing from localStorage on mount.
    setMemberships(getMemberships());
    setCurrent(getCurrentTenant());
  }, []);

  const currentMembership = memberships.find((m) => m.tenant_slug === current);

  async function pick(slug: string) {
    setOpen(false);
    if (slug === current) return;
    try {
      const res = await apiPost<SwitchTenantResponse>("/auth/switch-tenant", {
        tenant_slug: slug,
      });
      setSession({ access_token: res.access_token });
      setCurrentTenant(slug);
      setCurrent(slug);
      router.refresh();
    } catch {
      /* el handler global de api.ts redirige a login si 401 */
    }
  }

  if (memberships.length === 0) return null;

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-2 rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-left text-sm hover:bg-gray-100"
      >
        <span className="truncate font-medium text-gray-800">
          {currentMembership?.tenant_name ?? "Selecciona conjunto"}
        </span>
        <span className="text-xs text-gray-500">{open ? "▲" : "▼"}</span>
      </button>
      {open && (
        <ul className="absolute left-0 right-0 z-10 mt-1 max-h-72 overflow-y-auto rounded-md border border-gray-200 bg-white shadow-lg">
          {memberships.map((m) => (
            <li key={m.tenant_id}>
              <button
                type="button"
                onClick={() => pick(m.tenant_slug)}
                className={`flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-gray-50 ${
                  m.tenant_slug === current
                    ? "bg-blue-50 text-blue-700"
                    : "text-gray-700"
                }`}
              >
                <span
                  className="h-2 w-2 rounded-full"
                  style={{
                    backgroundColor: m.primary_color ?? "#9CA3AF",
                  }}
                />
                <span className="flex-1 truncate">{m.tenant_name}</span>
                <span className="text-xs text-gray-400">{m.role}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
