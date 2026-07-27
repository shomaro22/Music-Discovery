import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { saveQuestionAnswers } from "../api/question";
import { useAuth } from "../auth/AuthContext";
import type { Question, QuestionAnswers } from "../types/question";
import { AnswerConfirmPage } from "./AnswerConfirmPage";

vi.mock("../api/question", () => ({ saveQuestionAnswers: vi.fn() }));
vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));

const saveMock = vi.mocked(saveQuestionAnswers);
const useAuthMock = vi.mocked(useAuth);

const questions: Question[] = [1, 2, 3, 4, 5].map((value) => ({
  id: `q${value}`,
  code: value === 1 ? "MOOD" : value === 5 ? "DISCOVERY_LEVEL" : value === 4 ? "GENRE" : value === 2 ? "ELEMENT" : "ATMOSPHERE",
  title: `Q${value}`,
  answer_type: value === 1 || value === 5 ? "single" : "multiple",
  min_selections: 1,
  max_selections: value === 1 || value === 5 ? 1 : value === 4 ? 3 : null,
  sort_order: value,
  choices: [{ id: `c${value}`, code: `C${value}`, label: `Choice ${value}`, sort_order: 1 }],
}));
const answers: QuestionAnswers = { q1: ["c1"], q2: ["c2"], q3: ["c3"], q4: ["c4"], q5: ["c5"] };

beforeEach(() => {
  sessionStorage.clear();
  sessionStorage.setItem("MUSIC_DISCOVERY_QUESTION_ANSWERS:u1", JSON.stringify(answers));
  saveMock.mockReset();
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

describe("AnswerConfirmPage", () => {
  it("prevents duplicate submission and keeps answers when saving fails", async () => {
    let rejectSave: ((reason?: unknown) => void) | undefined;
    saveMock.mockReturnValue(new Promise((_, reject) => { rejectSave = reject; }));
    render(
      <MemoryRouter initialEntries={[{ pathname: "/questions/confirm", state: { questions, answers } }]}>
        <Routes><Route path="/questions/confirm" element={<AnswerConfirmPage />} /></Routes>
      </MemoryRouter>,
    );
    const button = screen.getByRole("button", { name: "この回答で保存" });
    fireEvent.click(button);
    fireEvent.click(button);
    expect(saveMock).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "保存中..." })).toBeDisabled();
    rejectSave?.(new Error("failed"));
    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
    expect(sessionStorage.getItem("MUSIC_DISCOVERY_QUESTION_ANSWERS:u1")).not.toBeNull();
  });

  it("removes the user-scoped Session Storage entry after a successful save", async () => {
    saveMock.mockResolvedValue({ answer_session_id: "session-1", profile_initialized: false, answered_at: "2026-07-27T00:00:00Z" });
    render(
      <MemoryRouter initialEntries={[{ pathname: "/questions/confirm", state: { questions, answers } }]}>
        <Routes>
          <Route path="/questions/confirm" element={<AnswerConfirmPage />} />
          <Route path="/recommendations" element={<p>saved</p>} />
        </Routes>
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByRole("button", { name: "この回答で保存" }));
    await waitFor(() => expect(screen.getByText("saved")).toBeInTheDocument());
    expect(sessionStorage.getItem("MUSIC_DISCOVERY_QUESTION_ANSWERS:u1")).toBeNull();
  });

});
