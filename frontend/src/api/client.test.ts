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

describe("request authentication recovery", () => {
  it("retries the original request once after a successful refresh", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ access_token: "new-token", token_type: "Bearer", expires_in: 300 }), { status: 200, headers: { "Content-Type": "application/json" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
    const client = await import("./client");
    client.setAccessToken("old-token");
    await expect(client.request<{ ok: boolean }>("/api/users/me", { method: "GET" })).resolves.toEqual({ ok: true });
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("does not refresh again when the retried request is still 401", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ access_token: "new-token", token_type: "Bearer", expires_in: 300 }), { status: 200, headers: { "Content-Type": "application/json" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 401, code: "UNAUTHORIZED", message: "unauthorized", request_id: "" }), { status: 401, headers: { "Content-Type": "application/json" } }));
    const client = await import("./client");
    client.setAccessToken("old-token");
    const listener = vi.fn();
    client.setAuthStateListener(listener);
    await expect(client.request("/api/users/me", { method: "GET" })).rejects.toBeInstanceOf(client.ApiRequestError);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(client.hasAccessToken()).toBe(false);
    expect(listener).toHaveBeenLastCalledWith(null, "expired");
  });

  it.each([403, 500])("does not refresh for status %s", async (status) => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ status, code: "ERROR", message: "failed", request_id: "" }), { status, headers: { "Content-Type": "application/json" } }));
    const client = await import("./client");
    client.setAccessToken("token");
    await expect(client.request("/api/users/me", { method: "GET" })).rejects.toBeInstanceOf(client.ApiRequestError);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("initializes expired auth state once when concurrent refresh fails", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = String(input);
      if (url.endsWith("/api/auth/refresh")) {
        return new Response(JSON.stringify({ status: 401, code: "INVALID_REFRESH_TOKEN", message: "expired", request_id: "" }), { status: 401, headers: { "Content-Type": "application/json" } });
      }
      return new Response(null, { status: 401 });
    });
    const client = await import("./client");
    const listener = vi.fn();
    client.setAuthStateListener(listener);
    client.setAccessToken("old-token");

    const results = await Promise.allSettled([
      client.request("/api/users/me", { method: "GET" }),
      client.request("/api/users/me", { method: "GET" }),
    ]);

    expect(results.every((result) => result.status === "rejected")).toBe(true);
    expect(listener.mock.calls.filter((call) => call[0] === null && call[1] === "expired")).toHaveLength(1);
  });

  it("notifies auth expiration when refresh authentication fails", async () => {
    vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 401, code: "INVALID_REFRESH_TOKEN", message: "expired", request_id: "" }), { status: 401, headers: { "Content-Type": "application/json" } }));
    const client = await import("./client");
    const listener = vi.fn();
    client.setAuthStateListener(listener);
    client.setAccessToken("old-token");
    await expect(client.request("/api/users/me", { method: "GET" })).rejects.toBeInstanceOf(client.ApiRequestError);
    expect(listener).toHaveBeenLastCalledWith(null, "expired");
  });
});
