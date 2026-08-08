import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useNavigate } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";
import { saveFeedback } from "../api/feedback";
import { getRecommendationHistoryDetail } from "../api/recommendationHistory";
import type { RecommendationHistoryDetailResponse } from "../types/recommendationHistory";
import { RecommendationHistoryDetailPage } from "./RecommendationHistoryDetailPage";

vi.mock("../api/recommendationHistory", () => ({
  getRecommendationHistoryDetail: vi.fn(),
}));
vi.mock("../api/feedback", () => ({ saveFeedback: vi.fn() }));

const getDetailMock = vi.mocked(getRecommendationHistoryDetail);
const saveFeedbackMock = vi.mocked(saveFeedback);

const detail: RecommendationHistoryDetailResponse = {
  recommendation_id: "rec-1",
  source: "DAILY_DISCOVERY",
  target_date: "2026-08-04",
  recommended_at: "2026-08-04T12:00:00Z",
  timezone: "Asia/Tokyo",
  recommendations: [
    {
      recommendation_detail_id: "detail-1",
      recommendation_type: "DAILY_MATCH_ONE",
      song_id: "song-1",
      song_name: "Song 1",
      artist_id: "artist-1",
      artist_name: "Artist 1",
      artwork_url: "https://example.com/song-1.jpg",
      spotify_url: "https://open.spotify.com/track/one",
      reason: "Reason 1",
      feedback_status: "LIKE",
    },
    {
      recommendation_detail_id: "detail-2",
      recommendation_type: "DAILY_MATCH_TWO",
      song_id: "song-2",
      song_name: "Song 2",
      artist_id: "artist-2",
      artist_name: "Artist 2",
      artwork_url: "",
      spotify_url: "https://open.spotify.com/track/two",
      reason: "Reason 2",
      feedback_status: null,
    },
    {
      recommendation_detail_id: "detail-3",
      recommendation_type: "DAILY_CHALLENGE",
      song_id: "song-3",
      song_name: "Song 3",
      artist_id: "artist-3",
      artist_name: "Artist 3",
      artwork_url: "https://example.com/song-3.jpg",
      spotify_url: "https://open.spotify.com/track/three",
      reason: "Reason 3",
      feedback_status: "MAYBE",
    },
  ],
};

function renderDetail(path = "/recommendation-history/rec-1?page=2&source=DAILY_DISCOVERY") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/recommendation-history/:recommendationId"
          element={<RecommendationHistoryDetailPage />}
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("RecommendationHistoryDetailPage", () => {
  beforeEach(() => {
    getDetailMock.mockReset();
    saveFeedbackMock.mockReset();
    getDetailMock.mockResolvedValue(detail);
  });

  it("3曲、表示名、理由、Spotify Link、Feedback、代替Artworkを表示する", async () => {
    renderDetail();

    expect(await screen.findByText("Song 1")).toBeInTheDocument();
    expect(screen.getByText("Song 2")).toBeInTheDocument();
    expect(screen.getByText("Song 3")).toBeInTheDocument();
    expect(screen.getAllByText("Daily Match")).toHaveLength(2);
    expect(screen.getByText("Daily Challenge")).toBeInTheDocument();
    expect(screen.getByText("Reason 1")).toBeInTheDocument();
    expect(screen.getByText(/対象日 2026-08-04/)).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Song 1 artwork" })).toHaveAttribute(
      "src",
      "https://example.com/song-1.jpg",
    );
    expect(screen.getByLabelText("artworkなし")).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: "Spotifyで聴く" })[0]).toHaveAttribute(
      "href",
      "https://open.spotify.com/track/one",
    );
    expect(screen.getAllByRole("button", { name: "好き LIKE" })[0]).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByRole("link", { name: "推薦履歴へ戻る" })).toHaveAttribute(
      "href",
      "/recommendation-history?page=2&source=DAILY_DISCOVERY",
    );
  });

  it("Feedback変更成功時に選択状態を更新する", async () => {
    const user = userEvent.setup();
    saveFeedbackMock.mockResolvedValue({
      feedback_id: "feedback-1",
      recommendation_detail_id: "detail-2",
      feedback_type: "SKIP",
      created: true,
      updated_at: "2026-08-05T00:00:00Z",
    });
    renderDetail();
    await screen.findByText("Song 2");

    const cards = screen.getAllByRole("article");
    await user.click(cards[1].querySelector<HTMLButtonElement>('button[aria-label="今回は違う SKIP"]')!);

    await waitFor(() =>
      expect(saveFeedbackMock).toHaveBeenCalledWith("detail-2", "SKIP"),
    );
    expect(cards[1].querySelector('button[aria-label="今回は違う SKIP"]')).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("Feedback変更失敗時に以前の選択状態を維持する", async () => {
    const user = userEvent.setup();
    saveFeedbackMock.mockRejectedValue(new Error("failed"));
    renderDetail();
    await screen.findByText("Song 1");

    const cards = screen.getAllByRole("article");
    const like = cards[0].querySelector<HTMLButtonElement>('button[aria-label="好き LIKE"]')!;
    const skip = cards[0].querySelector<HTMLButtonElement>('button[aria-label="今回は違う SKIP"]')!;
    expect(like).toHaveAttribute("aria-pressed", "true");

    await user.click(skip);

    expect(await screen.findByText("Feedbackを保存できませんでした")).toBeInTheDocument();
    expect(like).toHaveAttribute("aria-pressed", "true");
    expect(skip).toHaveAttribute("aria-pressed", "false");
  });

  it.each<[string, string, number]>([
    [
      "RECOMMENDATION_HISTORY_NOT_FOUND",
      "推薦履歴が見つかりません",
      404,
    ],
    [
      "RECOMMENDATION_HISTORY_ACCESS_DENIED",
      "この推薦履歴にはアクセスできません",
      403,
    ],
  ])("%sを専用画面として表示する", async (code, title, status) => {
    getDetailMock.mockRejectedValue(
      new ApiRequestError({
        status,
        code,
        message: "API error",
        request_id: "request-id",
      }),
    );
    renderDetail();

    expect(await screen.findByRole("heading", { name: title })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("API error");
  });

  it("その他のAPI Errorは共通見出しを表示する", async () => {
    getDetailMock.mockRejectedValue(
      new ApiRequestError({
        status: 500,
        code: "RECOMMENDATION_HISTORY_FETCH_FAILED",
        message: "取得失敗",
        request_id: "request-id",
      }),
    );
    renderDetail();
    expect(
      await screen.findByRole("heading", { name: "履歴を表示できません" }),
    ).toBeInTheDocument();
  });

  it("Recommendation ID変更時に旧Stateを消して再取得する", async () => {
    function RouteChanger() {
      const navigate = useNavigate();
      return (
        <>
          <button type="button" onClick={() => navigate("/recommendation-history/rec-2")}>change</button>
          <RecommendationHistoryDetailPage />
        </>
      );
    }

    getDetailMock
      .mockResolvedValueOnce(detail)
      .mockResolvedValueOnce({
        ...detail,
        recommendation_id: "rec-2",
        recommendations: detail.recommendations.map((item) => ({
          ...item,
          song_name: `${item.song_name} New`,
        })),
      });

    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/recommendation-history/rec-1"]}>
        <Routes>
          <Route path="/recommendation-history/:recommendationId" element={<RouteChanger />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Song 1")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "change" }));
    expect(await screen.findByText("Song 1 New")).toBeInTheDocument();
    expect(screen.queryByText(/^Song 1$/)).not.toBeInTheDocument();
  });
});
