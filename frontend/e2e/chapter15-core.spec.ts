import { createHmac, randomUUID } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import { expect, request as playwrightRequest, test, type APIRequestContext, type Page } from "@playwright/test";

function requiredEnv(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required for Chapter 15 E2E tests`);
  return value;
}

const apiURL = process.env.E2E_API_URL?.trim() || "http://localhost:8080";
const runID = requiredEnv("E2E_RUN_ID").replace(/[^a-zA-Z0-9_-]/g, "-");
const contentAdminEmail = requiredEnv("E2E_CONTENT_ADMIN_EMAIL");
const contentAdminPassword = requiredEnv("E2E_CONTENT_ADMIN_PASSWORD");
const systemAdminEmail = requiredEnv("E2E_SYSTEM_ADMIN_EMAIL");
const systemAdminPassword = requiredEnv("E2E_SYSTEM_ADMIN_PASSWORD");
const jwtSecret = process.env.E2E_JWT_SECRET?.trim() || "test-jwt-secret-at-least-32-characters-long";
const dailyPhase = requiredEnv("E2E_DAILY_PHASE");
const dailyExpectedDate = requiredEnv("E2E_DAILY_EXPECT_DATE");
const dailyUserEmail = scopedEmail(requiredEnv("E2E_DAILY_USER_EMAIL"), `daily-${runID}`);
const dailyUserPassword = requiredEnv("E2E_DAILY_USER_PASSWORD");
const dailySnapshotPath = scopedSnapshotPath(process.env.E2E_DAILY_SNAPSHOT?.trim() || ".e2e-daily-day1.json", runID);


function scopedEmail(baseEmail: string, suffix: string): string {
  const at = baseEmail.lastIndexOf("@");
  if (at <= 0) throw new Error(`invalid E2E email: ${baseEmail}`);
  return `${baseEmail.slice(0, at)}-${suffix}@${baseEmail.slice(at + 1)}`;
}

function scopedSnapshotPath(basePath: string, suffix: string): string {
  const jsonSuffix = ".json";
  if (basePath.endsWith(jsonSuffix)) {
    return `${basePath.slice(0, -jsonSuffix.length)}-${suffix}${jsonSuffix}`;
  }
  return `${basePath}-${suffix}`;
}

async function ensureUser(context: APIRequestContext, name: string, email: string, password: string): Promise<void> {
  const response = await context.post("/api/auth/signup", { data: { name, email, password } });
  if (response.status() !== 201 && response.status() !== 409) {
    throw new Error(`failed to prepare E2E user ${email}: ${response.status()} ${await response.text()}`);
  }
}

async function apiContext(): Promise<APIRequestContext> {
  return playwrightRequest.newContext({ baseURL: apiURL });
}

async function apiLogin(context: APIRequestContext, email: string, password: string): Promise<string> {
  const response = await context.post("/api/auth/login", { data: { email, password } });
  expect(response.status(), await response.text()).toBe(200);
  const body = (await response.json()) as { access_token: string };
  expect(body.access_token).toBeTruthy();
  return body.access_token;
}

function auth(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

function base64url(value: string): string {
  return Buffer.from(value).toString("base64url");
}

function expiredAccessToken(): string {
  const now = Math.floor(Date.now() / 1000);
  const header = base64url(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = base64url(JSON.stringify({
    iss: "music-discovery",
    aud: "music-discovery-frontend",
    sub: randomUUID(),
    role: "USER",
    tv: 0,
    iat: now - 3600,
    exp: now - 60,
  }));
  const unsigned = `${header}.${payload}`;
  const signature = createHmac("sha256", jwtSecret).update(unsigned).digest("base64url");
  return `${unsigned}.${signature}`;
}

async function loginInBrowser(page: Page, email: string, password: string): Promise<void> {
  await page.goto("/login");
  await page.getByLabel("メールアドレス").fill(email);
  await page.getByLabel("パスワード").fill(password);
  await page.getByRole("button", { name: "ログイン" }).click();
  await expect(page).toHaveURL(/\/$/);
}

async function answerQuestionsAndGenerateRecommendation(page: Page): Promise<void> {
  await page.getByRole("button", { name: "おすすめを探す" }).click();
  for (;;) {
    const selected = page.locator('[aria-checked="true"]');
    if ((await selected.count()) === 0) {
      await page.locator('[role="radio"], [role="checkbox"]').first().click();
    }
    const confirm = page.getByRole("button", { name: "回答を確認" });
    if (await confirm.isVisible()) {
      await confirm.click();
      break;
    }
    await page.getByRole("button", { name: "次へ" }).click();
  }
  await page.getByRole("button", { name: "この回答で保存" }).click();
  await expect(page.getByRole("heading", { name: "今日のおすすめ" })).toBeVisible();
}

async function signUpInBrowser(page: Page, name: string, email: string, password: string): Promise<void> {
  await page.goto("/signup");
  await page.getByLabel("名前").fill(name);
  await page.getByLabel("メールアドレス").fill(email);
  await page.getByLabel("パスワード", { exact: true }).fill(password);
  await page.getByLabel("パスワード確認").fill(password);
  await page.getByRole("button", { name: "アカウントを作成" }).click();
  await expect(page).toHaveURL(/\/$/);
}

test.describe.serial("Chapter 15 core E2E", () => {
  test("会員登録・ログアウト・通常ログイン・Refresh Token更新", async ({ page }) => {
    const email = `e2e-${Date.now()}@example.test`;
    const password = "StrongPassword123!";

    await signUpInBrowser(page, "E2E User", email, password);

    await page.getByRole("button", { name: "ログアウト" }).click();
    await expect(page).toHaveURL(/\/login/);

    await loginInBrowser(page, email, password);
    const refreshStatus = await page.evaluate(async (backendURL) => {
      const csrf = document.cookie.split("; ").find((entry) => entry.startsWith("csrf_token="))?.split("=")[1] ?? "";
      const response = await fetch(`${backendURL}/api/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: { "X-CSRF-Token": decodeURIComponent(csrf) },
      });
      return response.status;
    }, apiURL);
    expect(refreshStatus).toBe(200);
  });

  test("XSS文字列を利用者名に保存してもHTMLとして実行されない", async ({ page }) => {
    const email = `xss-e2e-${Date.now()}@example.test`;
    const password = "StrongPassword123!";
    const payload = '<img src=x onerror="window.__chapter15XSS=1">';

    await signUpInBrowser(page, payload, email, password);
    await expect(page.locator(".top-welcome-panel h2")).toContainText(payload);
    await expect(page.locator('img[src="x"]')).toHaveCount(0);
    const executed = await page.evaluate(() => Reflect.get(window, "__chapter15XSS"));
    expect(executed).toBeUndefined();
  });

  test("Content AdminでArtist・Song・TagをCRUDする", async () => {
    const context = await apiContext();
    try {
      const token = await apiLogin(context, contentAdminEmail, contentAdminPassword);
      const headers = auth(token);
      const suffix = Date.now().toString();

      const tagResponse = await context.post("/api/admin/tags", {
        headers,
        data: { name: `E2E Tag ${suffix}`, slug: `e2e-tag-${suffix}`, category: "genre", description: "chapter15 e2e" },
      });
      expect(tagResponse.status(), await tagResponse.text()).toBe(201);
      const tag = (await tagResponse.json()) as { id: string };

      const artistResponse = await context.post("/api/admin/artists", {
        headers,
        data: { spotify_artist_id: `e2e-artist-${suffix}`, name: `E2E Artist ${suffix}`, image_url: "", spotify_url: "https://open.spotify.com/artist/e2e", popularity: 60, tag_ids: [tag.id] },
      });
      expect(artistResponse.status(), await artistResponse.text()).toBe(201);
      const artist = (await artistResponse.json()) as { id: string };

      const artistUpdate = await context.patch(`/api/admin/artists/${artist.id}`, {
        headers,
        data: { spotify_artist_id: `e2e-artist-${suffix}`, name: `E2E Artist Updated ${suffix}`, image_url: "", spotify_url: "https://open.spotify.com/artist/e2e", popularity: 61, tag_ids: [tag.id] },
      });
      expect(artistUpdate.status(), await artistUpdate.text()).toBe(200);

      const songResponse = await context.post("/api/admin/songs", {
        headers,
        data: { artist_id: artist.id, spotify_track_id: `e2e-track-${suffix}`, name: `E2E Song ${suffix}`, album_name: "E2E Album", image_url: "", spotify_url: "https://open.spotify.com/track/e2e", preview_url: "", duration_ms: 180000, popularity: 50, release_date: "2026-01-01", explicit: false, tags: [{ tag_id: tag.id, weight: 1 }] },
      });
      expect(songResponse.status(), await songResponse.text()).toBe(201);
      const song = (await songResponse.json()) as { id: string };

      const songUpdate = await context.patch(`/api/admin/songs/${song.id}`, {
        headers,
        data: { artist_id: artist.id, spotify_track_id: `e2e-track-${suffix}`, name: `E2E Song Updated ${suffix}`, album_name: "E2E Album", image_url: "", spotify_url: "https://open.spotify.com/track/e2e", preview_url: "", duration_ms: 181000, popularity: 51, release_date: "2026-01-01", explicit: false, tags: [{ tag_id: tag.id, weight: 1 }] },
      });
      expect(songUpdate.status(), await songUpdate.text()).toBe(200);

      const tagUpdate = await context.patch(`/api/admin/tags/${tag.id}`, {
        headers,
        data: { name: `E2E Tag Updated ${suffix}`, slug: `e2e-tag-${suffix}`, category: "genre", description: "updated" },
      });
      expect(tagUpdate.status(), await tagUpdate.text()).toBe(200);

      expect((await context.delete(`/api/admin/songs/${song.id}`, { headers })).status()).toBe(204);
      expect((await context.delete(`/api/admin/artists/${artist.id}`, { headers })).status()).toBe(204);
      expect((await context.delete(`/api/admin/tags/${tag.id}`, { headers })).status()).toBe(204);
    } finally {
      await context.dispose();
    }
  });

  test("System AdminでUser停止・再開・Role・Audit Log・System Logを確認する", async () => {
    const context = await apiContext();
    try {
      const suffix = Date.now().toString();
      const signup = await context.post("/api/auth/signup", { data: { name: "System Target", email: `system-target-${suffix}@example.test`, password: "StrongPassword123!" } });
      expect(signup.status(), await signup.text()).toBe(201);
      const target = (await signup.json()) as { user: { id: string } };

      const token = await apiLogin(context, systemAdminEmail, systemAdminPassword);
      const headers = auth(token);
      expect((await context.get("/api/system/users?page=1&limit=20", { headers })).status()).toBe(200);

      const suspend = await context.patch(`/api/system/users/${target.user.id}/status`, { headers, data: { status: "SUSPENDED", reason: "chapter15 e2e" } });
      expect(suspend.status(), await suspend.text()).toBe(204);
      const resume = await context.patch(`/api/system/users/${target.user.id}/status`, { headers, data: { status: "ACTIVE", reason: "chapter15 e2e resume" } });
      expect(resume.status(), await resume.text()).toBe(204);

      const role = await context.patch(`/api/system/users/${target.user.id}/role`, { headers, data: { role: "CONTENT_ADMIN", reason: "chapter15 e2e" } });
      expect(role.status(), await role.text()).toBe(200);
      const roleBack = await context.patch(`/api/system/users/${target.user.id}/role`, { headers, data: { role: "USER", reason: "chapter15 e2e restore" } });
      expect(roleBack.status(), await roleBack.text()).toBe(200);

      expect((await context.get("/api/system/audit-logs?page=1&limit=20", { headers })).status()).toBe(200);
      expect((await context.get("/api/system/logs?page=1&limit=20", { headers })).status()).toBe(200);
    } finally {
      await context.dispose();
    }
  });

  test("不正Token・期限切れToken・利用停止Userを拒否する", async () => {
    const context = await apiContext();
    try {
      expect((await context.get("/api/users/me", { headers: auth("invalid.token.value") })).status()).toBe(401);
      expect((await context.get("/api/users/me", { headers: auth(expiredAccessToken()) })).status()).toBe(401);

      const suffix = Date.now().toString();
      const email = `suspended-e2e-${suffix}@example.test`;
      const password = "StrongPassword123!";
      const signup = await context.post("/api/auth/signup", { data: { name: "Suspended E2E", email, password } });
      expect(signup.status(), await signup.text()).toBe(201);
      const user = (await signup.json()) as { user: { id: string } };

      const systemToken = await apiLogin(context, systemAdminEmail, systemAdminPassword);
      const suspend = await context.patch(`/api/system/users/${user.user.id}/status`, { headers: auth(systemToken), data: { status: "SUSPENDED", reason: "security e2e" } });
      expect(suspend.status(), await suspend.text()).toBe(204);

      const login = await context.post("/api/auth/login", { data: { email, password } });
      expect(login.status(), await login.text()).toBe(403);
    } finally {
      await context.dispose();
    }
  });

  test("Feedback LIKE・MAYBE・SKIPとDaily Discovery同日固定を確認する", async ({ page }) => {
    const email = `daily-e2e-${Date.now()}@example.test`;
    const password = "StrongPassword123!";
    await signUpInBrowser(page, "Daily E2E User", email, password);
    await answerQuestionsAndGenerateRecommendation(page);

    await page.goto("/daily-discovery");
    await expect(page.getByRole("heading", { name: "今日の3曲" })).toBeVisible();
    const firstSongs = await page.locator(".recommendation-card").evaluateAll((cards) => cards.map((card) => card.textContent ?? ""));
    await page.reload();
    await expect(page.locator(".recommendation-card")).toHaveCount(3);
    const secondSongs = await page.locator(".recommendation-card").evaluateAll((cards) => cards.map((card) => card.textContent ?? ""));
    expect(secondSongs).toEqual(firstSongs);

    const cards = page.locator(".recommendation-card");
    const labels = [/好き LIKE/, /もう少し聴きたい MAYBE/, /今回は違う SKIP/];
    for (let index = 0; index < labels.length; index += 1) {
      const button = cards.nth(index).getByRole("button", { name: labels[index] });
      if (!(await button.getAttribute("aria-pressed"))?.includes("true")) await button.click();
      await expect(button).toHaveAttribute("aria-pressed", "true");
    }
  });

  test("Daily Discovery翌日更新で日付と3曲が変わる", async ({ page }) => {
    if (dailyPhase !== "day1" && dailyPhase !== "day2") {
      throw new Error(`E2E_DAILY_PHASE must be day1 or day2: ${dailyPhase}`);
    }

    if (dailyPhase === "day1") {
      const context = await apiContext();
      try {
        await ensureUser(context, "Daily Next Day User", dailyUserEmail, dailyUserPassword);
      } finally {
        await context.dispose();
      }
      await loginInBrowser(page, dailyUserEmail, dailyUserPassword);
      await answerQuestionsAndGenerateRecommendation(page);
    } else {
      await loginInBrowser(page, dailyUserEmail, dailyUserPassword);
    }

    await page.goto("/daily-discovery");
    await expect(page.getByRole("heading", { name: "今日の3曲" })).toBeVisible();
    await expect(page.locator(".recommendation-header p").last()).toHaveText(dailyExpectedDate);
    await expect(page.locator(".recommendation-card")).toHaveCount(3);
    const songs = await page.locator(".recommendation-card").evaluateAll((cards) =>
      cards.map((card) => card.textContent ?? ""),
    );

    if (dailyPhase === "day1") {
      writeFileSync(dailySnapshotPath, JSON.stringify(songs), "utf8");
      return;
    }

    const previousSongs = JSON.parse(readFileSync(dailySnapshotPath, "utf8")) as string[];
    expect(previousSongs).toHaveLength(3);
    expect(songs).toHaveLength(3);
    expect(songs).not.toEqual(previousSongs);
    expect(songs.filter((song) => previousSongs.includes(song))).toHaveLength(0);
  });

});
