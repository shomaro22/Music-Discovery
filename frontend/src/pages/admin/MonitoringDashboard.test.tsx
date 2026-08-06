import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getMonitoringAlerts, getMonitoringMetrics } from "../../api/admin";
import { MonitoringDashboard } from "./MonitoringDashboard";

vi.mock("../../api/admin", () => ({
  getMonitoringAlerts: vi.fn(),
  getMonitoringMetrics: vi.fn(),
}));

const metrics = {
  application: {
    request_count: 10, status_counts: { 200: 9, 500: 1 }, error_count: 1, error_rate: 0.1,
    response_time_p50_ms: 10, response_time_p95_ms: 20, response_time_p99_ms: 30,
    slow_request_count: 1, system_log_save_failures: 0, login_successes: 2, login_failures: 1,
    recommendation_count: 3, recommendation_failures: 1, fallback_count: 1, feedback_count: 4,
    daily_discovery_count: 2, spotify_api_errors: 0, rate_limit_exceeded: 0,
  },
  runtime: { goroutines: 5, heap_alloc_bytes: 1, heap_inuse_bytes: 2, system_bytes: 3 },
  database: { open_connections: 2, in_use_connections: 1, idle_connections: 1, wait_count: 0 },
  version: "test", generated_at: "2026-08-06T00:00:00Z",
};

afterEach(() => vi.clearAllMocks());

describe("MonitoringDashboard", () => {
  it("renders metrics and alerts after successful loading", async () => {
    vi.mocked(getMonitoringMetrics).mockResolvedValue(metrics);
    vi.mocked(getMonitoringAlerts).mockResolvedValue({ alerts: [{ code: "ERROR_RATE_HIGH", severity: "critical", status: "firing", message: "high", value: 0.1, threshold: 0.05, checked_at: "2026-08-06T00:00:00Z" }], generated_at: "2026-08-06T00:00:00Z" });
    render(<MonitoringDashboard />);
    expect(await screen.findByText("10.00%")).toBeInTheDocument();
    expect(screen.getByText("ERROR_RATE_HIGH")).toBeInTheDocument();
  });

  it("shows an error when loading fails", async () => {
    vi.mocked(getMonitoringMetrics).mockRejectedValue(new Error("failed"));
    vi.mocked(getMonitoringAlerts).mockRejectedValue(new Error("failed"));
    render(<MonitoringDashboard />);
    await waitFor(() => expect(screen.getByText("監視情報を取得できませんでした。")).toBeInTheDocument());
  });
});
