export const API_BASE_URL =
  process.env.API_INTERNAL_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8080";

export const TENANT_SLUG = process.env.TENANT_SLUG || "demo";

export const COOKIE_ACCESS = "ph_access";
export const COOKIE_REFRESH = "ph_refresh";
