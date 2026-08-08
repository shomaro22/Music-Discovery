import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { ForbiddenPage } from "./ForbiddenPage";

describe("ForbiddenPage", () => {
  it("shows a safe permission message and top button", () => {
    render(<MemoryRouter><ForbiddenPage /></MemoryRouter>);
    expect(screen.getByText(/権限がありません/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Topへ戻る" })).toBeInTheDocument();
    expect(screen.queryByText(/content_admin|system_admin/)).not.toBeInTheDocument();
  });
});
