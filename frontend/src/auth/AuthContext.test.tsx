import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";

import { AuthProvider, useAuth } from "./AuthContext";

const refreshMock = vi.fn();
const getCurrentUserMock = vi.fn();
const logoutMock = vi.fn();
const withdrawMock = vi.fn();
let authStateListener: ((token: string | null, reason?: "expired") => void) | null = null;
const bootstrapCSRFMock = vi.fn();

vi.mock("../api/csrf", () => ({ clearCSRFToken: vi.fn() }));
vi.mock("../api/user", () => ({
  login: vi.fn(),
  logout: () => logoutMock(),
  withdraw: (...args: unknown[]) => withdrawMock(...args),
  refresh: () => refreshMock(),
  getCurrentUser: () => getCurrentUserMock(),
}));
vi.mock("../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/client")>();
  return {
    ...actual,
    bootstrapCSRF: () => bootstrapCSRFMock(),
    setAccessToken: vi.fn(),
    setAuthStateListener: vi.fn((listener: ((token: string | null, reason?: "expired") => void) | null) => { authStateListener = listener; }),
  };
});

function Probe() {
  const auth = useAuth();
  useEffect(() => undefined, [auth]);
  return <p>{auth.isLoading ? "loading" : auth.authExpired ? "expired" : auth.user?.email ?? "anonymous"}</p>;
}

beforeEach(() => {
  bootstrapCSRFMock.mockReset();
  bootstrapCSRFMock.mockResolvedValue({ csrf_token: "csrf", refresh_available: true });
  refreshMock.mockReset();
  getCurrentUserMock.mockReset();
  logoutMock.mockReset();
  withdrawMock.mockReset();
  authStateListener = null;
  sessionStorage.clear();
});

describe("AuthProvider restore", () => {
  it("keeps loading true until refresh and current-user restoration complete", async () => {
    let resolveRefresh!: (value: unknown) => void;
    refreshMock.mockReturnValue(new Promise((resolve) => { resolveRefresh = resolve; }));
    getCurrentUserMock.mockResolvedValue({ id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" });

    render(<AuthProvider><Probe /></AuthProvider>);
    expect(screen.getByText("loading")).toBeInTheDocument();

    resolveRefresh({ access_token: "token", token_type: "Bearer", expires_in: 300 });
    await waitFor(() => expect(screen.getByText("taro@example.com")).toBeInTheDocument());
    expect(refreshMock).toHaveBeenCalledTimes(1);
    expect(getCurrentUserMock).toHaveBeenCalledTimes(1);
  });


  it("marks authentication expired when restoration refresh is unauthorized", async () => {
    refreshMock.mockRejectedValue(new ApiRequestError({ status: 401, code: "INVALID_REFRESH_TOKEN", message: "expired", request_id: "" }));
    render(<AuthProvider><Probe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("expired")).toBeInTheDocument());
  });


  it("clears user-scoped question answers when authentication expires", async () => {
    refreshMock.mockResolvedValue({ access_token: "token", token_type: "Bearer", expires_in: 300 });
    getCurrentUserMock.mockResolvedValue({ id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" });
    sessionStorage.setItem("MUSIC_DISCOVERY_QUESTION_ANSWERS:u1", JSON.stringify({ q1: ["c1"] }));

    render(<AuthProvider><Probe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("taro@example.com")).toBeInTheDocument());

    act(() => { authStateListener?.(null, "expired"); });

    await waitFor(() => expect(screen.getByText("expired")).toBeInTheDocument());
    expect(sessionStorage.getItem("MUSIC_DISCOVERY_QUESTION_ANSWERS:u1")).toBeNull();
  });

  it("does not mark authentication expired for a restoration server error", async () => {
    refreshMock.mockRejectedValue(new ApiRequestError({ status: 500, code: "INTERNAL_ERROR", message: "failed", request_id: "" }));
    render(<AuthProvider><Probe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("anonymous")).toBeInTheDocument());
    expect(screen.queryByText("expired")).not.toBeInTheDocument();
  });
  it("clears authentication when restoration fails", async () => {
    refreshMock.mockRejectedValue(new Error("refresh failed"));
    render(<AuthProvider><Probe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("anonymous")).toBeInTheDocument());
  });


  it.each(["", "guest", "owner"])("clears authentication when current-user returns invalid role %p", async (role) => {
    refreshMock.mockResolvedValue({ access_token: "token", token_type: "Bearer", expires_in: 300 });
    getCurrentUserMock.mockResolvedValue({ id: "u1", name: "Taro", email: "taro@example.com", role, status: "active" });

    render(<AuthProvider><Probe /></AuthProvider>);

    await waitFor(() => expect(screen.getByText("anonymous")).toBeInTheDocument());
    expect(screen.queryByText("taro@example.com")).not.toBeInTheDocument();
  });
});


