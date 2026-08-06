import { useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { ApiRequestError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import { ROLE } from "../../constants/role";

type AdminSection = "artists" | "songs" | "tags" | "users" | "audit" | "logs" | "monitoring";

interface AdminLayoutProps {
  title: string;
  activeSection: AdminSection;
  onSectionChange: (section: AdminSection) => void;
  children: ReactNode;
}

const contentItems: Array<{ id: AdminSection; label: string }> = [
  { id: "artists", label: "Artists" },
  { id: "songs", label: "Songs" },
  { id: "tags", label: "Tags" },
];

const systemItems: Array<{ id: AdminSection; label: string }> = [
  { id: "users", label: "Users" },
  { id: "audit", label: "Audit Log" },
  { id: "logs", label: "System Log" },
  { id: "monitoring", label: "Monitoring" },
];

export function AdminLayout({ title, activeSection, onSectionChange, children }: AdminLayoutProps) {
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState("");
  const items = user?.role === ROLE.SYSTEM_ADMIN ? systemItems : contentItems;

  const handleLogout = async (): Promise<void> => {
    if (isLoggingOut) return;
    setIsLoggingOut(true);
    setLogoutError("");
    try {
      await logout();
      navigate("/login", { replace: true });
    } catch (error: unknown) {
      const message = error instanceof ApiRequestError
        ? error.response.message
        : "ログアウト処理に失敗しました。再度お試しください。";
      setLogoutError(message);
      navigate("/login", { replace: true, state: { logoutError: message } });
    } finally {
      setIsLoggingOut(false);
    }
  };

  return (
    <div className="admin-shell">
      <aside className="admin-sidebar">
        <div>
          <div className="admin-brand-mark" aria-hidden="true">MD</div>
          <p className="admin-brand-name">Music Discovery</p>
          <p className="admin-role-label">{user?.role === ROLE.SYSTEM_ADMIN ? "System Admin" : "Content Admin"}</p>
        </div>

        <nav className="admin-navigation" aria-label="管理メニュー">
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              className={activeSection === item.id ? "admin-nav-item is-active" : "admin-nav-item"}
              onClick={() => onSectionChange(item.id)}
            >
              <span className="admin-nav-dot" aria-hidden="true" />
              {item.label}
            </button>
          ))}
        </nav>

        <div className="admin-sidebar-footer">
          <div className="admin-account">
            <div className="admin-avatar" aria-hidden="true">{user?.name?.slice(0, 1).toUpperCase() ?? "A"}</div>
            <div>
              <strong>{user?.name ?? "Administrator"}</strong>
              <span>{user?.email}</span>
            </div>
          </div>
          {logoutError && <p className="admin-sidebar-error" role="alert">{logoutError}</p>}
          <button className="admin-logout-button" type="button" onClick={() => void handleLogout()} disabled={isLoggingOut}>
            {isLoggingOut ? "ログアウト中..." : "ログアウト"}
          </button>
        </div>
      </aside>

      <div className="admin-workspace">
        <header className="admin-topbar">
          <div>
            <p className="admin-eyebrow">Administration</p>
            <h1>{title}</h1>
          </div>
          <div className="admin-topbar-account">
            <span className="admin-status-dot" aria-hidden="true" />
            {user?.email}
          </div>
        </header>
        <div className="admin-content">{children}</div>
      </div>
    </div>
  );
}
