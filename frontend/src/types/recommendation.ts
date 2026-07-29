export type RecommendationType = "YOUR_MATCH" | "NEW_FAVORITE" | "TRY_SOMETHING_NEW";
export type FeedbackStatus = "LIKE" | "MAYBE" | "SKIP";
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
  feedback_status: FeedbackStatus | null;
}
export interface Recommendation {
  recommendation_id: string;
  recommended_at: string;
  recommendations: RecommendationDetail[];
}
