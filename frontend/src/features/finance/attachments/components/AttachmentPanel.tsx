import { useEffect, useMemo, useState } from "react";
import {
  deleteFinanceAttachment,
  listFinanceAttachments,
  openFinanceAttachment,
  uploadFinanceAttachments
} from "../api/attachments.api";
import { FinanceAttachment, FinanceAttachmentOwnerType } from "../api/attachments.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";

type Props = {
  ownerType: FinanceAttachmentOwnerType;
  ownerID?: string | null;
  title?: string;
};

export function AttachmentPanel({ ownerType, ownerID, title }: Props): JSX.Element {
  const { locale, t } = useI18n();
  const [items, setItems] = useState<FinanceAttachment[]>([]);
  const [files, setFiles] = useState<File[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [itemToDelete, setItemToDelete] = useState<FinanceAttachment | null>(null);
  const [deletingID, setDeletingID] = useState<string | null>(null);

  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );

  async function reload(): Promise<void> {
    if (!ownerID) {
      setItems([]);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await listFinanceAttachments(ownerType, ownerID);
      setItems(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    reload().catch(() => undefined);
  }, [ownerType, ownerID, t]);

  async function handleUpload(): Promise<void> {
    if (!ownerID || files.length === 0) {
      return;
    }
    setUploading(true);
    setError(null);
    try {
      await uploadFinanceAttachments(ownerType, ownerID, files);
      setFiles([]);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setUploading(false);
    }
  }

  async function handleDelete(attachmentID: string): Promise<void> {
    if (!ownerID) {
      return;
    }
    setDeletingID(attachmentID);
    try {
      await deleteFinanceAttachment(ownerType, ownerID, attachmentID);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setDeletingID(null);
      setItemToDelete(null);
    }
  }

  async function handleView(attachmentID: string): Promise<void> {
    if (!ownerID) {
      return;
    }
    try {
      await openFinanceAttachment(ownerType, ownerID, attachmentID);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.downloadReportFailed"));
    }
  }

  return (
    <div className="card surface">
      <h3 className="form-title">{title ?? t("transactions.attachments.title")}</h3>
      {!ownerID ? <p className="page-subtitle">{t("attachments.hintSaveFirst")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}
      {ownerID ? (
        <div className="form-grid">
          <label className="form-field">
            {t("transactions.form.attachments")}
            <input
              className="input-control"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              multiple
              onChange={(event) => setFiles(Array.from(event.target.files ?? []))}
            />
            {files.length ? (
              <small className="page-subtitle">
                {files.length} {t("transactions.form.filesSelected")}
              </small>
            ) : null}
          </label>
          <button className="btn btn-primary" type="button" onClick={() => handleUpload().catch(() => undefined)} disabled={uploading || files.length === 0}>
            {uploading ? t("common.loading") : t("attachments.upload")}
          </button>
        </div>
      ) : null}
      {loading ? <p className="page-subtitle">{t("common.loading")}</p> : null}
      <div className="data-table-wrap table-mobile-stack">
        <table className="data-table">
          <thead>
            <tr>
              <th>{t("transactions.attachments.file")}</th>
              <th>{t("transactions.attachments.status")}</th>
              <th>{t("transactions.attachments.size")}</th>
              <th>{t("transactions.table.action")}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td data-label={t("transactions.attachments.file")} style={{ overflowWrap: "anywhere", wordBreak: "break-all", whiteSpace: "normal", minWidth: "120px", maxWidth: "250px" }}>
                  {item.original_filename}
                </td>
                <td data-label={t("transactions.attachments.status")}>{item.scan_status}</td>
                <td data-label={t("transactions.attachments.size")}>{numberFormatter.format(item.size_bytes)} bytes</td>
                <td data-label={t("transactions.table.action")}>
                  <div className="table-actions">
                    <button
                      className="btn btn-ghost-inline icon-btn"
                      type="button"
                      onClick={() => handleView(item.id).catch(() => undefined)}
                      title={t("transactions.attachments.view")}
                    >
                      <span aria-hidden="true" className="icon-eye">
                        <svg viewBox="0 0 24 24" className="table-icon">
                          <path
                            d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Zm10 3.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z"
                            fill="currentColor"
                          />
                        </svg>
                      </span>
                      {t("transactions.attachments.view")}
                    </button>
                    <button
                      className="btn btn-ghost-inline danger"
                      type="button"
                      onClick={() => setItemToDelete(item)}
                      disabled={deletingID === item.id}
                    >
                      {deletingID === item.id ? t("common.loading") : t("common.delete")}
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {!items.length ? (
              <tr>
                <td colSpan={4}>{t("common.noItems")}</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
      <DeleteConfirmModal
        isOpen={!!itemToDelete}
        title={t("common.delete")}
        message={`${t("common.delete")} "${itemToDelete?.original_filename}"?`}
        isLoading={deletingID !== null}
        onConfirm={() => itemToDelete && handleDelete(itemToDelete.id).catch(() => undefined)}
        onCancel={() => setItemToDelete(null)}
      />
    </div>
  );
}
