import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  deleteTransaction,
  getTransaction,
  listTransactionAttachments,
  openTransactionAttachment,
  updateTransaction,
  uploadTransactionAttachments
} from "../api/transaction.api";
import { Transaction, TransactionAttachment, UpdateTransactionPayload } from "../api/transaction.types";
import { TransactionForm } from "../components/TransactionForm";
import { useTransactionMasterData } from "../hooks/useTransactionMasterData";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { useToast } from "../../../../core/hooks/useToast";
import { setFlashToast } from "../../../../core/toast/flashToast";
import { PageHeader } from "../../../../core/components/PageHeader";

export function TransactionDetailPage(): JSX.Element {
  const { transactionID, tenantCode } = useParams();
  const navigate = useNavigate();
  const { locale, t } = useI18n();
  const { toasts, success, error: showError, remove } = useToast();
  const { accounts, categories, savings, loading: loadingMasterData, error: masterDataError } = useTransactionMasterData();
  const [item, setItem] = useState<Transaction | null>(null);
  const [attachments, setAttachments] = useState<TransactionAttachment[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [attachmentError, setAttachmentError] = useState<string | null>(null);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

  const isIgnorableAttachmentError = (err: unknown): boolean => {
    const message = err instanceof Error ? err.message.toLowerCase() : "";
    return (
      /f(?:i)?le.*required|required.*f(?:i)?le/.test(message) ||
      /f\w*\s*(is|are)?\s*required/.test(message) ||
      message.includes("file is required") ||
      message.includes("file required") ||
      message.includes("file_required") ||
      message.includes("fle is required") ||
      message.includes("invalid multipart") ||
      message.includes("bad request to api")
    );
  };

  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );

  async function loadAll(): Promise<void> {
    if (!transactionID) {
      setError(t("transactions.error.missingId"));
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const transaction = await getTransaction(transactionID);
      setItem(transaction);
      setAttachmentError(null);
      try {
        const attachmentItems = await listTransactionAttachments(transactionID);
        setAttachments(attachmentItems);
      } catch (attachmentErr) {
        setAttachments([]);
        setAttachmentError(attachmentErr instanceof Error ? attachmentErr.message : t("errors.loadAttachmentsFailed"));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveTransactionFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAll().catch(() => undefined);
  }, [transactionID, t]);

  async function handleUpdate(input: { payload: UpdateTransactionPayload; files: File[] }): Promise<void> {
    if (!transactionID) {
      return;
    }
    setSubmitting(true);
    setAttachmentError(null);
    try {
      await updateTransaction(transactionID, input.payload);
      if (input.files.length > 0) {
        try {
          await uploadTransactionAttachments(transactionID, input.files);
        } catch (uploadErr) {
          if (!isIgnorableAttachmentError(uploadErr)) {
            setAttachmentError(uploadErr instanceof Error ? uploadErr.message : t("errors.loadAttachmentsFailed"));
          }
        }
      }
      setFlashToast({ message: t("common.updateSuccess") || "Data berhasil diperbarui", type: "success" });
      if (tenantCode) {
        navigate(`/app/${tenantCode}/finance/transactions`, { replace: true });
        return;
      }
      navigate("..", { relative: "path" });
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : t("errors.saveTransactionFailed");
      showError(errMsg);
      throw err instanceof Error ? err : new Error(errMsg);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(): Promise<void> {
    if (!transactionID) {
      return;
    }
    setSubmitting(true);
    try {
      await deleteTransaction(transactionID);
      setFlashToast({ message: t("common.deleteSuccess"), type: "success" });
      if (tenantCode) {
        navigate(`/app/${tenantCode}/finance/transactions`, { replace: true });
        return;
      }
      navigate("..");
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.saveTransactionFailed"));
    } finally {
      setSubmitting(false);
      setDeleteConfirmOpen(false);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("transactions.detail.title")} 
        description={t("transactions.detail.subtitleNew")}
      >
        {item ? (
          <button className="btn-icon-danger" type="button" onClick={() => setDeleteConfirmOpen(true)} disabled={submitting} title={t("common.delete")}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
          </button>
        ) : null}
      </PageHeader>

      {loading || loadingMasterData ? <p className="alert info">{t("common.loading")}</p> : null}
      {masterDataError ? <p className="alert error">{masterDataError}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}
      {attachmentError ? <p className="alert info">{attachmentError}</p> : null}

      {!loading && !loadingMasterData && item ? (
        <div className="card-grid two-col">
          <div>
            <TransactionForm
              onSubmit={handleUpdate}
              accounts={accounts}
              categories={categories}
              initialValue={{
                account_id: item.account_id,
                category_id: item.category_id,
                category_name: item.category_name ?? undefined,
                savings_ids: item.savings_ids ?? [],
                type: item.type,
                amount_minor: item.amount_minor,
                currency: item.currency,
                input_date: item.input_date,
                transaction_date: item.transaction_date,
                description: item.description ?? "",
                merchant_name: item.merchant_name ?? "",
                receipt_number: item.receipt_number ?? "",
                payment_method: item.payment_method ?? "",
                subtotal_minor: item.subtotal_minor ?? 0,
                tax_minor: item.tax_minor ?? 0,
                service_charge_minor: item.service_charge_minor ?? 0,
                receipt_discount_minor: item.receipt_discount_minor ?? 0,
                items: item.items ?? []
              }}
              initialCategoryName={item.category_name ?? undefined}
              savingsOptions={savings}
              submitLabel={submitting ? t("common.loading") : t("transactions.form.update")}
              onCancel={() => navigate(tenantCode ? `/app/${tenantCode}/finance/transactions` : "..")}
              includeAttachmentUpload
              actorLabel={item.created_by_name || item.created_by || "-"}
            />
          </div>
          <div className="surface card">
            <h3 className="form-title">{t("transactions.attachments.title")}</h3>
            <p className="page-subtitle">{`TID ${item.tid} | ${t("transactions.table.inputDate")}: ${item.input_date}`}</p>
            <p className="page-subtitle">{`${t("transactions.detail.updatedAt")}: ${new Date(item.updated_at).toLocaleString(
              locale === "id" ? "id-ID" : "en-US"
            )}`}</p>
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
                  {attachments.map((attachment) => (
                    <tr key={attachment.id}>
                      <td data-label={t("transactions.attachments.file")}>{attachment.original_filename}</td>
                      <td data-label={t("transactions.attachments.status")}>{attachment.scan_status}</td>
                      <td data-label={t("transactions.attachments.size")}>{numberFormatter.format(attachment.size_bytes)} bytes</td>
                      <td data-label={t("transactions.table.action")}>
                        <button
                          className="btn btn-ghost-inline"
                          type="button"
                          onClick={() => openTransactionAttachment(item.id, attachment.id).catch(() => undefined)}
                        >
                          {t("transactions.attachments.view")}
                        </button>
                      </td>
                    </tr>
                  ))}
                  {!attachments.length ? (
                    <tr>
                      <td colSpan={4}>{t("common.noItems")}</td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      ) : null}
      <DeleteConfirmModal
        isOpen={deleteConfirmOpen}
        title={t("transactions.delete.title")}
        message={item ? `${t("transactions.delete.confirm")} TID ${item.tid}?` : t("transactions.delete.confirm")}
        isLoading={submitting}
        onConfirm={() => {
          handleDelete().catch(() => undefined);
        }}
        onCancel={() => {
          if (!submitting) {
            setDeleteConfirmOpen(false);
          }
        }}
      />
      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}
