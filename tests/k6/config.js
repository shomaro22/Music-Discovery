export const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export function requiredEnv(name) {
  const value = (__ENV[name] || "").trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

export const smokeThresholds = {
  http_req_failed: ["rate<0.01"],
  http_req_duration: ["p(95)<500"],
};

export const loadThresholds = {
  http_req_failed: ["rate<0.01"],
  transaction_errors: ["count==0"],
  "api_success_rate{api:login}": ["rate>0.99"],
  "api_success_rate{api:question}": ["rate>0.99"],
  "api_success_rate{api:recommendation}": ["rate>0.99"],
  "api_success_rate{api:daily_discovery}": ["rate>0.99"],
  "api_success_rate{api:feedback}": ["rate>0.99"],
  "api_success_rate{api:history}": ["rate>0.99"],
  "api_success_rate{api:content_admin}": ["rate>0.99"],
  "api_success_rate{api:system_admin}": ["rate>0.99"],
  "success_response_duration{api:login}": ["p(95)<800"],
  "success_response_duration{api:question}": ["p(95)<500"],
  "success_response_duration{api:history}": ["p(95)<500"],
  "success_response_duration{api:feedback}": ["p(95)<800"],
  "success_response_duration{api:content_admin}": ["p(95)<500"],
  "success_response_duration{api:system_admin}": ["p(95)<500"],
  "success_response_duration{api:recommendation}": ["p(95)<3000"],
  "success_response_duration{api:daily_discovery}": ["p(95)<3000"],
};

export function profileOptions(profile) {
  switch (profile) {
    case "stress":
      return { stages: [
        { duration: "1m", target: 20 },
        { duration: "2m", target: 50 },
        { duration: "1m", target: 80 },
        { duration: "1m", target: 0 },
      ] };
    case "spike":
      return { stages: [
        { duration: "10s", target: 5 },
        { duration: "10s", target: 100 },
        { duration: "30s", target: 100 },
        { duration: "20s", target: 0 },
      ] };
    case "soak":
      return { stages: [
        { duration: "1m", target: 10 },
        { duration: "30m", target: 10 },
        { duration: "1m", target: 0 },
      ] };
    default:
      return { stages: [
        { duration: "30s", target: 10 },
        { duration: "2m", target: 10 },
        { duration: "30s", target: 0 },
      ] };
  }
}