function LogoutProbe() {
  const auth = useAuth();
  return (
    <>
      <p>{auth.user?.email ?? "anonymous"}</p>
      <button type="button" onClick={() => { void auth.logout().catch(() => undefined); }}>logout</button>
    </>
  );
}

describe("AuthProvider logout", () => {
  it("clears frontend authentication after API success", async () => {
    refreshMock.mockResolvedValue({ access_token: "token", token_type: "Bearer", expires_in: 300 });
    getCurrentUserMock.mockResolvedValue({ id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" });
    logoutMock.mockResolvedValue(undefined);
    render(<AuthProvider><LogoutProbe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("taro@example.com")).toBeInTheDocument());
    screen.getByRole("button", { name: "logout" }).click();
    await waitFor(() => expect(screen.getByText("anonymous")).toBeInTheDocument());
    expect(logoutMock).toHaveBeenCalledTimes(1);
  });

  it("clears frontend authentication even when API fails", async () => {
    refreshMock.mockResolvedValue({ access_token: "token", token_type: "Bearer", expires_in: 300 });
    getCurrentUserMock.mockResolvedValue({ id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" });
    logoutMock.mockRejectedValue(new Error("logout failed"));
    render(<AuthProvider><LogoutProbe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("taro@example.com")).toBeInTheDocument());
    screen.getByRole("button", { name: "logout" }).click();
    await waitFor(() => expect(screen.getByText("anonymous")).toBeInTheDocument());
  });
});


function WithdrawProbe() {
  const auth = useAuth();
  return (
    <>
      <p data-testid="auth-state">{`${auth.user?.email ?? "anonymous"}|${auth.accessToken ?? "no-token"}|${auth.isAuthenticated}|${auth.isLoading}`}</p>
      <button type="button" onClick={() => { void auth.withdraw({ current_password: "password123" }).catch(() => undefined); }}>withdraw</button>
    </>
  );
}

describe("AuthProvider withdraw", () => {
  it("clears authentication after withdrawal succeeds without using global loading", async () => {
    refreshMock.mockResolvedValue({ access_token: "token", token_type: "Bearer", expires_in: 300 });
    getCurrentUserMock.mockResolvedValue({ id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" });
    let resolveWithdraw!: (value: { message: string }) => void;
    withdrawMock.mockReturnValue(new Promise((resolve) => { resolveWithdraw = resolve; }));
    const user = userEvent.setup();

    render(<AuthProvider><WithdrawProbe /></AuthProvider>);
    await waitFor(() => expect(screen.getByTestId("auth-state")).toHaveTextContent("taro@example.com|token|true|false"));

    await user.click(screen.getByRole("button", { name: "withdraw" }));
    expect(screen.getByTestId("auth-state")).toHaveTextContent("taro@example.com|token|true|false");

    resolveWithdraw({ message: "退会が完了しました。" });
    await waitFor(() => expect(screen.getByTestId("auth-state")).toHaveTextContent("anonymous|no-token|false|false"));
    expect(withdrawMock).toHaveBeenCalledWith({ current_password: "password123" });
  });

  it("preserves authentication after withdrawal fails", async () => {
    refreshMock.mockResolvedValue({ access_token: "token", token_type: "Bearer", expires_in: 300 });
    getCurrentUserMock.mockResolvedValue({ id: "u1", name: "Taro", email: "taro@example.com", role: "user", status: "active" });
    withdrawMock.mockRejectedValue(new ApiRequestError({ status: 500, code: "INTERNAL_ERROR", message: "failed", request_id: "" }));
    const user = userEvent.setup();

    render(<AuthProvider><WithdrawProbe /></AuthProvider>);
    await waitFor(() => expect(screen.getByTestId("auth-state")).toHaveTextContent("taro@example.com|token|true|false"));

    await user.click(screen.getByRole("button", { name: "withdraw" }));
    await waitFor(() => expect(withdrawMock).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("auth-state")).toHaveTextContent("taro@example.com|token|true|false");
  });
});
