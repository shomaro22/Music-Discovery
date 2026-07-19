import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { login as loginApi } from "../api/user";
import { AuthProvider, useAuth } from "./AuthContext";

vi.mock("../api/user", () => ({ login: vi.fn() }));
const mockedLogin = vi.mocked(loginApi);

function Consumer() {
  const auth = useAuth();
  return <><span>{auth.user?.email ?? "none"}</span><span>{auth.accessToken ?? "no-token"}</span><span>{String(auth.isAuthenticated)}</span><button onClick={() => auth.login({ email: "a@b.com", password: "password" })}>login</button><button onClick={auth.clearAuth}>clear</button></>;
}

beforeEach(() => { mockedLogin.mockReset(); localStorage.clear(); });
describe("AuthContext", () => {
  it("keeps login state only in memory", async () => {
    mockedLogin.mockResolvedValue({ access_token: "access", token_type: "Bearer", expires_in: 300, user: { id: "id", name: "Taro", email: "a@b.com", role: "user", status: "active" } });
    render(<AuthProvider><Consumer /></AuthProvider>);
    await userEvent.click(screen.getByRole("button", { name: "login" }));
    await waitFor(() => expect(screen.getByText("a@b.com")).toBeInTheDocument());
    expect(screen.getByText("access")).toBeInTheDocument(); expect(screen.getByText("true")).toBeInTheDocument();
    expect(localStorage.length).toBe(0);
    await userEvent.click(screen.getByRole("button", { name: "clear" })); expect(screen.getByText("none")).toBeInTheDocument();
  });
});
