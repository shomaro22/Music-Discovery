export const API_PATHS = {
  recommendations: "/api/recommendations",
  latestRecommendation: "/api/recommendations/latest",
  recommendation: (recommendationId: string) => `/api/recommendations/${recommendationId}`,
} as const;
