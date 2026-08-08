import { beforeEach, describe, expect, it, vi } from "vitest";
import { request } from "./client";
import { getDailyDiscoveries, getDailyDiscovery, getTodayDailyDiscovery } from "./dailyDiscovery";
import { getFeedback, saveFeedback } from "./feedback";
import { getLatestQuestionAnswers, getQuestions, saveQuestionAnswers } from "./question";
import { generateRecommendation, getLatestRecommendation, getRecommendation } from "./recommendation";

vi.mock("./client", () => ({ request: vi.fn() }));
const requestMock = vi.mocked(request);

beforeEach(() => {
  requestMock.mockReset();
  requestMock.mockResolvedValue({} as never);
});

describe("recommendation related API wrappers", () => {
  it("covers Daily Discovery APIs", async () => {
    await getTodayDailyDiscovery();
    await getDailyDiscoveries();
    await getDailyDiscoveries(3, 25);
    await getDailyDiscovery("rec-1");

    expect(requestMock).toHaveBeenNthCalledWith(1, "/api/daily-discovery", { method: "GET" });
    expect(requestMock).toHaveBeenNthCalledWith(2, "/api/daily-discoveries?page=1&limit=10", { method: "GET" });
    expect(requestMock).toHaveBeenNthCalledWith(3, "/api/daily-discoveries?page=3&limit=25", { method: "GET" });
    expect(requestMock).toHaveBeenNthCalledWith(4, "/api/daily-discoveries/rec-1", { method: "GET" });
  });

  it("covers Feedback APIs", async () => {
    await saveFeedback("detail-1", "LIKE");
    await getFeedback("detail-1");

    expect(requestMock).toHaveBeenNthCalledWith(1, "/api/recommendation-details/detail-1/feedback", {
      method: "PUT",
      body: JSON.stringify({ feedback_type: "LIKE" }),
    });
    expect(requestMock).toHaveBeenNthCalledWith(2, "/api/recommendation-details/detail-1/feedback", { method: "GET" });
  });

  it("covers Question APIs", async () => {
    const answers = [{ question_id: "q1", choice_ids: ["c1"] }];
    await getQuestions();
    await getLatestQuestionAnswers();
    await saveQuestionAnswers(answers);

    expect(requestMock).toHaveBeenNthCalledWith(1, "/api/questions", { method: "GET" });
    expect(requestMock).toHaveBeenNthCalledWith(2, "/api/user-answers/latest", { method: "GET" });
    expect(requestMock).toHaveBeenNthCalledWith(3, "/api/user-answers", {
      method: "POST",
      body: JSON.stringify({ answers }),
    });
  });

  it("covers Recommendation APIs", async () => {
    await generateRecommendation();
    await getLatestRecommendation();
    await getRecommendation("rec-2");

    expect(requestMock).toHaveBeenNthCalledWith(1, "/api/recommendations", { method: "POST" });
    expect(requestMock).toHaveBeenNthCalledWith(2, "/api/recommendations/latest", { method: "GET" });
    expect(requestMock).toHaveBeenNthCalledWith(3, "/api/recommendations/rec-2", { method: "GET" });
  });
});
