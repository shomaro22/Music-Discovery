import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { Navigation } from "./Navigation";
import { useAuth } from "../../hooks/useAuth";
import { useRole } from "../../hooks/useRole";

vi.mock("../../hooks/useAuth", () => ({ useAuth: vi.fn() }));
vi.mock("../../hooks/useRole", () => ({ useRole: vi.fn() }));
const useAuthMock = vi.mocked(useAuth);
const useRoleMock = vi.mocked(useRole);

function renderNavigation() { return render(<MemoryRouter><Navigation /></MemoryRouter>); }

describe("Navigation", () => {
  it("shows guest links only", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: false } as ReturnType<typeof useAuth>);
    useRoleMock.mockReturnValue({ isContentAdmin: false, isSystemAdmin: false } as ReturnType<typeof useRole>);
    renderNavigation();
    expect(screen.getByText("会員登録")).toBeInTheDocument();
    expect(screen.getByText("Login")).toBeInTheDocument();
    expect(screen.queryByText("Content管理")).not.toBeInTheDocument();
  });
  it("separates content and system navigation", () => {
    useAuthMock.mockReturnValue({ isLoading: false, isAuthenticated: true } as ReturnType<typeof useAuth>);
    useRoleMock.mockReturnValue({ isContentAdmin: true, isSystemAdmin: false } as ReturnType<typeof useRole>);
    const { rerender } = renderNavigation();
    expect(screen.getByText("Content管理")).toBeInTheDocument();
    expect(screen.queryByText("System管理")).not.toBeInTheDocument();
    useRoleMock.mockReturnValue({ isContentAdmin: false, isSystemAdmin: true } as ReturnType<typeof useRole>);
    rerender(<MemoryRouter><Navigation /></MemoryRouter>);
    expect(screen.queryByText("Content管理")).not.toBeInTheDocument();
    expect(screen.getByText("System管理")).toBeInTheDocument();
  });
});
