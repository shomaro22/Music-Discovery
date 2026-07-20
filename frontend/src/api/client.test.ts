import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

beforeEach(() => {
  vi.resetModules();
  document.cookie = "csrf_token=test-csrf; path=/";
});

afterEach(() => {
  vi.restoreAllMocks();
  document.cookie = "csrf_token=; Max-Age=0; path=/";
});

describe("refreshAccessToken", () => {
  it("returns the backend RefreshResponse without inventing expires_in", async () => {
    const response = { access_token: "new-token", token_type: "Bearer" as const, expires_in: 123 };
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify(response), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));
    const { refreshAccessToken } = await import("./client");
    await expect(refreshAccessToken()).resolves.toEqual(response);
  });

  it("uses one in-flight Refresh request for concurrent callers", async () => {
    let resolveResponse!: (response: Response) => void;
    const pending = new Promise<Response>((resolve) => { resolveResponse = resolve; });
    const fetchMock = vi.spyOn(globalThis, "fetch").mockReturnValue(pending);
    const { refreshAccessToken } = await import("./client");

    const first = refreshAccessToken();
    const second = refreshAccessToken();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const payload = { access_token: "shared-token", token_type: "Bearer" as const, expires_in: 240 };
    resolveResponse(new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } }));
    await expect(Promise.all([first, second])).resolves.toEqual([payload, payload]);
  });
});
