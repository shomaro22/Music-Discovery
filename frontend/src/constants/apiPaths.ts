export const API_PATHS = {
  passwordResetRequest: "/api/auth/password-reset/request",
  passwordResetConfirm: "/api/auth/password-reset/confirm",
  questions: "/api/questions",
  latestQuestionAnswers: "/api/user-answers/latest",
  saveQuestionAnswers: "/api/user-answers",
  recommendations: "/api/recommendations",
  latestRecommendation: "/api/recommendations/latest",
  recommendation: (recommendationId: string) => `/api/recommendations/${recommendationId}`,
  dailyDiscovery: "/api/daily-discovery",
  dailyDiscoveries: "/api/daily-discoveries",
  dailyDiscoveryDetail: (recommendationId: string) => `/api/daily-discoveries/${recommendationId}`,
  recommendationHistory: "/api/recommendation-history",
  recommendationHistoryDetail: (recommendationId: string) => `/api/recommendation-history/${recommendationId}`,
  feedback: (recommendationDetailId: string) =>
    `/api/recommendation-details/${recommendationDetailId}/feedback`,
} as const;
