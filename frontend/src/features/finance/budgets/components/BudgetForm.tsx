import { FormEvent, useMemo, useState } from "react";
import { BudgetPayload } from "../api/budgets.types";
import { FinanceCategory } from "../../masterdata/api/masterdata.types";
import { useI18n } from "../../../../core/i18n/i18n";

type Props = {
  categories: FinanceCategory[];
  initialValue?: Partial<BudgetPayload>;
  initialCategoryName?: string;
  submitLabel: string;
  onSubmit: (payload: BudgetPayload) => Promise<void>;
  onCancel?: () => void;
};

const now = new Date();
const firstDayOfMonth = new Date(now.getFullYear(), now.getMonth(), 1).toISOString().split("T")[0];

const defaultForm: BudgetPayload = {
  name: "",
  category_id: "",
  category_name: "",
  amount_limit_minor: 0,
  currency: "IDR",
  period: "monthly",
  start_date: firstDayOfMonth,
  end_date: "",
  alert_threshold_pct: 80,
  notes: "",
  status: "active"
};

export function BudgetForm({ categories, initialValue, initialCategoryName, submitLabel, onSubmit, onCancel }: Props): JSX.Element {
  const { t } = useI18n();
  const [form, setForm] = useState<BudgetPayload>({
    ...defaultForm,
    ...initialValue
  });
  const [selectedCategoryID, setSelectedCategoryID] = useState<string>(initialValue?.category_id ?? "");
  const [customCategoryName, setCustomCategoryName] = useState(() => {
    if (initialValue?.category_id) {
      return "";
    }
    return initialCategoryName ?? initialValue?.category_name ?? "";
  });
  const [error, setError] = useState<string | null>(null);

  const categoryOptions = useMemo(
    () => categories.filter((category) => category.category_type === "expense"),
    [categories]
  );

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError(null);
    const matchedCategory = categoryOptions.find((category) => category.id === selectedCategoryID);
    const normalizedCategoryName = customCategoryName.trim();

    try {
      await onSubmit({
        ...form,
        category_id: matchedCategory?.id,
        category_name: matchedCategory ? matchedCategory.name : normalizedCategoryName || undefined,
        end_date: form.end_date || undefined,
        alert_threshold_pct: form.alert_threshold_pct || undefined,
        notes: form.notes || undefined
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    }
  }

  return (
    <div className="surface card">
      <form className="form-grid" onSubmit={(event) => handleSubmit(event).catch(() => undefined)}>
        <label className="form-field">
          {t("budgets.form.name")}
          <input
            className="input-control"
            value={form.name}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, name: event.target.value }))}
            required
          />
        </label>
        <div className="form-field">
          <span>{t("budgets.form.category")}</span>
          <div className="checkbox-grid budget-category-grid">
            {categoryOptions.map((category) => {
              const checked = selectedCategoryID === category.id;
              return (
                <label key={category.id} className={`check-card${checked ? " is-checked" : ""}`}>
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => {
                      setSelectedCategoryID((prev) => (prev === category.id ? "" : category.id));
                      setCustomCategoryName("");
                    }}
                  />
                  <span>{category.name}</span>
                </label>
              );
            })}
          </div>
          <input
            className="input-control category-custom-input"
            value={customCategoryName}
            onChange={(event) => {
              setCustomCategoryName(event.target.value);
              if (event.target.value.trim()) {
                setSelectedCategoryID("");
              }
            }}
            placeholder={t("budgets.form.categoryCustom")}
          />
        </div>
        <label className="form-field">
          {t("budgets.form.limit")}
          <input
            className="input-control"
            type="number"
            value={form.amount_limit_minor}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, amount_limit_minor: Number(event.target.value) }))}
            required
          />
        </label>
        <label className="form-field">
          {t("budgets.form.currency")}
          <input
            className="input-control"
            value={form.currency}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, currency: event.target.value }))}
            required
          />
        </label>
        <label className="form-field">
          {t("budgets.form.period")}
          <select
            className="input-control"
            value={form.period}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, period: event.target.value }))}
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
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, start_date: event.target.value }))}
            required
          />
        </label>
        <label className="form-field">
          {t("budgets.form.endDate")}
          <input
            className="input-control"
            type="date"
            value={form.end_date ?? ""}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, end_date: event.target.value }))}
          />
        </label>
        <label className="form-field">
          {t("budgets.form.alert")}
          <input
            className="input-control"
            type="number"
            value={form.alert_threshold_pct ?? 80}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, alert_threshold_pct: Number(event.target.value) }))}
          />
        </label>
        <label className="form-field">
          {t("budgets.form.status")}
          <select
            className="input-control"
            value={form.status ?? "active"}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, status: event.target.value }))}
          >
            <option value="active">{t("budgets.status.active")}</option>
            <option value="paused">{t("budgets.status.paused")}</option>
            <option value="ended">{t("budgets.status.ended")}</option>
          </select>
        </label>
        <label className="form-field full-width">
          {t("budgets.form.notes")}
          <textarea
            className="input-control"
            rows={3}
            value={form.notes ?? ""}
            onChange={(event) => setForm((prev: BudgetPayload) => ({ ...prev, notes: event.target.value }))}
            placeholder={t("budgets.form.notes")}
          />
        </label>
        {error ? <p className="alert error">{error}</p> : null}
        <div className="form-actions" style={{ marginTop: "1rem" }}>
          <button className="btn btn-primary" type="submit">
            {submitLabel}
          </button>
          {onCancel && (
            <button className="btn btn-secondary" type="button" onClick={onCancel}>
              {t("common.cancel")}
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
