import { useCallback, useEffect, useState, type ReactNode } from "react";
import { useSearchParams } from "react-router-dom";
import { ApiRequestError } from "../../api/client";

interface Props<T> {
  title: string;
  load: (keyword: string, page: number) => Promise<{ items: T[]; page: number; total_pages: number }>;
  columns: { label: string; render: (item: T) => ReactNode }[];
  rowKey: (item: T) => string;
  actions?: (item: T, reload: () => void) => ReactNode;
  filters?: ReactNode;
  filterKey?: string;
}

export function AdminListPage<T>({ title, load, columns, rowKey, actions, filters, filterKey = "" }: Props<T>) {
  const [params, setParams] = useSearchParams();
  const [items, setItems] = useState<T[]>([]);
  const [keyword, setKeyword] = useState(params.get("keyword") ?? "");
  const [page, setPage] = useState(Number(params.get("page") ?? 1) || 1);
  const [pages, setPages] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(() => {
    setLoading(true);
    setError("");
    void load(keyword, page)
      .then((response) => {
        setItems(response.items);
        setPages(Math.max(1, response.total_pages));
        const next = new URLSearchParams(params);
        if (keyword.trim()) next.set("keyword", keyword.trim());
        else next.delete("keyword");
        next.set("page", String(page));
        setParams(next, { replace: true });
      })
      .catch((error: unknown) => {
        setError(error instanceof ApiRequestError ? error.response.message : "取得に失敗しました");
      })
      .finally(() => setLoading(false));
  }, [load, keyword, page, filterKey, params, setParams]);

  useEffect(() => {
    reload();
  }, [page, filterKey]);

  return (
    <main className="admin-panel">
      <div className="admin-section-heading">
        <div>
          <p className="admin-section-kicker">Management</p>
          <h2>{title}</h2>
        </div>
      </div>

      <form
        className="admin-filter-card"
        onSubmit={(event) => {
          event.preventDefault();
          if (page === 1) reload();
          else setPage(1);
        }}
      >
        <label className="admin-control admin-search-control">
          キーワード
          <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="名前・メールアドレスなど" />
        </label>
        {filters}
        <button className="admin-primary-button" type="submit" disabled={loading}>検索</button>
      </form>

      {error && <p className="admin-alert" role="alert">{error}</p>}

      <section className="admin-table-card">
        {loading ? (
          <div className="admin-empty-state"><strong>読み込み中...</strong></div>
        ) : items.length === 0 ? (
          <div className="admin-empty-state">
            <strong>対象データはありません</strong>
            <span>検索条件を変更して再度お試しください。</span>
          </div>
        ) : (
          <div className="admin-table-scroll">
            <table className="admin-table">
              <thead><tr>{columns.map((column) => <th key={column.label}>{column.label}</th>)}{actions && <th>操作</th>}</tr></thead>
              <tbody>{items.map((item) => <tr key={rowKey(item)}>{columns.map((column) => <td key={column.label}>{column.render(item)}</td>)}{actions && <td><div className="admin-row-actions">{actions(item, reload)}</div></td>}</tr>)}</tbody>
            </table>
          </div>
        )}
        <div className="admin-pagination">
          <button type="button" disabled={page <= 1 || loading} onClick={() => setPage((value) => value - 1)}>前へ</button>
          <span>{page} / {pages}</span>
          <button type="button" disabled={page >= pages || loading} onClick={() => setPage((value) => value + 1)}>次へ</button>
        </div>
      </section>
    </main>
  );
}
