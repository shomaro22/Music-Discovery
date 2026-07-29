import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { saveQuestionAnswers } from "../api/question";
import { useAuth } from "../auth/AuthContext";
import type { Question, QuestionAnswers } from "../types/question";
import { areAllQuestionAnswersValid, clearQuestionAnswers } from "../utils/questionAnswers";

interface ConfirmState { questions: Question[]; answers: QuestionAnswers; }

export function AnswerConfirmPage() {
  const { user } = useAuth(); const navigate = useNavigate(); const location = useLocation();
  const state = location.state as ConfirmState | null;
  const [isSaving, setIsSaving] = useState(false); const [error, setError] = useState("");
  if (!state || !areAllQuestionAnswersValid(state.questions, state.answers)) return <main className="question-page"><div className="question-card"><p role="alert">回答が不足しています。</p><button type="button" onClick={() => navigate("/questions")}>質問へ戻る</button></div></main>;
  const save = async (): Promise<void> => {
    if (!user || isSaving) return;
    setIsSaving(true); setError("");
    try {
      await saveQuestionAnswers(state.questions.map((question) => ({ question_id: question.id, choice_ids: state.answers[question.id] ?? [] })));
      clearQuestionAnswers(user.id); navigate("/recommendations", { replace: true, state: { generate: true } });
    } catch (caught: unknown) { setError(caught instanceof ApiRequestError ? caught.message : "回答を保存できませんでした"); }
    finally { setIsSaving(false); }
  };
  return <main className="question-page"><section className="question-card confirm-card"><p className="question-code">Confirm</p><h1>回答内容の確認</h1>
    {error && <p role="alert" className="question-error">{error}</p>}
    <div className="answer-summary">{state.questions.map((question) => <article key={question.id}><div><h2>{question.title}</h2><p>{(state.answers[question.id] ?? []).map((id) => question.choices.find((choice) => choice.id === id)?.label).filter(Boolean).join("、")}</p></div><button type="button" onClick={() => navigate("/questions", { state: { editQuestionId: question.id, returnToConfirm: true } })}>修正</button></article>)}</div>
    <div className="question-actions"><button type="button" onClick={() => navigate("/questions")}>戻る</button><button type="button" className="primary" disabled={isSaving} onClick={() => void save()}>{isSaving ? "保存中..." : "この回答で保存"}</button></div>
  </section></main>;
}
