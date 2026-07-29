import { API_PATHS } from "../constants/apiPaths";
import type { Recommendation } from "../types/recommendation";
import { request } from "./client";
export function generateRecommendation(): Promise<Recommendation> { return request<Recommendation>(API_PATHS.recommendations, { method: "POST" }); }
export function getLatestRecommendation(): Promise<Recommendation> { return request<Recommendation>(API_PATHS.latestRecommendation, { method: "GET" }); }
export function getRecommendation(recommendationId: string): Promise<Recommendation> { return request<Recommendation>(API_PATHS.recommendation(recommendationId), { method: "GET" }); }
