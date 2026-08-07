import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { WithdrawPage } from "./WithdrawPage";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));
const mockedUseAuth = vi.mocked(useAuth);
const withdrawMock = vi.fn<ReturnType<typeof useAuth>["withdraw"]>();

function renderPage() {
  return render(<MemoryRouter initialEntries={["/withdraw"]}><Routes><Route path="/withdraw" element={<WithdrawPage />} /><Route path="/login" element={<p>login</p>} /></Routes></MemoryRouter>);
}

beforeEach(() => {
  withdrawMock.mockReset();
  mockedUseAuth.mockReturnValue({
    user: { id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" },
    accessToken: "token", isAuthenticated: true, isLoading: false, authExpired: false,
    signUp: vi.fn(), login: vi.fn(), logout: vi.fn(), withdraw: withdrawMock, updateProfile: vi.fn(), clearAuth: vi.fn(),
  });
});

describe("WithdrawPage", () => {
  it("toggles current password visibility", async () => {
    const user = userEvent.setup(); renderPage();
    const input = screen.getByLabelText<HTMLInputElement>("現在のパスワード");
    expect(input).toHaveAttribute("type", "password");
    await user.click(screen.getByRole("button", { name: "パスワードを表示する" }));
    expect(input).toHaveAttribute("type", "text");
    await user.click(screen.getByRole("button", { name: "パスワードを非表示にする" }));
    expect(input).toHaveAttribute("type", "password");
  });

  it("keeps the page, password, and authentication state after password mismatch", async () => {
    withdrawMock.mockRejectedValue(new ApiRequestError({ status: 401, code: "CURRENT_PASSWORD_MISMATCH", message: "mismatch", request_id: "r1" }));
    const user = userEvent.setup(); renderPage();
    const input = screen.getByLabelText<HTMLInputElement>("現在のパスワード");
    await user.type(input, "wrong-password");
    await user.click(screen.getByRole("button", { name: "退会する" }));
    await user.click(screen.getByRole("button", { name: "退会を確定" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("現在のパスワードが正しくありません"));
    expect(input).toHaveValue("wrong-password");
    expect(screen.queryByText("login")).not.toBeInTheDocument();
    expect(mockedUseAuth().isAuthenticated).toBe(true);
  });

  it("prevents duplicate submissions and redirects after success", async () => {
    let resolve!: (value: string) => void;
    withdrawMock.mockReturnValue(new Promise<string>((r) => { resolve = r; }));
    const user = userEvent.setup(); renderPage();
    await user.type(screen.getByLabelText("現在のパスワード"), "password");
    await user.click(screen.getByRole("button", { name: "退会する" }));
    const confirm = screen.getByRole("button", { name: "退会を確定" });
    await user.click(confirm); await user.click(confirm);
    expect(withdrawMock).toHaveBeenCalledTimes(1);
    expect(withdrawMock).toHaveBeenCalledWith({ current_password: "password" });
    resolve("退会が完了しました。");
    await waitFor(() => expect(screen.getByText("login")).toBeInTheDocument());
  });
});
