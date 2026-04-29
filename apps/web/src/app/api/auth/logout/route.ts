import { NextResponse } from "next/server";
import {
  API_BASE_URL,
  TENANT_SLUG,
  COOKIE_ACCESS,
  COOKIE_REFRESH,
} from "@/lib/api/config";
import { cookies } from "next/headers";

export async function POST() {
  const store = await cookies();
  const token = store.get(COOKIE_ACCESS)?.value;

  if (token) {
    await fetch(`${API_BASE_URL}/auth/logout`, {
      method: "POST",
      headers: {
        "X-Tenant-Slug": TENANT_SLUG,
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
    }).catch(() => undefined);
  }

  const response = NextResponse.json({ ok: true });
  response.cookies.delete(COOKIE_ACCESS);
  response.cookies.delete(COOKIE_REFRESH);
  return response;
}
