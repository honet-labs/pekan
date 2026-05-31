import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useBudgets } from "../hooks/useBudgets";
import { BudgetTable } from "../components/BudgetTable";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { consumeFlashToast } from "../../../../core/toast/flashToast";
import { EntityTransactionsModal } from "../../transactions/components/EntityTransactionsModal";
import { Budget } from "../api/budgets.types";

import { Pagination } from "../../../../core/components/Pagination";
import { PageHeader } from "../../../../core/components/PageHeader";

export function BudgetsListPage(): JSX.Element {
  const { t } = useI18n();
  const { items, loading, error, page, pageSize, total, setPage, remove } = useBudgets();
  const { toasts, success, error: showError, remove: removeToast } = useToast();
  const [activeTab, setActiveTab] = useState<"active" | "history">("active");
  const [transactionViewEntity, setTransactionViewEntity] = useState<Budget | null>(null);

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

  const activeItems = items.filter(item => item.status !== "ended");
  const historyItems = items.filter(item => item.status === "ended");
  const displayedItems = activeTab === "active" ? activeItems : historyItems;

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
          <div style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            flexWrap: "wrap",
            gap: "1rem",
            marginBottom: "1rem"
          }}>
            <h3 className="form-title" style={{ margin: 0 }}>{t("budgets.table.title")}</h3>
            
            {/* Tabs */}
            <div style={{
              display: "flex",
              background: "var(--surface-soft)",
              padding: "4px",
              borderRadius: "10px",
              border: "1px solid var(--border)"
            }}>
              <button
                type="button"
                onClick={() => setActiveTab("active")}
                style={{
                  padding: "6px 16px",
                  borderRadius: "8px",
                  border: "none",
                  background: activeTab === "active" ? "var(--primary)" : "transparent",
                  color: activeTab === "active" ? "#fff" : "var(--text-muted)",
                  fontWeight: 600,
                  fontSize: "0.875rem",
                  cursor: "pointer",
                  transition: "all 0.2s"
                }}
              >
                Aktif ({activeItems.length})
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("history")}
                style={{
                  padding: "6px 16px",
                  borderRadius: "8px",
                  border: "none",
                  background: activeTab === "history" ? "var(--primary)" : "transparent",
                  color: activeTab === "history" ? "#fff" : "var(--text-muted)",
                  fontWeight: 600,
                  fontSize: "0.875rem",
                  cursor: "pointer",
                  transition: "all 0.2s"
                }}
              >
                Riwayat ({historyItems.length})
              </button>
            </div>
          </div>

          <BudgetTable 
            items={displayedItems} 
            onDelete={handleDelete} 
            onViewTransactions={(item) => setTransactionViewEntity(item)} 
          />
          <Pagination
            currentPage={page}
            pageSize={pageSize}
            totalItems={activeTab === "active" ? activeItems.length : historyItems.length}
            onPageChange={setPage}
            disabled={loading}
          />
        </div>
      ) : null}
      <EntityTransactionsModal
        isOpen={!!transactionViewEntity}
        title={`${t("nav.budgets")}: ${transactionViewEntity?.name}`}
        type="budget"
        entityId={transactionViewEntity?.category_id || ""}
        onClose={() => setTransactionViewEntity(null)}
      />
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}
