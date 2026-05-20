import { useEffect } from "react";
import { Link } from "react-router-dom";
import { useBudgets } from "../hooks/useBudgets";
import { BudgetTable } from "../components/BudgetTable";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { consumeFlashToast } from "../../../../core/toast/flashToast";

import { Pagination } from "../../../../core/components/Pagination";
import { PageHeader } from "../../../../core/components/PageHeader";

export function BudgetsListPage(): JSX.Element {
  const { t } = useI18n();
  const { items, loading, error, page, pageSize, total, setPage, remove } = useBudgets();
  const { toasts, success, error: showError, remove: removeToast } = useToast();

  useEffect(() => {
    const flash = consumeFlashToast();
    if (!flash) {
      return;
    }
    if (flash.type === "error") {
      showError(flash.message);
      return;
    }
    success(flash.message);
  }, [showError, success]);

  async function handleDelete(budgetID: string): Promise<void> {
    try {
      await remove(budgetID);
      success(t("common.deleteSuccess"));
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadBudgetsFailed"));
      throw err;
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("budgets.title")} 
        description={t("budgets.subtitle")}
      >
        <Link to="new" className="btn btn-primary">
          {t("budgets.create")}
        </Link>
      </PageHeader>

      {loading && items.length === 0 ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      {!error && (items.length > 0 || !loading) ? (
        <div className="surface card">
          <h3 className="form-title">{t("budgets.table.title")}</h3>
          <BudgetTable items={items} onDelete={handleDelete} />
          <Pagination
            currentPage={page}
            pageSize={pageSize}
            totalItems={total}
            onPageChange={setPage}
            disabled={loading}
          />
        </div>
      ) : null}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}
