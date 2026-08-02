import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { saveFeedback } from "../../api/feedback";
import type { RecommendationDetail } from "../../types/recommendation";
import { RecommendationCard } from "./RecommendationCard";

vi.mock("../../api/feedback", () => ({ saveFeedback: vi.fn() }));
const saveFeedbackMock = vi.mocked(saveFeedback);

const detail: RecommendationDetail = {
  recommendation_detail_id: "detail-1",
  recommendation_type: "YOUR_MATCH",
  song_id: "song-1",
  song_name: "Song",
  artist_id: "artist-1",
  artist_name: "Artist",
  artwork_url: "",
  spotify_url: "https://open.spotify.com/track/1",
  reason: "reason",
  feedback_status: null,
};

beforeEach(() => vi.clearAllMocks());

describe("RecommendationCard Feedback", () => {
  it("LIKE、MAYBE、SKIPを表示し、選択値を送信する", async () => {
    saveFeedbackMock.mockResolvedValue({ feedback_id: "f1", recommendation_detail_id: "detail-1", feedback_type: "LIKE", created: true, updated_at: "2026-07-31T00:00:00Z" });
    render(<RecommendationCard detail={detail} />);
    expect(screen.getByRole("button", { name: /好き LIKE/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /もう少し聴きたい MAYBE/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /今回は違う SKIP/ })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /好き LIKE/ }));
    await waitFor(() => expect(saveFeedbackMock).toHaveBeenCalledWith("detail-1", "LIKE"));
    expect(screen.getByRole("button", { name: /好き LIKE/ })).toHaveAttribute("aria-pressed", "true");
  });

  it("通信中は対象Cardだけを無効化し、同じFeedbackは再送信しない", async () => {
    let resolveRequest: ((value: Awaited<ReturnType<typeof saveFeedback>>) => void) | undefined;
    saveFeedbackMock.mockImplementation(() => new Promise((resolve) => { resolveRequest = resolve; }));
    render(<RecommendationCard detail={detail} />);
    fireEvent.click(screen.getByRole("button", { name: /好き LIKE/ }));
    expect(screen.getByRole("group", { name: "この推薦はどうでしたか？" })).toBeDisabled();
    resolveRequest?.({ feedback_id: "f1", recommendation_detail_id: "detail-1", feedback_type: "LIKE", created: true, updated_at: "2026-07-31T00:00:00Z" });
    await waitFor(() => expect(screen.getByRole("button", { name: /好き LIKE/ })).toHaveAttribute("aria-pressed", "true"));
    fireEvent.click(screen.getByRole("button", { name: /好き LIKE/ }));
    expect(saveFeedbackMock).toHaveBeenCalledTimes(1);
  });

  it("変更成功時に選択状態を更新する", async () => {
    saveFeedbackMock.mockResolvedValue({ feedback_id: "f1", recommendation_detail_id: "detail-1", feedback_type: "SKIP", created: false, updated_at: "2026-07-31T00:00:00Z" });
    render(<RecommendationCard detail={{ ...detail, feedback_status: "LIKE" }} />);
    fireEvent.click(screen.getByRole("button", { name: /今回は違う SKIP/ }));
    await waitFor(() => expect(screen.getByRole("button", { name: /今回は違う SKIP/ })).toHaveAttribute("aria-pressed", "true"));
    expect(screen.getByRole("button", { name: /好き LIKE/ })).toHaveAttribute("aria-pressed", "false");
  });

  it("保存失敗時は変更前の状態を維持し、再試行できる", async () => {
    saveFeedbackMock.mockRejectedValueOnce(new Error("failed")).mockResolvedValueOnce({ feedback_id: "f1", recommendation_detail_id: "detail-1", feedback_type: "MAYBE", created: false, updated_at: "2026-07-31T00:00:00Z" });
    render(<RecommendationCard detail={{ ...detail, feedback_status: "LIKE" }} />);
    fireEvent.click(screen.getByRole("button", { name: /もう少し聴きたい MAYBE/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Feedbackを保存できませんでした");
    expect(screen.getByRole("button", { name: /好き LIKE/ })).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: /もう少し聴きたい MAYBE/ }));
    await waitFor(() => expect(saveFeedbackMock).toHaveBeenCalledTimes(2));
  });
  it("Daily Discoveryの推薦種別ラベルを表示する", () => {
    render(<RecommendationCard detail={{ ...detail, recommendation_type: "DAILY_CHALLENGE" }} />);
    expect(screen.getByText("Daily Challenge")).toBeInTheDocument();
  });

});
