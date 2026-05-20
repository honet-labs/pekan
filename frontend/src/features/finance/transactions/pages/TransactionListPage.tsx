import { useEffect } from "react";
import { Link } from "react-router-dom";
import { TransactionTable } from "../components/TransactionTable";
import { useTransactionList } from "../hooks/useTransactionList";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { consumeFlashToast } from "../../../../core/toast/flashToast";

import { Pagination } from "../../../../core/components/Pagination";
import { PageHeader } from "../../../../core/components/PageHeader";

export function TransactionListPage(): JSX.Element {
  const { items, loading, error, query, type, from, to, page, pageSize, total, setQuery, setType, setFrom, setTo, setPage, reload, remove } = useTransactionList();
  const { t } = useI18n();
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

  async function handleDelete(transactionID: string): Promise<void> {
    try {
      await remove(transactionID);
      success(t("common.deleteSuccess"));
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.deleteTransactionFailed"));
      throw err;
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("transactions.title")} 
        description={t("transactions.subtitle")}
      >
        <Link to="new" className="btn btn-primary">
          {t("transactions.create")}
        </Link>
      </PageHeader>
      <div className="surface card">
        <div className="filter-row">
          <label className="form-field">
            {t("transactions.filter.search")}
            <input
              className="input-control"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("transactions.filter.searchPlaceholder")}
            />
          </label>
          <label className="form-field">
            {t("transactions.form.type")}
            <select className="input-control" value={type} onChange={(event) => setType(event.target.value as "" | "income" | "expense" | "transfer" | "savings")}>
              <option value="">{t("transactions.filter.allTypes")}</option>
              <option value="expense">{t("transactions.type.expense")}</option>
              <option value="income">{t("transactions.type.income")}</option>
              <option value="transfer">{t("transactions.type.transfer")}</option>
              <option value="savings">{t("transactions.type.savings")}</option>
            </select>
          </label>
          <label className="form-field">
            {t("dashboard.from")}
            <input className="input-control" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
          </label>
          <label className="form-field">
            {t("dashboard.to")}
            <input className="input-control" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
          </label>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => {
              reload().catch(() => undefined);
            }}
          >
            {t("transactions.filter.apply")}
          </button>
        </div>
      </div>
      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}
      {!loading && !error ? (
        <>
          <TransactionTable items={items} onDelete={handleDelete} />
          <Pagination
            currentPage={page}
            pageSize={pageSize}
            totalItems={total}
            onPageChange={setPage}
            disabled={loading}
          />
        </>
      ) : null}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}
