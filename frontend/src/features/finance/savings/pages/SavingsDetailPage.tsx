import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { deleteSavings, getSavings, updateSavings } from "../api/savings.api";
import { Savings } from "../api/savings.types";
import { SavingsForm } from "../components/SavingsForm";
import { RelatedTransactionsPanel } from "../components/RelatedTransactionsPanel";
import { useI18n } from "../../../../core/i18n/i18n";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { setFlashToast } from "../../../../core/toast/flashToast";
import { AttachmentPanel } from "../../attachments/components/AttachmentPanel";
import { PageHeader } from "../../../../core/components/PageHeader";

export function SavingsDetailPage(): JSX.Element {
  const { t } = useI18n();
  const { savingsID, tenantCode } = useParams();
  const navigate = useNavigate();
  const { toasts, success, error, remove } = useToast();
  const [item, setItem] = useState<Savings | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function load(): Promise<void> {
    if (!savingsID) {
      setErrorMsg(t("errors.saveDataFailed"));
      setLoading(false);
      return;
    }
    setLoading(true);
    setErrorMsg(null);
    try {
      const result = await getSavings(savingsID);
      setItem(result);
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : t("errors.saveDataFailed");
      setErrorMsg(errMsg);
      error(errMsg);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, [savingsID, t]);

  async function handleUpdate(payload: Parameters<typeof updateSavings>[1]): Promise<void> {
    if (!savingsID) {
      return;
    }
    try {
      await updateSavings(savingsID, payload);
      setFlashToast({ message: t("common.updateSuccess") || "Data berhasil disimpan", type: "success" });
      navigate(`/app/${tenantCode ?? "default"}/finance/savings`, { replace: true });
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : t("errors.saveDataFailed");
      error(errMsg);
    }
  }

  async function handleDelete(): Promise<void> {
    if (!savingsID) {
      return;
    }
    setDeleting(true);
    try {
      await deleteSavings(savingsID);
      setFlashToast({ message: t("common.deleteSuccess"), type: "success" });
      navigate(`/app/${tenantCode ?? "default"}/finance/savings`, { replace: true });
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : t("errors.saveDataFailed");
      error(errMsg);
    } finally {
      setDeleting(false);
      setDeleteConfirmOpen(false);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("savings.form.update")} 
        description={t("savings.subtitle")}
      >
        <button className="btn-icon-danger" type="button" onClick={() => setDeleteConfirmOpen(true)} title={t("common.delete")}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
        </button>
      </PageHeader>

      {item ? (
        <div style={{ marginBottom: '1rem' }}>
          <p className="page-subtitle">{`${t("transactions.detail.updatedAt")}: ${new Date(item.updated_at ?? Date.now()).toLocaleString()}`}</p>
        </div>
      ) : null}

      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {errorMsg ? <p className="alert error">{errorMsg}</p> : null}

      {!loading && item ? (
        <>
          <div className="card-grid two-col tight">
            <SavingsForm
              initialValue={{
                name: item.name,
                target_amount_minor: item.target_amount_minor,
                current_amount_minor: item.current_amount_minor,
                currency: item.currency,
                start_date: item.start_date ?? "",
                target_date: item.target_date ?? "",
                status: item.status
              }}
              submitLabel={t("savings.form.updateBtn")}
              onSubmit={handleUpdate}
              onCancel={() => navigate("..")}
            />
            <AttachmentPanel ownerType="savings" ownerID={item.id} title={`${t("nav.savings")} ${t("transactions.attachments.title")}`} />
          </div>
          <RelatedTransactionsPanel savingsID={item.id} />
        </>
      ) : null}

      <DeleteConfirmModal
        isOpen={deleteConfirmOpen}
        title={t("savings.delete.title")}
        message={`${t("savings.delete.confirm")} "${item?.name}"?`}
        isLoading={deleting}
        onConfirm={handleDelete}
        onCancel={() => setDeleteConfirmOpen(false)}
      />

      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}

