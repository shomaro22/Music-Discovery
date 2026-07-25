import { describe, expect, it } from "vitest";
import type { Question } from "../types/question";
import { areAllQuestionAnswersValid, isQuestionAnswerValid } from "./questionAnswers";

function question(id: string, min: number, max: number | null): Question {
  return { id, code: id, title: id, answer_type: max === 1 ? "single" : "multiple", min_selections: min, max_selections: max, sort_order: 1, choices: [] };
}

describe("question answer validation", () => {
  it("Q4 rejects a fourth choice", () => {
    expect(isQuestionAnswerValid(question("q4", 1, 3), ["1", "2", "3", "4"])).toBe(false);
  });

  it("requires all five questions", () => {
    const questions = [question("q1", 1, 1), question("q2", 1, null), question("q3", 1, null), question("q4", 1, 3), question("q5", 1, 1)];
    expect(areAllQuestionAnswersValid(questions, { q1: ["1"], q2: ["2"], q3: ["3"], q4: ["4"], q5: ["5"] })).toBe(true);
    expect(areAllQuestionAnswersValid(questions, { q1: ["1"] })).toBe(false);
  });
});
