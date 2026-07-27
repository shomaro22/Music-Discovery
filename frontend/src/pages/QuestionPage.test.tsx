import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getLatestQuestionAnswers, getQuestions } from "../api/question";
import { useAuth } from "../auth/AuthContext";
import type { Question } from "../types/question";
import { QuestionPage } from "./QuestionPage";

vi.mock("../api/question", () => ({ getQuestions: vi.fn(), getLatestQuestionAnswers: vi.fn() }));
vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));

const getQuestionsMock = vi.mocked(getQuestions);
const getLatestMock = vi.mocked(getLatestQuestionAnswers);
const useAuthMock = vi.mocked(useAuth);

const questions: Question[] = [
  { id: "q1", code: "MOOD", title: "今日の気分", answer_type: "single", min_selections: 1, max_selections: 1, sort_order: 1, choices: [{ id: "c1", code: "MOOD_CALM", label: "落ち着きたい", sort_order: 1 }] },
  { id: "q2", code: "ELEMENT", title: "重視する要素", answer_type: "multiple", min_selections: 1, max_selections: null, sort_order: 2, choices: [{ id: "c2", code: "ELEMENT_MELODY", label: "メロディ", sort_order: 1 }] },
  { id: "q3", code: "ATMOSPHERE", title: "雰囲気", answer_type: "multiple", min_selections: 1, max_selections: null, sort_order: 3, choices: [{ id: "c3", code: "ATMOSPHERE_CALM", label: "落ち着いた", sort_order: 1 }] },
  { id: "q4", code: "GENRE", title: "ジャンル", answer_type: "multiple", min_selections: 1, max_selections: 3, sort_order: 4, choices: [{ id: "c4", code: "GENRE_POP", label: "ポップ", sort_order: 1 }] },
  { id: "q5", code: "DISCOVERY_LEVEL", title: "探索度", answer_type: "single", min_selections: 1, max_selections: 1, sort_order: 5, choices: [{ id: "c5", code: "DISCOVERY_MEDIUM", label: "少し新しい音楽", sort_order: 1 }] },
];

beforeEach(() => {
  sessionStorage.clear();
  getQuestionsMock.mockResolvedValue({ questions });
  getLatestMock.mockResolvedValue({ answers: [] });
  useAuthMock.mockReturnValue({
    user: { id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" },
    accessToken: "token",
    isAuthenticated: true,
    isLoading: false,
    authExpired: false,
    signUp: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    withdraw: vi.fn(),
    clearAuth: vi.fn(),
  });
});

describe("QuestionPage", () => {
  it("restores the edited answer and returns directly to confirmation", async () => {
    sessionStorage.setItem("MUSIC_DISCOVERY_QUESTION_ANSWERS:u1", JSON.stringify({ q1: ["c1"], q2: ["c2"], q3: ["c3"], q4: ["c4"], q5: ["c5"] }));
    render(
      <MemoryRouter initialEntries={[{ pathname: "/questions", state: { editQuestionId: "q2", returnToConfirm: true } }]}>
        <Routes>
          <Route path="/questions" element={<QuestionPage />} />
          <Route path="/questions/confirm" element={<p>confirm page</p>} />
        </Routes>
      </MemoryRouter>,
    );
    await waitFor(() => expect(screen.getByRole("heading", { name: "重視する要素" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "確認画面へ戻る" }));
    expect(screen.getByText("confirm page")).toBeInTheDocument();
  });
});
