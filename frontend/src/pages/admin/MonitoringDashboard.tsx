import { useCallback, useEffect, useState } from "react";
import { getMonitoringAlerts, getMonitoringMetrics } from "../../api/admin";
import type { AlertResponse, MonitoringMetrics } from "../../types/admin";

export function MonitoringDashboard() {
  const [metrics, setMetrics] = useState<MonitoringMetrics | null>(null);
  const [alerts, setAlerts] = useState<AlertResponse | null>(null);
  const [error, setError] = useState("");
  const load = useCallback(async () => {
    try {
      const [nextMetrics, nextAlerts] = await Promise.all([getMonitoringMetrics(), getMonitoringAlerts()]);
      setMetrics(nextMetrics); setAlerts(nextAlerts); setError("");
    } catch { setError("監視情報を取得できませんでした。"); }
  }, []);
  useEffect(() => { void load(); const id = window.setInterval(() => void load(), 30000); return () => window.clearInterval(id); }, [load]);
  if (!metrics) return <section><h2>Monitoring Dashboard</h2><p>{error || "読み込み中..."}</p></section>;
  const app = metrics.application;
  return <section>
    <div className="admin-page-heading"><div><p className="admin-eyebrow">Observability</p><h2>Monitoring Dashboard</h2></div><button type="button" onClick={() => void load()}>更新</button></div>
    {error && <p role="alert">{error}</p>}
    <div className="admin-summary-grid">
      <article><span>Requests</span><strong>{app.request_count}</strong></article>
      <article><span>Error Rate</span><strong>{(app.error_rate * 100).toFixed(2)}%</strong></article>
      <article><span>p95</span><strong>{app.response_time_p95_ms.toFixed(1)} ms</strong></article>
      <article><span>Slow Requests</span><strong>{app.slow_request_count}</strong></article>
      <article><span>DB Connections</span><strong>{metrics.database.open_connections}</strong></article>
      <article><span>Goroutines</span><strong>{metrics.runtime.goroutines}</strong></article>
    </div>
    <h3>業務メトリクス</h3>
    <table><tbody>
      <tr><th>Login 成功 / 失敗</th><td>{app.login_successes} / {app.login_failures}</td></tr>
      <tr><th>推薦 成功 / 失敗 / Fallback</th><td>{app.recommendation_count} / {app.recommendation_failures} / {app.fallback_count}</td></tr>
      <tr><th>Feedback / Daily Discovery</th><td>{app.feedback_count} / {app.daily_discovery_count}</td></tr>
      <tr><th>Spotify Error / Rate Limit</th><td>{app.spotify_api_errors} / {app.rate_limit_exceeded}</td></tr>
      <tr><th>System Log保存失敗</th><td>{app.system_log_save_failures}</td></tr>
    </tbody></table>
    <h3>Alerts</h3>
    <table><thead><tr><th>Status</th><th>Severity</th><th>Code</th><th>Message</th></tr></thead><tbody>
      {alerts?.alerts.map((alert) => <tr key={alert.code}><td>{alert.status}</td><td>{alert.severity}</td><td>{alert.code}</td><td>{alert.message}</td></tr>)}
    </tbody></table>
    <p>Version: {metrics.version} / 更新: {new Date(metrics.generated_at).toLocaleString()}</p>
  </section>;
}
