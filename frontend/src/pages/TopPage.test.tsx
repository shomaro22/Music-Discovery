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
  return <><p>login page</p>{state?.logoutError && <p role="alert">{state.logoutError}</p>}</>;
}

function renderTopPage() {
  useAuthMock.mockReturnValue({
    user: { id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" },
    accessToken: "access-token",
    isAuthenticated: true,
    isLoading: false,
    login: vi.fn(),
    logout: logoutMock,
    clearAuth: vi.fn(),
  });

  return render(
    <MemoryRouter initialEntries={["/"]}>
      <Routes>
        <Route path="/" element={<TopPage />} />
        <Route path="/login" element={<LoginResult />} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  logoutMock.mockReset();
});

describe("TopPage logout", () => {
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
