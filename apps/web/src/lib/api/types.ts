export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance?: string;
}

export interface LoginRequest {
  identifier: string;
  password: string;
}

export interface LoginResponse {
  mfa_required: boolean;
  pre_auth_token?: string;
  access_token?: string;
  refresh_token?: string;
  expires_in?: number;
  token_type?: string;
}

export interface MFAVerifyRequest {
  pre_auth_token: string;
  code: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

export interface MeResponse {
  id: string;
  document_type: string;
  document_number: string;
  names: string;
  last_names: string;
  email?: string | null;
  phone?: string | null;
  mfa_enrolled_at?: string | null;
  last_login_at?: string | null;
}

export interface Unit {
  id: string;
  structure_id?: string | null;
  code: string;
  type: string;
  area_m2?: number | null;
  bedrooms?: number | null;
  coefficient?: number | null;
  status: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface CreateUnitRequest {
  structure_id?: string | null;
  code: string;
  type: string;
  area_m2?: number | null;
  bedrooms?: number | null;
  coefficient?: number | null;
}

export interface ListUnitsResponse {
  items: Unit[];
}

export interface PersonInUnit {
  user_id: string;
  full_name: string;
  document: string;
  role_in_unit: string;
  is_primary: boolean;
  since_date: string;
}

export interface PeopleInUnitResponse {
  unit_id: string;
  people: PersonInUnit[];
}

export interface Vehicle {
  id: string;
  plate: string;
  type: string;
  brand?: string | null;
  model?: string | null;
  color?: string | null;
  year?: number | null;
  status: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface CreateVehicleRequest {
  plate: string;
  type: string;
  brand?: string | null;
  model?: string | null;
  color?: string | null;
  year?: number | null;
}

export interface ListVehiclesResponse {
  items: Vehicle[];
  total: number;
}

export interface PackageItem {
  id: string;
  unit_id: string;
  recipient_name: string;
  category_id?: string | null;
  carrier?: string | null;
  tracking_number?: string | null;
  received_evidence_url?: string | null;
  received_by_user_id: string;
  received_at: string;
  delivered_at?: string | null;
  returned_at?: string | null;
  status: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface ListPackagesResponse {
  items: PackageItem[];
  total: number;
}

export interface PackageCategory {
  id: string;
  name: string;
  requires_evidence: boolean;
}

export interface ListCategoriesResponse {
  items: PackageCategory[];
  total: number;
}

export interface AudienceTarget {
  target_type: string;
  target_id?: string | null;
}

export interface Announcement {
  id: string;
  title: string;
  body: string;
  published_by: string;
  published_at: string;
  pinned: boolean;
  expires_at?: string | null;
  status: string;
  audiences: AudienceTarget[];
  created_at: string;
  updated_at: string;
  version: number;
}

export interface AnnouncementFeedResponse {
  items: Announcement[];
  total: number;
}

export interface VisitorEntry {
  id: string;
  unit_id?: string | null;
  pre_registration_id?: string | null;
  visitor_full_name: string;
  visitor_document_type?: string | null;
  visitor_document_number: string;
  photo_url?: string | null;
  guard_id: string;
  entry_time: string;
  exit_time?: string | null;
  source: string;
  notes?: string | null;
  status: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface ListVisitorEntriesResponse {
  items: VisitorEntry[];
  total: number;
}

export interface BlacklistEntry {
  id: string;
  document_type: string;
  document_number: string;
  full_name?: string | null;
  reason: string;
  reported_by_unit_id?: string | null;
  reported_by_user_id?: string | null;
  expires_at?: string | null;
  status: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface ListBlacklistResponse {
  items: BlacklistEntry[];
  total: number;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListRolesResponse {
  items: Role[];
}

export interface Permission {
  id: string;
  namespace: string;
  description: string;
}

export interface ListPermissionsResponse {
  items: Permission[];
}
