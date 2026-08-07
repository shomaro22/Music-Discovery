import { useCallback, useState, type ReactNode } from "react";
import {
  changeUserRole,
  getAuditLog,
  getSystemLog,
  getUser,
  listAuditLogs,
  listSystemLogs,
  listUsers,
  setUserStatus,
} from "../../api/admin";
import type { AdminUser, AuditLog, SystemLog } from "../../types/admin";
import { AdminLayout } from "../../components/admin/AdminLayout";
import { DateDropdownInput } from "../../components/common/DateDropdownInput";
import { AdminListPage } from "./AdminListPage";
import { MonitoringDashboard } from "./MonitoringDashboard";

type Tab = "users" | "audit" | "logs" | "monitoring";

const targetTypeLabel = (value: string): string => ({
  artist: "Artist",
  song: "Song",
  tag: "Tag",
  user: "User",
  role: "Role",
}[value] ?? (value || "不明"));


const roleLabel = (value: AdminUser["role"]): string => ({
  user: "一般ユーザー",
  content_admin: "Content管理者",
  system_admin: "System管理者",
}[value]);

const auditActionLabel = (value: string): string => ({
  artist_created: "Artistを登録",
  artist_updated: "Artistを編集",
  artist_deleted: "Artistを削除",
  song_created: "Songを登録",
  song_updated: "Songを編集",
  song_deleted: "Songを削除",
  tag_created: "Tagを登録",
  tag_updated: "Tagを編集",
  tag_deleted: "Tagを削除",
  user_suspended: "Userを停止",
  user_activated: "Userを再開",
  user_role_changed: "User Roleを変更",
}[value] ?? value);

function AuditTarget({ value }: { value: AuditLog }) {
  const name = value.target_name && value.target_name !== value.target_id ? value.target_name : "名称不明";
  return <div className="admin-target-cell"><strong>{targetTypeLabel(value.target_type)}: {name}</strong>{value.target_detail && <span>{value.target_detail}</span>}<small>ID: {value.target_id}</small></div>;
}

function SystemLogUser({ value }: { value: SystemLog }) {
  if (!value.user_id) return <span>-</span>;
  return <div className="admin-target-cell"><strong>{value.user_name || "User"}</strong>{value.user_email && <span>{value.user_email}</span>}<small>ID: {value.user_id}</small></div>;
}

const startOfDate = (value: string): string | undefined => value ? new Date(`${value}T00:00:00`).toISOString() : undefined;
const endOfDate = (value: string): string | undefined => value ? new Date(`${value}T23:59:59.999`).toISOString() : undefined;

export function SystemAdminPage() {
  const [tab, setTab] = useState<Tab>("users");
  return (
    <AdminLayout title="System管理" activeSection={tab} onSectionChange={(section) => setTab(section as Tab)}>
      {tab === "users" && <Users />}
      {tab === "audit" && <Audit />}
      {tab === "logs" && <Logs />}
      {tab === "monitoring" && <MonitoringDashboard />}
    </AdminLayout>
  );
}

