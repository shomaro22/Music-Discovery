export type FeedbackType = "LIKE" | "MAYBE" | "SKIP";
export type FeedbackStatus = "NONE" | FeedbackType;

export interface SaveFeedbackRequest {
  feedback_type: FeedbackType;
}

export interface SaveFeedbackResponse {
  feedback_id: string;
  recommendation_detail_id: string;
  feedback_type: FeedbackType;
  created: boolean;
  updated_at: string;
}

export interface FeedbackDetailResponse {
  recommendation_detail_id: string;
  feedback_status: FeedbackStatus;
  updated_at: string | null;
}
