import { afterEach, describe, expect, it, vi } from "vitest";
import { logout, signUp, withdraw } from "./user";
import { ApiRequestError } from "./client";
import { setCSRFToken } from "./csrf";

afterEach(() => vi.restoreAllMocks());

describe("signUp", () => {
  it("posts JSON to the signup endpoint", async () => {
    const response = {
      access_token: "access-token",
      token_type: "Bearer",
      expires_in: 300,
      csrf_token: "csrf-value",
      user: {
        id: "7c4ef6d5-cac5-4fca-b15f-f1641cdada23",
        name: "Taro",
        email: "taro@example.com",
        role: "user",
        status: "active",
        created_at: "2026-07-18T00:00:00Z",
      },
    };
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(
        new Response(JSON.stringify(response), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      );

    await expect(
      signUp({ name: "Taro", email: "taro@example.com", password: "password" }),
    ).resolves.toEqual(response);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];

    expect(url).toBe("http://localhost:8080/api/auth/signup");
    expect(options.method).toBe("POST");
    expect(options.credentials).toBe("include");
    expect(options.body).toBe(
      JSON.stringify({
        name: "Taro",
        email: "taro@example.com",
        password: "password",
      }),
    );

    const headers = new Headers(options.headers);
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("throws ApiRequestError for API errors", async () => {
    const apiError = {
      status: 409,
      code: "EMAIL_ALREADY_EXISTS",
      message: "登録済みです",
      request_id: "req-1",
    };
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify(apiError), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await expect(
      signUp({ name: "Taro", email: "taro@example.com", password: "password" }),
    ).rejects.toBeInstanceOf(ApiRequestError);
  });
});


describe("logout", () => {
  it("posts with the CSRF header and accepts 204", async () => {
    setCSRFToken("csrf-value");
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));

    await expect(logout()).resolves.toBeUndefined();

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://localhost:8080/api/auth/logout");
    expect(options.method).toBe("POST");
    expect(options.credentials).toBe("include");
    expect(new Headers(options.headers).get("X-CSRF-Token")).toBe("csrf-value");
  });

  it("bootstraps a CSRF pair before logout when memory state is empty", async () => {
    setCSRFToken(null);
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ csrf_token: "bootstrapped", refresh_available: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));

    await expect(logout()).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls[0][0]).toBe("http://localhost:8080/api/auth/csrf");
    const [, logoutOptions] = fetchMock.mock.calls[1] as [string, RequestInit];
    expect(new Headers(logoutOptions.headers).get("X-CSRF-Token")).toBe("bootstrapped");
  });

  it("re-bootstraps once when the server rejects a stale CSRF pair", async () => {
    setCSRFToken("stale");
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ status: 403, code: "CSRF_TOKEN_INVALID", message: "invalid", request_id: "req" }), {
          status: 403,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ csrf_token: "fresh", refresh_available: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));

    await expect(logout()).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledTimes(3);
    const [, retryOptions] = fetchMock.mock.calls[2] as [string, RequestInit];
    expect(new Headers(retryOptions.headers).get("X-CSRF-Token")).toBe("fresh");
  });
});


describe("withdraw", () => {
  it("deletes the current user with snake_case current_password", async () => {
    setCSRFToken("csrf-value");
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ message: "退会が完了しました。" }), { status: 200, headers: { "Content-Type": "application/json" } }));
    await expect(withdraw({ current_password: "password" })).resolves.toEqual({ message: "退会が完了しました。" });
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://localhost:8080/api/users/me");
    expect(options.method).toBe("DELETE");
    expect(options.body).toBe(JSON.stringify({ current_password: "password" }));
    expect(new Headers(options.headers).get("X-CSRF-Token")).toBe("csrf-value");
  });
});
