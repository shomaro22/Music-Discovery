import type { FeedbackStatus } from "./feedback";
import type { RecommendationDetail } from "./recommendation";

export type RecommendationSource = "NORMAL" | "DAILY_DISCOVERY";

export interface RecommendationHistorySong {
  recommendation_detail_id: string;
  display_order: number;
  song_id: string;
  song_name: string;
  artist_id: string;
  artist_name: string;
  artwork_url: string;
  feedback_status: FeedbackStatus;
}

export interface RecommendationHistoryItem {
  recommendation_id: string;
  source: RecommendationSource;
  target_date?: string;
  recommended_at: string;
  songs: RecommendationHistorySong[];
}

export interface RecommendationHistoryListResponse {
  items: RecommendationHistoryItem[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface RecommendationHistoryDetailResponse {
  recommendation_id: string;
  source: RecommendationSource;
  target_date?: string;
  recommended_at: string;
  recommendations: RecommendationDetail[];
}

export interface RecommendationHistoryQuery {
  page: number;
  limit: number;
  source?: RecommendationSource;
  startDate?: string;
  endDate?: string;
}