function Users() {
  const [role, setRole] = useState("");
  const [status, setStatus] = useState("");
  const [deleted, setDeleted] = useState(false);
  const [detail, setDetail] = useState<AdminUser | null>(null);
  const [sort, setSort] = useState("created_at");
  const [order, setOrder] = useState("desc");
  const [refreshKey, setRefreshKey] = useState(0);
  const [pendingAction, setPendingAction] = useState<{ user: AdminUser; type: "status" | "role"; next: string } | null>(null);
  const load = useCallback(
    (keyword: string, page: number) => listUsers(keyword, page, { role, status, include_deleted: deleted, sort, order }),
    [role, status, deleted, sort, order],
  );

  return (
    <>
      <AdminListPage<AdminUser>
        title="User管理"
        load={load}
        filterKey={`${role}:${status}:${deleted}:${sort}:${order}:${refreshKey}`}
        filters={
          <>
            <select aria-label="権限" value={role} onChange={(event) => setRole(event.target.value)}>
              <option value="">全権限</option><option value="user">一般ユーザー</option><option value="content_admin">Content管理者</option><option value="system_admin">System管理者</option>
            </select>
            <select aria-label="状態" value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="">全状態</option><option value="active">active</option><option value="suspended">suspended</option>
            </select>
            <label className="admin-checkbox-row admin-filter-checkbox"><input className="admin-checkbox" type="checkbox" checked={deleted} onChange={(event) => setDeleted(event.target.checked)} />退会済みを含む</label>
            <select aria-label="並び順" value={sort} onChange={(event) => setSort(event.target.value)}><option value="created_at">登録日時</option><option value="name">名前</option><option value="email">Email</option></select>
            <select aria-label="昇順降順" value={order} onChange={(event) => setOrder(event.target.value)}><option value="desc">降順</option><option value="asc">昇順</option></select>
          </>
        }
        rowKey={(value) => value.id}
        columns={[
          { label: "名前", render: (value) => value.name },
          { label: "Email", render: (value) => value.email },
          { label: "Role", render: (value) => <span className={`admin-role-badge admin-role-${value.role}`}>{roleLabel(value.role)}</span> },
          { label: "状態", render: (value) => value.deleted_at ? `退会済み (${new Date(value.deleted_at).toLocaleString()})` : value.status },
        ]}
        actions={(value) => (
          <>
            <button type="button" onClick={() => void getUser(value.id).then(setDetail)}>詳細</button>
            {!value.deleted_at && <button type="button" onClick={() => setPendingAction({ user: value, type: "status", next: value.status === "active" ? "suspended" : "active" })}>{value.status === "active" ? "停止" : "再開"}</button>}
            {!value.deleted_at && <select className="admin-role-select" aria-label={`${value.email}のRole`} value={value.role} onChange={(event) => {
              const next = event.target.value as AdminUser["role"];
              if (next !== value.role) setPendingAction({ user: value, type: "role", next });
            }}><option value="user">一般ユーザー</option><option value="content_admin">Content管理者</option><option value="system_admin">System管理者</option></select>}
          </>
        )}
      />
      {pendingAction && <UserActionDialog action={pendingAction} onClose={() => setPendingAction(null)} onComplete={() => { setPendingAction(null); setRefreshKey((value) => value + 1); }} />}
      {detail && <DetailDialog title="User詳細" onClose={() => setDetail(null)}><dl><dt>名前</dt><dd>{detail.name}</dd><dt>Email</dt><dd>{detail.email}</dd><dt>Role</dt><dd><span className={`admin-role-badge admin-role-${detail.role}`}>{roleLabel(detail.role)}</span></dd><dt>状態</dt><dd>{detail.status}</dd><dt>停止理由</dt><dd>{detail.suspended_reason || "-"}</dd><dt>登録日時</dt><dd>{new Date(detail.created_at).toLocaleString()}</dd><dt>更新日時</dt><dd>{new Date(detail.updated_at).toLocaleString()}</dd></dl></DetailDialog>}
    </>
  );
}

function Audit() {
  const [action, setAction] = useState("");
  const [actor, setActor] = useState("");
  const [target, setTarget] = useState("");
  const [requestId, setRequestId] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [detail, setDetail] = useState<AuditLog | null>(null);
  const [sort, setSort] = useState("created_at");
  const [order, setOrder] = useState("desc");
  const load = useCallback((keyword: string, page: number) => listAuditLogs(keyword, page, { action, actor_id: actor, target_type: target, request_id: requestId, from: startOfDate(from), to: endOfDate(to), sort, order }), [action, actor, target, requestId, from, to, sort, order]);
  return <>
    <section className="admin-explainer"><strong>Audit Logとは</strong><p>管理者が「誰が・誰または何に・何をしたか」を確認する操作履歴です。操作対象はUUIDだけでなく、Userなら名前・メールアドレス、Artist/Song/Tagなら名称も表示します。</p></section>
    <AdminListPage<AuditLog> title="Audit Log" load={load} filterKey={`${action}:${actor}:${target}:${requestId}:${from}:${to}:${sort}:${order}`} filters={<>
      <input aria-label="Action" placeholder="Action（例: artist_updated）" value={action} onChange={(e) => setAction(e.target.value)} />
      <input aria-label="操作したUser ID" placeholder="操作したUser UUID" value={actor} onChange={(e) => setActor(e.target.value)} />
      <select aria-label="操作対象" value={target} onChange={(e) => setTarget(e.target.value)}><option value="">全操作対象</option><option value="artist">Artist</option><option value="song">Song</option><option value="tag">Tag</option><option value="user">User</option></select>
      <input aria-label="Request ID" placeholder="Request ID" value={requestId} onChange={(e) => setRequestId(e.target.value)} />
      <DateDropdownInput id="audit-from" label="開始日" value={from} onChange={setFrom} />
      <DateDropdownInput id="audit-to" label="終了日" value={to} onChange={setTo} />
      <select aria-label="並び順" value={sort} onChange={(e) => setSort(e.target.value)}><option value="created_at">日時</option><option value="action">Action</option></select>
      <select aria-label="昇順降順" value={order} onChange={(e) => setOrder(e.target.value)}><option value="desc">降順</option><option value="asc">昇順</option></select>
    </>} rowKey={(value) => value.id} columns={[
      { label: "操作した管理者", render: (value) => `${value.actor_name || value.actor_user_id} (${value.actor_role})` },
      { label: "操作内容", render: (value) => <div className="admin-action-cell"><strong>{auditActionLabel(value.action)}</strong><small>{value.action}</small></div> },
      { label: "操作対象", render: (value) => <AuditTarget value={value} /> },
      { label: "Request ID", render: (value) => value.request_id },
      { label: "日時", render: (value) => new Date(value.created_at).toLocaleString() },
    ]} actions={(value) => <button type="button" onClick={() => void getAuditLog(value.id).then(setDetail)}>詳細</button>} />
    {detail && <DetailDialog title="Audit Log詳細" onClose={() => setDetail(null)}><dl><dt>操作した管理者</dt><dd>{detail.actor_name} ({detail.actor_email}) / {detail.actor_role}</dd><dt>操作内容</dt><dd>{auditActionLabel(detail.action)} <small>({detail.action})</small></dd><dt>操作対象</dt><dd>{targetTypeLabel(detail.target_type)} / {detail.target_id}</dd><dt>理由</dt><dd>{detail.reason || "-"}</dd><dt>概要</dt><dd>{detail.summary || "-"}</dd><dt>Request ID</dt><dd>{detail.request_id}</dd></dl><h3>変更前</h3><pre>{JSON.stringify(detail.before, null, 2)}</pre><h3>変更後</h3><pre>{JSON.stringify(detail.after, null, 2)}</pre></DetailDialog>}
  </>;
}

