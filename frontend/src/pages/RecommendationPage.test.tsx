import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";
import { generateRecommendation, getLatestRecommendation } from "../api/recommendation";
import type { Recommendation } from "../types/recommendation";
import { RecommendationPage } from "./RecommendationPage";

vi.mock("../api/recommendation", () => ({
  generateRecommendation: vi.fn(),
  getLatestRecommendation: vi.fn(),
}));

const generateMock = vi.mocked(generateRecommendation);
const latestMock = vi.mocked(getLatestRecommendation);

const recommendation: Recommendation = {
  recommendation_id: "r1",
  recommended_at: "2026-07-29T00:00:00Z",
  recommendations: [
    {
      recommendation_detail_id: "d1",
      recommendation_type: "YOUR_MATCH",
      song_id: "s1",
      song_name: "Song One",
      artist_id: "a1",
      artist_name: "Artist One",
      artwork_url: "",
      spotify_url: "https://open.spotify.com/track/1",
      reason: "reason",
      feedback_status: null,
    },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
});

function renderPage(initialEntry: string | { pathname: string; state?: unknown } = "/recommendations") {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/recommendations" element={<RecommendationPage />} />
        <Route path="/questions" element={<p>questions page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("RecommendationPage", () => {
  it("generates a recommendation when navigation state requests generation", async () => {
    generateMock.mockResolvedValue(recommendation);
    renderPage({ pathname: "/recommendations", state: { generate: true } });

    await waitFor(() => expect(generateMock).toHaveBeenCalledTimes(1));
    expect(latestMock).not.toHaveBeenCalled();
    expect(await screen.findByText("Song One")).toBeInTheDocument();
  });

  it("loads the latest recommendation without generation state", async () => {
    latestMock.mockResolvedValue(recommendation);
    renderPage();

    await waitFor(() => expect(latestMock).toHaveBeenCalledTimes(1));
    expect(generateMock).not.toHaveBeenCalled();
  });

  it("retries the failed generation with the generation API", async () => {
    generateMock
      .mockRejectedValueOnce(new Error("failed"))
      .mockResolvedValueOnce(recommendation);
    renderPage({ pathname: "/recommendations", state: { generate: true } });

    fireEvent.click(await screen.findByRole("button", { name: "再試行" }));
    await waitFor(() => expect(generateMock).toHaveBeenCalledTimes(2));
    expect(latestMock).not.toHaveBeenCalled();
  });

  it("retries the failed latest request with the latest API", async () => {
    latestMock
      .mockRejectedValueOnce(new Error("failed"))
      .mockResolvedValueOnce(recommendation);
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "再試行" }));
    await waitFor(() => expect(latestMock).toHaveBeenCalledTimes(2));
    expect(generateMock).not.toHaveBeenCalled();
  });

  it("moves to questions when answers are required", async () => {
    generateMock.mockRejectedValue(
      new ApiRequestError({
        status: 400,
        code: "QUESTION_ANSWER_REQUIRED",
        message: "answers required",
        request_id: "req-1",
      }),
    );
    renderPage({ pathname: "/recommendations", state: { generate: true } });

    expect(await screen.findByText("questions page")).toBeInTheDocument();
  });
});
