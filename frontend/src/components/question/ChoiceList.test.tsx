import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Question } from "../../types/question";
import { ChoiceList } from "./ChoiceList";

const choices = ["1", "2", "3", "4"].map((id, index) => ({ id, code: `C${id}`, label: `Choice ${id}`, sort_order: index + 1 }));

describe("ChoiceList", () => {
  it("replaces the selected value for single choice", async () => {
    const onChange = vi.fn();
    const question: Question = { id: "q1", code: "MOOD", title: "Mood", answer_type: "single", min_selections: 1, max_selections: 1, sort_order: 1, choices };
    render(<ChoiceList question={question} selectedIds={["1"]} onChange={onChange} />);
    await userEvent.click(screen.getByRole("radio", { name: "Choice 2" }));
    expect(onChange).toHaveBeenCalledWith(["2"]);
  });

  it("disables a fourth choice when max is three", () => {
    const question: Question = { id: "q4", code: "GENRE", title: "Genre", answer_type: "multiple", min_selections: 1, max_selections: 3, sort_order: 4, choices };
    render(<ChoiceList question={question} selectedIds={["1", "2", "3"]} onChange={vi.fn()} />);
    expect(screen.getByRole("checkbox", { name: "Choice 4" })).toBeDisabled();
  });
});
