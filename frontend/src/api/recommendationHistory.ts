import { API_PATHS } from "../constants/apiPaths";
import type {
  RecommendationHistoryDetailResponse,
  RecommendationHistoryListResponse,
  RecommendationHistoryQuery,
} from "../types/recommendationHistory";
import { request } from "./client";

export function getRecommendationHistory(query: RecommendationHistoryQuery): Promise<RecommendationHistoryListResponse> {
  const params = new URLSearchParams({ page: String(query.page), limit: String(query.limit) });
  if (query.source) params.set("source", query.source);
  if (query.startDate) params.set("startDate", query.startDate);
  if (query.endDate) params.set("endDate", query.endDate);
  return request<RecommendationHistoryListResponse>(`${API_PATHS.recommendationHistory}?${params.toString()}`, { method: "GET" });
}

export function getRecommendationHistoryDetail(recommendationId: string): Promise<RecommendationHistoryDetailResponse> {
  return request<RecommendationHistoryDetailResponse>(API_PATHS.recommendationHistoryDetail(recommendationId), { method: "GET" });
}
