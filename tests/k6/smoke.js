import http from "k6/http";
import { check } from "k6";
import { BASE_URL, smokeThresholds } from "./config.js";

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: smokeThresholds,
};

export default function () {
  const response = http.get(`${BASE_URL}/api/health`);
  check(response, {
    "health returns success": (r) => r.status >= 200 && r.status < 300,
  });
}
