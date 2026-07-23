import { request } from "./client";
import type { AdminUser, Artist, AuditLog, PageResponse, Song, SystemLog, Tag } from "../types/admin";

function queryString(keyword: string, page: number): string { const p = new URLSearchParams({ page: String(page), limit: "20" }); if (keyword.trim()) p.set("keyword", keyword.trim()); return p.toString(); }
export const listArtists=(keyword="",page=1)=>request<PageResponse<Artist>>(`/api/admin/artists?${queryString(keyword,page)}`,{method:"GET"});
export const saveArtist=(value: Partial<Artist>,id?:string)=>request<Artist>(id?`/api/admin/artists/${id}`:"/api/admin/artists",{method:id?"PATCH":"POST",body:JSON.stringify(value)});
export const deleteArtist=(id:string)=>request<void>(`/api/admin/artists/${id}`,{method:"DELETE"});
export const listSongs=(keyword="",page=1)=>request<PageResponse<Song>>(`/api/admin/songs?${queryString(keyword,page)}`,{method:"GET"});
export const saveSong=(value: Partial<Song>,id?:string)=>request<Song>(id?`/api/admin/songs/${id}`:"/api/admin/songs",{method:id?"PATCH":"POST",body:JSON.stringify(value)});
export const deleteSong=(id:string)=>request<void>(`/api/admin/songs/${id}`,{method:"DELETE"});
export const listTags=(keyword="",page=1)=>request<PageResponse<Tag>>(`/api/admin/tags?${queryString(keyword,page)}`,{method:"GET"});
export const saveTag=(value: Partial<Tag>,id?:string)=>request<Tag>(id?`/api/admin/tags/${id}`:"/api/admin/tags",{method:id?"PATCH":"POST",body:JSON.stringify(value)});
export const deleteTag=(id:string)=>request<void>(`/api/admin/tags/${id}`,{method:"DELETE"});
export const listUsers=(keyword="",page=1)=>request<PageResponse<AdminUser>>(`/api/system/users?${queryString(keyword,page)}`,{method:"GET"});
export const setUserStatus=(id:string,status:AdminUser["status"])=>request<void>(`/api/system/users/${id}/status`,{method:"PATCH",body:JSON.stringify({status})});
export const changeUserRole=(id:string,role:AdminUser["role"])=>request<void>(`/api/system/users/${id}/role`,{method:"PATCH",body:JSON.stringify({role})});
export const listAuditLogs=(keyword="",page=1)=>request<PageResponse<AuditLog>>(`/api/system/audit-logs?${queryString(keyword,page)}`,{method:"GET"});
export const listSystemLogs=(keyword="",page=1)=>request<PageResponse<SystemLog>>(`/api/system/logs?${queryString(keyword,page)}`,{method:"GET"});
