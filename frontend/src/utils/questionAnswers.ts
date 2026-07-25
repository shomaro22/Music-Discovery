import type { Question, QuestionAnswers } from "../types/question";

const STORAGE_PREFIX = "MUSIC_DISCOVERY_QUESTION_ANSWERS";

export function questionAnswersStorageKey(userId: string): string { return `${STORAGE_PREFIX}:${userId}`; }

export function loadQuestionAnswers(userId: string): QuestionAnswers {
  const raw = sessionStorage.getItem(questionAnswersStorageKey(userId));
  if (!raw) return {};
  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) return {};
    const result: QuestionAnswers = {};
    for (const [questionId, value] of Object.entries(parsed)) {
      if (Array.isArray(value) && value.every((item) => typeof item === "string")) result[questionId] = value;
    }
    return result;
  } catch { return {}; }
}

export function saveQuestionAnswersToSession(userId: string, answers: QuestionAnswers): void {
  sessionStorage.setItem(questionAnswersStorageKey(userId), JSON.stringify(answers));
}

export function clearQuestionAnswers(userId: string): void { sessionStorage.removeItem(questionAnswersStorageKey(userId)); }

export function isQuestionAnswerValid(question: Question, choiceIds: string[]): boolean {
  if (choiceIds.length < question.min_selections) return false;
  return question.max_selections === null || choiceIds.length <= question.max_selections;
}

export function areAllQuestionAnswersValid(questions: Question[], answers: QuestionAnswers): boolean {
  return questions.length === 5 && questions.every((question) => isQuestionAnswerValid(question, answers[question.id] ?? []));
}
