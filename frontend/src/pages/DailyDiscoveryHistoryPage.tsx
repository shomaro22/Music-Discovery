import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { getDailyDiscoveries } from "../api/dailyDiscovery";
import type { DailyDiscoveryList } from "../types/recommendation";

const PAGE_LIMIT = 10;

export function DailyDiscoveryHistoryPage() {
  const [data, setData] = useState<DailyDiscoveryList | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");

    getDailyDiscoveries(page, PAGE_LIMIT)
      .then((response) => {
        if (active) setData(response);
      })
      .catch((caught: unknown) => {
        if (active) {
          setError(
            caught instanceof Error
              ? caught.message
              : "履歴を取得できませんでした",
          );
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [page]);

  return (
    <main className="recommendation-page">
      <section className="recommendation-header">
        <h1>Daily Discovery履歴</h1>
        <Link to="/daily-discovery">今日の3曲へ戻る</Link>
      </section>

      {loading && <p aria-busy="true">履歴を読み込んでいます...</p>}
      {error && <p role="alert">{error}</p>}
      {!loading && data?.items.length === 0 && <p>履歴はまだありません。</p>}

      {!loading &&
        data?.items.map((item) => (
          <article
            className="recommendation-panel"
            key={item.recommendation_id}
          >
            <h2>{item.target_date}</h2>
            <p>
              {item.recommendations.map((song) => song.song_name).join(" / ")}
            </p>
            <Link to={`/daily-discoveries/${item.recommendation_id}`}>
              詳細を見る
            </Link>
          </article>
        ))}

      <div>
        <button
          disabled={loading || page <= 1}
          onClick={() => setPage((value) => value - 1)}
        >
          前へ
        </button>
        <button
          disabled={loading || !data || page >= data.total_pages}
          onClick={() => setPage((value) => value + 1)}
        >
          次へ
        </button>
      </div>
    </main>
  );
}
