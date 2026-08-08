import { API_PATHS } from "../constants/apiPaths";
import type {
  LatestQuestionAnswersResponse,
  QuestionAnswer,
  QuestionListResponse,
  SaveQuestionAnswersResponse,
} from "../types/question";
import { request } from "./client";

export function getQuestions(): Promise<QuestionListResponse> {
  return request<QuestionListResponse>(API_PATHS.questions, { method: "GET" });
}

export function getLatestQuestionAnswers(): Promise<LatestQuestionAnswersResponse> {
  return request<LatestQuestionAnswersResponse>(API_PATHS.latestQuestionAnswers, {
    method: "GET",
  });
}

export function saveQuestionAnswers(
  answers: QuestionAnswer[],
): Promise<SaveQuestionAnswersResponse> {
  return request<SaveQuestionAnswersResponse>(API_PATHS.saveQuestionAnswers, {
    method: "POST",
    body: JSON.stringify({ answers }),
  });
}
