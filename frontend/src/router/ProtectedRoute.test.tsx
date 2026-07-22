import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { useAuth } from "../auth/AuthContext";
import { ProtectedRoute } from "./ProtectedRoute";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));
const useAuthMock = vi.mocked(useAuth);

function CurrentPath() {
  const location = useLocation();
  const state = location.state as { from?: { pathname?: string }; authExpired?: boolean } | null;
  return <><p>{location.pathname}</p><p>{state?.from?.pathname ?? "no-from"}</p><p>{state?.authExpired ? "expired" : "not-expired"}</p></>;
}

function renderRoute(initialEntry = "/account") {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/login" element={<><p>login page</p><CurrentPath /></>} />
        <Route path="/account" element={<ProtectedRoute><p>protected page</p></ProtectedRoute>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("ProtectedRoute", () => {
  it("shows loading while authentication restoration is running", () => {
    useAuthMock.mockReturnValue({ isLoading: true, isAuthenticated: false } as ReturnType<typeof useAuth>);
    renderRoute();
    expect(screen.getByRole("status")).toHaveTextContent("認証情報を確認しています");
    expect(screen.queryByText("login page")).not.toBeInTheDocument();
  });

  it("redirects direct access to login after restoration when unauthenticated", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: false, authExpired: true } as ReturnType<typeof useAuth>);
    renderRoute();
    expect(screen.getByText("login page")).toBeInTheDocument();
    expect(screen.getByText("/login")).toBeInTheDocument();
    expect(screen.getByText("/account")).toBeInTheDocument();
    expect(screen.getByText("expired")).toBeInTheDocument();
  });

  it("renders protected content when authenticated", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: true } as ReturnType<typeof useAuth>);
    renderRoute();
    expect(screen.getByText("protected page")).toBeInTheDocument();
  });
});
