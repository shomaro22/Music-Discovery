import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { ApiRequestError } from "../api/client";
import { getLatestQuestionAnswers, getQuestions } from "../api/question";
import { useAuth } from "../auth/AuthContext";
import { ChoiceList } from "../components/question/ChoiceList";
import { QuestionProgress } from "../components/question/QuestionProgress";
import type { Question, QuestionAnswers } from "../types/question";
import { isQuestionAnswerValid, loadQuestionAnswers, saveQuestionAnswersToSession } from "../utils/questionAnswers";

interface QuestionLocationState { editQuestionId?: string; loadLatest?: boolean; returnToConfirm?: boolean; }

export function QuestionPage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const state = location.state as QuestionLocationState | null;
  const [questions, setQuestions] = useState<Question[]>([]);
  const [answers, setAnswers] = useState<QuestionAnswers>({});
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  const load = async (): Promise<void> => {
    if (!user) return;
    setIsLoading(true); setError("");
    try {
      const questionResponse = await getQuestions();
      const sorted = [...questionResponse.questions].sort((a, b) => a.sort_order - b.sort_order);
      let restored = loadQuestionAnswers(user.id);
      if (state?.loadLatest && Object.keys(restored).length === 0) {
        const latest = await getLatestQuestionAnswers();
        restored = Object.fromEntries(latest.answers.map((answer) => [answer.question_id, answer.choice_ids]));
      }
      setQuestions(sorted); setAnswers(restored);
      const targetIndex = state?.editQuestionId ? sorted.findIndex((question) => question.id === state.editQuestionId) : 0;
      setCurrentIndex(targetIndex >= 0 ? targetIndex : 0);
    } catch (caught: unknown) { setError(caught instanceof ApiRequestError ? caught.message : "質問を取得できませんでした"); }
    finally { setIsLoading(false); }
  };

  useEffect(() => { void load(); }, [user?.id]);
  useEffect(() => { if (user && !isLoading) saveQuestionAnswersToSession(user.id, answers); }, [answers, isLoading, user]);

  const question = questions[currentIndex];
  const selectedIds = useMemo(() => question ? answers[question.id] ?? [] : [], [answers, question]);
  if (isLoading) return <main className="question-page"><p>質問を読み込んでいます...</p></main>;
  if (error) return <main className="question-page"><div className="question-card"><p role="alert">{error}</p><button type="button" onClick={() => void load()}>再取得</button></div></main>;
  if (!question) return <main className="question-page"><p role="alert">質問設定が不足しています。</p></main>;

  const isValid = isQuestionAnswerValid(question, selectedIds);
  const isLast = currentIndex === questions.length - 1;
  return <main className="question-page"><div className="question-page-toolbar"><button type="button" className="question-top-button" onClick={() => navigate("/")}>TOPページへ戻る</button></div><section className="question-card">
    <QuestionProgress current={currentIndex + 1} total={questions.length} />
    <p className="question-code">Q{currentIndex + 1}</p><h1>{question.title}</h1>
    <p className="question-rule">{question.answer_type === "single" ? "1つ選択してください" : question.max_selections === null ? `${question.min_selections}つ以上選択してください` : `${question.min_selections}〜${question.max_selections}つ選択してください`}</p>
    <ChoiceList question={question} selectedIds={selectedIds} onChange={(ids) => setAnswers((current) => ({ ...current, [question.id]: ids }))} />
    <div className="question-actions"><button type="button" disabled={currentIndex === 0 && !state?.returnToConfirm} onClick={() => state?.returnToConfirm ? navigate("/questions/confirm", { state: { questions, answers } }) : setCurrentIndex((value) => value - 1)}>戻る</button><button type="button" className="primary" disabled={!isValid} onClick={() => state?.returnToConfirm ? navigate("/questions/confirm", { state: { questions, answers } }) : isLast ? navigate("/questions/confirm", { state: { questions, answers } }) : setCurrentIndex((value) => value + 1)}>{state?.returnToConfirm ? "確認画面へ戻る" : isLast ? "回答を確認" : "次へ"}</button></div>
  </section></main>;
}
