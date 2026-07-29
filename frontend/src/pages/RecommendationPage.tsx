import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { generateRecommendation, getLatestRecommendation } from "../api/recommendation";
import { RecommendationList } from "../components/recommendation/RecommendationList";
import type { Recommendation } from "../types/recommendation";
interface RecommendationLocationState { generate?: boolean; }
export function RecommendationPage() {
  const location = useLocation(); const navigate = useNavigate(); const started = useRef(false);
  const [recommendation,setRecommendation] = useState<Recommendation|null>(null); const [loading,setLoading] = useState(true); const [error,setError] = useState(""); const [generateMode,setGenerateMode] = useState(false);
  const load = async (generate: boolean): Promise<void> => { setGenerateMode(generate); setLoading(true); setError(""); try { setRecommendation(await (generate ? generateRecommendation() : getLatestRecommendation())); if (generate) navigate(location.pathname,{replace:true,state:null}); } catch (caught: unknown) { if (caught instanceof ApiRequestError && caught.response.code === "QUESTION_ANSWER_REQUIRED") { navigate("/questions",{replace:true}); return; } setError(caught instanceof ApiRequestError ? caught.message : "推薦結果を取得できませんでした"); } finally { setLoading(false); } };
  useEffect(() => { if (started.current) return; started.current=true; const state=location.state as RecommendationLocationState|null; void load(state?.generate === true); }, []);
  if (loading) return <main className="recommendation-page"><section className="recommendation-panel" aria-busy="true"><div className="recommendation-loader"/><h1>あなたの3曲を選んでいます</h1><p>質問への回答と好みから推薦を計算中です。</p></section></main>;
  if (error) return <main className="recommendation-page"><section className="recommendation-panel"><h1>推薦を表示できませんでした</h1><p role="alert">{error}</p><button type="button" className="primary" onClick={() => void load(generateMode)}>再試行</button></section></main>;
  return <main className="recommendation-page"><section className="recommendation-header"><p className="question-code">Music Discovery</p><h1>今日のおすすめ</h1><p>あなたの今と、まだ知らないお気に入りへ。</p></section>{recommendation && <RecommendationList recommendations={recommendation.recommendations}/>}</main>;
}
