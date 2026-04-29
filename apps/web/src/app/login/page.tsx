"use client";

import { Suspense, useState, type FormEvent } from "react";
import { useRouter, useSearchParams } from "next/navigation";

export default function LoginPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-slate-50" />}>
      <LoginForm />
    </Suspense>
  );
}

function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const next = params.get("next") || "/dashboard";

  const [identifier, setIdentifier] = useState("admin@demo.ph.localhost");
  const [password, setPassword] = useState("admin123");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ identifier, password }),
      });

      if (!res.ok) {
        const problem = await res
          .json()
          .catch(() => ({ detail: "Error inesperado" }));
        setError(problem.detail || problem.title || "Error al iniciar sesion");
        return;
      }

      const data = await res.json();
      if (data.mfa_required) {
        setError(
          "MFA requerido (no soportado en esta UI todavia). Usuario sin MFA por favor.",
        );
        return;
      }

      router.push(next);
      router.refresh();
    } catch (err) {
      const msg =
        err instanceof Error ? err.message : "Error de red al iniciar sesion";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-slate-50 via-white to-slate-100 px-4">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-600 text-white shadow-md shadow-indigo-200">
            <span className="text-lg font-bold">PH</span>
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">
            Propiedad Horizontal
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Plataforma de administracion del conjunto
          </p>
        </div>

        <div className="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
          {error && (
            <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label
                htmlFor="identifier"
                className="mb-1.5 block text-sm font-medium text-slate-700"
              >
                Correo o documento
              </label>
              <input
                id="identifier"
                type="text"
                required
                autoComplete="username"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-100"
                placeholder="correo@ejemplo.com"
              />
              <p className="mt-1 text-xs text-slate-400">
                Tambien acepta CC:12345678
              </p>
            </div>

            <div>
              <label
                htmlFor="password"
                className="mb-1.5 block text-sm font-medium text-slate-700"
              >
                Contrasena
              </label>
              <input
                id="password"
                type="password"
                required
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-100"
                placeholder="********"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="mt-2 w-full rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-200 disabled:opacity-50"
            >
              {loading ? "Ingresando..." : "Iniciar sesion"}
            </button>
          </form>
        </div>

        <p className="mt-6 text-center text-xs text-slate-400">
          Demo cargado con admin@demo.ph.localhost / admin123
        </p>
      </div>
    </div>
  );
}
