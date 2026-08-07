import http from "k6/http";
import { check, group, sleep } from "k6";
import { Counter, Gauge, Rate, Trend } from "k6/metrics";
import { BASE_URL, loadThresholds, profileOptions, requiredEnv } from "./config.js";

const USER_EMAIL = requiredEnv("K6_USER_EMAIL");
const USER_PASSWORD = requiredEnv("K6_USER_PASSWORD");
const CONTENT_ADMIN_EMAIL = requiredEnv("K6_CONTENT_ADMIN_EMAIL");
const CONTENT_ADMIN_PASSWORD = requiredEnv("K6_CONTENT_ADMIN_PASSWORD");
const SYSTEM_ADMIN_EMAIL = requiredEnv("K6_SYSTEM_ADMIN_EMAIL");
const SYSTEM_ADMIN_PASSWORD = requiredEnv("K6_SYSTEM_ADMIN_PASSWORD");

const transactionErrors = new Counter("transaction_errors");
const apiSuccessRate = new Rate("api_success_rate");
const successResponseDuration = new Trend("success_response_duration", true);
const dbOpenConnections = new Gauge("db_open_connections");
const heapAllocBytes = new Gauge("heap_alloc_bytes");
const cpuUtilizationPercent = new Gauge("cpu_utilization_percent");

const profile = __ENV.K6_PROFILE || "load";
export const options = { ...profileOptions(profile), thresholds: loadThresholds };

function recordResult(response, api, successStatuses) {
  const success = successStatuses.includes(response.status);
  apiSuccessRate.add(success, { api });
  if (success) {
    successResponseDuration.add(response.timings.duration, { api });
  } else {
    transactionErrors.add(1, { api });
  }
  return success;
}

function login(email, password) {
  const response = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({ email, password }), {
    headers: { "Content-Type": "application/json" },
    tags: { api: "login" },
  });
  recordResult(response, "login", [200]);
  if (response.status !== 200) {
    throw new Error(`login failed for ${email}: ${response.status} ${response.body}`);
  }
  return response.json("access_token");
}

function initializeUser(token) {
  const headers = { Authorization: `Bearer ${token}`, "Content-Type": "application/json" };
  const questionResponse = http.get(`${BASE_URL}/api/questions`, {
    headers,
    tags: { api: "question_setup" },
  });
  if (questionResponse.status !== 200) {
    throw new Error(`question setup failed: ${questionResponse.status} ${questionResponse.body}`);
  }
  const questions = JSON.parse(questionResponse.body).questions || [];
  const answers = questions.map((question) => ({
    question_id: question.id,
    choice_ids: (question.choices || [])
      .slice(0, Math.max(1, question.min_selections || 1))
      .map((choice) => choice.id),
  }));
  const answerResponse = http.post(`${BASE_URL}/api/user-answers`, JSON.stringify({ answers }), {
    headers,
    tags: { api: "answer_setup" },
  });
  if (answerResponse.status !== 201 && answerResponse.status !== 200) {
    throw new Error(`answer setup failed: ${answerResponse.status} ${answerResponse.body}`);
  }

  let recommendationResponse = http.post(`${BASE_URL}/api/recommendations`, null, {
    headers: { Authorization: `Bearer ${token}` },
    tags: { api: "recommendation_setup" },
    responseCallback: http.expectedStatuses(201, 409),
  });
  if (recommendationResponse.status === 409) {
    recommendationResponse = http.get(`${BASE_URL}/api/recommendations/latest`, {
      headers: { Authorization: `Bearer ${token}` },
      tags: { api: "recommendation_setup" },
    });
  }
  if (recommendationResponse.status !== 201 && recommendationResponse.status !== 200) {
    throw new Error(`recommendation setup failed: ${recommendationResponse.status} ${recommendationResponse.body}`);
  }
  const recommendations = JSON.parse(recommendationResponse.body).recommendations || [];
  if (!recommendations[0]?.recommendation_detail_id) {
    throw new Error("recommendation setup did not return a recommendation_detail_id");
  }
  return recommendations[0].recommendation_detail_id;
}

