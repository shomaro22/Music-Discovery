import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { getLatestQuestionAnswers } from "../api/question";

export function RecommendationPlaceholderPage() {
  const navigate = useNavigate();
  const [isChecking, setIsChecking] = useState(true);

  useEffect(() => {
    let active = true;
    const checkAnswers = async (): Promise<void> => {
      try {
        const latest = await getLatestQuestionAnswers();
        if (active && latest.answers.length !== 5) navigate("/questions", { replace: true });
      } catch {
        if (active) navigate("/questions", { replace: true });
      } finally {
        if (active) setIsChecking(false);
      }
    };
    void checkAnswers();
    return () => { active = false; };
  }, [navigate]);

  if (isChecking) return <main className="question-page"><p>回答状況を確認しています...</p></main>;

  return <main className="question-page"><section className="question-card"><p className="question-code">Saved</p><h1>回答を保存しました</h1><p>第10章の楽曲推薦機能で、この回答を使って3曲を生成します。</p><button type="button" className="primary" onClick={() => navigate("/questions", { state: { loadLatest: true } })}>回答を変更する</button></section></main>;
}
