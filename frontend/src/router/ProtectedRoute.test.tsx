import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { useAuth } from "../auth/AuthContext";
import { ProtectedRoute } from "./ProtectedRoute";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));
const useAuthMock = vi.mocked(useAuth);

function CurrentPath() {
  return <p>{useLocation().pathname}</p>;
}

function renderRoute(initialEntry = "/") {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/login" element={<><p>login page</p><CurrentPath /></>} />
        <Route path="/" element={<ProtectedRoute><p>protected page</p></ProtectedRoute>} />
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
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: false } as ReturnType<typeof useAuth>);
    renderRoute();
    expect(screen.getByText("login page")).toBeInTheDocument();
    expect(screen.getByText("/login")).toBeInTheDocument();
  });

  it("renders protected content when authenticated", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: true } as ReturnType<typeof useAuth>);
    renderRoute();
    expect(screen.getByText("protected page")).toBeInTheDocument();
  });
});
