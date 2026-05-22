import { FormEvent, useState } from "react";
import { SavingsPayload } from "../api/savings.types";
import { useI18n } from "../../../../core/i18n/i18n";

type Props = {
  initialValue?: Partial<SavingsPayload>;
  submitLabel: string;
  onSubmit: (payload: SavingsPayload) => Promise<void>;
  onCancel?: () => void;
};

const defaultForm: SavingsPayload = {
  name: "",
  target_amount_minor: 0,
  current_amount_minor: 0,
  currency: "IDR",
  start_date: "",
  target_date: "",
  notes: "",
  status: "active"
};

export function SavingsForm({ initialValue, submitLabel, onSubmit, onCancel }: Props): JSX.Element {
  const { t } = useI18n();
  const [form, setForm] = useState<any>({
    ...defaultForm,
    ...initialValue,
    target_amount_minor: initialValue?.target_amount_minor?.toString() ?? "",
    current_amount_minor: initialValue?.current_amount_minor?.toString() ?? "0"
  });
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    if (isSubmitting) return;
    setIsSubmitting(true);
    setError(null);
    try {
      await onSubmit({
        ...form,
        target_amount_minor: Number(form.target_amount_minor) || 0,
        current_amount_minor: Number(form.current_amount_minor) || 0,
        start_date: form.start_date || undefined,
        target_date: form.target_date || undefined,
        notes: form.notes || undefined
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="surface card">
      <form className="form-grid" onSubmit={(event) => handleSubmit(event).catch(() => undefined)}>
        <label className="form-field">
          {t("savings.form.name")}
          <input
            className="input-control"
            value={form.name}
            onChange={(event) => setForm((prev: any) => ({ ...prev, name: event.target.value }))}
            required
          />
        </label>
        <label className="form-field">
          {t("savings.form.target")}
          <input
            className="input-control"
            type="number"
            value={form.target_amount_minor}
            onChange={(event) => setForm((prev: any) => ({ ...prev, target_amount_minor: event.target.value }))}
            placeholder="0"
          />
        </label>
        <label className="form-field">
          {t("savings.form.current")}
          <input
            className="input-control"
            type="number"
            value={form.current_amount_minor}
            onChange={(event) => setForm((prev: any) => ({ ...prev, current_amount_minor: event.target.value }))}
            placeholder="0"
          />
        </label>
        <label className="form-field">
          {t("savings.form.currency")}
          <input
            className="input-control"
            value={form.currency}
            onChange={(event) => setForm((prev: any) => ({ ...prev, currency: event.target.value }))}
            required
          />
        </label>
        <label className="form-field">
          {t("savings.form.startDate")}
          <input
            className="input-control"
            type="date"
            value={form.start_date ?? ""}
            onChange={(event) => setForm((prev: any) => ({ ...prev, start_date: event.target.value }))}
          />
        </label>
        <label className="form-field">
          {t("savings.form.targetDate")}
          <input
            className="input-control"
            type="date"
            value={form.target_date ?? ""}
            onChange={(event) => setForm((prev: any) => ({ ...prev, target_date: event.target.value }))}
          />
        </label>
        <label className="form-field">
          {t("savings.form.status")}
          <select
            className="input-control"
            value={form.status ?? "active"}
            onChange={(event) => setForm((prev: any) => ({ ...prev, status: event.target.value }))}
          >
            <option value="active">{t("savings.status.active")}</option>
            <option value="completed">{t("savings.status.completed")}</option>
            <option value="cancelled">{t("savings.status.cancelled")}</option>
          </select>
        </label>
        <label className="form-field full-width">
          {t("savings.form.notes")}
          <textarea
            className="input-control"
            rows={3}
            value={form.notes ?? ""}
            onChange={(event) => setForm((prev: any) => ({ ...prev, notes: event.target.value }))}
            placeholder={t("savings.form.notes")}
          />
        </label>
        {error ? <p className="alert error">{error}</p> : null}
        <div className="form-actions" style={{ marginTop: "1rem" }}>
          <button className="btn btn-primary" type="submit" disabled={isSubmitting}>
            {isSubmitting ? t("common.loading") : submitLabel}
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

