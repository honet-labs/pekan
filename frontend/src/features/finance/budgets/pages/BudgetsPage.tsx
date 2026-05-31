import { useMemo, useState } from "react";
import { useBudgets } from "../hooks/useBudgets";
import { Budget } from "../api/budgets.types";
import { useFinanceMasterData } from "../../masterdata/hooks/useFinanceMasterData";
import { useI18n } from "../../../../core/i18n/i18n";
import { AttachmentPanel } from "../../attachments/components/AttachmentPanel";
import { EntityTransactionsModal } from "../../transactions/components/EntityTransactionsModal";

const initialForm = {
  name: "",
  category_id: "",
  category_name: "",
  amount_limit_minor: 0,
  currency: "IDR",
  period: "monthly",
  start_date: "",
  end_date: "",
  alert_threshold_pct: 80,
  status: "active"
};

export function BudgetsPage(): JSX.Element {
  const { locale, t } = useI18n();
  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );
  const { items, loading, error, create, update, remove } = useBudgets();
  const { categories } = useFinanceMasterData();
  const [form, setForm] = useState(initialForm);
  const [editing, setEditing] = useState<Budget | null>(null);
  const [selectedBudgetID, setSelectedBudgetID] = useState<string | null>(null);
  const [transactionViewEntity, setTransactionViewEntity] = useState<Budget | null>(null);

  const categoryOptions = useMemo(
    () => categories.filter((category) => category.category_type === "expense"),
    [categories]
  );

  const title = useMemo(() => (editing ? t("budgets.form.update") : t("budgets.form.create")), [editing, t]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    try {
      const normalizedCategoryName = form.category_name.trim();
      const matchedCategory = categoryOptions.find((category) => category.name.toLowerCase() === normalizedCategoryName.toLowerCase());
      const payload = {
        name: form.name,
        category_id: matchedCategory?.id,
        category_name: normalizedCategoryName ? (matchedCategory ? matchedCategory.name : normalizedCategoryName) : undefined,
        amount_limit_minor: Number(form.amount_limit_minor),
        currency: form.currency,
        period: form.period,
        start_date: form.start_date,
        end_date: form.end_date || undefined,
        alert_threshold_pct: form.alert_threshold_pct || undefined,
        status: form.status
      };
      if (editing) {
        const updated = await update(editing.id, payload);
        setSelectedBudgetID(updated.id);
      } else {
        const created = await create(payload);
        setSelectedBudgetID(created.id);
      }
      setEditing(null);
      setForm(initialForm);
    } catch {
      // Hook already stores error state; keep page stable.
    }
  };

  const startEdit = (item: Budget) => {
    const categoryName = item.category_name ?? categories.find((category) => category.id === item.category_id)?.name ?? "";
    setEditing(item);
    setSelectedBudgetID(item.id);
    setForm({
      name: item.name,
      category_id: item.category_id ?? "",
      category_name: categoryName,
      amount_limit_minor: item.amount_limit_minor,
      currency: item.currency,
      period: item.period,
      start_date: item.start_date,
      end_date: item.end_date ?? "",
      alert_threshold_pct: item.alert_threshold_pct ?? 80,
      status: item.status
    });
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
        <form className="card surface form-grid" onSubmit={handleSubmit}>
          <h3 className="form-title">{title}</h3>
          <label className="form-field">
            {t("budgets.form.name")}
            <input
              className="input-control"
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
              required
            />
          </label>
          <label className="form-field">
            {t("budgets.form.category")}
            <input
              className="input-control"
              list="budget-category-options"
              value={form.category_name}
              onChange={(event) => setForm({ ...form, category_name: event.target.value })}
              placeholder={t("transactions.form.categoryPlaceholder")}
            />
            <datalist id="budget-category-options">
              {categoryOptions.map((category) => (
                <option key={category.id} value={category.name} />
              ))}
            </datalist>
          </label>
          <label className="form-field">
            {t("budgets.form.limit")}
            <input
              className="input-control"
              type="number"
              value={form.amount_limit_minor}
              onChange={(event) => setForm({ ...form, amount_limit_minor: Number(event.target.value) })}
              required
            />
          </label>
          <label className="form-field">
            {t("budgets.form.currency")}
            <input
              className="input-control"
              value={form.currency}
              onChange={(event) => setForm({ ...form, currency: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("budgets.form.period")}
            <select
              className="input-control"
              value={form.period}
              onChange={(event) => setForm({ ...form, period: event.target.value })}
            >
              <option value="monthly">{t("budgets.period.monthly")}</option>
              <option value="weekly">{t("budgets.period.weekly")}</option>
              <option value="yearly">{t("budgets.period.yearly")}</option>
              <option value="custom">{t("budgets.period.custom")}</option>
            </select>
          </label>
          <label className="form-field">
            {t("budgets.form.startDate")}
            <input
              className="input-control"
              type="date"
              value={form.start_date}
              onChange={(event) => setForm({ ...form, start_date: event.target.value })}
              required
            />
          </label>
          <label className="form-field">
            {t("budgets.form.endDate")}
            <input
              className="input-control"
              type="date"
              value={form.end_date}
              onChange={(event) => setForm({ ...form, end_date: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("budgets.form.alert")}
            <input
              className="input-control"
              type="number"
              value={form.alert_threshold_pct}
              onChange={(event) => setForm({ ...form, alert_threshold_pct: Number(event.target.value) })}
            />
          </label>
          <label className="form-field">
            {t("budgets.form.status")}
            <select
              className="input-control"
              value={form.status}
              onChange={(event) => setForm({ ...form, status: event.target.value })}
            >
              <option value="active">{t("budgets.status.active")}</option>
              <option value="paused">{t("budgets.status.paused")}</option>
              <option value="ended">{t("budgets.status.ended")}</option>
            </select>
          </label>
          <button className="btn btn-primary" type="submit">
            {editing ? t("budgets.form.updateBtn") : t("budgets.form.createBtn")}
          </button>
        </form>

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
                      <td>{item.ida}</td>
                      <td>{item.name}</td>
                      <td>{item.category_name ?? t("budgets.form.allCategories")}</td>
                      <td>Rp {numberFormatter.format(item.amount_limit_minor)}</td>
                      <td>
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
                      <td>{t(`budgets.period.${item.period}`)}</td>
                      <td>{t(`budgets.status.${item.status}`)}</td>
                      <td>
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

