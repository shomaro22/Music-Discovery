import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
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
  const [keywordInput, setKeywordInput] = useState("");
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const [pages, setPages] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const requestSequence = useRef(0);

  const reload = useCallback(async () => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setError("");
    try {
      const response = await load(keyword, page);
      if (sequence !== requestSequence.current) return;
      setItems(response.items);
      const nextPages = Math.max(1, response.total_pages);
      setPages(nextPages);

      // If deletion/filtering leaves us beyond the final page, move to the
      // nearest valid page and let the effect reload it.
      if (page > nextPages) {
        setPage(nextPages);
        return;
      }

      const next = new URLSearchParams(params);
      if (keyword) next.set("keyword", keyword);
      else next.delete("keyword");
      next.set("page", String(page));
      setParams(next, { replace: true });
    } catch (caught: unknown) {
      if (sequence !== requestSequence.current) return;
      setError(caught instanceof ApiRequestError ? caught.response.message : "取得に失敗しました");
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [load, keyword, page, params, setParams]);

  useEffect(() => {
    void reload();
  }, [page, keyword, filterKey]); // load is represented by filterKey; avoids URL-search-param reload loops.

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
          const nextKeyword = keywordInput.trim();
          if (page !== 1) setPage(1);
          if (nextKeyword !== keyword) setKeyword(nextKeyword);
          else void reload();
        }}
      >
        <label className="admin-control admin-search-control">
          キーワード
          <input value={keywordInput} onChange={(event) => setKeywordInput(event.target.value)} placeholder="名前・メールアドレスなど" />
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
              <tbody>{items.map((item) => <tr key={rowKey(item)}>{columns.map((column) => <td key={column.label}>{column.render(item)}</td>)}{actions && <td><div className="admin-row-actions">{actions(item, () => void reload())}</div></td>}</tr>)}</tbody>
            </table>
          </div>
        )}
        <div className="admin-pagination">
          <button type="button" disabled={page <= 1 || loading} onClick={() => setPage((value) => Math.max(1, value - 1))}>前へ</button>
          <span>{page} / {pages}</span>
          <button type="button" disabled={page >= pages || loading} onClick={() => setPage((value) => Math.min(pages, value + 1))}>次へ</button>
        </div>
      </section>
    </main>
  );
}
