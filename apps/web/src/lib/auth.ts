// Helpers de auth/JWT post-Fase 16 (ADR 0007).
//
// Despues de POST /auth/login el backend devuelve:
//   { access_token, refresh_token, memberships[], needs_tenant }
// Si needs_tenant=false (memberships.length === 1) el cliente puede
// llamar a switchTenant inmediatamente. Si true, redirige a /select-tenant.

const ACCESS_KEY = "ph_access_token";
const REFRESH_KEY = "ph_refresh_token";
const MEMBERSHIPS_KEY = "ph_memberships";
const CURRENT_KEY = "ph_current_tenant";

export interface MembershipDTO {
  tenant_id: string;
  tenant_slug: string;
  tenant_name: string;
  logo_url?: string | null;
  primary_color?: string | null;
  role: string;
  status: string;
}

export interface AuthSession {
  accessToken: string;
  refreshToken?: string;
  memberships: MembershipDTO[];
  currentTenantSlug: string | null;
}

const isBrowser = () => typeof window !== "undefined";

export function setSession(s: {
  access_token: string;
  refresh_token?: string;
  memberships?: MembershipDTO[];
}) {
  if (!isBrowser()) return;
  localStorage.setItem(ACCESS_KEY, s.access_token);
  if (s.refresh_token) localStorage.setItem(REFRESH_KEY, s.refresh_token);
  if (s.memberships) {
    localStorage.setItem(MEMBERSHIPS_KEY, JSON.stringify(s.memberships));
  }
}

export function setCurrentTenant(slug: string) {
  if (!isBrowser()) return;
  localStorage.setItem(CURRENT_KEY, slug);
}

export function clearSession() {
  if (!isBrowser()) return;
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
  localStorage.removeItem(MEMBERSHIPS_KEY);
  localStorage.removeItem(CURRENT_KEY);
}

export function getAccessToken(): string | null {
  if (!isBrowser()) return null;
  return localStorage.getItem(ACCESS_KEY);
}

export function getRefreshToken(): string | null {
  if (!isBrowser()) return null;
  return localStorage.getItem(REFRESH_KEY);
}

export function getMemberships(): MembershipDTO[] {
  if (!isBrowser()) return [];
  const raw = localStorage.getItem(MEMBERSHIPS_KEY);
  if (!raw) return [];
  try {
    return JSON.parse(raw) as MembershipDTO[];
  } catch {
    return [];
  }
}

export function getCurrentTenant(): string | null {
  if (!isBrowser()) return null;
  return localStorage.getItem(CURRENT_KEY);
}

export function isAuthenticated(): boolean {
  return getAccessToken() !== null;
}
