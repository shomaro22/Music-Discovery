import { Link } from "react-router-dom";
import { useAuth } from "../../hooks/useAuth";
import { useRole } from "../../hooks/useRole";
import { ROLE } from "../../constants/role";

export function Navigation() {
  const { isAuthenticated, isLoading, user } = useAuth();
  const { isContentAdmin, isSystemAdmin } = useRole();
  if (isLoading) return null;
  return (
    <nav aria-label="メインナビゲーション">
      {!isAuthenticated ? <><Link to="/signup">会員登録</Link><Link to="/login">Login</Link></> : <>
        <Link to="/">Top</Link>
        {user?.role === ROLE.USER && <Link to="/questions">質問回答</Link>}
        {isContentAdmin && <Link to="/admin/content">Content管理</Link>}
        {isSystemAdmin && <Link to="/system/users">System管理</Link>}
      </>}
    </nav>
  );
}
