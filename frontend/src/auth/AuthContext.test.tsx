import { render, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuthProvider, useAuth } from "./AuthContext";

const refreshMock = vi.fn();
const getCurrentUserMock = vi.fn();

vi.mock("../api/csrf", () => ({ getCSRFCookie: () => "csrf" }));
vi.mock("../api/user", () => ({
  login: vi.fn(),
  refresh: () => refreshMock(),
  getCurrentUser: () => getCurrentUserMock(),
}));
vi.mock("../api/client", () => ({
  setAccessToken: vi.fn(),
  setAuthStateListener: vi.fn(),
}));

function Probe() {
  const auth = useAuth();
  useEffect(() => undefined, [auth]);
  return <p>{auth.isLoading ? "loading" : auth.user?.email ?? "anonymous"}</p>;
}

beforeEach(() => {
  refreshMock.mockReset();
  getCurrentUserMock.mockReset();
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

  it("clears authentication when restoration fails", async () => {
    refreshMock.mockRejectedValue(new Error("refresh failed"));
    render(<AuthProvider><Probe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("anonymous")).toBeInTheDocument());
  });
});
