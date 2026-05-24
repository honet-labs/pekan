import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { deleteBudget, getBudget, updateBudget } from "../api/budgets.api";
import { Budget } from "../api/budgets.types";
import { useFinanceMasterData } from "../../masterdata/hooks/useFinanceMasterData";
import { BudgetForm } from "../components/BudgetForm";
import { useI18n } from "../../../../core/i18n/i18n";
import { AttachmentPanel } from "../../attachments/components/AttachmentPanel";
import { ToastContainer } from "../../../../core/components/Toast";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { useToast } from "../../../../core/hooks/useToast";
import { setFlashToast } from "../../../../core/toast/flashToast";
import { PageHeader } from "../../../../core/components/PageHeader";

export function BudgetDetailPage(): JSX.Element {
  const { t } = useI18n();
  const { toasts, success, error: showError, remove } = useToast();
  const { budgetID, tenantCode } = useParams();
  const navigate = useNavigate();
  const { categories, loading: loadingMasterData, error: masterDataError } = useFinanceMasterData();
  const [item, setItem] = useState<Budget | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function load(): Promise<void> {
    if (!budgetID) {
      setError(t("errors.loadBudgetsFailed"));
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await getBudget(budgetID);
      setItem(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.loadBudgetsFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, [budgetID, t]);

  async function handleUpdate(payload: Parameters<typeof updateBudget>[1]): Promise<void> {
    if (!budgetID) {
      return;
    }
    try {
      await updateBudget(budgetID, payload);
      setFlashToast({ message: t("common.updateSuccess") || "Data berhasil diperbarui", type: "success" });
      navigate(`/app/${tenantCode ?? "default"}/finance/budgets`);
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : t("errors.loadBudgetsFailed");
      showError(errMsg);
      throw err instanceof Error ? err : new Error(errMsg);
    }
  }

  async function handleDelete(): Promise<void> {
    if (!budgetID) {
      return;
    }
    setDeleting(true);
    try {
      await deleteBudget(budgetID);
      setFlashToast({ message: t("common.deleteSuccess"), type: "success" });
      navigate(`/app/${tenantCode ?? "default"}/finance/budgets`, { replace: true });
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadBudgetsFailed"));
    } finally {
      setDeleting(false);
      setDeleteConfirmOpen(false);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("budgets.form.update")} 
        description={t("budgets.subtitle")}
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

      {loading || loadingMasterData ? <p className="alert info">{t("common.loading")}</p> : null}
      {masterDataError ? <p className="alert error">{masterDataError}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      {!loading && !loadingMasterData && item ? (
        <div className="card-grid two-col tight">
          <BudgetForm
            categories={categories}
            initialValue={{
              name: item.name,
              category_id: item.category_id ?? "",
              category_name: item.category_name ?? "",
              amount_limit_minor: item.amount_limit_minor,
              currency: item.currency,
              period: item.period,
              start_date: item.start_date,
              end_date: item.end_date ?? "",
              alert_threshold_pct: item.alert_threshold_pct ?? 80,
              notes: item.notes ?? "",
              status: item.status
            }}
            initialCategoryName={item.category_name ?? ""}
            submitLabel={t("budgets.form.updateBtn")}
            onSubmit={handleUpdate}
            onCancel={() => navigate("..")}
          />
          <AttachmentPanel ownerType="budgets" ownerID={item.id} title={`${t("nav.budgets")} ${t("transactions.attachments.title")}`} />
        </div>
      ) : null}
      <DeleteConfirmModal
        isOpen={deleteConfirmOpen}
        title={t("budgets.delete.title")}
        message={item ? `${t("budgets.delete.confirm")} "${item.name}"?` : t("budgets.delete.confirm")}
        isLoading={deleting}
        onConfirm={() => {
          handleDelete().catch(() => undefined);
        }}
        onCancel={() => {
          if (!deleting) {
            setDeleteConfirmOpen(false);
          }
        }}
      />
      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}

