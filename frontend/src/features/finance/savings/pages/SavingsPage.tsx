import { useMemo, useState } from "react";
import { useSavings } from "../hooks/useSavings";
import { Savings } from "../api/savings.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { AttachmentPanel } from "../../attachments/components/AttachmentPanel";
import { EntityTransactionsModal } from "../../transactions/components/EntityTransactionsModal";

const initialForm = {
  name: "",
  target_amount_minor: 0,
  current_amount_minor: 0,
  currency: "IDR",
  start_date: "",
  target_date: "",
  status: "active"
};

function resolveProgressPercent(item: Savings): number {
  if (typeof item.progress_percent === "number" && Number.isFinite(item.progress_percent)) {
    return item.progress_percent;
  }
  if (item.target_amount_minor <= 0) {
    return 0;
  }
  return (item.current_amount_minor / item.target_amount_minor) * 100;
}

export function SavingsPage(): JSX.Element {
  const { locale, t } = useI18n();
  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );
  const { items, loading, error, create, update, remove } = useSavings();
  const [form, setForm] = useState(initialForm);
  const [editing, setEditing] = useState<Savings | null>(null);
  const [selectedSavingsID, setSelectedSavingsID] = useState<string | null>(null);
  const [transactionViewEntity, setTransactionViewEntity] = useState<Savings | null>(null);

  const title = useMemo(
    () => (editing ? t("savings.form.update") : t("savings.form.create")),
    [editing, t]
  );

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    try {
      const payload = {
        name: form.name,
        target_amount_minor: Number(form.target_amount_minor),
        current_amount_minor: Number(form.current_amount_minor),
        currency: form.currency,
        start_date: form.start_date || undefined,
        target_date: form.target_date || undefined,
        status: form.status
      };
      if (editing) {
        const updated = await update(editing.id, payload);
        setSelectedSavingsID(updated.id);
      } else {
        const created = await create(payload);
        setSelectedSavingsID(created.id);
      }
      setForm(initialForm);
      setEditing(null);
    } catch {
      // Error state is managed in hook to keep page stable.
    }
  };

  const startEdit = (item: Savings) => {
    setEditing(item);
    setSelectedSavingsID(item.id);
    setForm({
      name: item.name,
      target_amount_minor: item.target_amount_minor,
      current_amount_minor: item.current_amount_minor,
      currency: item.currency,
      start_date: item.start_date ?? "",
      target_date: item.target_date ?? "",
      status: item.status
    });
  };

  return (
    <section className="page-section">
      <header className="page-header">
        <div>
          <h1 className="page-title">{t("savings.title")}</h1>
          <p className="page-subtitle">{t("savings.subtitle")}</p>
        </div>
      </header>

      {error ? <div className="alert error">{error}</div> : null}

      <div className="card-grid two-col tight">
        <form className="card surface form-grid" onSubmit={handleSubmit}>
          <h3 className="form-title">{title}</h3>
          <label className="form-field">
            {t("savings.form.name")}
            <input
              className="input-control"
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
              required
            />
          </label>
          <label className="form-field">
            {t("savings.form.target")}
            <input
              className="input-control"
              type="number"
              value={form.target_amount_minor}
              onChange={(event) => setForm({ ...form, target_amount_minor: Number(event.target.value) })}
              required
            />
          </label>
          <label className="form-field">
            {t("savings.form.current")}
            <input
              className="input-control"
              type="number"
              value={form.current_amount_minor}
              onChange={(event) => setForm({ ...form, current_amount_minor: Number(event.target.value) })}
            />
          </label>
          <label className="form-field">
            {t("savings.form.currency")}
            <input
              className="input-control"
              value={form.currency}
              onChange={(event) => setForm({ ...form, currency: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("savings.form.startDate")}
            <input
              className="input-control"
              type="date"
              value={form.start_date}
              onChange={(event) => setForm({ ...form, start_date: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("savings.form.targetDate")}
            <input
              className="input-control"
              type="date"
              value={form.target_date}
              onChange={(event) => setForm({ ...form, target_date: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("savings.form.status")}
            <select
              className="input-control"
              value={form.status}
              onChange={(event) => setForm({ ...form, status: event.target.value })}
            >
              <option value="active">{t("savings.status.active")}</option>
              <option value="completed">{t("savings.status.completed")}</option>
              <option value="cancelled">{t("savings.status.cancelled")}</option>
            </select>
          </label>
          <button className="btn btn-primary" type="submit">
            {editing ? t("savings.form.updateBtn") : t("savings.form.createBtn")}
          </button>
        </form>

        <div className="stack-col">
          <div className="card surface">
            <h3 className="form-title">{t("savings.table.title")}</h3>
            {loading ? <p className="page-subtitle">{t("common.loading")}</p> : null}
            <div className="data-table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t("savings.table.id")}</th>
                    <th>{t("savings.table.name")}</th>
                    <th>{t("savings.table.target")}</th>
                    <th>{t("savings.table.current")}</th>
                    <th>{t("savings.table.progress")}</th>
                    <th>{t("savings.table.status")}</th>
                    <th>{t("savings.table.action")}</th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((item) => {
                    const progressPercent = resolveProgressPercent(item);
                    return (
                      <tr key={item.id}>
                        <td>{item.sid}</td>
                        <td>{item.name}</td>
                        <td>Rp {numberFormatter.format(item.target_amount_minor)}</td>
                        <td>Rp {numberFormatter.format(item.current_amount_minor)}</td>
                        <td>
                          <div className="progress-cell">
                            <span>{progressPercent.toFixed(2)}%</span>
                            <div className="progress-track">
                              <span
                                className="progress-fill"
                                style={{ width: `${Math.max(0, Math.min(progressPercent, 100))}%` }}
                              />
                            </div>
                          </div>
                        </td>
                        <td>{t(`savings.status.${item.status}`)}</td>
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
                            <button className="btn btn-ghost-inline icon-btn" type="button" onClick={() => setSelectedSavingsID(item.id)}>
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
                    );
                  })}
                  {!items.length ? (
                    <tr>
                      <td colSpan={7}>{t("common.noItems")}</td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </div>
          <AttachmentPanel ownerType="savings" ownerID={selectedSavingsID} title={`${t("nav.savings")} ${t("transactions.attachments.title")}`} />
        </div>
      </div>

      <EntityTransactionsModal
        isOpen={!!transactionViewEntity}
        title={`${t("nav.savings")}: ${transactionViewEntity?.name}`}
        type="savings"
        entityId={transactionViewEntity?.id || ""}
        onClose={() => setTransactionViewEntity(null)}
      />
    </section>
  );
}

