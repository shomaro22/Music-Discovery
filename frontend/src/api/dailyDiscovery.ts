import { API_PATHS } from "../constants/apiPaths";
import type { DailyDiscovery, DailyDiscoveryList } from "../types/recommendation";
import { request } from "./client";
export const getTodayDailyDiscovery = (): Promise<DailyDiscovery> => request<DailyDiscovery>(API_PATHS.dailyDiscovery, { method: "GET" });
export const getDailyDiscoveries = (page = 1, limit = 10): Promise<DailyDiscoveryList> => request<DailyDiscoveryList>(`${API_PATHS.dailyDiscoveries}?page=${page}&limit=${limit}`, { method: "GET" });
export const getDailyDiscovery = (id: string): Promise<DailyDiscovery> => request<DailyDiscovery>(API_PATHS.dailyDiscoveryDetail(id), { method: "GET" });
