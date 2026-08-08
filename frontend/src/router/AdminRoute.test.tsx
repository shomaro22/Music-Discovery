import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { useAuth } from "../hooks/useAuth";
import { AdminRoute } from "./AdminRoute";

vi.mock("../hooks/useAuth", () => ({ useAuth: vi.fn() }));
const useAuthMock = vi.mocked(useAuth);

function renderRoute(allowedRoles: readonly ("user" | "content_admin" | "system_admin")[] = ["content_admin"]) {
  return render(<MemoryRouter initialEntries={["/admin"]}><Routes>
    <Route path="/login" element={<p>login</p>} />
    <Route path="/forbidden" element={<p>forbidden</p>} />
    <Route path="/admin" element={<AdminRoute allowedRoles={allowedRoles}><p>admin</p></AdminRoute>} />
  </Routes></MemoryRouter>);
}

describe("AdminRoute", () => {
  it("shows loading while auth is restoring", () => {
    useAuthMock.mockReturnValue({ isLoading: true, isAuthenticated: false, user: null } as ReturnType<typeof useAuth>);
    renderRoute(); expect(screen.queryByText("admin")).not.toBeInTheDocument();
  });
  it("redirects an unauthenticated user to login", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: false, user: null, authExpired: false } as ReturnType<typeof useAuth>);
    renderRoute(); expect(screen.getByText("login")).toBeInTheDocument();
  });
  it("renders an allowed role", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: true, user: { role: "content_admin" } } as ReturnType<typeof useAuth>);
    renderRoute(); expect(screen.getByText("admin")).toBeInTheDocument();
  });
  it("redirects an authenticated but unauthorized user to forbidden", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: true, user: { role: "user" } } as ReturnType<typeof useAuth>);
    renderRoute(); expect(screen.getByText("forbidden")).toBeInTheDocument();
  });
  it("fails closed for an empty allowed role list", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: true, user: { role: "system_admin" } } as ReturnType<typeof useAuth>);
    renderRoute([]); expect(screen.getByText("forbidden")).toBeInTheDocument();
  });
  it("allows system admin only when explicitly listed", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: true, user: { role: "system_admin" } } as ReturnType<typeof useAuth>);
    const { unmount } = renderRoute(["content_admin"]); expect(screen.getByText("forbidden")).toBeInTheDocument(); unmount();
    renderRoute(["content_admin", "system_admin"]); expect(screen.getByText("admin")).toBeInTheDocument();
  });
});
