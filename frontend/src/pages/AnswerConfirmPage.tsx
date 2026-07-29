import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { getQuestions, saveQuestionAnswers } from "../api/question";
import { useAuth } from "../auth/AuthContext";
import { ROUTES } from "../constants/routes";
import type { Question, QuestionAnswers } from "../types/question";
import { areAllQuestionAnswersValid, clearQuestionAnswers, loadQuestionAnswers } from "../utils/questionAnswers";

interface ConfirmState { questions: Question[]; answers: QuestionAnswers; }

export function AnswerConfirmPage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const locationState = location.state as ConfirmState | null;
  const [confirmState, setConfirmState] = useState<ConfirmState | null>(locationState);
  const [isLoading, setIsLoading] = useState(locationState === null);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (locationState || !user) {
      setIsLoading(false);
      return;
    }

    let active = true;
    const restore = async (): Promise<void> => {
      setIsLoading(true);
      setError("");
      try {
        const response = await getQuestions();
        if (!active) return;
        const questions = [...response.questions].sort((a, b) => a.sort_order - b.sort_order);
        const answers = loadQuestionAnswers(user.id);
        setConfirmState({ questions, answers });
      } catch (caught: unknown) {
        if (active) setError(caught instanceof ApiRequestError ? caught.message : "回答内容を復元できませんでした");
      } finally {
        if (active) setIsLoading(false);
      }
    };
    void restore();
    return () => { active = false; };
  }, [locationState, user]);

  if (isLoading) return <main className="question-page"><p>回答内容を読み込んでいます...</p></main>;
  if (error) return <main className="question-page"><div className="question-card"><p role="alert">{error}</p><button type="button" onClick={() => navigate(ROUTES.questions)}>質問へ戻る</button></div></main>;
  if (!confirmState || !areAllQuestionAnswersValid(confirmState.questions, confirmState.answers)) return <main className="question-page"><div className="question-card"><p role="alert">回答が不足しています。</p><button type="button" onClick={() => navigate(ROUTES.questions)}>質問へ戻る</button></div></main>;

  const save = async (): Promise<void> => {
    if (!user || isSaving) return;
    setIsSaving(true);
    setError("");
    try {
      await saveQuestionAnswers(confirmState.questions.map((question) => ({ question_id: question.id, choice_ids: confirmState.answers[question.id] ?? [] })));
      clearQuestionAnswers(user.id);
      navigate(ROUTES.recommendations, { replace: true });
    } catch (caught: unknown) {
      setError(caught instanceof ApiRequestError ? caught.message : "回答を保存できませんでした");
    } finally {
      setIsSaving(false);
    }
  };

  return <main className="question-page"><section className="question-card confirm-card"><p className="question-code">Confirm</p><h1>回答内容の確認</h1>
    {error && <p role="alert" className="question-error">{error}</p>}
    <div className="answer-summary">{confirmState.questions.map((question) => <article key={question.id}><div><h2>{question.title}</h2><p>{(confirmState.answers[question.id] ?? []).map((id) => question.choices.find((choice) => choice.id === id)?.label).filter(Boolean).join("、")}</p></div><button type="button" onClick={() => navigate(ROUTES.questions, { state: { editQuestionId: question.id, returnToConfirm: true } })}>修正</button></article>)}</div>
    <div className="question-actions"><button type="button" onClick={() => navigate(ROUTES.questions)}>戻る</button><button type="button" className="primary" disabled={isSaving} onClick={() => void save()}>{isSaving ? "保存中..." : "この回答で保存"}</button></div>
  </section></main>;
}
