import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiRequestError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { SignupPage } from "./SignupPage";

vi.mock("../hooks/useAuth", () => ({ useAuth: vi.fn() }));
const mockedUseAuth = vi.mocked(useAuth);
const mockedSignUp = vi.fn<ReturnType<typeof useAuth>["signUp"]>();

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/signup"]}>
      <Routes>
        <Route path="/signup" element={<SignupPage />} />
        <Route path="/" element={<p>トップ画面</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve;
  });
  return { promise, resolve };
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText("名前"), "Taro");
  await user.type(screen.getByLabelText("メールアドレス"), "taro@example.com");
  await user.type(screen.getByLabelText("パスワード"), "password");
  await user.type(screen.getByLabelText("パスワード確認"), "password");
}

beforeEach(() => {
  mockedSignUp.mockReset();
  mockedUseAuth.mockReturnValue({
    user: null,
    accessToken: null,
    isAuthenticated: false,
    isLoading: false,
      authExpired: false,
    signUp: mockedSignUp,
    login: vi.fn(),
    logout: vi.fn(),
    clearAuth: vi.fn(),
  });
});

describe("SignupPage", () => {
  it("renders the branded signup UI and accessible fields", () => {
    renderPage();

    expect(
      screen.getByRole("heading", {
        name: "アカウントを作成",
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", {
        name: /Start Your\s*Discovery\./,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("あなたの次の一曲を、ここから見つけよう。"),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("名前")).toHaveAttribute(
      "autocomplete",
      "name",
    );
    expect(screen.getByLabelText("メールアドレス")).toHaveAttribute(
      "autocomplete",
      "email",
    );
    expect(screen.getByLabelText("パスワード")).toHaveAttribute(
      "autocomplete",
      "new-password",
    );
    expect(screen.getByLabelText("パスワード確認")).toHaveAttribute(
      "autocomplete",
      "new-password",
    );
    expect(
      screen.getByRole("button", { name: "アカウントを作成" }),
    ).toBeInTheDocument();
  });

  it("toggles password visibility without submitting", async () => {
    const user = userEvent.setup();
    renderPage();

    const password = screen.getByLabelText<HTMLInputElement>("パスワード");
    expect(password).toHaveAttribute("type", "password");

    await user.click(
      screen.getByRole("button", { name: "パスワードを表示する" }),
    );
    expect(password).toHaveAttribute("type", "text");
    expect(mockedSignUp).not.toHaveBeenCalled();

    await user.click(
      screen.getByRole("button", { name: "パスワードを非表示にする" }),
    );
    expect(password).toHaveAttribute("type", "password");
  });

  it("validates name boundaries", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(
      screen.getByLabelText("メールアドレス"),
      "user@example.com",
    );
    await user.type(screen.getByLabelText("パスワード"), "password");
    await user.type(screen.getByLabelText("パスワード確認"), "password");
    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));

    expect(
      screen.getByText("名前は1文字以上50文字以内で入力してください"),
    ).toBeInTheDocument();
    expect(mockedSignUp).not.toHaveBeenCalled();
  });

  it("shows an error for an invalid email format", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("名前"), "Taro");
    await user.type(screen.getByLabelText("メールアドレス"), "invalid-email");
    await user.type(screen.getByLabelText("パスワード"), "password");
    await user.type(screen.getByLabelText("パスワード確認"), "password");
    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));

    expect(
      screen.getByText("有効なメールアドレスを入力してください"),
    ).toBeInTheDocument();
    expect(mockedSignUp).not.toHaveBeenCalled();
  });

  it("shows an error when the password is too short", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("名前"), "Taro");
    await user.type(
      screen.getByLabelText("メールアドレス"),
      "taro@example.com",
    );
    await user.type(screen.getByLabelText("パスワード"), "1234567");
    await user.type(screen.getByLabelText("パスワード確認"), "1234567");
    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));

    expect(
      screen.getByText("パスワードは8文字以上で入力してください"),
    ).toBeInTheDocument();
    expect(mockedSignUp).not.toHaveBeenCalled();
  });

  it("rejects mismatched password confirmation", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("名前"), "Taro");
    await user.type(
      screen.getByLabelText("メールアドレス"),
      "user@example.com",
    );
    await user.type(screen.getByLabelText("パスワード"), "password");
    await user.type(screen.getByLabelText("パスワード確認"), "different");
    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));

    expect(screen.getByText("パスワードが一致しません")).toBeInTheDocument();
    expect(mockedSignUp).not.toHaveBeenCalled();
  });

  it("does not trim password and normalizes email", async () => {
    const user = userEvent.setup();
    renderPage();
    mockedSignUp.mockResolvedValueOnce(undefined);

    await user.type(screen.getByLabelText("名前"), " Taro ");
    await user.type(
      screen.getByLabelText("メールアドレス"),
      " USER@EXAMPLE.COM ",
    );
    await user.type(screen.getByLabelText("パスワード"), " 123456 ");
    await user.type(screen.getByLabelText("パスワード確認"), " 123456 ");
    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));

    await waitFor(() =>
      expect(mockedSignUp).toHaveBeenCalledWith({
        name: "Taro",
        email: "user@example.com",
        password: " 123456 ",
      }),
    );
  });

  it("prevents duplicate submission and redirects after success", async () => {
    const deferred = createDeferred<void>();
    mockedSignUp.mockReturnValueOnce(deferred.promise);
    const user = userEvent.setup();
    renderPage();
    await fillValidForm(user);

    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));
    await waitFor(() => expect(mockedSignUp).toHaveBeenCalledTimes(1));

    const pendingButton = screen.getByRole("button", { name: "登録中..." });
    expect(pendingButton).toBeDisabled();
    await user.click(pendingButton);
    expect(mockedSignUp).toHaveBeenCalledTimes(1);

    deferred.resolve();
    expect(await screen.findByText("トップ画面")).toBeInTheDocument();
  });

  it("shows duplicate email error beside the email field and clears passwords", async () => {
    mockedSignUp.mockRejectedValueOnce(
      new ApiRequestError({
        status: 409,
        code: "EMAIL_ALREADY_EXISTS",
        message: "このメールアドレスは既に登録されています",
        request_id: "req-1",
      }),
    );
    const user = userEvent.setup();
    renderPage();
    await fillValidForm(user);

    const email = screen.getByLabelText<HTMLInputElement>("メールアドレス");
    const emailField = email.closest(".field");
    const password = screen.getByLabelText<HTMLInputElement>("パスワード");
    const confirmation =
      screen.getByLabelText<HTMLInputElement>("パスワード確認");

    expect(emailField).not.toBeNull();

    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));

    const duplicateError = await screen.findByText(
      "このメールアドレスは既に登録されています",
    );

    expect(
      within(emailField as HTMLElement).getByText(
        "このメールアドレスは既に登録されています",
      ),
    ).toBe(duplicateError);
    expect(email).toHaveAccessibleDescription(
      "このメールアドレスは既に登録されています",
    );
    expect(
      screen.queryByText("このメールアドレスは既に登録されています", {
        selector: ".api-alert",
      }),
    ).not.toBeInTheDocument();
    expect(password).toHaveValue("");
    expect(confirmation).toHaveValue("");
  });

  it("shows a general API error", async () => {
    mockedSignUp.mockRejectedValueOnce(
      new ApiRequestError({
        status: 500,
        code: "INTERNAL_ERROR",
        message: "サーバー内部でエラーが発生しました",
        request_id: "req-2",
      }),
    );
    const user = userEvent.setup();
    renderPage();
    await fillValidForm(user);

    await user.click(screen.getByRole("button", { name: "アカウントを作成" }));

    const apiError =
      await screen.findByText("サーバー内部でエラーが発生しました");
    expect(apiError).toHaveClass("api-alert");
    expect(apiError).toHaveAttribute("role", "alert");
  });
});
