import type { FeedbackType } from "./feedback";
import type { RecommendationType } from "./recommendation";
import type { RecommendationDetail } from "./recommendation";

export type RecommendationSource = "NORMAL" | "DAILY_DISCOVERY";

export interface RecommendationHistorySong {
  recommendation_detail_id: string;
  recommendation_type: RecommendationType;
  display_order: number;
  song_id: string;
  song_name: string;
  artist_id: string;
  artist_name: string;
  artwork_url: string;
  reason: string;
  feedback_status: FeedbackType | null;
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
  timezone: string;
}

export interface RecommendationHistoryDetailResponse {
  recommendation_id: string;
  source: RecommendationSource;
  target_date?: string;
  recommended_at: string;
  recommendations: RecommendationDetail[];
  timezone: string;
}

export interface RecommendationHistoryQuery {
  page: number;
  limit: number;
  source?: RecommendationSource;
  startDate?: string;
  endDate?: string;
}
