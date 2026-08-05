import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";
import { getRecommendationHistory } from "../api/recommendationHistory";
import type { RecommendationHistoryListResponse } from "../types/recommendationHistory";
import { RecommendationHistoryPage } from "./RecommendationHistoryPage";

vi.mock("../api/recommendationHistory", () => ({
  getRecommendationHistory: vi.fn(),
}));
const getHistoryMock = vi.mocked(getRecommendationHistory);

const response: RecommendationHistoryListResponse = {
  items: [
    {
      recommendation_id: "normal-1",
      source: "NORMAL",
      recommended_at: "2026-08-04T12:00:00Z",
      songs: [
        {
          recommendation_detail_id: "normal-detail-1",
          recommendation_type: "YOUR_MATCH",
          display_order: 1,
          song_id: "normal-song-1",
          song_name: "Normal Song One",
          artist_id: "normal-artist-1",
          artist_name: "Normal Artist One",
          artwork_url: "https://example.com/normal-one.jpg",
          reason: "好みに一致したため",
          feedback_status: null,
        },
        {
          recommendation_detail_id: "normal-detail-2",
          recommendation_type: "NEW_FAVORITE",
          display_order: 2,
          song_id: "normal-song-2",
          song_name: "Normal Song Two",
          artist_id: "normal-artist-2",
          artist_name: "Normal Artist Two",
          artwork_url: "https://example.com/normal-two.jpg",
          reason: "新しいお気に入り候補です",
          feedback_status: "LIKE",
        },
        {
          recommendation_detail_id: "normal-detail-3",
          recommendation_type: "TRY_SOMETHING_NEW",
          display_order: 3,
          song_id: "normal-song-3",
          song_name: "Normal Song Three",
          artist_id: "normal-artist-3",
          artist_name: "Normal Artist Three",
          artwork_url: "",
          reason: "少し新しい方向性です",
          feedback_status: "MAYBE",
        },
      ],
    },
    {
      recommendation_id: "daily-1",
      source: "DAILY_DISCOVERY",
      target_date: "2026-08-05",
      recommended_at: "2026-08-05T00:30:00Z",
      songs: [
        {
          recommendation_detail_id: "daily-detail-1",
          recommendation_type: "DAILY_MATCH_ONE",
          display_order: 1,
          song_id: "daily-song-1",
          song_name: "Daily Song One",
          artist_id: "daily-artist-1",
          artist_name: "Daily Artist One",
          artwork_url: "https://example.com/daily-one.jpg",
          reason: "今日の気分に合います",
          feedback_status: "SKIP",
        },
        {
          recommendation_detail_id: "daily-detail-2",
          recommendation_type: "DAILY_MATCH_TWO",
          display_order: 2,
          song_id: "daily-song-2",
          song_name: "Daily Song Two",
          artist_id: "daily-artist-2",
          artist_name: "Daily Artist Two",
          artwork_url: "",
          reason: "継続して聴きやすい曲です",
          feedback_status: null,
        },
        {
          recommendation_detail_id: "daily-detail-3",
          recommendation_type: "DAILY_CHALLENGE",
          display_order: 3,
          song_id: "daily-song-3",
          song_name: "Daily Song Three",
          artist_id: "daily-artist-3",
          artist_name: "Daily Artist Three",
          artwork_url: "https://example.com/daily-three.jpg",
          reason: "今日の挑戦枠です",
          feedback_status: "LIKE",
        },
      ],
    },
  ],
  page: 1,
  limit: 10,
  total: 12,
  total_pages: 2,
  timezone: "Asia/Tokyo",
};

