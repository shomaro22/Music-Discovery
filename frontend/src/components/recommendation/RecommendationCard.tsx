import { useState } from "react";
import { ApiRequestError } from "../../api/client";
import { saveFeedback } from "../../api/feedback";
import type { FeedbackStatus, FeedbackType } from "../../types/feedback";
import type {
  RecommendationDetail,
  RecommendationType,
} from "../../types/recommendation";
import { FeedbackButtons } from "./FeedbackButtons";

const LABELS: Record<RecommendationType, string> = {
  YOUR_MATCH: "Your Match",
  NEW_FAVORITE: "New Favorite",
  TRY_SOMETHING_NEW: "Try Something New",
  DAILY_MATCH_ONE: "Daily Match",
  DAILY_MATCH_TWO: "Daily Match",
  DAILY_CHALLENGE: "Daily Challenge",
};

interface RecommendationCardProps {
  detail: RecommendationDetail;
}

export function RecommendationCard({ detail }: RecommendationCardProps) {
  const initialStatus: FeedbackStatus = detail.feedback_status ?? "NONE";
  const [feedbackStatus, setFeedbackStatus] =
    useState<FeedbackStatus>(initialStatus);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const selectFeedback = async (feedbackType: FeedbackType): Promise<void> => {
    if (saving || feedbackStatus === feedbackType) return;

    const previous = feedbackStatus;
    setSaving(true);
    setError("");

    try {
      const response = await saveFeedback(
        detail.recommendation_detail_id,
        feedbackType,
      );
      setFeedbackStatus(response.feedback_type);
    } catch (caught: unknown) {
      setFeedbackStatus(previous);
      setError(
        caught instanceof ApiRequestError
          ? caught.message
          : "Feedbackを保存できませんでした",
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <article className="recommendation-card">
      <p className="recommendation-type">
        {LABELS[detail.recommendation_type]}
      </p>
      {detail.artwork_url ? (
        <img
          src={detail.artwork_url}
          alt={`${detail.song_name} artwork`}
        />
      ) : (
        <div
          className="recommendation-artwork-placeholder"
          aria-label="artworkなし"
        >
          ♪
        </div>
      )}
      <div className="recommendation-card-body">
        <h2>{detail.song_name}</h2>
        <p className="recommendation-artist">{detail.artist_name}</p>
        <p>{detail.reason}</p>
        <a
          className="spotify-button"
          href={detail.spotify_url}
          target="_blank"
          rel="noreferrer"
        >
          Spotifyで聴く
        </a>
        <FeedbackButtons
          current={feedbackStatus}
          disabled={saving}
          error={error}
          onSelect={(value) => void selectFeedback(value)}
        />
      </div>
    </article>
  );
}
