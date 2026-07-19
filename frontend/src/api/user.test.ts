import { afterEach, describe, expect, it, vi } from "vitest";
import { signUp } from "./user";
import { ApiRequestError } from "./client";

afterEach(() => vi.restoreAllMocks());

describe("signUp", () => {
  it("posts JSON to the signup endpoint", async () => {
    const response = {
      id: "7c4ef6d5-cac5-4fca-b15f-f1641cdada23",
      name: "Taro",
      email: "taro@example.com",
      role: "user",
      status: "active",
      created_at: "2026-07-18T00:00:00Z",
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
