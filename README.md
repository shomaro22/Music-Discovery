# Music Discovery

## 概要

Music Discovery は、ユーザーの質問回答・過去の推薦履歴・Feedbackをもとに、音楽との新しい出会いを支援する推薦アプリケーションです。

通常推薦に加えて、1日3曲を提示する DailyDiscovery、推薦履歴、Feedback 学習、Spotifyを利用した候補曲管理、コンテンツ管理者・システム管理者向けの管理機能を備えています。

主な特徴:

- 質問回答をもとにしたユーザープロファイル生成
- `YOUR_MATCH` / `NEW_FAVORITE` / `TRY_SOMETHING_NEW`
  の3枠による通常推薦
- 1日3曲の Daily Discovery
- 推薦履歴・Daily Discovery 履歴の閲覧
- LIKE / DISLIKE 等の Feedback を推薦条件へ反映
- Genre / Artist / Song / Element / Atmosphere
  等を利用した推薦スコアリング
- Discovery Level に応じた探索度の調整
- 直近推薦曲の重複抑制、Artist 多様性、候補不足時の段階的緩和
- Spotify API を利用した Artist / Track 検索
- コンテンツ管理者による Artist / Song / Tag 管理
- システム管理者による User / Role / Log / Metrics / Alert 管理
- JWT + Refresh Token + CSRF 対策による認証
- Redis を利用した Rate Limit・推薦生成 Lock
- Health / Readiness / Metrics / Alert による運用監視
- k6・Playwright・Vitest・Go test によるテスト

## 想定ユーザー

- USER --- 質問回答、音楽推薦、DailyDiscovery、Feedback、推薦履歴、プロフィール管理
- CONTENT_ADMIN --- Artist / Song / Tag の管理、Spotifyからの楽曲・Artist 検索
- SYSTEM_ADMIN --- User / Role、Audit Log / System Log、Metrics / Alert の管理

## 主なフロー

1.  ユーザーが新規登録またはログイン
2.  質問に回答し、音楽嗜好を表す User Profile を生成・更新
3.  推薦を生成し、3種類の推薦枠から楽曲を提示
4.  推薦結果に Feedback を登録
5.  Feedback を Genre / Artist / Song / Element / Atmosphere 等の重みに反映
6.  次回以降の推薦で学習済みの重み、履歴、Discovery Level を利用
7.  Daily Discovery では1日3曲を生成し、過去の提示曲との重複や Artist 偏りを抑制
8.  USER は通常推薦・Daily Discovery の履歴を確認
9.  CONTENT_ADMIN は推薦候補となる Artist / Song / Tag を管理
10. SYSTEM_ADMIN はユーザー、権限、ログ、Metrics、Alert を管理

## 推薦の考え方

Music Discovery
では、単純な人気順ではなく、ユーザーの回答・Feedback・推薦履歴を組み合わせて候補曲を評価します。

通常推薦は次の3枠で構成されます。

---

推薦タイプ 目的

---

`YOUR_MATCH` 現在の嗜好との一致度を重視した推薦

`NEW_FAVORITE` 嗜好との接点を保ちながら、新しいArtist / Song を発見する推薦

`TRY_SOMETHING_NEW` Element / Atmosphere 等の接点を残しつつ探索性を高める推薦

---

推薦処理では、Genre 一致、関連 Genre、Artist / Song の学習済みWeight、Feedback Penalty、Novelty、過去の推薦履歴、Discovery Level
などを考慮します。

また、同じ曲を短期間に繰り返し提示しないこと、特定 Artistに偏らないこと、候補不足時には条件を段階的に緩和することを考慮した構成になっています。

## Daily Discovery

Daily Discovery は USER 向けの「今日の3曲」機能です。

- 原則として1日3曲を生成
- 同日中は既存の生成結果を再利用
- 通常枠と Challenge 枠を組み合わせて選曲
- 直近の推薦履歴を考慮して重複を抑制
- Artist の多様性を考慮
- Discovery Level に応じて探索範囲を調整
- 履歴一覧・詳細画面から過去の Daily Discovery を確認可能

## Feedback 学習

推薦詳細に対して登録された Feedbackは、次回以降の推薦条件へ反映されます。

主な学習対象:

- Genre
- Artist
- Song
- Element
- Atmosphere
- Discovery Adjustment

Feedbackによる増減値は保存され、推薦時のスコアリング・Penalty・探索度調整に利用されます。

## 技術スタック

### Backend

- Go 1.26.5
- Echo v4
- GORM
- PostgreSQL 17
- Redis
- pgx
- JWT Bearer Authentication
- bcrypt
- Spotify Web API

