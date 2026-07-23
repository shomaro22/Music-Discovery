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
import { AdminListPage } from "./AdminListPage";

type Tab = "users" | "audit" | "logs";

export function SystemAdminPage() {
  const [tab, setTab] = useState<Tab>("users");
  return (
    <AdminLayout title="System管理" activeSection={tab} onSectionChange={(section) => setTab(section as Tab)}>
      {tab === "users" && <Users />}
      {tab === "audit" && <Audit />}
      {tab === "logs" && <Logs />}
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
  const load = useCallback(
    (keyword: string, page: number) => listUsers(keyword, page, { role, status, include_deleted: deleted, sort, order }),
    [role, status, deleted, sort, order],
  );

  return (
    <>
      <AdminListPage<AdminUser>
        title="User管理"
        load={load}
        filterKey={`${role}:${status}:${deleted}:${sort}:${order}`}
        filters={
          <>
            <select aria-label="権限" value={role} onChange={(event) => setRole(event.target.value)}>
              <option value="">全権限</option><option value="user">user</option><option value="content_admin">content_admin</option><option value="system_admin">system_admin</option>
            </select>
            <select aria-label="状態" value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="">全状態</option><option value="active">active</option><option value="suspended">suspended</option>
            </select>
            <label><input type="checkbox" checked={deleted} onChange={(event) => setDeleted(event.target.checked)} />退会済みを含む</label>
            <select aria-label="並び順" value={sort} onChange={(event) => setSort(event.target.value)}><option value="created_at">登録日時</option><option value="name">名前</option><option value="email">Email</option></select>
            <select aria-label="昇順降順" value={order} onChange={(event) => setOrder(event.target.value)}><option value="desc">降順</option><option value="asc">昇順</option></select>
          </>
        }
        rowKey={(value) => value.id}
        columns={[
          { label: "名前", render: (value) => value.name },
          { label: "Email", render: (value) => value.email },
          { label: "Role", render: (value) => value.role },
          { label: "状態", render: (value) => value.deleted_at ? `退会済み (${new Date(value.deleted_at).toLocaleString()})` : value.status },
        ]}
        actions={(value, reload) => (
          <>
            <button type="button" onClick={() => void getUser(value.id).then(setDetail)}>詳細</button>
            {!value.deleted_at && <button type="button" onClick={() => {
              const next = value.status === "active" ? "suspended" : "active";
              const reason = window.prompt(next === "suspended" ? "利用停止理由（必須）" : "再開理由") ?? "";
              if (next === "suspended" && !reason.trim()) return;
              void setUserStatus(value.id, next, reason).then(reload);
            }}>{value.status === "active" ? "停止" : "再開"}</button>}
            {!value.deleted_at && <select aria-label={`${value.email}のRole`} value={value.role} onChange={(event) => {
              const next = event.target.value as AdminUser["role"];
              if (next === value.role) return;
              const reason = window.prompt("Role変更理由（必須）") ?? "";
              if (!reason.trim()) return;
              void changeUserRole(value.id, next, reason).then(reload);
            }}><option value="user">user</option><option value="content_admin">content_admin</option><option value="system_admin">system_admin</option></select>}
          </>
        )}
      />
      {detail && <DetailDialog title="User詳細" onClose={() => setDetail(null)}><dl><dt>名前</dt><dd>{detail.name}</dd><dt>Email</dt><dd>{detail.email}</dd><dt>Role</dt><dd>{detail.role}</dd><dt>状態</dt><dd>{detail.status}</dd><dt>停止理由</dt><dd>{detail.suspended_reason || "-"}</dd><dt>登録日時</dt><dd>{new Date(detail.created_at).toLocaleString()}</dd><dt>更新日時</dt><dd>{new Date(detail.updated_at).toLocaleString()}</dd></dl></DetailDialog>}
    </>
  );
}

function Audit() {
  const [action, setAction] = useState(""); const [actor, setActor] = useState(""); const [target, setTarget] = useState(""); const [requestId, setRequestId] = useState(""); const [from, setFrom] = useState(""); const [to, setTo] = useState(""); const [detail, setDetail] = useState<AuditLog | null>(null); const [sort, setSort] = useState("created_at"); const [order, setOrder] = useState("desc");
  const load = useCallback((keyword: string, page: number) => listAuditLogs(keyword, page, { action, actor_id: actor, target_type: target, request_id: requestId, from: from ? new Date(from).toISOString() : undefined, to: to ? new Date(to).toISOString() : undefined, sort, order }), [action, actor, target, requestId, from, to, sort, order]);
  return <><AdminListPage<AuditLog> title="Audit Log" load={load} filterKey={`${action}:${actor}:${target}:${requestId}:${from}:${to}:${sort}:${order}`} filters={<><input placeholder="action" value={action} onChange={(e) => setAction(e.target.value)} /><input placeholder="actor UUID" value={actor} onChange={(e) => setActor(e.target.value)} /><input placeholder="target type" value={target} onChange={(e) => setTarget(e.target.value)} /><input placeholder="request ID" value={requestId} onChange={(e) => setRequestId(e.target.value)} /><input type="datetime-local" value={from} onChange={(e) => setFrom(e.target.value)} /><input type="datetime-local" value={to} onChange={(e) => setTo(e.target.value)} /><select value={sort} onChange={(e) => setSort(e.target.value)}><option value="created_at">日時</option><option value="action">Action</option></select><select value={order} onChange={(e) => setOrder(e.target.value)}><option value="desc">降順</option><option value="asc">昇順</option></select></>} rowKey={(value) => value.id} columns={[{ label: "Actor", render: (value) => `${value.actor_name || value.actor_user_id} (${value.actor_role})` }, { label: "Action", render: (value) => value.action }, { label: "Target", render: (value) => `${value.target_type}:${value.target_id}` }, { label: "Request ID", render: (value) => value.request_id }, { label: "日時", render: (value) => new Date(value.created_at).toLocaleString() }]} actions={(value) => <button type="button" onClick={() => void getAuditLog(value.id).then(setDetail)}>詳細</button>} />{detail && <DetailDialog title="Audit Log詳細" onClose={() => setDetail(null)}><dl><dt>Actor</dt><dd>{detail.actor_name} ({detail.actor_email}) / {detail.actor_role}</dd><dt>Action</dt><dd>{detail.action}</dd><dt>Target</dt><dd>{detail.target_type}:{detail.target_id}</dd><dt>理由</dt><dd>{detail.reason || "-"}</dd><dt>概要</dt><dd>{detail.summary || "-"}</dd><dt>Request ID</dt><dd>{detail.request_id}</dd></dl><h3>変更前</h3><pre>{JSON.stringify(detail.before, null, 2)}</pre><h3>変更後</h3><pre>{JSON.stringify(detail.after, null, 2)}</pre></DetailDialog>}</>;
}

function Logs() {
  const [level, setLevel] = useState(""); const [category, setCategory] = useState(""); const [event, setEvent] = useState(""); const [user, setUser] = useState(""); const [requestId, setRequestId] = useState(""); const [status, setStatus] = useState(""); const [service, setService] = useState(""); const [method, setMethod] = useState(""); const [path, setPath] = useState(""); const [from, setFrom] = useState(""); const [to, setTo] = useState(""); const [detail, setDetail] = useState<SystemLog | null>(null); const [sort, setSort] = useState("created_at"); const [order, setOrder] = useState("desc");
  const load = useCallback((keyword: string, page: number) => listSystemLogs(keyword, page, { level, category, event, user_id: user, request_id: requestId, status_code: status || undefined, external_service: service, method, path, from: from ? new Date(from).toISOString() : undefined, to: to ? new Date(to).toISOString() : undefined, sort, order }), [level, category, event, user, requestId, status, service, method, path, from, to, sort, order]);
  return <><AdminListPage<SystemLog> title="System Log" load={load} filterKey={`${level}:${category}:${event}:${user}:${requestId}:${status}:${service}:${method}:${path}:${from}:${to}:${sort}:${order}`} filters={<><input placeholder="level" value={level} onChange={(e) => setLevel(e.target.value)} /><input placeholder="category" value={category} onChange={(e) => setCategory(e.target.value)} /><input placeholder="event" value={event} onChange={(e) => setEvent(e.target.value)} /><input placeholder="user UUID" value={user} onChange={(e) => setUser(e.target.value)} /><input placeholder="request ID" value={requestId} onChange={(e) => setRequestId(e.target.value)} /><input type="number" placeholder="status" value={status} onChange={(e) => setStatus(e.target.value)} /><input placeholder="external service" value={service} onChange={(e) => setService(e.target.value)} /><select value={method} onChange={(e) => setMethod(e.target.value)}><option value="">全Method</option><option>GET</option><option>POST</option><option>PATCH</option><option>DELETE</option></select><input placeholder="path" value={path} onChange={(e) => setPath(e.target.value)} /><input type="datetime-local" value={from} onChange={(e) => setFrom(e.target.value)} /><input type="datetime-local" value={to} onChange={(e) => setTo(e.target.value)} /><select value={sort} onChange={(e) => setSort(e.target.value)}><option value="created_at">日時</option><option value="level">Level</option><option value="status_code">Status</option></select><select value={order} onChange={(e) => setOrder(e.target.value)}><option value="desc">降順</option><option value="asc">昇順</option></select></>} rowKey={(value) => value.id} columns={[{ label: "Level", render: (value) => value.level }, { label: "Event", render: (value) => `${value.category}/${value.event}` }, { label: "HTTP", render: (value) => `${value.method || "-"} ${value.path || "-"} ${value.status_code || "-"}` }, { label: "Service", render: (value) => value.external_service || "-" }, { label: "概要", render: (value) => value.message }, { label: "日時", render: (value) => new Date(value.created_at).toLocaleString() }]} actions={(value) => <button type="button" onClick={() => void getSystemLog(value.id).then(setDetail)}>詳細</button>} />{detail && <DetailDialog title="System Log詳細" onClose={() => setDetail(null)}><dl><dt>Level</dt><dd>{detail.level}</dd><dt>Category / Event</dt><dd>{detail.category} / {detail.event}</dd><dt>Message</dt><dd>{detail.message}</dd><dt>HTTP</dt><dd>{detail.method} {detail.path} {detail.status_code}</dd><dt>Request ID</dt><dd>{detail.request_id}</dd><dt>External Service</dt><dd>{detail.external_service || "-"}</dd><dt>Duration</dt><dd>{detail.duration_ms ?? 0} ms</dd></dl><h3>Error detail</h3><pre>{detail.error_detail || "-"}</pre><h3>Metadata</h3><pre>{JSON.stringify(detail.metadata, null, 2)}</pre><h3>Stack trace</h3><pre>{detail.stack_trace || "-"}</pre></DetailDialog>}</>;
}

function DetailDialog({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return <div className="admin-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.currentTarget === event.target) onClose(); }}><section className="admin-dialog" role="dialog" aria-modal="true" aria-label={title}><header><h2>{title}</h2><button type="button" onClick={onClose}>閉じる</button></header>{children}</section></div>;
}
