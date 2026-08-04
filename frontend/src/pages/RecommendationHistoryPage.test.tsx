import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getRecommendationHistory } from "../api/recommendationHistory";
import { RecommendationHistoryPage } from "./RecommendationHistoryPage";

vi.mock("../api/recommendationHistory", () => ({ getRecommendationHistory: vi.fn() }));
const getHistoryMock = vi.mocked(getRecommendationHistory);

const response = {
  items: [{
    recommendation_id: "rec-1",
    source: "NORMAL" as const,
    recommended_at: "2026-08-04T12:00:00Z",
    songs: [{
      recommendation_detail_id: "detail-1",
      recommendation_type: "Your Match" as const,
      display_order: 1,
      song_id: "song-1",
      song_name: "Song One",
      artist_id: "artist-1",
      artist_name: "Artist One",
      artwork_url: "",
      reason: "好みに一致したため",
      feedback_status: null,
    }],
  }],
  page: 1,
  limit: 10,
  total: 1,
  total_pages: 1,
  timezone: "Asia/Tokyo",
};

describe("RecommendationHistoryPage", () => {
  beforeEach(() => { getHistoryMock.mockResolvedValue(response); });

  it("履歴一覧に推薦種別、理由、Feedbackを表示する", async () => {
    render(<MemoryRouter><RecommendationHistoryPage /></MemoryRouter>);
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();
    expect(await screen.findByText("Song One")).toBeInTheDocument();
    expect(screen.getByText("Your Match")).toBeInTheDocument();
    expect(screen.getByText("好みに一致したため")).toBeInTheDocument();
    expect(screen.getByText("未回答")).toBeInTheDocument();
  });

  it("検索時にPageを1へ戻して条件付きAPIを呼び出す", async () => {
    const user = userEvent.setup();
    render(<MemoryRouter initialEntries={["/recommendation-history?page=3"]}><RecommendationHistoryPage /></MemoryRouter>);
    await screen.findByText("Song One");
    await user.selectOptions(screen.getByLabelText("推薦区分"), "NORMAL");
    await user.click(screen.getByRole("button", { name: "検索" }));
    await waitFor(() => expect(getHistoryMock).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, source: "NORMAL" })));
  });

  it("0件時に空状態と質問回答導線を表示する", async () => {
    getHistoryMock.mockResolvedValue({ ...response, items: [], total: 0, total_pages: 0 });
    render(<MemoryRouter><RecommendationHistoryPage /></MemoryRouter>);
    expect(await screen.findByText("履歴はまだありません")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "質問回答へ進む" })).toBeInTheDocument();
  });
});
