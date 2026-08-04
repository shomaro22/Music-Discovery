import { FormEvent, useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { getRecommendationHistory } from "../api/recommendationHistory";
import { ROUTES } from "../constants/routes";
import type { FeedbackStatus } from "../types/feedback";
import type { RecommendationType } from "../types/recommendation";
import type { RecommendationHistoryItem, RecommendationSource } from "../types/recommendationHistory";

const LIMIT = 10;
const sourceLabel = (source: RecommendationSource) => source === "NORMAL" ? "通常推薦" : "Daily Discovery";
const FEEDBACK_LABELS: Record<FeedbackStatus, string> = {
  NONE: "未回答",
  LIKE: "好き",
  MAYBE: "まあまあ",
  SKIP: "スキップ",
};

const RECOMMENDATION_TYPE_LABELS: Record<RecommendationType, string> = {
  YOUR_MATCH: "Your Match",
  NEW_FAVORITE: "New Favorite",
  TRY_SOMETHING_NEW: "Try Something New",
  DAILY_MATCH_ONE: "Daily Match",
  DAILY_MATCH_TWO: "Daily Match",
  DAILY_CHALLENGE: "Daily Challenge",
};

const feedbackLabel = (status: FeedbackStatus | null): string =>
  FEEDBACK_LABELS[status ?? "NONE"];

export function RecommendationHistoryPage() {
  const [params, setParams] = useSearchParams();
  const page = Math.max(1, Number(params.get("page") ?? "1") || 1);
  const [items, setItems] = useState<RecommendationHistoryItem[]>([]);
  const [totalPages, setTotalPages] = useState(0);
  const [total, setTotal] = useState(0);
  const [timezone, setTimezone] = useState("Asia/Tokyo");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [source, setSource] = useState(params.get("source") ?? "");
  const [startDate, setStartDate] = useState(params.get("startDate") ?? "");
  const [endDate, setEndDate] = useState(params.get("endDate") ?? "");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const response = await getRecommendationHistory({
        page, limit: LIMIT,
        source: (params.get("source") || undefined) as RecommendationSource | undefined,
        startDate: params.get("startDate") || undefined,
        endDate: params.get("endDate") || undefined,
      });
      setItems(response.items); setTotalPages(response.total_pages); setTotal(response.total); setTimezone(response.timezone);
    } catch (caught: unknown) {
      setError(caught instanceof ApiRequestError ? caught.message : "推薦履歴を取得できませんでした");
    } finally { setLoading(false); }
  }, [page, params]);

  useEffect(() => { void load(); }, [load]);

  const search = (event: FormEvent) => {
    event.preventDefault();
    const next = new URLSearchParams();
    next.set("page", "1");
    if (source) next.set("source", source);
    if (startDate) next.set("startDate", startDate);
    if (endDate) next.set("endDate", endDate);
    setParams(next);
  };
  const reset = () => { setSource(""); setStartDate(""); setEndDate(""); setParams({ page: "1" }); };
  const changePage = (nextPage: number) => {
    const next = new URLSearchParams(params); next.set("page", String(nextPage)); setParams(next); window.scrollTo({ top: 0, behavior: "smooth" });
  };

  return <main className="history-page">
    <div className="history-shell">
      <header className="history-header"><div><p className="history-kicker">Recommendation Archive</p><h1>推薦履歴</h1><p>通常推薦とDaily Discoveryをまとめて振り返れます。</p></div><Link className="recommendation-home-button" to={ROUTES.top}>TOPページへ戻る</Link></header>
      <form className="history-filter" onSubmit={search}>
        <label>推薦区分<select value={source} onChange={(e) => setSource(e.target.value)}><option value="">すべて</option><option value="NORMAL">通常推薦</option><option value="DAILY_DISCOVERY">Daily Discovery</option></select></label>
        <label>開始日<input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} /></label>
        <label>終了日<input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} /></label>
        <div className="history-filter-actions"><button type="submit">検索</button><button type="button" onClick={reset}>リセット</button></div>
      </form>
      {loading && <p className="history-state">読み込み中...</p>}
      {!loading && error && <div className="history-state"><p>{error}</p><button onClick={() => void load()}>再取得</button></div>}
      {!loading && !error && items.length === 0 && <div className="history-state"><h2>履歴はまだありません</h2><p>質問に回答して最初の推薦を受け取ってください。</p><Link to={ROUTES.questions}>質問回答へ進む</Link></div>}
      {!loading && !error && items.length > 0 && <><p className="history-count">{total}件の履歴</p><section className="history-list">{items.map((item) => <Link className="history-card" key={item.recommendation_id} to={`/recommendation-history/${item.recommendation_id}?${params.toString()}`}>
        <div className="history-card-heading"><span>{sourceLabel(item.source)}</span><time>{new Intl.DateTimeFormat("ja-JP", { dateStyle: "medium", timeStyle: "short", timeZone: timezone }).format(new Date(item.recommended_at))}</time>{item.target_date && <small>対象日 {item.target_date}</small>}</div>
        <div className="history-song-grid">{item.songs.map((song) => <article key={song.recommendation_detail_id}>{song.artwork_url ? <img src={song.artwork_url} alt="" /> : <div className="history-artwork-placeholder">♪</div>}<div><h2>{song.song_name || "利用できない楽曲"}</h2><p>{song.artist_name || "不明なArtist"}</p><p className="history-recommendation-type">{RECOMMENDATION_TYPE_LABELS[song.recommendation_type]}</p><p className="history-reason">{song.reason}</p><small>{feedbackLabel(song.feedback_status)}</small></div></article>)}</div>
        <span className="history-detail-link">詳細を見る →</span>
      </Link>)}</section>
      <nav className="history-pagination" aria-label="ページ移動"><button disabled={page <= 1} onClick={() => changePage(page - 1)}>前へ</button><span>{page} / {Math.max(totalPages, 1)}</span><button disabled={totalPages === 0 || page >= totalPages} onClick={() => changePage(page + 1)}>次へ</button></nav></>}
    </div>
  </main>;
}
