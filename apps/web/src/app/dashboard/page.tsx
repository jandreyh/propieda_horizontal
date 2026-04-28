"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

interface MeResponse {
  id: string;
  full_name: string;
  email?: string;
  role: string;
}

export default function DashboardPage() {
  const [user, setUser] = useState<MeResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<MeResponse>("/v1/me")
      .then(setUser)
      .catch((err) =>
        setError(err instanceof Error ? err.message : "Error cargando perfil"),
      );
  }, []);

  if (error) {
    return (
      <main className="flex min-h-screen items-center justify-center">
        <p className="text-red-600">{error}</p>
      </main>
    );
  }

  if (!user) {
    return (
      <main className="flex min-h-screen items-center justify-center">
        <p>Cargando...</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-3xl p-8">
      <h1 className="text-2xl font-semibold">Dashboard</h1>
      <dl className="mt-4 space-y-2">
        <div>
          <dt className="text-sm text-gray-500">Nombre</dt>
          <dd>{user.full_name}</dd>
        </div>
        <div>
          <dt className="text-sm text-gray-500">Rol</dt>
          <dd>{user.role}</dd>
        </div>
        {user.email && (
          <div>
            <dt className="text-sm text-gray-500">Email</dt>
            <dd>{user.email}</dd>
          </div>
        )}
      </dl>
    </main>
  );
}
