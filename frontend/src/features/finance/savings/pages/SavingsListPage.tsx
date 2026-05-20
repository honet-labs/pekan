import { useEffect } from "react";
import { Link } from "react-router-dom";
import { useSavings } from "../hooks/useSavings";
import { useI18n } from "../../../../core/i18n/i18n";
import { PageHeader } from "../../../../core/components/PageHeader";
import { SavingsTable } from "../components/SavingsTable";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { consumeFlashToast } from "../../../../core/toast/flashToast";

import { Pagination } from "../../../../core/components/Pagination";

export function SavingsListPage(): JSX.Element {
  const { t } = useI18n();
  const { items, loading, error, page, pageSize, total, setPage, remove } = useSavings();
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

  async function handleDelete(id: string): Promise<void> {
    try {
      await remove(id);
      success(t("common.deleteSuccess"));
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadSavingsFailed"));
      throw err;
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("savings.title")} 
        description={t("savings.subtitle")}
      >
        <Link to="new" className="btn btn-primary">
          {t("savings.create")}
        </Link>
      </PageHeader>

      {loading && items.length === 0 ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      {!error && (items.length > 0 || !loading) ? (
        <div className="surface card">
          <h3 className="form-title">{t("savings.table.title")}</h3>
          <SavingsTable items={items} onDelete={handleDelete} />
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
