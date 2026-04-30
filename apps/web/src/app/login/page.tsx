"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { apiPost } from "@/lib/api";
import {
  setSession,
  setCurrentTenant,
  type MembershipDTO,
} from "@/lib/auth";

interface LoginResponse {
  access_token?: string;
  refresh_token?: string;
  token_type?: string;
  expires_in?: number;
  memberships?: MembershipDTO[];
  needs_tenant?: boolean;
  mfa_required?: boolean;
  pre_auth_token?: string;
}

interface SwitchTenantResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  current_tenant: MembershipDTO;
}

const DOC_TYPES = ["CC", "CE", "PA", "TI", "RC", "NIT"] as const;

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [docType, setDocType] = useState<typeof DOC_TYPES[number]>("CC");
  const [docNumber, setDocNumber] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const res = await apiPost<LoginResponse>("/auth/login", {
        email,
        document_type: docType,
        document_number: docNumber,
        password,
      });

      if (res.mfa_required) {
        setError("MFA aun no soportado en esta interfaz. Contactar al admin.");
        return;
      }
      if (!res.access_token) {
        setError("Respuesta inesperada del servidor.");
        return;
      }

      setSession({
        access_token: res.access_token,
        refresh_token: res.refresh_token,
        memberships: res.memberships ?? [],
      });

      const memberships = res.memberships ?? [];
      if (memberships.length === 1) {
        // Auto-switch al unico tenant.
        const slug = memberships[0].tenant_slug;
        const switched = await apiPost<SwitchTenantResponse>(
          "/auth/switch-tenant",
          { tenant_slug: slug },
        );
        setSession({ access_token: switched.access_token });
        setCurrentTenant(slug);
        router.push("/dashboard");
      } else if (memberships.length > 1) {
        router.push("/select-tenant");
      } else {
        setError("Tu cuenta no tiene acceso a ningun conjunto.");
      }
    } catch (err: unknown) {
      const apiErr = err as { detail?: string; title?: string };
      setError(apiErr?.detail || apiErr?.title || "Error al iniciar sesion");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-100">
      <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-md">
        <h1 className="mb-2 text-center text-2xl font-bold text-gray-900">
          Propiedad Horizontal
        </h1>
        <p className="mb-6 text-center text-sm text-gray-500">
          Ingresa tu correo, documento y contrasena
        </p>

        {error && (
          <div className="mb-4 rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="email"
              className="mb-1 block text-sm font-medium text-gray-700"
            >
              Correo electronico
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              placeholder="correo@ejemplo.com"
            />
          </div>

          <div className="flex gap-3">
            <div className="w-32">
              <label
                htmlFor="docType"
                className="mb-1 block text-sm font-medium text-gray-700"
              >
                Tipo
              </label>
              <select
                id="docType"
                value={docType}
                onChange={(e) =>
                  setDocType(e.target.value as typeof DOC_TYPES[number])
                }
                className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              >
                {DOC_TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex-1">
              <label
                htmlFor="docNumber"
                className="mb-1 block text-sm font-medium text-gray-700"
              >
                Numero de documento
              </label>
              <input
                id="docNumber"
                type="text"
                required
                value={docNumber}
                onChange={(e) => setDocNumber(e.target.value)}
                className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
                placeholder="1234567890"
              />
            </div>
          </div>

          <div>
            <label
              htmlFor="password"
              className="mb-1 block text-sm font-medium text-gray-700"
            >
              Contrasena
            </label>
            <input
              id="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              placeholder="********"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:outline-none disabled:opacity-50"
          >
            {loading ? "Ingresando..." : "Iniciar sesion"}
          </button>
        </form>
      </div>
    </div>
  );
}
