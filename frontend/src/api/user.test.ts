import { afterEach, describe, expect, it, vi } from "vitest";
import { login, signUp } from "./user";
import { ApiRequestError } from "./client";

afterEach(() => vi.restoreAllMocks());

describe("user API", () => {
  it("posts signup JSON", async () => {
    const response = { id: "id", name: "Taro", email: "taro@example.com", role: "user", status: "active", created_at: "now" };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify(response), { status: 201, headers: { "Content-Type": "application/json" } }));
    await expect(signUp({ name: "Taro", email: "taro@example.com", password: "password" })).resolves.toEqual(response);
    expect(fetchMock).toHaveBeenCalledWith("http://localhost:8080/api/auth/signup", expect.objectContaining({ method: "POST", credentials: "include" }));
  });

  it("posts login JSON with credentials include", async () => {
    const response = { access_token: "access", token_type: "Bearer", expires_in: 300, user: { id: "id", name: "Taro", email: "taro@example.com", role: "user", status: "active" } };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify(response), { status: 200, headers: { "Content-Type": "application/json" } }));
    await expect(login({ email: "taro@example.com", password: "password" })).resolves.toEqual(response);
    expect(fetchMock).toHaveBeenCalledWith("http://localhost:8080/api/auth/login", expect.objectContaining({
      method: "POST", credentials: "include", body: JSON.stringify({ email: "taro@example.com", password: "password" }), headers: expect.objectContaining({ "Content-Type": "application/json" }),
    }));
  });

  it("throws ApiRequestError", async () => {
    const apiError = { status: 401, code: "INVALID_CREDENTIALS", message: "認証失敗", request_id: "req-1" };
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify(apiError), { status: 401, headers: { "Content-Type": "application/json" } }));
    await expect(login({ email: "a@b.com", password: "password" })).rejects.toBeInstanceOf(ApiRequestError);
  });
});
