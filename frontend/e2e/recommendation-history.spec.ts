import { expect, request as playwrightRequest, type Page, test } from "@playwright/test";

const requiredEnvNames = ["E2E_RUN_ID", "E2E_USER_EMAIL", "E2E_USER_PASSWORD"] as const;
const missingRequiredEnv = requiredEnvNames.filter((name) => !process.env[name]?.trim());
test.skip(missingRequiredEnv.length > 0, `Recommendation history E2E requires: ${missingRequiredEnv.join(", ")}`);

function envOr(name: string, fallback: string): string {
  return process.env[name]?.trim() || fallback;
}

const apiURL = envOr("E2E_API_URL", "http://localhost:8080");
const runID = envOr("E2E_RUN_ID", "local").replace(/[^a-zA-Z0-9_-]/g, "-");
const userPassword = envOr("E2E_USER_PASSWORD", "unused");
const userEmail = scopedEmail(envOr("E2E_USER_EMAIL", "user@example.invalid"), `history-${runID}`);

function scopedEmail(baseEmail: string, suffix: string): string {
  const at = baseEmail.lastIndexOf("@");
  if (at <= 0) throw new Error(`invalid E2E email: ${baseEmail}`);
  return `${baseEmail.slice(0, at)}-${suffix}@${baseEmail.slice(at + 1)}`;
}

async function login(page: Page, email: string, password: string): Promise<void> {
  await page.goto("/login");
  await page.getByLabel("メールアドレス").fill(email);
  await page.getByLabel("パスワード", { exact: true }).fill(password);
  await page.getByRole("button", { name: "ログイン" }).click();
  await expect(page).toHaveURL(/\/$/);
}

async function answerAllQuestions(page: Page): Promise<void> {
  await page.getByRole("button", { name: "おすすめを探す" }).click();
  const saveButton = page.getByRole("button", { name: "この回答で保存" });

  for (let step = 0; step < 20; step += 1) {
    if (await saveButton.isVisible()) break;

    const choices = page.locator('[role="radio"], [role="checkbox"]');
    await expect(choices.first()).toBeVisible();
    if ((await page.locator('[aria-checked="true"]').count()) === 0) {
      await choices.first().click();
    }

    const confirm = page.getByRole("button", { name: "回答を確認" });
    if (await confirm.isVisible()) {
      await expect(confirm).toBeEnabled();
      await confirm.click();
      await expect(saveButton).toBeVisible();
      break;
    }

    const next = page.getByRole("button", { name: "次へ" });
    await expect(next).toBeVisible();
    await expect(next).toBeEnabled();
    await next.click();
  }

  await expect(saveButton).toBeVisible();
  await saveButton.click();
  await expect(page.getByRole("heading", { name: "今日のおすすめ" })).toBeVisible();
}

async function registerFeedbackForVisibleCards(page: Page): Promise<void> {
  const cards = page.locator(".recommendation-card");
  await expect(cards).toHaveCount(3);
  for (let index = 0; index < 3; index += 1) {
    const button = cards.nth(index).getByRole("button", { name: /好き LIKE/ });
    if ((await button.getAttribute("aria-pressed")) !== "true") {
      await button.click();
    }
    await expect(button).toHaveAttribute("aria-pressed", "true");
  }
}

async function openHistory(page: Page): Promise<void> {
  await page.goto("/");
  await page.getByRole("button", { name: "推薦履歴を開く" }).click();
  await expect(page.getByRole("heading", { name: "推薦履歴" })).toBeVisible();
}

test.describe.serial("推薦履歴", () => {
  test.beforeAll(async () => {
    const context = await playwrightRequest.newContext({ baseURL: apiURL });
    try {
      const response = await context.post("/api/auth/signup", {
        data: { name: "History E2E User", email: userEmail, password: userPassword },
      });
      if (response.status() !== 201 && response.status() !== 409) {
        throw new Error(`failed to prepare history E2E user: ${response.status()} ${await response.text()}`);
      }
    } finally {
      await context.dispose();
    }
  });

  test("通常推薦・Daily Discovery・Feedbackを準備して一覧と詳細を確認する", async ({
    page,
  }) => {
    await login(page, userEmail, userPassword);
    await answerAllQuestions(page);
    await registerFeedbackForVisibleCards(page);

    await page.goto("/");
    await page.getByRole("button", { name: "Daily Discoveryを開く" }).click();
    await expect(page.getByRole("heading", { name: "今日の3曲" })).toBeVisible();
    await registerFeedbackForVisibleCards(page);

    await openHistory(page);
    const cards = page.locator(".history-card");
    await expect(cards).toHaveCount(2);
    await expect(page.getByText("通常推薦").first()).toBeVisible();
    await expect(page.getByText("Daily Discovery").first()).toBeVisible();

    const times = await page.locator(".history-card time").allTextContents();
    expect(times.length).toBeGreaterThanOrEqual(2);

    await cards.first().click();
    await expect(page.getByRole("heading", { name: "推薦履歴の詳細" })).toBeVisible();
    await expect(page.locator(".recommendation-card")).toHaveCount(3);
    await expect(page.getByRole("link", { name: "Spotifyで聴く" }).first()).toHaveAttribute(
      "target",
      "_blank",
    );

    const firstCard = page.locator(".recommendation-card").first();
    await firstCard.getByRole("button", { name: /今回は違う SKIP/ }).click();
    await expect(
      firstCard.getByRole("button", { name: /今回は違う SKIP/ }),
    ).toHaveAttribute("aria-pressed", "true");
  });

  test("Source・期間・Pagination・Browser Backで条件を維持する", async ({ page }) => {
    await login(page, userEmail, userPassword);
    await openHistory(page);

    await page.getByLabel("推薦区分").selectOption("DAILY_DISCOVERY");
    await page.getByRole("combobox", { name: "開始日 年" }).fill("2026");
    await page.getByRole("combobox", { name: "開始日 月" }).fill("01");
    await page.getByRole("combobox", { name: "開始日 日" }).fill("01");
    await page.getByRole("combobox", { name: "終了日 年" }).fill("2026");
    await page.getByRole("combobox", { name: "終了日 月" }).fill("12");
    await page.getByRole("combobox", { name: "終了日 日" }).fill("31");
    await page.getByRole("button", { name: "検索" }).click();
    await expect(page).toHaveURL(/source=DAILY_DISCOVERY/);
    await expect(page).toHaveURL(/startDate=2026-01-01/);
    await expect(page).toHaveURL(/endDate=2026-12-31/);

    const next = page.getByRole("button", { name: "次へ" });
    if (await next.isEnabled()) {
      await next.click();
      await expect(page).toHaveURL(/page=2/);
    }

    await page.locator(".history-card").first().click();
    await page.goBack();
    await expect(page).toHaveURL(/source=DAILY_DISCOVERY/);

    await page.getByRole("button", { name: "リセット" }).click();
    await expect(page).not.toHaveURL(/source=/);
  });

});
