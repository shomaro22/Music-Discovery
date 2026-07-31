import { API_PATHS } from "../constants/apiPaths";
import type { FeedbackDetailResponse, FeedbackType, SaveFeedbackResponse } from "../types/feedback";
import { request } from "./client";

export function saveFeedback(recommendationDetailId: string, feedbackType: FeedbackType): Promise<SaveFeedbackResponse> {
  return request<SaveFeedbackResponse>(API_PATHS.feedback(recommendationDetailId), {
    method: "PUT",
    body: JSON.stringify({ feedback_type: feedbackType }),
  });
}

export function getFeedback(recommendationDetailId: string): Promise<FeedbackDetailResponse> {
  return request<FeedbackDetailResponse>(API_PATHS.feedback(recommendationDetailId), { method: "GET" });
}
