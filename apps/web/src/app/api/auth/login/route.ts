import { NextRequest, NextResponse } from "next/server";
import {
  API_BASE_URL,
  TENANT_SLUG,
  COOKIE_ACCESS,
  COOKIE_REFRESH,
} from "@/lib/api/config";
import type { LoginResponse, ProblemDetails } from "@/lib/api/types";

interface ClientLoginBody {
  identifier?: string;
  email?: string;
  password?: string;
}

export async function POST(req: NextRequest) {
  const body = (await req.json()) as ClientLoginBody;
  const identifier = body.identifier ?? body.email ?? "";
  const password = body.password ?? "";

  if (!identifier || !password) {
    return NextResponse.json(
      { type: "about:blank", title: "Bad Request", status: 400, detail: "identifier y password son requeridos" },
      { status: 400 },
    );
  }

  const res = await fetch(`${API_BASE_URL}/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-Slug": TENANT_SLUG,
    },
    body: JSON.stringify({ identifier, password }),
    cache: "no-store",
  });

  if (!res.ok) {
    let problem: ProblemDetails;
    try {
      problem = await res.json();
    } catch {
      problem = {
        type: "about:blank",
        title: res.statusText,
        status: res.status,
        detail: res.statusText,
      };
    }
    return NextResponse.json(problem, { status: res.status });
  }

  const data = (await res.json()) as LoginResponse;

  if (data.mfa_required) {
    return NextResponse.json(
      { mfa_required: true, pre_auth_token: data.pre_auth_token },
      { status: 200 },
    );
  }

  if (!data.access_token || !data.refresh_token) {
    return NextResponse.json(
      {
        type: "about:blank",
        title: "Bad gateway",
        status: 502,
        detail: "respuesta inesperada del API",
      },
      { status: 502 },
    );
  }

  const response = NextResponse.json({ ok: true }, { status: 200 });
  const isProd = process.env.NODE_ENV === "production";
  response.cookies.set(COOKIE_ACCESS, data.access_token, {
    httpOnly: true,
    secure: isProd,
    sameSite: "lax",
    path: "/",
    maxAge: data.expires_in ?? 3600,
  });
  response.cookies.set(COOKIE_REFRESH, data.refresh_token, {
    httpOnly: true,
    secure: isProd,
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 30,
  });
  return response;
}