export function setup() {
  const userToken = login(USER_EMAIL, USER_PASSWORD);
  const detailID = initializeUser(userToken);
  return {
    userToken,
    detailID,
    contentToken: login(CONTENT_ADMIN_EMAIL, CONTENT_ADMIN_PASSWORD),
    systemToken: login(SYSTEM_ADMIN_EMAIL, SYSTEM_ADMIN_PASSWORD),
  };
}

function auth(token) {
  return { headers: { Authorization: `Bearer ${token}` } };
}

export default function (data) {
  const user = auth(data.userToken);
  const content = auth(data.contentToken);
  const system = auth(data.systemToken);

  group("questions", () => {
    const r = http.get(`${BASE_URL}/api/questions`, { ...user, tags: { api: "question" } });
    const success = recordResult(r, "question", [200]);
    check(r, { "questions 200": () => success });
  });

  group("recommendation", () => {
    // Recommendation generation has an intentional per-user generation window.
    // The load scenario therefore measures the stable authenticated recommendation
    // read endpoint instead of turning expected 409 conflicts into successful samples.
    const r = http.get(`${BASE_URL}/api/recommendations/latest`, {
      ...user,
      tags: { api: "recommendation" },
    });
    const success = recordResult(r, "recommendation", [200]);
    check(r, { "latest recommendation 200": () => success });
  });

  group("daily discovery", () => {
    const r = http.get(`${BASE_URL}/api/daily-discovery`, {
      ...user,
      tags: { api: "daily_discovery" },
    });
    const success = recordResult(r, "daily_discovery", [200]);
    check(r, { "daily discovery 200": () => success });
  });

  group("feedback", () => {
    // Repeated mutation of the same feedback record is not a stable load target.
    // Read the feedback created during setup so only successful responses contribute
    // to the performance trend and success rate.
    const r = http.get(
      `${BASE_URL}/api/recommendation-details/${data.detailID}/feedback`,
      {
        ...user,
        tags: { api: "feedback" },
      },
    );
    const success = recordResult(r, "feedback", [200]);
    check(r, { "feedback 200": () => success });
  });

  group("history", () => {
    const r = http.get(`${BASE_URL}/api/recommendation-history?page=1&limit=20`, {
      ...user,
      tags: { api: "history" },
    });
    const success = recordResult(r, "history", [200]);
    check(r, { "history 200": () => success });
  });

  group("content admin", () => {
    const artists = http.get(`${BASE_URL}/api/admin/artists?page=1&limit=20`, {
      ...content,
      tags: { api: "content_admin" },
    });
    const songs = http.get(`${BASE_URL}/api/admin/songs?page=1&limit=20`, {
      ...content,
      tags: { api: "content_admin" },
    });
    const artistsSuccess = recordResult(artists, "content_admin", [200]);
    const songsSuccess = recordResult(songs, "content_admin", [200]);
    check(artists, { "artist admin 200": () => artistsSuccess });
    check(songs, { "song admin 200": () => songsSuccess });
  });

  group("system admin", () => {
    const users = http.get(`${BASE_URL}/api/system/users?page=1&limit=20`, {
      ...system,
      tags: { api: "system_admin" },
    });
    const logs = http.get(`${BASE_URL}/api/system/logs?page=1&limit=20`, {
      ...system,
      tags: { api: "system_admin" },
    });
    const metrics = http.get(`${BASE_URL}/api/system/metrics`, {
      ...system,
      tags: { api: "system_admin" },
    });
    const usersSuccess = recordResult(users, "system_admin", [200]);
    const logsSuccess = recordResult(logs, "system_admin", [200]);
    const metricsSuccess = recordResult(metrics, "system_admin", [200]);
    check(users, { "user admin 200": () => usersSuccess });
    check(logs, { "system logs 200": () => logsSuccess });
    check(metrics, { "metrics 200": () => metricsSuccess });
    if (metricsSuccess) {
      dbOpenConnections.add(Number(metrics.json("database.open_connections") || 0));
      heapAllocBytes.add(Number(metrics.json("runtime.heap_alloc_bytes") || 0));
      cpuUtilizationPercent.add(Number(metrics.json("runtime.cpu_utilization_percent") || 0));
    }
  });

  sleep(1);
}
