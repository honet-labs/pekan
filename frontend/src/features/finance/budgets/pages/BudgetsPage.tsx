import { useMemo, useState } from "react";
import { useBudgets } from "../hooks/useBudgets";
import { Budget } from "../api/budgets.types";
import { useFinanceMasterData } from "../../masterdata/hooks/useFinanceMasterData";
import { useI18n } from "../../../../core/i18n/i18n";
import { AttachmentPanel } from "../../attachments/components/AttachmentPanel";
import { EntityTransactionsModal } from "../../transactions/components/EntityTransactionsModal";
import { BudgetForm } from "../components/BudgetForm";

export function BudgetsPage(): JSX.Element {
  const { locale, t } = useI18n();
  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );
  const { items, loading, error, create, update, remove } = useBudgets();
  const { categories } = useFinanceMasterData();
  const [editing, setEditing] = useState<Budget | null>(null);
  const [selectedBudgetID, setSelectedBudgetID] = useState<string | null>(null);
  const [transactionViewEntity, setTransactionViewEntity] = useState<Budget | null>(null);

  const title = useMemo(() => (editing ? t("budgets.form.update") : t("budgets.form.create")), [editing, t]);

  const startEdit = (item: Budget) => {
    setEditing(item);
    setSelectedBudgetID(item.id);
  };

  return (
    <section className="page-section">
      <header className="page-header">
        <div>
          <h1 className="page-title">{t("budgets.title")}</h1>
          <p className="page-subtitle">{t("budgets.subtitle")}</p>
        </div>
      </header>

      {error ? <div className="alert error">{error}</div> : null}

      <div className="card-grid two-col tight">
        <div className="stack-col">
          <h3 className="form-title" style={{ margin: 0, marginBottom: "0.5rem" }}>{title}</h3>
          <BudgetForm
            key={editing ? editing.id : "new"}
            categories={categories}
            initialValue={editing ? {
              name: editing.name,
              category_id: editing.category_id ?? "",
              category_name: editing.category_name ?? "",
              amount_limit_minor: editing.amount_limit_minor,
              currency: editing.currency,
              period: editing.period,
              start_date: editing.start_date,
              end_date: editing.end_date ?? "",
              alert_threshold_pct: editing.alert_threshold_pct ?? 80,
              notes: editing.notes ?? "",
              status: editing.status
            } : undefined}
            initialCategoryName={editing?.category_name ?? ""}
            submitLabel={editing ? t("budgets.form.updateBtn") : t("budgets.form.createBtn")}
            onSubmit={async (payload) => {
              if (editing) {
                const updated = await update(editing.id, payload);
                setSelectedBudgetID(updated.id);
                setEditing(null);
              } else {
                const created = await create(payload);
                setSelectedBudgetID(created.id);
              }
            }}
            onCancel={editing ? () => setEditing(null) : undefined}
          />
        </div>

        <div className="stack-col">
          <div className="card surface">
            <h3 className="form-title">{t("budgets.table.title")}</h3>
            {loading ? <p className="page-subtitle">{t("common.loading")}</p> : null}
            <div className="data-table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>IDA</th>
                    <th>{t("budgets.table.name")}</th>
                    <th>{t("budgets.form.category")}</th>
                    <th>{t("budgets.table.limit")}</th>
                    <th>{t("budgets.table.progress")}</th>
                    <th>{t("budgets.table.period")}</th>
                    <th>{t("budgets.table.status")}</th>
                    <th>{t("budgets.table.action")}</th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((item) => (
                    <tr key={item.id}>
                      <td data-label="IDA">{item.ida}</td>
                      <td data-label={t("budgets.table.name")}>{item.name}</td>
                      <td data-label={t("budgets.form.category")}>{item.category_name ?? t("budgets.form.allCategories")}</td>
                      <td data-label={t("budgets.table.limit")}>Rp {numberFormatter.format(item.amount_limit_minor)}</td>
                      <td data-label={t("budgets.table.progress")}>
                        <div className="progress-cell">
                          <span>{Number(item.progress_percent ?? 0).toFixed(2)}%</span>
                          <div className="progress-track">
                            <span
                              className="progress-fill"
                              style={{ width: `${Math.max(0, Math.min(Number(item.progress_percent ?? 0), 100))}%` }}
                            />
                          </div>
                        </div>
                      </td>
                      <td data-label={t("budgets.table.period")}>{t(`budgets.period.${item.period}`)}</td>
                      <td data-label={t("budgets.table.status")}>{t(`budgets.status.${item.status}`)}</td>
                      <td data-label={t("budgets.table.action")}>
                        <div className="table-actions">
                          <button
                            className="btn btn-primary-light"
                            type="button"
                            title="Lihat Transaksi"
                            onClick={() => setTransactionViewEntity(item)}
                            style={{ 
                              fontSize: '11px', 
                              padding: '4px 10px', 
                              border: '1px solid var(--primary-color)', 
                              borderRadius: '4px',
                              display: 'inline-flex',
                              alignItems: 'center',
                              gap: '4px',
                              color: 'var(--primary-color)',
                              fontWeight: 600,
                              background: 'rgba(var(--primary-color-rgb), 0.1)',
                              marginRight: '8px'
                            }}
                          >
                            <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                              <line x1="8" y1="6" x2="21" y2="6" /><line x1="8" y1="12" x2="21" y2="12" /><line x1="8" y1="18" x2="21" y2="18" />
                              <line x1="3" y1="6" x2="3.01" y2="6" /><line x1="3" y1="12" x2="3.01" y2="12" /><line x1="3" y1="18" x2="3.01" y2="18" />
                            </svg>
                            Lihat Transaksi
                          </button>
                          <button className="btn btn-ghost-inline" type="button" onClick={() => startEdit(item)}>
                            {t("common.edit")}
                          </button>
                          <button className="btn btn-ghost-inline icon-btn" type="button" onClick={() => setSelectedBudgetID(item.id)}>
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
                          <button className="btn btn-ghost-inline danger" type="button" onClick={() => remove(item.id).catch(() => undefined)}>
                            {t("common.delete")}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {!items.length ? (
                    <tr>
                      <td colSpan={8}>{t("common.noItems")}</td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </div>
          <AttachmentPanel ownerType="budgets" ownerID={selectedBudgetID} title={`${t("nav.budgets")} ${t("transactions.attachments.title")}`} />
        </div>
      </div>

      <EntityTransactionsModal
        isOpen={!!transactionViewEntity}
        title={`${t("nav.budgets")}: ${transactionViewEntity?.name}`}
        type="budget"
        entityId={transactionViewEntity?.category_id || ""}
        startDate={transactionViewEntity?.start_date}
        endDate={transactionViewEntity?.end_date}
        onClose={() => setTransactionViewEntity(null)}
      />
    </section>
  );
}

