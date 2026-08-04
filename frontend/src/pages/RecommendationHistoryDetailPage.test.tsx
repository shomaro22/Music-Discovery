import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getRecommendationHistoryDetail } from "../api/recommendationHistory";
import { RecommendationHistoryDetailPage } from "./RecommendationHistoryDetailPage";

vi.mock("../api/recommendationHistory", () => ({ getRecommendationHistoryDetail: vi.fn() }));
vi.mock("../components/recommendation/RecommendationCard", () => ({ RecommendationCard: ({ detail }: { detail: { song_name: string; reason: string } }) => <article><h2>{detail.song_name}</h2><p>{detail.reason}</p></article> }));
const getDetailMock = vi.mocked(getRecommendationHistoryDetail);

const detail = {
  recommendation_id: "rec-1",
  source: "DAILY_DISCOVERY" as const,
  target_date: "2026-08-04",
  recommended_at: "2026-08-04T12:00:00Z",
  timezone: "Asia/Tokyo",
  recommendations: [1, 2, 3].map((position) => ({
    recommendation_detail_id: `detail-${position}`,
    recommendation_type: position === 3 ? "DAILY_CHALLENGE" as const : position === 1 ? "DAILY_MATCH_ONE" as const : "DAILY_MATCH_TWO" as const,
    song_id: `song-${position}`,
    song_name: `Song ${position}`,
    artist_id: `artist-${position}`,
    artist_name: `Artist ${position}`,
    artwork_url: "",
    spotify_url: "https://open.spotify.com/track/test",
    reason: `Reason ${position}`,
    feedback_status: null,
  })),
};

describe("RecommendationHistoryDetailPage", () => {
  beforeEach(() => { getDetailMock.mockResolvedValue(detail); });

  it("詳細3曲、理由、対象日、一覧へ戻る導線を表示する", async () => {
    render(<MemoryRouter initialEntries={["/recommendation-history/rec-1?page=2&source=DAILY_DISCOVERY"]}><Routes><Route path="/recommendation-history/:recommendationId" element={<RecommendationHistoryDetailPage />} /></Routes></MemoryRouter>);
    expect(await screen.findByText("Song 1")).toBeInTheDocument();
    expect(screen.getByText("Song 2")).toBeInTheDocument();
    expect(screen.getByText("Song 3")).toBeInTheDocument();
    expect(screen.getByText("Reason 1")).toBeInTheDocument();
    expect(screen.getByText(/対象日 2026-08-04/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "推薦履歴へ戻る" })).toHaveAttribute("href", "/recommendation-history?page=2&source=DAILY_DISCOVERY");
  });

  it("APIエラーを表示する", async () => {
    getDetailMock.mockRejectedValue(new Error("failed"));
    render(<MemoryRouter initialEntries={["/recommendation-history/rec-1"]}><Routes><Route path="/recommendation-history/:recommendationId" element={<RecommendationHistoryDetailPage />} /></Routes></MemoryRouter>);
    expect(await screen.findByText("履歴を表示できません")).toBeInTheDocument();
  });
});
