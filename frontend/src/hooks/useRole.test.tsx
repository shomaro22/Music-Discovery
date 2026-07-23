import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useAuth } from "./useAuth";
import { useRole } from "./useRole";

vi.mock("./useAuth", () => ({ useAuth: vi.fn() }));
const useAuthMock = vi.mocked(useAuth);

describe("useRole", () => {
  it("does not implicitly treat system admin as content admin", () => {
    useAuthMock.mockReturnValue({ user: { role: "system_admin" } } as ReturnType<typeof useAuth>);
    const { result } = renderHook(() => useRole());
    expect(result.current.isSystemAdmin).toBe(true);
    expect(result.current.isContentAdmin).toBe(false);
    expect(result.current.hasAnyRole(["content_admin"])).toBe(false);
  });
});
