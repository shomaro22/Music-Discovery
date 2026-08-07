import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { confirmPasswordReset, requestPasswordReset } from "../api/user";
import { useAuth } from "../auth/AuthContext";
import { ForgotPasswordPage } from "./ForgotPasswordPage";
import { ResetPasswordPage } from "./ResetPasswordPage";

vi.mock("../api/user", () => ({
  requestPasswordReset: vi.fn(),
  confirmPasswordReset: vi.fn(),
}));
vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));

const requestMock = vi.mocked(requestPasswordReset);
const confirmMock = vi.mocked(confirmPasswordReset);
const useAuthMock = vi.mocked(useAuth);
const logoutMock = vi.fn();

describe("Password reset pages", () => {
  beforeEach(() => {
    requestMock.mockReset();
    confirmMock.mockReset();
    logoutMock.mockReset();
    logoutMock.mockResolvedValue(undefined);
    useAuthMock.mockReturnValue({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      isLoading: false,
      authExpired: false,
      signUp: vi.fn(),
      login: vi.fn(),
      logout: logoutMock,
      withdraw: vi.fn(),
      updateProfile: vi.fn(),
      clearAuth: vi.fn(),
    });
  });

  it("prefills the email address passed from the profile page", () => {
    render(
      <MemoryRouter initialEntries={[{ pathname: "/forgot-password", state: { email: "user@example.com" } }]}>
        <Routes><Route path="/forgot-password" element={<ForgotPasswordPage />} /></Routes>
      </MemoryRouter>,
    );

    expect(screen.getByLabelText("メールアドレス")).toHaveValue("user@example.com");
  });

  it("requests a reset and exposes the development link when returned", async () => {
    requestMock.mockResolvedValue({ message: "案内を送信しました。", reset_token: "dev-token" });
    const user = userEvent.setup();
    render(<MemoryRouter initialEntries={["/forgot-password"]}><Routes><Route path="/forgot-password" element={<ForgotPasswordPage />} /><Route path="/reset-password" element={<p>reset destination</p>} /></Routes></MemoryRouter>);

    await user.type(screen.getByLabelText("メールアドレス"), " USER@EXAMPLE.COM ");
    await user.click(screen.getByRole("button", { name: "再設定メールを送る" }));

    expect(requestMock).toHaveBeenCalledWith({ email: "user@example.com" });
    expect(await screen.findByText("案内を送信しました。")).toBeInTheDocument();
    await user.click(screen.getByRole("link", { name: "パスワード再設定へ進む" }));
    expect(screen.getByText("reset destination")).toBeInTheDocument();
  });

  it("logs out an authenticated user before returning to login", async () => {
    useAuthMock.mockReturnValue({
      user: { id: "u1", name: "User", email: "user@example.com", role: "user", status: "active" },
      accessToken: "token",
      isAuthenticated: true,
      isLoading: false,
      authExpired: false,
      signUp: vi.fn(),
      login: vi.fn(),
      logout: logoutMock,
      withdraw: vi.fn(),
      updateProfile: vi.fn(),
      clearAuth: vi.fn(),
    });
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/forgot-password"]}>
        <Routes>
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/login" element={<p>login destination</p>} />
        </Routes>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "ログインへ戻る" }));

    expect(logoutMock).toHaveBeenCalledTimes(1);
    expect(await screen.findByText("login destination")).toBeInTheDocument();
  });

  it("confirms a new password with the token in the URL", async () => {
    confirmMock.mockResolvedValue({ message: "パスワードを変更しました。" });
    const user = userEvent.setup();
    render(<MemoryRouter initialEntries={["/reset-password?token=reset-token"]}><Routes><Route path="/reset-password" element={<ResetPasswordPage />} /></Routes></MemoryRouter>);

    await user.type(screen.getByLabelText("新しいパスワード"), "new-password");
    await user.type(screen.getByLabelText("新しいパスワード確認"), "new-password");
    await user.click(screen.getByRole("button", { name: "パスワードを変更" }));

    expect(confirmMock).toHaveBeenCalledWith({ token: "reset-token", new_password: "new-password" });
    expect(await screen.findByText("パスワードを変更しました。")).toBeInTheDocument();
  });
});