function Logs() {
  const [level, setLevel] = useState(""); const [category, setCategory] = useState(""); const [event, setEvent] = useState(""); const [user, setUser] = useState(""); const [requestId, setRequestId] = useState(""); const [status, setStatus] = useState(""); const [service, setService] = useState(""); const [method, setMethod] = useState(""); const [path, setPath] = useState(""); const [from, setFrom] = useState(""); const [to, setTo] = useState(""); const [detail, setDetail] = useState<SystemLog | null>(null); const [sort, setSort] = useState("created_at"); const [order, setOrder] = useState("desc");
  const load = useCallback((keyword: string, page: number) => listSystemLogs(keyword, page, { level, category, event, user_id: user, request_id: requestId, status_code: status || undefined, external_service: service, method, path, from: startOfDate(from), to: endOfDate(to), sort, order }), [level, category, event, user, requestId, status, service, method, path, from, to, sort, order]);
  return <>
    <section className="admin-explainer"><strong>System Logとは</strong><p>APIや外部サービスで発生した処理・エラーの記録です。Userが紐づくログでは名前・メールアドレス・UUIDを併記し、誰の処理なのか確認できます。Request IDは1回のHTTPリクエストを追跡する識別子です。</p></section>
    <AdminListPage<SystemLog> title="System Log" load={load} filterKey={`${level}:${category}:${event}:${user}:${requestId}:${status}:${service}:${method}:${path}:${from}:${to}:${sort}:${order}`} filters={<>
      <input aria-label="Level" placeholder="Level（INFO / ERROR）" value={level} onChange={(e) => setLevel(e.target.value)} />
      <input aria-label="Category" placeholder="Category" value={category} onChange={(e) => setCategory(e.target.value)} />
      <input aria-label="Event" placeholder="Event" value={event} onChange={(e) => setEvent(e.target.value)} />
      <input aria-label="User ID" placeholder="User UUID" value={user} onChange={(e) => setUser(e.target.value)} />
      <input aria-label="Request ID" placeholder="Request ID" value={requestId} onChange={(e) => setRequestId(e.target.value)} />
      <input aria-label="HTTP Status" type="number" placeholder="HTTP Status" value={status} onChange={(e) => setStatus(e.target.value)} />
      <input aria-label="External Service" placeholder="External Service" value={service} onChange={(e) => setService(e.target.value)} />
      <select aria-label="HTTP Method" value={method} onChange={(e) => setMethod(e.target.value)}><option value="">全Method</option><option>GET</option><option>POST</option><option>PATCH</option><option>DELETE</option></select>
      <input aria-label="Path" placeholder="Path" value={path} onChange={(e) => setPath(e.target.value)} />
      <DateDropdownInput id="system-log-from" label="開始日" value={from} onChange={setFrom} />
      <DateDropdownInput id="system-log-to" label="終了日" value={to} onChange={setTo} />
      <select aria-label="並び順" value={sort} onChange={(e) => setSort(e.target.value)}><option value="created_at">日時</option><option value="level">Level</option><option value="status_code">Status</option></select>
      <select aria-label="昇順降順" value={order} onChange={(e) => setOrder(e.target.value)}><option value="desc">降順</option><option value="asc">昇順</option></select>
    </>} rowKey={(value) => value.id} columns={[
      { label: "Level", render: (value) => value.level },
      { label: "User", render: (value) => <SystemLogUser value={value} /> },
      { label: "処理内容", render: (value) => `${value.category}/${value.event}` },
      { label: "HTTP", render: (value) => `${value.method || "-"} ${value.path || "-"} ${value.status_code || "-"}` },
      { label: "外部Service", render: (value) => value.external_service || "-" },
      { label: "概要", render: (value) => value.message },
      { label: "日時", render: (value) => new Date(value.created_at).toLocaleString() },
    ]} actions={(value) => <button type="button" onClick={() => void getSystemLog(value.id).then(setDetail)}>詳細</button>} />
    {detail && <DetailDialog title="System Log詳細" onClose={() => setDetail(null)}><dl><dt>Level</dt><dd>{detail.level}</dd><dt>Category / Event</dt><dd>{detail.category} / {detail.event}</dd><dt>Message</dt><dd>{detail.message}</dd><dt>HTTP</dt><dd>{detail.method} {detail.path} {detail.status_code}</dd><dt>Request ID</dt><dd>{detail.request_id}</dd><dt>External Service</dt><dd>{detail.external_service || "-"}</dd><dt>Duration</dt><dd>{detail.duration_ms ?? 0} ms</dd></dl><h3>Error detail</h3><pre>{detail.error_detail || "-"}</pre><h3>Metadata</h3><pre>{JSON.stringify(detail.metadata, null, 2)}</pre><h3>Stack trace</h3><pre>{detail.stack_trace || "-"}</pre></DetailDialog>}
  </>;
}

