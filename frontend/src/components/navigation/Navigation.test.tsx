import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuth } from "../../hooks/useAuth";
import { useRole } from "../../hooks/useRole";
import { Navigation } from "./Navigation";

vi.mock("../../hooks/useAuth", () => ({ useAuth: vi.fn() }));
vi.mock("../../hooks/useRole", () => ({ useRole: vi.fn() }));

const useAuthMock = vi.mocked(useAuth);
const useRoleMock = vi.mocked(useRole);

beforeEach(() => {
  useAuthMock.mockReturnValue({
    isAuthenticated: true,
    isLoading: false,
    user: { role: "user" },
  } as ReturnType<typeof useAuth>);
});

describe("Navigation Daily Discovery role", () => {
  it("一般ユーザーにDaily Discovery導線を表示する", () => {
    useRoleMock.mockReturnValue({
      isUser: true,
      isContentAdmin: false,
      isSystemAdmin: false,
    } as ReturnType<typeof useRole>);

    render(<MemoryRouter><Navigation /></MemoryRouter>);

    expect(screen.getByRole("link", { name: "Daily Discovery" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "履歴" })).toBeInTheDocument();
  });

  it("管理者にDaily Discovery導線を表示しない", () => {
    useAuthMock.mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      user: { role: "content_admin" },
    } as ReturnType<typeof useAuth>);
    useRoleMock.mockReturnValue({
      isUser: false,
      isContentAdmin: true,
      isSystemAdmin: false,
    } as ReturnType<typeof useRole>);

    render(<MemoryRouter><Navigation /></MemoryRouter>);

    expect(screen.queryByRole("link", { name: "Daily Discovery" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "履歴" })).not.toBeInTheDocument();
  });
});
