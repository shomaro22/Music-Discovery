import { useState } from "react";
import { ApiRequestError } from "../../api/client";

interface Props {
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => Promise<void>;
  onClose: () => void;
}

export function ConfirmActionDialog({ title, message, confirmLabel = "削除する", onConfirm, onClose }: Props) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const submit = async () => {
    setBusy(true);
    setError("");
    try {
      await onConfirm();
      onClose();
    } catch (caught: unknown) {
      setError(caught instanceof ApiRequestError ? caught.response.message : "処理に失敗しました。もう一度お試しください。");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="admin-dialog-backdrop" role="presentation" onMouseDown={(event) => { if (event.currentTarget === event.target && !busy) onClose(); }}>
      <section className="admin-dialog admin-confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="admin-confirm-title">
        <header>
          <h2 id="admin-confirm-title">{title}</h2>
          <button type="button" onClick={onClose} disabled={busy}>閉じる</button>
        </header>
        <p>{message}</p>
        {error && <p className="admin-alert" role="alert">{error}</p>}
        <div className="admin-dialog-actions">
          <button type="button" onClick={onClose} disabled={busy}>キャンセル</button>
          <button className="admin-danger-button" type="button" onClick={() => void submit()} disabled={busy}>{busy ? "処理中..." : confirmLabel}</button>
        </div>
      </section>
    </div>
  );
}
