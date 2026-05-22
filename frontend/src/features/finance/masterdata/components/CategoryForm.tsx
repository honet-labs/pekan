import { FormEvent, useState } from "react";

import { CreateCategoryPayload } from "../api/masterdata.types";
import { useI18n } from "../../../../core/i18n/i18n";

type Props = {
  disabled?: boolean;
  onSubmit: (payload: CreateCategoryPayload) => Promise<void>;
};

export function CategoryForm({ disabled, onSubmit }: Props): JSX.Element {
  const { t } = useI18n();
  const [form, setForm] = useState<CreateCategoryPayload>({
    name: "",
    category_type: "expense",
    parent_id: null
  });
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    if (isSubmitting) return;
    setIsSubmitting(true);
    setError(null);
    try {
      await onSubmit(form);
      setForm({
        name: "",
        category_type: form.category_type,
        parent_id: null
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.createCategoryFailed"));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <form className="form-grid" onSubmit={handleSubmit}>
      <h3 className="form-title">{t("categories.form.title")}</h3>
      <label className="form-field">
        {t("categories.form.name")}
        <input
          className="input-control"
          value={form.name}
          onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
          required
          disabled={disabled}
        />
      </label>
      <label className="form-field">
        {t("categories.form.type")}
        <select
          className="input-control"
          value={form.category_type}
          onChange={(e) =>
            setForm((prev) => ({
              ...prev,
              category_type: e.target.value as CreateCategoryPayload["category_type"]
            }))
          }
          disabled={disabled}
        >
          <option value="expense">{t("categories.type.expense")}</option>
          <option value="income">{t("categories.type.income")}</option>
        </select>
      </label>
      {error ? <p className="alert error">{error}</p> : null}
      <button className="btn btn-primary" type="submit" disabled={disabled || isSubmitting}>
        {isSubmitting ? t("common.loading") : t("categories.form.save")}
      </button>
    </form>
  );
}

