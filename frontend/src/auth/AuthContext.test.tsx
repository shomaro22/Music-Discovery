import { render, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "../api/client";

import { AuthProvider, useAuth } from "./AuthContext";

const refreshMock = vi.fn();
const getCurrentUserMock = vi.fn();
const logoutMock = vi.fn();

vi.mock("../api/csrf", () => ({ getCSRFCookie: () => "csrf" }));
vi.mock("../api/user", () => ({
  login: vi.fn(),
  logout: () => logoutMock(),
  refresh: () => refreshMock(),
  getCurrentUser: () => getCurrentUserMock(),
}));
vi.mock("../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/client")>();
  return {
    ...actual,
    setAccessToken: vi.fn(),
    setAuthStateListener: vi.fn(),
  };
});

function Probe() {
  const auth = useAuth();
  useEffect(() => undefined, [auth]);
  return <p>{auth.isLoading ? "loading" : auth.authExpired ? "expired" : auth.user?.email ?? "anonymous"}</p>;
}

beforeEach(() => {
  refreshMock.mockReset();
  getCurrentUserMock.mockReset();
  logoutMock.mockReset();
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
