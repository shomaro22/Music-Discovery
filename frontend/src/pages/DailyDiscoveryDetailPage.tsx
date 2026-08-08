import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { getDailyDiscovery } from "../api/dailyDiscovery";
import { RecommendationList } from "../components/recommendation/RecommendationList";
import type { DailyDiscovery } from "../types/recommendation";

export function DailyDiscoveryDetailPage() {
  const { recommendationId = "" } = useParams<{
    recommendationId: string;
  }>();
  const [data, setData] = useState<DailyDiscovery | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");

    getDailyDiscovery(recommendationId)
      .then((response) => {
        if (active) setData(response);
      })
      .catch((caught: unknown) => {
        if (!active) return;
        if (
          caught instanceof ApiRequestError &&
          caught.response.code === "DAILY_DISCOVERY_ACCESS_DENIED"
        ) {
          setError("このDaily Discoveryを表示する権限がありません。");
          return;
        }
        setError(
          caught instanceof Error
            ? caught.message
            : "詳細を取得できませんでした",
        );
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [recommendationId]);

  return (
    <main className="recommendation-page">
      <section className="recommendation-header">
        <h1>Daily Discovery詳細</h1>
        <p>{data?.target_date}</p>
        <Link to="/daily-discoveries">履歴へ戻る</Link>
      </section>
      {loading && <p aria-busy="true">詳細を読み込んでいます...</p>}
      {error && <p role="alert">{error}</p>}
      {!loading && data && (
        <RecommendationList recommendations={data.recommendations} />
      )}
    </main>
  );
}
