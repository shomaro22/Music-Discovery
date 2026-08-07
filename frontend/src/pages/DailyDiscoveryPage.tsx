import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { ApiRequestError } from "../api/client";
import { getTodayDailyDiscovery } from "../api/dailyDiscovery";
import { RecommendationList } from "../components/recommendation/RecommendationList";
import { ROUTES } from "../constants/routes";
import type { DailyDiscovery } from "../types/recommendation";

const QUESTION_REQUIRED_CODES = new Set([
  "QUESTION_ANSWER_REQUIRED",
  "USER_PROFILE_REQUIRED",
]);

export function DailyDiscoveryPage() {
  const navigate = useNavigate();
  const [data, setData] = useState<DailyDiscovery | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError("");

    try {
      setData(await getTodayDailyDiscovery());
    } catch (caught: unknown) {
      if (
        caught instanceof ApiRequestError &&
        QUESTION_REQUIRED_CODES.has(caught.response.code)
      ) {
        navigate(ROUTES.questions, { replace: true });
        return;
      }

      if (
        caught instanceof ApiRequestError &&
        caught.response.code === "DAILY_DISCOVERY_CANDIDATES_NOT_FOUND"
      ) {
        setError(
          "今日の推薦候補が不足しています。時間を置いて再度お試しください。",
        );
        return;
      }

      setError(
        caught instanceof ApiRequestError
          ? caught.message
          : "Daily Discoveryを取得できませんでした",
      );
    } finally {
      setLoading(false);
    }
  }, [navigate]);

  useEffect(() => {
    void load();
  }, [load]);

  const topButton = (
    <button
      type="button"
      className="recommendation-home-button"
      onClick={() => navigate(ROUTES.top)}
    >
      TOPページへ戻る
    </button>
  );

  if (loading) {
    return (
      <main className="recommendation-page">
        {topButton}
        <section className="recommendation-panel" aria-busy="true">
          <div className="recommendation-loader" />
          <h1>今日の3曲を選んでいます</h1>
        </section>
      </main>
    );
  }

  if (error) {
    return (
      <main className="recommendation-page">
        {topButton}
        <section className="recommendation-panel">
          <h1>Daily Discovery</h1>
          <p role="alert">{error}</p>
          <button className="primary" type="button" onClick={() => void load()}>
            再取得
          </button>
        </section>
      </main>
    );
  }

  return (
    <main className="recommendation-page">
      {topButton}
      <section className="recommendation-header">
        <p className="question-code">Daily Discovery</p>
        <h1>今日の3曲</h1>
        <p>{data?.target_date}</p>
        <Link className="recommendation-history-link" to={ROUTES.dailyDiscoveryHistory}>過去のDaily Discovery</Link>
      </section>

      {data && <RecommendationList recommendations={data.recommendations} />}
    </main>
  );
}
