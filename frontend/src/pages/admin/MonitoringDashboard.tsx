import { useCallback, useEffect, useState } from "react";
import { getMonitoringAlerts, getMonitoringMetrics } from "../../api/admin";
import type { AlertResponse, MonitoringMetrics } from "../../types/admin";

const alertStatusLabel = (status: string) => status === "firing" ? "要確認" : "正常";
const severityLabel = (severity: string) => ({ critical: "重大", warning: "警告", info: "情報" }[severity] ?? severity);

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
  if (!metrics) return <section className="monitoring-dashboard"><h2>Monitoring Dashboard</h2><p>{error || "読み込み中..."}</p></section>;
  const app = metrics.application;
  return <section className="monitoring-dashboard">
    <div className="admin-page-heading"><div><p className="admin-eyebrow">System Health</p><h2>Monitoring Dashboard</h2><p>APIの利用状況・エラー・処理速度・DB状態を確認する画面です。30秒ごとに自動更新されます。</p></div><button type="button" onClick={() => void load()}>今すぐ更新</button></div>
    {error && <p className="admin-alert" role="alert">{error}</p>}
    <div className="admin-summary-grid">
      <article><span>APIリクエスト数</span><strong>{app.request_count}</strong><small>起動後に受け付けたHTTPリクエスト</small></article>
      <article><span>エラー率</span><strong>{(app.error_rate * 100).toFixed(2)}%</strong><small>HTTPエラーになったリクエストの割合</small></article>
      <article><span>応答時間 p95</span><strong>{app.response_time_p95_ms.toFixed(1)} ms</strong><small>95%のリクエストがこの時間以内に完了</small></article>
      <article><span>遅いリクエスト</span><strong>{app.slow_request_count}</strong><small>基準時間を超えたAPI処理</small></article>
      <article><span>DB接続数</span><strong>{metrics.database.open_connections}</strong><small>PostgreSQLへの現在のOpen Connection</small></article>
      <article><span>Goroutine</span><strong>{metrics.runtime.goroutines}</strong><small>Goで現在動作している並行処理数</small></article>
    </div>
    <section className="monitoring-section">
      <div><h3>機能別の処理件数</h3><p>ユーザー操作や推薦処理が正常に動いているかを見るための集計です。</p></div>
      <table className="monitoring-table"><tbody>
        <tr><th>ログイン</th><td>成功 {app.login_successes} / 失敗 {app.login_failures}</td></tr>
        <tr><th>推薦生成</th><td>成功 {app.recommendation_count} / 失敗 {app.recommendation_failures} / Fallback {app.fallback_count}</td></tr>
        <tr><th>Feedback</th><td>{app.feedback_count} 件</td></tr>
        <tr><th>Daily Discovery</th><td>{app.daily_discovery_count} 件</td></tr>
        <tr><th>Spotify API</th><td>Error {app.spotify_api_errors} / Rate Limit {app.rate_limit_exceeded}</td></tr>
        <tr><th>System Log保存失敗</th><td>{app.system_log_save_failures} 件</td></tr>
      </tbody></table>
    </section>
    <section className="monitoring-section">
      <div><h3>Alerts</h3><p>設定したしきい値を超えた項目を表示します。「要確認」がある場合はMessageと値を確認してください。</p></div>
      <table className="monitoring-table"><thead><tr><th>状態</th><th>重要度</th><th>監視項目</th><th>内容</th><th>現在値 / しきい値</th></tr></thead><tbody>
        {(alerts?.alerts ?? []).map((alert) => <tr key={alert.code}><td>{alertStatusLabel(alert.status)}</td><td>{severityLabel(alert.severity)}</td><td>{alert.code}</td><td>{alert.message}</td><td>{alert.value} / {alert.threshold}</td></tr>)}
      </tbody></table>
    </section>
    <p className="monitoring-meta">API Version: {metrics.version} / 最終更新: {new Date(metrics.generated_at).toLocaleString()}</p>
  </section>;
}
