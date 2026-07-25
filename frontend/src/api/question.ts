import { request } from "./client";
import type { LatestQuestionAnswersResponse, QuestionAnswer, QuestionListResponse, SaveQuestionAnswersResponse } from "../types/question";

export const QUESTION_API_PATHS = {
  questions: "/api/questions",
  latestAnswers: "/api/user-answers/latest",
  saveAnswers: "/api/user-answers",
} as const;

export function getQuestions(): Promise<QuestionListResponse> {
  return request<QuestionListResponse>(QUESTION_API_PATHS.questions, { method: "GET" });
}

export function getLatestQuestionAnswers(): Promise<LatestQuestionAnswersResponse> {
  return request<LatestQuestionAnswersResponse>(QUESTION_API_PATHS.latestAnswers, { method: "GET" });
}

export function saveQuestionAnswers(answers: QuestionAnswer[]): Promise<SaveQuestionAnswersResponse> {
  return request<SaveQuestionAnswersResponse>(QUESTION_API_PATHS.saveAnswers, { method: "POST", body: JSON.stringify({ answers }) });
}
