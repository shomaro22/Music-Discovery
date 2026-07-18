import { useLocation } from "react-router-dom";

export function LoginPage() {
  const location = useLocation();
  const signedUp = Boolean((location.state as { signedUp?: boolean } | null)?.signedUp);
  return <main><h1>ログイン</h1>{signedUp && <p>会員登録が完了しました。ログインしてください。</p>}<p>ログイン機能は第2章で実装します。</p></main>;
}
