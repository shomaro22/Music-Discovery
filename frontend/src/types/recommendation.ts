import type { FeedbackType } from "./feedback";
export type RecommendationType = "YOUR_MATCH" | "NEW_FAVORITE" | "TRY_SOMETHING_NEW" | "DAILY_MATCH_ONE" | "DAILY_MATCH_TWO" | "DAILY_CHALLENGE";
export interface RecommendationDetail {
  recommendation_detail_id: string;
  recommendation_type: RecommendationType;
  song_id: string;
  song_name: string;
  artist_id: string;
  artist_name: string;
  artwork_url: string;
  spotify_url: string;
  reason: string;
  feedback_status: FeedbackType | null;
}
export interface Recommendation {
  recommendation_id: string;
  recommended_at: string;
  recommendations: RecommendationDetail[];
}

export interface DailyDiscovery extends Recommendation { target_date: string; generated: boolean; }
export interface DailyDiscoveryList { items: DailyDiscovery[]; page: number; limit: number; total: number; total_pages: number; }
