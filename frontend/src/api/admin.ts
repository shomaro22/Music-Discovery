import { request } from "./client";
import type {
  AdminUser,
  Artist,
  AuditLog,
  PageResponse,
  Song,
  SpotifyArtistSearchItem,
  SpotifySearchResponse,
  SpotifyTrackSearchItem,
  SystemLog,
  Tag,
  TagRef,
} from "../types/admin";

export type ArtistSaveRequest = Omit<Partial<Artist>, "tags"> & {
  tag_ids?: string[];
};
export type SongSaveRequest = Omit<Partial<Song>, "tags" | "artist"> & {
  tags?: { tag_id: string; weight: number }[];
};
export type Query = Record<string, string | number | boolean | undefined>;
function qs(query: Query): string {
  const p = new URLSearchParams();
  Object.entries(query).forEach(([k, v]) => {
    if (v !== undefined && v !== "") p.set(k, String(v));
  });
  return p.toString();
}
const list = <T>(path: string, query: Query = {}) =>
  request<PageResponse<T>>(`${path}?${qs({ page: 1, limit: 20, ...query })}`, {
    method: "GET",
  });
export const listArtists = (keyword = "", page = 1, query: Query = {}) =>
  list<Artist>("/api/admin/artists", { keyword, page, ...query });
export const getArtist = (id: string) =>
  request<Artist>(`/api/admin/artists/${id}`, { method: "GET" });
export const saveArtist = (value: ArtistSaveRequest, id?: string) =>
  request<Artist>(id ? `/api/admin/artists/${id}` : "/api/admin/artists", {
    method: id ? "PATCH" : "POST",
    body: JSON.stringify(value),
  });
export const deleteArtist = (id: string) =>
  request<void>(`/api/admin/artists/${id}`, { method: "DELETE" });
export const listSongs = (keyword = "", page = 1, query: Query = {}) =>
  list<Song>("/api/admin/songs", { keyword, page, ...query });
export const getSong = (id: string) =>
  request<Song>(`/api/admin/songs/${id}`, { method: "GET" });
export const saveSong = (value: SongSaveRequest, id?: string) =>
  request<Song>(id ? `/api/admin/songs/${id}` : "/api/admin/songs", {
    method: id ? "PATCH" : "POST",
    body: JSON.stringify(value),
  });
export const deleteSong = (id: string) =>
  request<void>(`/api/admin/songs/${id}`, { method: "DELETE" });
export const listTags = (keyword = "", page = 1, query: Query = {}) =>
  list<Tag>("/api/admin/tags", { keyword, page, ...query });
export const listTagOptions = () =>
  request<TagRef[]>("/api/admin/tags/options", { method: "GET" });
export const getTag = (id: string) =>
  request<Tag>(`/api/admin/tags/${id}`, { method: "GET" });
export const saveTag = (value: Partial<Tag>, id?: string) =>
  request<Tag>(id ? `/api/admin/tags/${id}` : "/api/admin/tags", {
    method: id ? "PATCH" : "POST",
    body: JSON.stringify(value),
  });
export const deleteTag = (id: string) =>
  request<void>(`/api/admin/tags/${id}`, { method: "DELETE" });
export const listUsers = (keyword = "", page = 1, query: Query = {}) =>
  list<AdminUser>("/api/system/users", { keyword, page, ...query });
export const getUser = (id: string) =>
  request<AdminUser>(`/api/system/users/${id}`, { method: "GET" });
export const setUserStatus = (
  id: string,
  status: AdminUser["status"],
  reason: string,
) =>
  request<void>(`/api/system/users/${id}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status, reason }),
  });
export const changeUserRole = (
  id: string,
  role: AdminUser["role"],
  reason: string,
) =>
  request<{ user_id: string; before_role: string; after_role: string }>(
    `/api/system/users/${id}/role`,
    { method: "PATCH", body: JSON.stringify({ role, reason }) },
  );
export const listAuditLogs = (keyword = "", page = 1, query: Query = {}) =>
  list<AuditLog>("/api/system/audit-logs", { keyword, page, ...query });
export const getAuditLog = (id: string) =>
  request<AuditLog>(`/api/system/audit-logs/${id}`, { method: "GET" });
export const listSystemLogs = (keyword = "", page = 1, query: Query = {}) =>
  list<SystemLog>("/api/system/logs", { keyword, page, ...query });
export const getSystemLog = (id: string) =>
  request<SystemLog>(`/api/system/logs/${id}`, { method: "GET" });

export const searchSpotifyArtists = (query: string, limit = 10) =>
  request<SpotifySearchResponse<SpotifyArtistSearchItem>>(
    `/api/admin/spotify/artists?${qs({ q: query, limit })}`,
    { method: "GET" },
  );

export const searchSpotifyTracks = (query: string, limit = 10) =>
  request<SpotifySearchResponse<SpotifyTrackSearchItem>>(
    `/api/admin/spotify/tracks?${qs({ q: query, limit })}`,
    { method: "GET" },
  );
