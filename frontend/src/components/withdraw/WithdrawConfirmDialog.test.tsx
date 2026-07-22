import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { WithdrawConfirmDialog } from "./WithdrawConfirmDialog";

describe("WithdrawConfirmDialog", () => {
  it("does not render when closed", () => {
    render(<WithdrawConfirmDialog isOpen={false} isSubmitting={false} onConfirm={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes with Escape before the API request starts", () => {
    const onCancel = vi.fn();
    render(<WithdrawConfirmDialog isOpen isSubmitting={false} onConfirm={vi.fn()} onCancel={onCancel} />);

    fireEvent.keyDown(document, { key: "Escape" });

    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("does not close with Escape while submitting", () => {
    const onCancel = vi.fn();
    render(<WithdrawConfirmDialog isOpen isSubmitting onConfirm={vi.fn()} onCancel={onCancel} />);

    fireEvent.keyDown(document, { key: "Escape" });

    expect(onCancel).not.toHaveBeenCalled();
  });

  it("does not submit when cancel is selected", async () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const user = userEvent.setup();
    render(<WithdrawConfirmDialog isOpen isSubmitting={false} onConfirm={onConfirm} onCancel={onCancel} />);

    await user.click(screen.getByRole("button", { name: "キャンセル" }));

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("disables both actions while submitting", () => {
    render(<WithdrawConfirmDialog isOpen isSubmitting onConfirm={vi.fn()} onCancel={vi.fn()} />);

    expect(screen.getByRole("button", { name: "キャンセル" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "退会処理中..." })).toBeDisabled();
  });
});
