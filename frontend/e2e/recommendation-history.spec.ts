import { expect, test } from "@playwright/test";

test.describe("推薦履歴", () => {
  test.skip(!process.env.E2E_BASE_URL, "E2E_BASE_URL is not set");

  test("通常推薦へ絞り込み、詳細から戻って条件を維持する", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("メールアドレス").fill(process.env.E2E_USER_EMAIL ?? "user@example.com");
    await page.getByLabel("パスワード").fill(process.env.E2E_USER_PASSWORD ?? "password123");
    await page.getByRole("button", { name: "ログイン" }).click();

    await page.getByRole("button", { name: "推薦履歴を開く" }).click();
    await expect(page.getByRole("heading", { name: "推薦履歴" })).toBeVisible();
    await page.getByLabel("推薦区分").selectOption("NORMAL");
    await page.getByRole("button", { name: "検索" }).click();
    await expect(page).toHaveURL(/source=NORMAL/);

    const firstCard = page.locator(".history-card").first();
    await firstCard.click();
    await expect(page.getByRole("heading", { name: "推薦履歴の詳細" })).toBeVisible();
    await page.getByRole("link", { name: "推薦履歴へ戻る" }).click();
    await expect(page).toHaveURL(/source=NORMAL/);
  });
});
