import { useNavigate } from "react-router-dom";

export function ForbiddenPage() {
  const navigate = useNavigate();
  return (
    <main className="top-page">
      <section className="top-card" aria-labelledby="forbidden-title">
        <p className="top-kicker">403 Forbidden</p>
        <h1 id="forbidden-title">このページを表示する権限がありません</h1>
        <p className="top-description">アクセス可能な画面から操作を続けてください。</p>
        <button type="button" className="top-logout-button" onClick={() => navigate("/", { replace: true })}>
          Topへ戻る
        </button>
      </section>
    </main>
  );
}
