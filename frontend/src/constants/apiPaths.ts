export const API_PATHS = {
  questions: "/api/questions",
  latestQuestionAnswers: "/api/user-answers/latest",
  saveQuestionAnswers: "/api/user-answers",
  recommendations: "/api/recommendations",
  latestRecommendation: "/api/recommendations/latest",
  recommendation: (recommendationId: string) => `/api/recommendations/${recommendationId}`,
  dailyDiscovery: "/api/daily-discovery",
  dailyDiscoveries: "/api/daily-discoveries",
  dailyDiscoveryDetail: (recommendationId: string) => `/api/daily-discoveries/${recommendationId}`,
  feedback: (recommendationDetailId: string) =>
    `/api/recommendation-details/${recommendationDetailId}/feedback`,
} as const;
