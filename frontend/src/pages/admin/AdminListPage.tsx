import { useEffect, useState, type ReactNode } from "react";
import { ApiRequestError } from "../../api/client";

interface Props<T> { title: string; load: (keyword:string,page:number)=>Promise<{items:T[];page:number;total_pages:number}>; columns: {label:string; render:(item:T)=>ReactNode}[]; rowKey:(item:T)=>string; actions?:(item:T,reload:()=>void)=>ReactNode }
export function AdminListPage<T>({title,load,columns,rowKey,actions}:Props<T>) {
 const [items,setItems]=useState<T[]>([]); const [keyword,setKeyword]=useState(""); const [page,setPage]=useState(1); const [pages,setPages]=useState(1); const [loading,setLoading]=useState(false); const [error,setError]=useState("");
 const reload=()=>{setLoading(true);setError("");void load(keyword,page).then(r=>{setItems(r.items);setPages(Math.max(1,r.total_pages))}).catch((e:unknown)=>setError(e instanceof ApiRequestError?e.response.message:"取得に失敗しました")).finally(()=>setLoading(false))};
 useEffect(reload,[page]);
 return <main className="admin-page"><h1>{title}</h1><form onSubmit={e=>{e.preventDefault();setPage(1);reload()}}><label>検索 <input value={keyword} onChange={e=>setKeyword(e.target.value)} /></label><button type="submit" disabled={loading}>検索</button></form>{error&&<p role="alert">{error}</p>}{loading?<p>読み込み中...</p>:items.length===0?<p>対象データはありません。</p>:<table><thead><tr>{columns.map(c=><th key={c.label}>{c.label}</th>)}{actions&&<th>操作</th>}</tr></thead><tbody>{items.map(item=><tr key={rowKey(item)}>{columns.map(c=><td key={c.label}>{c.render(item)}</td>)}{actions&&<td>{actions(item,reload)}</td>}</tr>)}</tbody></table>}<div><button disabled={page<=1} onClick={()=>setPage(v=>v-1)}>前へ</button><span>{page} / {pages}</span><button disabled={page>=pages} onClick={()=>setPage(v=>v+1)}>次へ</button></div></main>
}
