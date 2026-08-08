import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { getDailyDiscoveries } from "../api/dailyDiscovery";
import { ROUTES } from "../constants/routes";
import type { DailyDiscoveryList } from "../types/recommendation";

const PAGE_LIMIT = 10;

export function DailyDiscoveryHistoryPage() {
  const [data, setData] = useState<DailyDiscoveryList | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true); setError("");
    getDailyDiscoveries(page, PAGE_LIMIT)
      .then((response) => { if (active) setData(response); })
      .catch((caught: unknown) => { if (active) setError(caught instanceof Error ? caught.message : "履歴を取得できませんでした"); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [page]);

  return (
    <main className="history-page">
      <div className="history-shell">
        <Link className="recommendation-home-button" to={ROUTES.top}>TOPページへ戻る</Link>
        <header className="history-header">
          <div><p className="history-kicker">Daily Discovery Archive</p><h1>Daily Discovery履歴</h1><p>これまでの「今日の3曲」をジャケットと一緒に振り返れます。</p></div>
        </header>
        <Link className="history-secondary-link" to={ROUTES.dailyDiscovery}>今日の3曲へ戻る</Link>

        {loading && <p className="history-state" aria-busy="true">履歴を読み込んでいます...</p>}
        {!loading && error && <p className="history-state" role="alert">{error}</p>}
        {!loading && !error && data?.items.length === 0 && <div className="history-state"><h2>履歴はまだありません。</h2><p>Daily Discoveryを利用すると、ここに過去の3曲が表示されます。</p></div>}

        {!loading && !error && data && data.items.length > 0 && <>
          <p className="history-count">{data.total}件の履歴</p>
          <section className="history-list">
            {data.items.map((item) => (
              <Link className="history-card" key={item.recommendation_id} to={`/daily-discoveries/${item.recommendation_id}`}>
                <div className="history-card-heading"><span>Daily Discovery</span><time>{item.target_date}</time></div>
                <div className="history-song-grid">
                  {item.recommendations.map((song) => (
                    <article key={song.recommendation_detail_id}>
                      {song.artwork_url ? <img src={song.artwork_url} alt="" /> : <div className="history-artwork-placeholder">♪</div>}
                      <div><h2>{song.song_name}</h2><p>{song.artist_name}</p><p className="history-recommendation-type">{song.recommendation_type === "DAILY_CHALLENGE" ? "Daily Challenge" : "Daily Match"}</p><p className="history-reason">{song.reason}</p></div>
                    </article>
                  ))}
                </div>
                <span className="history-detail-link">詳細を見る →</span>
              </Link>
            ))}
          </section>
          <nav className="history-pagination" aria-label="Daily Discoveryページ移動">
            <button type="button" disabled={loading || page <= 1} onClick={() => setPage((value) => value - 1)}>前へ</button>
            <span>{page} / {Math.max(data.total_pages, 1)}</span>
            <button type="button" disabled={loading || page >= data.total_pages} onClick={() => setPage((value) => value + 1)}>次へ</button>
          </nav>
        </>}
      </div>
    </main>
  );
}