### Frontend

- React
- TypeScript
- Vite
- React Router
- Vitest
- Testing Library
- Playwright

### Infrastructure / Test

- Docker
- Docker Compose
- Vercel（Frontend 設定）
- Render（Backend API）
- k6
- Shell Security Test
- OpenAPI 3.1

## アーキテクチャ

Backend は責務ごとにレイヤを分離しています。

---

ディレクトリ 主な責務

---

`backend/entity` User、Question、Recommendation、Feedback等のドメインモデル・enum
`backend/controller` HTTP Request / Response、入力変換、Usecase 呼び出し
`backend/usecase` 推薦、DailyDiscovery、Feedback、認証、管理機能等の業務ロジック
`backend/repository` PostgreSQL / Redis に対するデータアクセス
`backend/middleware` JWT 認証、Role、Rate Limit、CSRF、Request ID、Metrics等
`backend/validator` User 入力等のバリデーション
`backend/spotify` Spotify API との通信
`backend/migrate` DB Migration、質問・開発用 User / Catalog の Seed
`backend/router` API Route と Middleware の組み立て
`backend/db` Database 接続
`backend/test` Backend Test 共通処理

`frontend/src/pages` 各画面
`frontend/src/api` Backend API 呼び出し
`frontend/src/auth` Frontend 認証状態管理
`frontend/src/router` Route Guard、Role 別 Routing

`tests/k6` Smoke / Load Test
`tests/security` Security Test

---

Backend の基本的な処理フローは次の通りです。

`Router → Middleware → Controller → Usecase → Repository → PostgreSQL / Redis`

外部の楽曲候補取得では、Usecase から Spotify Client を利用します。

## 認証・認可

認証は Access Token と Refresh Token を利用します。

- Access Token: JWT Bearer Token
- Refresh Token: Cookie
- Refresh Token Rotation / Reuse Detection
- CSRF Token による Cookie 利用 API の保護
- bcrypt による Password Hash
- Role Based Access Control
- Redis ベースの Rate Limit
- CORS Origin 制限
- Request Body Size 制限

主な Role:

Role 権限

---

`USER` 通常機能、Daily Discovery、履歴、Feedback
`CONTENT_ADMIN` Artist / Song / Tag / Spotify 管理
`SYSTEM_ADMIN` User / Role / Log / Monitoring 管理

## デプロイ

OpenAPI には Backend の Production Server として以下が定義されています。

- Backend API (Render): `https://music-discovery-1s2i.onrender.com`
- Health Check: `https://music-discovery-1s2i.onrender.com/health`
- Readiness Check: `https://music-discovery-1s2i.onrender.com/ready`

Frontend には `vercel.json` があり、React Router の SPA Routingに対応する rewrite が設定されています。

## 起動（Docker Compose）

リポジトリ直下で実行します。

```bash
docker compose up -d --build
```

起動するサービス:

サービス URL / Port

---

Frontend `http://localhost:5173`
Backend API `http://localhost:8080`
PostgreSQL `localhost:5433` → Container `5432`
Redis `localhost:6379`

状態確認:

```bash
docker compose ps
```

ログ確認:

```bash
docker compose logs -f api
```

停止:

```bash
docker compose down
```

DB Volume も削除して初期化する場合:

```bash
docker compose down -v
```

## Backend 起動（個別）

Docker Compose を利用せず Backend を直接起動する場合は、PostgreSQL / Redis を利用可能な状態にしたうえで環境変数を設定します。

```bash
cd backend
cp .env.example .env
go mod download
go run .
```

Docker Compose 内から接続する `.env.example` の既定値では PostgreSQLHost が `postgres`、Redis Host が `redis` になっています。
Backend を Mac上で直接起動する場合は、接続先をローカル向けに変更してください。

例:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5433/music_discovery?sslmode=disable
REDIS_ADDR=localhost:6379
CORS_ALLOW_ORIGIN=http://localhost:5173
```

## Frontend 起動（個別）

```bash
cd frontend
npm ci
npm run dev
```

開発サーバー:

```text
http://localhost:5173
```

Backend の接続先は `VITE_API_BASE_URL` で指定します。

例:

```env
VITE_API_BASE_URL=http://localhost:8080
```

## 主な環境変数

`backend/.env.example` を基準に設定します。

変数 説明

---

`APP_ENV` 実行環境
`APP_PORT` Backend API Port
`DATABASE_URL` PostgreSQL 接続文字列
`JWT_SECRET` Access Token 用 Secret
`REFRESH_TOKEN_SECRET` Refresh Token 用 Secret
`ACCESS_TOKEN_EXPIRES_IN` Access Token 有効期限
`REFRESH_TOKEN_EXPIRES_IN` Refresh Token 有効期限
`REDIS_ADDR` Redis 接続先
`CORS_ALLOW_ORIGIN` 許可する Frontend Origin。カンマ区切り対応
`SPOTIFY_CLIENT_ID` Spotify Client ID
`SPOTIFY_CLIENT_SECRET` Spotify Client Secret
`SPOTIFY_MARKET` Spotify Market。既定値は `JP`
`RECOMMENDATION_CANDIDATE_LIMIT` 推薦候補取得上限
`RECOMMENDATION_DUPLICATE_DAYS` 重複抑制期間
`RECOMMENDATION_RECENCY_DAYS` 履歴の参照期間
`RECOMMENDATION_GENERATION_WINDOW_SECONDS` 推薦生成競合判定 Window
`RECOMMENDATION_LOCK_TTL_SECONDS` 推薦 Lock TTL
`SERVICE_NAME` Service 名
`APP_VERSION` Application Version
`LOG_LEVEL` Log Level
`SLOW_REQUEST_THRESHOLD_MS` Slow Request 判定値
`HEALTH_CHECK_TIMEOUT_MS` Health Check Timeout
`SYSTEM_LOG_RETENTION_DAYS` System Log 保持日数
`ERROR_SECURITY_LOG_RETENTION_DAYS` Error / Security Log 保持日数
`AUDIT_LOG_RETENTION_DAYS` Audit Log 保持日数
`ALERT_ERROR_RATE_THRESHOLD` Error Rate Alert 閾値
`ALERT_P95_THRESHOLD_MS` Response p95 Alert 閾値
`ALERT_EVALUATION_INTERVAL_SECONDS` Alert 評価間隔
`ALERT_NOTIFICATION_TIMEOUT_MS` Alert 通知 Timeout
`ALERT_WEBHOOK_URL` Alert 通知先 Webhook

Spotify の認証情報が必要な管理機能を利用する場合は `SPOTIFY_CLIENT_ID` / `SPOTIFY_CLIENT_SECRET` を設定してください。

## 開発用アカウント

`SEED_DEVELOPMENT_USERS=true` で起動した場合、以下の開発用ユーザーがSeed されます。

Role Email Password

---

USER `user@example.com` `Password123!`
CONTENT_ADMIN `content-admin@example.com` `Password123!`
SYSTEM_ADMIN `system-admin@example.com` `Password123!`

Development User Seed は Production 環境では拒否される実装です。

開発用 Catalog を投入する場合は `SEED_DEVELOPMENT_CATALOG=true` を利用します。

## 主な API

### Monitoring

Method Endpoint 内容

---

GET `/health` Liveness Check
GET `/ready` PostgreSQL / Redis を含む Readiness Check

### Authentication / User

Method Endpoint 内容

---

POST `/api/auth/signup` 新規登録
POST `/api/auth/login` ログイン
POST `/api/auth/refresh` Access Token 更新
POST `/api/auth/logout` ログアウト
POST `/api/auth/password-reset/request` Password Reset 要求
POST `/api/auth/password-reset/confirm` Password Reset 確定
GET `/api/users/me` 自分の User 情報取得
PATCH `/api/users/me` Profile 更新
DELETE `/api/users/me` 退会

### Question / Recommendation

Method Endpoint 内容

---

GET `/api/questions` 質問一覧
GET `/api/user-answers/latest` 最新回答取得
POST `/api/user-answers` 回答保存
POST `/api/recommendations` 推薦生成
GET `/api/recommendations/latest` 最新推薦取得
GET `/api/recommendations/:recommendationId` 推薦詳細取得
GET `/api/recommendation-history` 推薦履歴一覧
GET `/api/recommendation-history/:recommendationId` 推薦履歴詳細

### Daily Discovery / Feedback

---

Method Endpoint 内容

---

GET `/api/daily-discovery` 今日の3曲
GET `/api/daily-discoveries` Daily Discovery 履歴
GET `/api/daily-discoveries/:recommendationId` Daily Discovery 詳細
PUT `/api/recommendation-details/:recommendationDetailId/feedback` Feedback 保存
GET `/api/recommendation-details/:recommendationDetailId/feedback` Feedback 取得

---

### Content Admin

Base path: `/api/admin`

- Artist CRUD
- Song CRUD
- Tag CRUD
- Spotify Artist Search
- Spotify Track Search

### System Admin

Base path: `/api/system`

- User 一覧・詳細・Status 変更
- Role 一覧・Role 変更
- Audit Log
- System Log
- Metrics
- Alert 一覧
- Alert Test

API の詳細な Request / Response Schema は `openapi.yaml` を参照してください。

## テスト

### Backend

```bash
cd backend
gofmt -w .
go vet ./...
go test ./...
```

Integration Test を個別に利用する場合は、対象テストの前提となる PostgreSQL 等を起動してください。

### Frontend

```bash
cd frontend
npm ci
npm test
npm run build
```

Type Check:

```bash
npm run typecheck
```

Coverage:

```bash
npm run test:coverage
```

CI 相当の Frontend Test:

```bash
npm run test:ci
```

### E2E

Frontend の Playwright Test:

```bash
cd frontend
npm run test:e2e
```

### k6 Smoke Test

API を起動後、リポジトリ直下から実行します。

```bash
k6 run tests/k6/smoke.js
```

既定の `BASE_URL`:

```text
http://localhost:8080
```

### k6 Load Test

必要な E2E 用アカウント情報を環境変数に設定したうえで実行します。

```bash
k6 run tests/k6/load.js
```

`tests/k6/config.js` では、通常 API の成功率や p95 Response Time に Threshold が定義されています。

### Security Test

Security Test では、JWT 改ざん、Refresh Token Rotation / Reuse、CSRF、SQL Injection Probe、CORS、XSS 回避実装、Logout 後 Token
無効化等を確認します。

例:

```bash
SECURITY_USER_EMAIL=user@example.com \
SECURITY_USER_PASSWORD='Password123!' \
sh tests/security/security.sh
```

## CI 用 Docker Compose

CI / E2E 向けには `docker-compose.ci.yml` が用意されています。

```bash
docker compose -f docker-compose.ci.yml up -d --build
```

CI 用構成では以下を行います。

- PostgreSQL / Redis の Health Check
- API `/health` の Health Check
- Development User / Catalog Seed
- Test 環境用 Rate Limit Capacity の拡張
- Frontend / API の起動

## Monitoring / Alert

公開 Monitoring Endpoint:

- `GET /health`
- `GET /ready`

SYSTEM_ADMIN 向け:

- `GET /api/system/metrics`
- `GET /api/system/alerts`
- `POST /api/system/alerts/test`

Alert では Error Rate、Response p95、依存サービス、Log
保存等の状態を監視する構成です。

## Project Structure

```text
Music-Discovery/
├── backend/
│   ├── controller/
│   ├── db/
│   ├── entity/
│   ├── middleware/
│   ├── migrate/
│   ├── repository/
│   ├── router/
│   ├── spotify/
│   ├── test/
│   ├── usecase/
│   ├── validator/
│   ├── main.go
│   └── server.go
├── frontend/
│   ├── e2e/
│   ├── scripts/
│   ├── src/
│   │   ├── api/
│   │   ├── auth/
│   │   ├── components/
│   │   ├── constants/
│   │   ├── pages/
│   │   └── router/
│   ├── package.json
│   └── vercel.json
├── tests/
│   ├── k6/
│   ├── security/
│   └── spotify-mock/
├── Dockerfile
├── docker-compose.yml
├── docker-compose.ci.yml
└── openapi.yaml
```

## API / 仕様

- OpenAPI: `openapi.yaml`
- Backend 環境変数サンプル: `backend/.env.example`
- DB Migration / Seed: `backend/migrate/`
- API Routing: `backend/router/router.go`
- 推薦ロジック: `backend/usecase/recommendation_usecase.go`
- 推薦スコアリング: `backend/usecase/recommendation_calculator.go`
- 推薦条件生成: `backend/usecase/recommendation_condition_builder.go`
- Daily Discovery: `backend/usecase/daily_discovery_usecase.go`
- Feedback: `backend/usecase/feedback_usecase.go`
- Monitoring / Alert: `backend/usecase/health_usecase.go`,
  `backend/usecase/metrics_usecase.go`,
  `backend/usecase/alert_monitor_usecase.go`

## 注意点

- `backend/.env` には Secret を含める可能性があるため、公開 Repository
  へ実値を Commit しないでください。
- Production では十分な長さのランダムな `JWT_SECRET` /
  `REFRESH_TOKEN_SECRET` を設定してください。
- Development User / Catalog Seed は本番用途では使用しないでください。
- `CORS_ALLOW_ORIGIN` は本番 Frontend の Origin のみに制限してください。
- Spotify 管理検索を利用する場合は Spotify API Credentials が必要です。
- PostgreSQL の公開 Port は Docker Compose 開発環境では `5433` です。
- API の正確な仕様は実装と `openapi.yaml` を基準にしてください。
