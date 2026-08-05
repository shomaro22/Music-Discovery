import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuth } from "../auth/AuthContext";
import { TopPage } from "./TopPage";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));
const useAuthMock = vi.mocked(useAuth);
const logoutMock = vi.fn();

function LoginResult() {
  const location = useLocation();
  const state = location.state as { logoutError?: string } | null;

  return (
    <>
      <p>login page</p>
      {state?.logoutError && <p role="alert">{state.logoutError}</p>}
    </>
  );
}

function QuestionsResult() {
  const location = useLocation();
  const state = location.state as { loadLatest?: boolean } | null;

  return (
    <>
      <p>questions page</p>
      {state?.loadLatest && <p>latest answers enabled</p>}
    </>
  );
}

function renderTopPage() {
  useAuthMock.mockReturnValue({
    user: {
      id: "u1",
      name: "Taro",
      email: "taro@example.com",
      role: "user",
      status: "active",
    },
    accessToken: "access-token",
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

  return render(
    <MemoryRouter initialEntries={["/"]}>
      <Routes>
        <Route path="/" element={<TopPage />} />
        <Route path="/login" element={<LoginResult />} />
        <Route path="/withdraw" element={<p>withdraw page</p>} />
        <Route path="/profile" element={<p>profile page</p>} />
        <Route path="/questions" element={<QuestionsResult />} />
        <Route path="/recommendation-history" element={<p>history page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  logoutMock.mockReset();
});

describe("TopPage", () => {
  it("shows the shortened slogan", () => {
    renderTopPage();
    expect(
      screen.getByRole("heading", { name: "次の一曲を、見つけよう。" }),
    ).toBeInTheDocument();
  });

  it("moves to the profile page", () => {
    renderTopPage();
    fireEvent.click(screen.getByRole("button", { name: "プロフィールを変更" }));
    expect(screen.getByText("profile page")).toBeInTheDocument();
  });

  it("moves to the questions page and loads the latest answers", () => {
    renderTopPage();
    fireEvent.click(screen.getByRole("button", { name: "おすすめを探す" }));
    expect(screen.getByText("questions page")).toBeInTheDocument();
    expect(screen.getByText("latest answers enabled")).toBeInTheDocument();
  });

  it("does not show a separate preference edit button", () => {
    renderTopPage();
    expect(
      screen.queryByRole("button", { name: "好みを変更" }),
    ).not.toBeInTheDocument();
  });


  it("moves to recommendation history from the top page", () => {
    renderTopPage();
    fireEvent.click(screen.getByRole("button", { name: "推薦履歴を開く" }));
    expect(screen.getByText("history page")).toBeInTheDocument();
  });

  it("moves to the withdraw page", () => {
    renderTopPage();
    fireEvent.click(screen.getByRole("button", { name: "退会手続きへ" }));
    expect(screen.getByText("withdraw page")).toBeInTheDocument();
  });

  it("moves to login after successful logout", async () => {
    logoutMock.mockResolvedValue(undefined);
    renderTopPage();
    fireEvent.click(screen.getByRole("button", { name: "ログアウト" }));
    await waitFor(() => expect(screen.getByText("login page")).toBeInTheDocument());
    expect(logoutMock).toHaveBeenCalledTimes(1);
  });

  it("moves to login with an error after failed logout", async () => {
    logoutMock.mockRejectedValue(new Error("logout failed"));
    renderTopPage();
    fireEvent.click(screen.getByRole("button", { name: "ログアウト" }));
    await waitFor(() => expect(screen.getByText("login page")).toBeInTheDocument());
    expect(screen.getByRole("alert")).toHaveTextContent("ログアウト処理に失敗しました");
  });

  it("disables the button while logout is running", () => {
    logoutMock.mockReturnValue(new Promise(() => undefined));
    renderTopPage();
    fireEvent.click(screen.getByRole("button", { name: "ログアウト" }));
    expect(screen.getByRole("button", { name: "ログアウト中..." })).toBeDisabled();
  });

  it("calls the logout API only once on repeated clicks", () => {
    logoutMock.mockReturnValue(new Promise(() => undefined));
    renderTopPage();
    const button = screen.getByRole("button", { name: "ログアウト" });
    fireEvent.click(button);
    fireEvent.click(button);
    expect(logoutMock).toHaveBeenCalledTimes(1);
  });
});
