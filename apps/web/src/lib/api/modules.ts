import { apiGet, apiPost, apiDelete } from "./server";
import type {
  Unit,
  CreateUnitRequest,
  ListUnitsResponse,
  PeopleInUnitResponse,
  Vehicle,
  CreateVehicleRequest,
  ListVehiclesResponse,
  PackageItem,
  ListPackagesResponse,
  ListCategoriesResponse,
  Announcement,
  AnnouncementFeedResponse,
  ListVisitorEntriesResponse,
  ListBlacklistResponse,
  ListRolesResponse,
  ListPermissionsResponse,
  MeResponse,
} from "./types";

export const me = () => apiGet<MeResponse>("/me");

export const listUnits = () => apiGet<ListUnitsResponse>("/units");
export const getUnit = (id: string) => apiGet<Unit>(`/units/${id}`);
export const getUnitPeople = (id: string) =>
  apiGet<PeopleInUnitResponse>(`/units/${id}/people`);
export const createUnit = (body: CreateUnitRequest) =>
  apiPost<Unit>("/units", body);

export const listVehicles = () => apiGet<ListVehiclesResponse>("/vehicles");
export const createVehicle = (body: CreateVehicleRequest) =>
  apiPost<Vehicle>("/vehicles", body);

export const listPackages = (params?: { unit_id?: string; status?: string }) =>
  apiGet<ListPackagesResponse>("/packages", params);
export const getPackage = (id: string) => apiGet<PackageItem>(`/packages/${id}`);
export const listPackageCategories = () =>
  apiGet<ListCategoriesResponse>("/package-categories");

export const listAnnouncementsFeed = () =>
  apiGet<AnnouncementFeedResponse>("/announcements/feed");
export const getAnnouncement = (id: string) =>
  apiGet<Announcement>(`/announcements/${id}`);

export const listActiveVisits = () =>
  apiGet<ListVisitorEntriesResponse>("/visits/active");
export const listBlacklist = () => apiGet<ListBlacklistResponse>("/blacklist");

export const listRoles = () => apiGet<ListRolesResponse>("/roles");
export const listPermissions = () =>
  apiGet<ListPermissionsResponse>("/permissions");

export const archiveAnnouncement = (id: string) =>
  apiDelete<void>(`/announcements/${id}`);
