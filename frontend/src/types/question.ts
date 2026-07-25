export type QuestionAnswerType = "single" | "multiple";

export interface QuestionChoice {
  id: string;
  code: string;
  label: string;
  sort_order: number;
}

export interface Question {
  id: string;
  code: string;
  title: string;
  answer_type: QuestionAnswerType;
  min_selections: number;
  max_selections: number | null;
  sort_order: number;
  choices: QuestionChoice[];
}

export interface QuestionListResponse { questions: Question[]; }
export interface QuestionAnswer { question_id: string; choice_ids: string[]; }
export interface LatestQuestionAnswersResponse { answers: QuestionAnswer[]; answered_at?: string; }
export interface SaveQuestionAnswersResponse { answer_session_id: string; profile_initialized: boolean; answered_at: string; }
export type QuestionAnswers = Record<string, string[]>;
