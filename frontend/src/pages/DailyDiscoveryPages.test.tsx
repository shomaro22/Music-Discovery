import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";
import { getDailyDiscoveries, getDailyDiscovery, getTodayDailyDiscovery } from "../api/dailyDiscovery";
import { DailyDiscoveryPage } from "./DailyDiscoveryPage";
import { DailyDiscoveryHistoryPage } from "./DailyDiscoveryHistoryPage";
import { DailyDiscoveryDetailPage } from "./DailyDiscoveryDetailPage";

vi.mock("../api/dailyDiscovery", () => ({
  getTodayDailyDiscovery: vi.fn(),
  getDailyDiscoveries: vi.fn(),
  getDailyDiscovery: vi.fn(),
}));
vi.mock("../components/recommendation/RecommendationList", () => ({
  RecommendationList: ({ recommendations }: { recommendations: Array<{ song_name: string }> }) => (
    <div data-testid="recommendations">{recommendations.map((item) => item.song_name).join(",")}</div>
  ),
}));

const todayMock = vi.mocked(getTodayDailyDiscovery);
const listMock = vi.mocked(getDailyDiscoveries);
const detailMock = vi.mocked(getDailyDiscovery);

const recommendation = {
  recommendation_detail_id: "detail-1",
  recommendation_type: "DAILY_MATCH_ONE" as const,
  song_id: "song-1",
  song_name: "Daily Song",
  artist_id: "artist-1",
  artist_name: "Artist",
  artwork_url: "",
  spotify_url: "https://open.spotify.com/track/1",
  reason: "reason",
  feedback_status: null,
};
const daily = {
  recommendation_id: "daily-1",
  target_date: "2026-08-07",
  generated: true,
  recommended_at: "2026-08-07T00:00:00Z",
  recommendations: [recommendation],
};

beforeEach(() => {
  todayMock.mockReset();
  listMock.mockReset();
  detailMock.mockReset();
});

describe("Daily Discovery pages", () => {
  it("loads today's Daily Discovery and returns to top", async () => {
    todayMock.mockResolvedValue(daily);
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/daily-discovery"]}>
        <Routes>
          <Route path="/daily-discovery" element={<DailyDiscoveryPage />} />
          <Route path="/" element={<p>top page</p>} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("今日の3曲を選んでいます")).toBeInTheDocument();
    expect(await screen.findByText("2026-08-07")).toBeInTheDocument();
    expect(screen.getByTestId("recommendations")).toHaveTextContent("Daily Song");
    await user.click(screen.getByRole("button", { name: "TOPページへ戻る" }));
    expect(screen.getByText("top page")).toBeInTheDocument();
  });

  it("redirects to questions when a profile is required", async () => {
    todayMock.mockRejectedValue(new ApiRequestError({ status: 422, code: "QUESTION_ANSWER_REQUIRED", message: "answer required", request_id: "req-1" }));
    render(
      <MemoryRouter initialEntries={["/daily-discovery"]}>
        <Routes>
          <Route path="/daily-discovery" element={<DailyDiscoveryPage />} />
          <Route path="/questions" element={<p>questions page</p>} />
        </Routes>
      </MemoryRouter>,
    );
    expect(await screen.findByText("questions page")).toBeInTheDocument();
  });

  it("shows candidate shortage and retries", async () => {
    todayMock
      .mockRejectedValueOnce(new ApiRequestError({ status: 422, code: "DAILY_DISCOVERY_CANDIDATES_NOT_FOUND", message: "none", request_id: "req-2" }))
      .mockResolvedValueOnce(daily);
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/daily-discovery"]}>
        <Routes><Route path="/daily-discovery" element={<DailyDiscoveryPage />} /></Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("今日の推薦候補が不足しています。時間を置いて再度お試しください。")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "再取得" }));
    expect(await screen.findByText("2026-08-07")).toBeInTheDocument();
    expect(todayMock).toHaveBeenCalledTimes(2);
  });

  it("paginates Daily Discovery history", async () => {
    listMock
      .mockResolvedValueOnce({ items: [daily], page: 1, limit: 10, total: 2, total_pages: 2 })
      .mockResolvedValueOnce({ items: [], page: 2, limit: 10, total: 2, total_pages: 2 });
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/daily-discoveries"]}>
        <Routes><Route path="/daily-discoveries" element={<DailyDiscoveryHistoryPage />} /></Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Daily Song")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "次へ" }));
    await waitFor(() => expect(listMock).toHaveBeenLastCalledWith(2, 10));
    expect(await screen.findByText("履歴はまだありません。")).toBeInTheDocument();
  });

  it("renders detail and access denied errors", async () => {
    detailMock.mockResolvedValueOnce(daily);
    const first = render(
      <MemoryRouter initialEntries={["/daily-discoveries/daily-1"]}>
        <Routes><Route path="/daily-discoveries/:recommendationId" element={<DailyDiscoveryDetailPage />} /></Routes>
      </MemoryRouter>,
    );
    expect(await screen.findByText("2026-08-07")).toBeInTheDocument();
    expect(detailMock).toHaveBeenCalledWith("daily-1");
    first.unmount();

    detailMock.mockRejectedValueOnce(new ApiRequestError({ status: 403, code: "DAILY_DISCOVERY_ACCESS_DENIED", message: "denied", request_id: "req-3" }));
    render(
      <MemoryRouter initialEntries={["/daily-discoveries/daily-2"]}>
        <Routes><Route path="/daily-discoveries/:recommendationId" element={<DailyDiscoveryDetailPage />} /></Routes>
      </MemoryRouter>,
    );
    expect(await screen.findByText("このDaily Discoveryを表示する権限がありません。")).toBeInTheDocument();
  });
});
