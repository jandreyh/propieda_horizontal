"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiPost } from "@/lib/api";
import {
  getMemberships,
  isAuthenticated,
  setCurrentTenant,
  setSession,
  type MembershipDTO,
} from "@/lib/auth";

interface SwitchTenantResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  current_tenant: MembershipDTO;
}

export default function SelectTenantPage() {
  const router = useRouter();
  const [memberships, setMemberships] = useState<MembershipDTO[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busySlug, setBusySlug] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }
    // eslint-disable-next-line react-hooks/set-state-in-effect -- syncing from localStorage on mount.
    setMemberships(getMemberships());
  }, [router]);

  async function selectTenant(slug: string) {
    setBusySlug(slug);
    setError(null);
    try {
      const res = await apiPost<SwitchTenantResponse>("/auth/switch-tenant", {
        tenant_slug: slug,
      });
      setSession({ access_token: res.access_token });
      setCurrentTenant(slug);
      router.push("/dashboard");
    } catch (err: unknown) {
      const apiErr = err as { detail?: string; title?: string };
      setError(apiErr?.detail || apiErr?.title || "Error al seleccionar conjunto");
    } finally {
      setBusySlug(null);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-100 px-4 py-12">
      <div className="w-full max-w-2xl">
        <h1 className="mb-2 text-center text-2xl font-bold text-gray-900">
          Selecciona un conjunto
        </h1>
        <p className="mb-6 text-center text-sm text-gray-500">
          Tienes acceso a {memberships.length}{" "}
          {memberships.length === 1 ? "conjunto" : "conjuntos"}.
        </p>

        {error && (
          <div className="mb-4 rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {memberships.map((m) => (
            <li key={m.tenant_id}>
              <button
                type="button"
                onClick={() => selectTenant(m.tenant_slug)}
                disabled={busySlug !== null}
                className="flex w-full items-center gap-3 rounded-lg bg-white p-4 text-left shadow-sm hover:bg-gray-50 disabled:opacity-50"
                style={{
                  borderLeft: m.primary_color
                    ? `4px solid ${m.primary_color}`
                    : undefined,
                }}
              >
                {m.logo_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={m.logo_url}
                    alt={m.tenant_name}
                    className="h-10 w-10 rounded object-cover"
                  />
                ) : (
                  <div className="flex h-10 w-10 items-center justify-center rounded bg-blue-100 text-sm font-medium text-blue-700">
                    {m.tenant_name.slice(0, 2).toUpperCase()}
                  </div>
                )}
                <div className="flex-1">
                  <div className="font-medium text-gray-900">
                    {m.tenant_name}
                  </div>
                  <div className="text-xs text-gray-500">
                    {m.role} · {busySlug === m.tenant_slug ? "..." : m.tenant_slug}
                  </div>
                </div>
              </button>
            </li>
          ))}
        </ul>

        {memberships.length === 0 && (
          <div className="rounded border border-yellow-300 bg-yellow-50 px-4 py-3 text-center text-sm text-yellow-800">
            No tienes membresias activas. Contacta al administrador del
            conjunto para que te vincule por tu codigo unico.
          </div>
        )}
      </div>
    </div>
  );
}