function renderPage(path = "/recommendation-history") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/recommendation-history"
          element={<RecommendationHistoryPage />}
        />
        <Route path="/questions" element={<p>questions page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("RecommendationHistoryPage", () => {
  beforeEach(() => {
    getHistoryMock.mockReset();
    getHistoryMock.mockResolvedValue(response);
    vi.stubGlobal("scrollTo", vi.fn());
  });

  it("通常推薦とDaily Discoveryを表示し各履歴Cardに3曲を表示する", async () => {
    renderPage();
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();
    expect(await screen.findByText("Normal Song One")).toBeInTheDocument();

    const cards = screen.getAllByRole("link", { name: /詳細を見る/ });
    expect(cards).toHaveLength(2);
    expect(within(cards[0]).getAllByRole("article")).toHaveLength(3);
    expect(within(cards[1]).getAllByRole("article")).toHaveLength(3);
    expect(within(cards[0]).getByText("通常推薦")).toBeInTheDocument();
    expect(within(cards[1]).getByText("Daily Discovery")).toBeInTheDocument();
    expect(screen.getByText("対象日 2026-08-05")).toBeInTheDocument();
  });

  it("推薦枠、理由、Feedback、Artworkと代替表示を表示する", async () => {
    renderPage();
    await screen.findByText("Normal Song One");

    expect(screen.getByText("Your Match")).toBeInTheDocument();
    expect(screen.getByText("New Favorite")).toBeInTheDocument();
    expect(screen.getByText("Try Something New")).toBeInTheDocument();
    expect(screen.getAllByText("Daily Match")).toHaveLength(2);
    expect(screen.getByText("Daily Challenge")).toBeInTheDocument();
    expect(screen.getByText("好みに一致したため")).toBeInTheDocument();
    expect(screen.getAllByText("未回答")).toHaveLength(2);
    expect(screen.getAllByText("好き")).toHaveLength(2);
    expect(screen.getByText("スキップ")).toBeInTheDocument();
    expect(document.querySelector('img[src="https://example.com/normal-one.jpg"]')).toBeInTheDocument();
    expect(screen.getAllByText("♪")).toHaveLength(2);
  });

  it("URL Queryから絞り込み条件とPageを復元する", async () => {
    renderPage(
      "/recommendation-history?page=2&source=DAILY_DISCOVERY&startDate=2026-08-01&endDate=2026-08-05",
    );
    await screen.findByText("Normal Song One");

    expect(screen.getByLabelText("推薦区分")).toHaveValue("DAILY_DISCOVERY");
    expect(screen.getByLabelText("開始日")).toHaveValue("2026-08-01");
    expect(screen.getByLabelText("終了日")).toHaveValue("2026-08-05");
    expect(getHistoryMock).toHaveBeenCalledWith({
      page: 2,
      limit: 10,
      source: "DAILY_DISCOVERY",
      startDate: "2026-08-01",
      endDate: "2026-08-05",
    });
  });

  it("Source・開始日・終了日検索時にPageを1へ戻す", async () => {
    const user = userEvent.setup();
    renderPage("/recommendation-history?page=3");
    await screen.findByText("Normal Song One");

    await user.selectOptions(screen.getByLabelText("推薦区分"), "NORMAL");
    await user.type(screen.getByLabelText("開始日"), "2026-08-01");
    await user.type(screen.getByLabelText("終了日"), "2026-08-05");
    await user.click(screen.getByRole("button", { name: "検索" }));

    await waitFor(() =>
      expect(getHistoryMock).toHaveBeenLastCalledWith({
        page: 1,
        limit: 10,
        source: "NORMAL",
        startDate: "2026-08-01",
        endDate: "2026-08-05",
      }),
    );
  });

  it("Filterリセット時に全条件を解除しPage 1を取得する", async () => {
    const user = userEvent.setup();
    renderPage(
      "/recommendation-history?page=2&source=NORMAL&startDate=2026-08-01&endDate=2026-08-05",
    );
    await screen.findByText("Normal Song One");
    await user.click(screen.getByRole("button", { name: "リセット" }));

    await waitFor(() =>
      expect(getHistoryMock).toHaveBeenLastCalledWith({
        page: 1,
        limit: 10,
        source: undefined,
        startDate: undefined,
        endDate: undefined,
      }),
    );
    expect(screen.getByLabelText("推薦区分")).toHaveValue("");
    expect(screen.getByLabelText("開始日")).toHaveValue("");
    expect(screen.getByLabelText("終了日")).toHaveValue("");
  });

  it("前・次Page操作でFilter条件を維持する", async () => {
    const user = userEvent.setup();
    getHistoryMock.mockResolvedValue({ ...response, page: 2, total_pages: 3 });
    renderPage("/recommendation-history?page=2&source=NORMAL&startDate=2026-08-01");
    await screen.findByText("Normal Song One");

    await user.click(screen.getByRole("button", { name: "次へ" }));
    await waitFor(() =>
      expect(getHistoryMock).toHaveBeenLastCalledWith(
        expect.objectContaining({
          page: 3,
          source: "NORMAL",
          startDate: "2026-08-01",
        }),
      ),
    );
    expect(window.scrollTo).toHaveBeenCalled();
  });

  it("API Error後に再取得できる", async () => {
    const user = userEvent.setup();
    getHistoryMock
      .mockRejectedValueOnce(
        new ApiRequestError({
          status: 500,
          code: "RECOMMENDATION_HISTORY_FETCH_FAILED",
          message: "履歴取得失敗",
          request_id: "request-id",
        }),
      )
      .mockResolvedValueOnce(response);
    renderPage();

    expect(await screen.findByText("履歴取得失敗")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "再取得" }));
    expect(await screen.findByText("Normal Song One")).toBeInTheDocument();
    expect(getHistoryMock).toHaveBeenCalledTimes(2);
  });

  it("APIのTimezoneを使って推薦日時を表示する", async () => {
    getHistoryMock.mockResolvedValue({ ...response, timezone: "America/New_York" });
    renderPage();
    await screen.findByText("Normal Song One");

    const expected = new Intl.DateTimeFormat("ja-JP", {
      dateStyle: "medium",
      timeStyle: "short",
      timeZone: "America/New_York",
    }).format(new Date("2026-08-04T12:00:00Z"));
    expect(screen.getByText(expected)).toBeInTheDocument();
  });

  it("0件時に空状態と質問回答導線を表示する", async () => {
    const user = userEvent.setup();
    getHistoryMock.mockResolvedValue({
      ...response,
      items: [],
      total: 0,
      total_pages: 0,
    });
    renderPage();
    expect(await screen.findByText("履歴はまだありません")).toBeInTheDocument();
    await user.click(screen.getByRole("link", { name: "質問回答へ進む" }));
    expect(screen.getByText("questions page")).toBeInTheDocument();
  });
});
