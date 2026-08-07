import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { LoginPage } from "./LoginPage";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));
const mockedUseAuth = vi.mocked(useAuth);
const loginMock = vi.fn();

function authValue(overrides: Partial<ReturnType<typeof useAuth>> = {}): ReturnType<typeof useAuth> {
  return {
    user: null,
    accessToken: null,
    isAuthenticated: false,
    isLoading: false,
    authExpired: false,
    signUp: vi.fn(),
    login: loginMock,
    logout: vi.fn(),
    withdraw: vi.fn(),
    updateProfile: vi.fn(),
    clearAuth: vi.fn(),
    ...overrides,
  };
}

function renderPage(initialEntry: string | { pathname: string; state?: unknown } = "/login") {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<p>user top</p>} />
        <Route path="/admin/content" element={<p>content admin</p>} />
        <Route path="/system/users" element={<p>system admin</p>} />
        <Route path="/protected" element={<p>protected destination</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  loginMock.mockReset();
  mockedUseAuth.mockReturnValue(authValue());
});

describe("LoginPage", () => {
  it("validates fields before calling login", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("メールアドレス"), "invalid");
    await user.type(screen.getByLabelText("パスワード"), "short");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    expect(screen.getByText("有効なメールアドレスを入力してください")).toBeInTheDocument();
    expect(screen.getByText("パスワードは8文字以上で入力してください")).toBeInTheDocument();
    expect(loginMock).not.toHaveBeenCalled();
  });

  it("toggles password visibility", async () => {
    const user = userEvent.setup();
    renderPage();
    const input = screen.getByLabelText<HTMLInputElement>("パスワード");

    expect(input).toHaveAttribute("type", "password");
    await user.click(screen.getByRole("button", { name: "パスワードを表示する" }));
    expect(input).toHaveAttribute("type", "text");
    await user.click(screen.getByRole("button", { name: "パスワードを非表示にする" }));
    expect(input).toHaveAttribute("type", "password");
  });

  it("normalizes email and returns to the requested page after login", async () => {
    loginMock.mockResolvedValue({
      id: "user-1",
      name: "User",
      email: "user@example.com",
      role: "user",
      status: "active",
    });
    const user = userEvent.setup();
    renderPage({ pathname: "/login", state: { from: { pathname: "/protected" } } });

    await user.type(screen.getByLabelText("メールアドレス"), " USER@EXAMPLE.COM ");
    await user.type(screen.getByLabelText("パスワード"), "password");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    await waitFor(() => expect(loginMock).toHaveBeenCalledWith({ email: "user@example.com", password: "password" }));
    expect(await screen.findByText("protected destination")).toBeInTheDocument();
  });

  it("routes administrators to their role destination", () => {
    mockedUseAuth.mockReturnValue(authValue({
      isAuthenticated: true,
      user: {
        id: "admin-1",
        name: "Admin",
        email: "admin@example.com",
        role: "content_admin",
        status: "active",
        },
    }));
    renderPage();
    expect(screen.getByText("content admin")).toBeInTheDocument();
  });

  it("shows API error and request id", async () => {
    loginMock.mockRejectedValue(
      new ApiRequestError({
        status: 401,
        code: "UNAUTHORIZED",
        message: "認証に失敗しました",
        request_id: "req-login-1",
      }),
    );
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("メールアドレス"), "user@example.com");
    await user.type(screen.getByLabelText("パスワード"), "password");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    expect(await screen.findByText("認証に失敗しました")).toBeInTheDocument();
    expect(screen.getByText("Request ID: req-login-1")).toBeInTheDocument();
    expect(screen.getByLabelText<HTMLInputElement>("パスワード")).toHaveValue("");
  });
});