function UserActionDialog({ action, onClose, onComplete }: { action: { user: AdminUser; type: "status" | "role"; next: string }; onClose: () => void; onComplete: () => void }) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const isSuspend = action.type === "status" && action.next === "suspended";
  const title = action.type === "role" ? "Roleを変更" : isSuspend ? "ユーザーを停止" : "ユーザーを再開";
  const submit = async (): Promise<void> => {
    if (!reason.trim()) { setError("理由を入力してください。"); return; }
    setBusy(true); setError("");
    try {
      if (action.type === "role") await changeUserRole(action.user.id, action.next as AdminUser["role"], reason.trim());
      else await setUserStatus(action.user.id, action.next as "active" | "suspended", reason.trim());
      onComplete();
    } catch { setError("更新に失敗しました。時間をおいて再度お試しください。"); } finally { setBusy(false); }
  };
  return <div className="admin-dialog-backdrop" role="presentation"><section className="admin-dialog admin-action-dialog" role="dialog" aria-modal="true" aria-labelledby="user-action-title"><header><h2 id="user-action-title">{title}</h2><button type="button" onClick={onClose} disabled={busy}>閉じる</button></header><p><strong>{action.user.name}</strong>（{action.user.email}）</p><label className="admin-reason-field"><span>変更理由</span><small>監査ログに保存されるため、停止・再開・Role変更の理由を具体的に入力してください。</small><textarea autoFocus rows={5} value={reason} onChange={(event) => setReason(event.target.value)} placeholder="例：利用規約違反の確認が取れたため、一時的にアカウントを停止" /></label>{error && <p className="admin-alert" role="alert">{error}</p>}<div className="admin-dialog-actions"><button type="button" onClick={onClose} disabled={busy}>キャンセル</button><button className="admin-primary-button" type="button" onClick={() => void submit()} disabled={busy}>{busy ? "更新中..." : "実行する"}</button></div></section></div>;
}

function DetailDialog({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return <div className="admin-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.currentTarget === event.target) onClose(); }}><section className="admin-dialog" role="dialog" aria-modal="true" aria-label={title}><header><h2>{title}</h2><button type="button" onClick={onClose}>閉じる</button></header>{children}</section></div>;
}
