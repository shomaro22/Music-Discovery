import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

beforeEach(async () => {
  vi.resetModules();
  const csrf = await import("./csrf");
  csrf.setCSRFToken("test-csrf");
});

afterEach(() => {
  vi.restoreAllMocks();

});

describe("refreshAccessToken", () => {
  it("returns the backend RefreshResponse without inventing expires_in", async () => {
    const response = { access_token: "new-token", token_type: "Bearer" as const, expires_in: 123, csrf_token: "csrf-next" };
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

    const payload = { access_token: "shared-token", token_type: "Bearer" as const, expires_in: 240, csrf_token: "csrf-next" };
    resolveResponse(new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } }));
    await expect(Promise.all([first, second])).resolves.toEqual([payload, payload]);
  });
});

describe("request authentication recovery", () => {
  it("retries the original request once after a successful refresh", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ access_token: "new-token", token_type: "Bearer", expires_in: 300, csrf_token: "csrf-next" }), { status: 200, headers: { "Content-Type": "application/json" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
    const client = await import("./client");
    client.setAccessToken("old-token");
    await expect(client.request<{ ok: boolean }>("/api/users/me", { method: "GET" })).resolves.toEqual({ ok: true });
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("does not refresh again when the retried request is still 401", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ access_token: "new-token", token_type: "Bearer", expires_in: 300, csrf_token: "csrf-next" }), { status: 200, headers: { "Content-Type": "application/json" } }))
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

  it("keeps the access token and auth state on 403", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ status: 403, code: "FORBIDDEN", message: "forbidden", request_id: "" }), {
        status: 403,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const client = await import("./client");
    const listener = vi.fn();
    client.setAuthStateListener(listener);
    client.setAccessToken("token");

    await expect(client.request("/api/system/users", { method: "GET" })).rejects.toMatchObject({
      response: { status: 403, code: "FORBIDDEN" },
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(client.hasAccessToken()).toBe(true);
    expect(listener).not.toHaveBeenCalledWith(null, "expired");
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

describe("withdraw request authentication recovery", () => {
  it("does not refresh or retry for CURRENT_PASSWORD_MISMATCH", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ status: 401, code: "CURRENT_PASSWORD_MISMATCH", message: "password mismatch", request_id: "" }), {
        status: 401,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const client = await import("./client");
    const listener = vi.fn();
    client.setAuthStateListener(listener);
    client.setAccessToken("old-token");

    await expect(client.request("/api/users/me", {
      method: "DELETE",
      body: JSON.stringify({ current_password: "wrong" }),
    })).rejects.toMatchObject({ response: { code: "CURRENT_PASSWORD_MISMATCH" } });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(String(fetchMock.mock.calls[0][0])).toContain("/api/users/me");
    expect(client.hasAccessToken()).toBe(true);
    expect(listener).not.toHaveBeenCalledWith(null, "expired");
  });

  it("refreshes and retries DELETE exactly once for UNAUTHORIZED", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 401, code: "UNAUTHORIZED", message: "expired", request_id: "" }), { status: 401, headers: { "Content-Type": "application/json" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ access_token: "new-token", token_type: "Bearer", expires_in: 300, csrf_token: "csrf-next" }), { status: 200, headers: { "Content-Type": "application/json" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ message: "退会が完了しました。" }), { status: 200, headers: { "Content-Type": "application/json" } }));
    const client = await import("./client");
    client.setAccessToken("old-token");

    await expect(client.request<{ message: string }>("/api/users/me", {
      method: "DELETE",
      body: JSON.stringify({ current_password: "password123" }),
    })).resolves.toEqual({ message: "退会が完了しました。" });

    const withdrawCalls = fetchMock.mock.calls.filter(([input]) => String(input).endsWith("/api/users/me"));
    const refreshCalls = fetchMock.mock.calls.filter(([input]) => String(input).endsWith("/api/auth/refresh"));
    expect(withdrawCalls).toHaveLength(2);
    expect(refreshCalls).toHaveLength(1);
    expect(new Headers(withdrawCalls[1][1]?.headers).get("Authorization")).toBe("Bearer new-token");
  });
});


describe("request authentication expiration boundaries", () => {
  it("does not notify expiration for a login 401 without an Access Token", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ status: 401, code: "INVALID_CREDENTIALS", message: "invalid", request_id: "" }), {
        status: 401,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const client = await import("./client");
    const listener = vi.fn();
    client.setAuthStateListener(listener);

    await expect(client.request("/api/auth/login", { method: "POST" })).rejects.toBeInstanceOf(client.ApiRequestError);

    expect(listener).not.toHaveBeenCalledWith(null, "expired");
  });

  it("clears authentication for a 401 when the request had an Access Token", async () => {
    vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 401, code: "UNAUTHORIZED", message: "expired", request_id: "" }), { status: 401, headers: { "Content-Type": "application/json" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 401, code: "INVALID_REFRESH_TOKEN", message: "expired", request_id: "" }), { status: 401, headers: { "Content-Type": "application/json" } }));
    const client = await import("./client");
    const listener = vi.fn();
    client.setAuthStateListener(listener);
    client.setAccessToken("token");

    await expect(client.request("/api/users/me", { method: "GET" })).rejects.toBeInstanceOf(client.ApiRequestError);

    expect(client.hasAccessToken()).toBe(false);
    expect(listener).toHaveBeenLastCalledWith(null, "expired");
  });

  it("clears authentication for USER_SUSPENDED even without an Access Token", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ status: 403, code: "USER_SUSPENDED", message: "suspended", request_id: "" }), {
        status: 403,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const client = await import("./client");
    const listener = vi.fn();
    client.setAuthStateListener(listener);

    await expect(client.request("/api/recommendation-history", { method: "GET" })).rejects.toBeInstanceOf(client.ApiRequestError);

    expect(listener).toHaveBeenLastCalledWith(null, "expired");
  });
});

describe("request CSRF header", () => {
  it("automatically attaches the in-memory CSRF token to PATCH requests", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ id: "user-1" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const client = await import("./client");
    client.setAccessToken("access-token");

    await client.request("/api/users/me", {
      method: "PATCH",
      body: JSON.stringify({ name: "updated" }),
    });

    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers);
    expect(headers.get("X-CSRF-Token")).toBe("test-csrf");
    expect(headers.get("Authorization")).toBe("Bearer access-token");
  });

  it("automatically attaches the in-memory CSRF token to logout", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(null, { status: 204 }),
    );
    const client = await import("./client");

    await client.request("/api/auth/logout", { method: "POST" });

    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers);
    expect(headers.get("X-CSRF-Token")).toBe("test-csrf");
  });

  it("does not attach CSRF to signup/login", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const client = await import("./client");

    await client.request("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email: "user@example.com", password: "password" }),
    });

    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers);
    expect(headers.get("X-CSRF-Token")).toBeNull();
  });
});
