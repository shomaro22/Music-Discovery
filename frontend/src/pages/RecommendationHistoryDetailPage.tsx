import { useEffect, useState } from "react";
import { Link, useLocation, useParams } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { getRecommendationHistoryDetail } from "../api/recommendationHistory";
import { RecommendationCard } from "../components/recommendation/RecommendationCard";
import { ROUTES } from "../constants/routes";
import type { RecommendationHistoryDetailResponse } from "../types/recommendationHistory";

function historyErrorTitle(errorCode: string): string {
  switch (errorCode) {
    case "RECOMMENDATION_HISTORY_NOT_FOUND":
      return "推薦履歴が見つかりません";
    case "RECOMMENDATION_HISTORY_ACCESS_DENIED":
      return "この推薦履歴にはアクセスできません";
    default:
      return "履歴を表示できません";
  }
}

export function RecommendationHistoryDetailPage() {
  const { recommendationId = "" } = useParams();
  const location = useLocation();
  const [value, setValue] =
    useState<RecommendationHistoryDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [errorCode, setErrorCode] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    setErrorCode("");
    setValue(null);

    getRecommendationHistoryDetail(recommendationId)
      .then((result) => {
        if (active) setValue(result);
      })
      .catch((caught: unknown) => {
        if (!active) return;
        if (caught instanceof ApiRequestError) {
          setErrorCode(caught.response.code);
          setError(caught.message);
          return;
        }
        setError("推薦履歴を取得できませんでした");
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [recommendationId]);

  const query = location.search;

  return (
    <main className="history-page">
      <div className="history-shell">
        <Link
          className="recommendation-home-button"
          to={`${ROUTES.recommendationHistory}${query}`}
        >
          推薦履歴へ戻る
        </Link>

        {loading && <p className="history-state">読み込み中...</p>}

        {!loading && error && (
          <div className="history-state" role="alert">
            <h1>{historyErrorTitle(errorCode)}</h1>
            <p>{error}</p>
          </div>
        )}

        {!loading && value && (
          <>
            <header className="history-header">
              <div>
                <p className="history-kicker">
                  {value.source === "NORMAL" ? "通常推薦" : "Daily Discovery"}
                </p>
                <h1>推薦履歴の詳細</h1>
                <p>
                  {new Intl.DateTimeFormat("ja-JP", {
                    dateStyle: "medium",
                    timeStyle: "short",
                    timeZone: value.timezone,
                  }).format(new Date(value.recommended_at))}
                  {value.target_date ? ` / 対象日 ${value.target_date}` : ""}
                </p>
              </div>
            </header>
            <section className="recommendation-list">
              {value.recommendations.map((detail) => (
                <RecommendationCard
                  key={detail.recommendation_detail_id}
                  detail={detail}
                />
              ))}
            </section>
          </>
        )}
      </div>
    </main>
  );
}
