import { FormEvent, useState } from "react";

import { CreateAccountPayload } from "../api/masterdata.types";
import { useI18n } from "../../../../core/i18n/i18n";

type Props = {
  disabled?: boolean;
  onSubmit: (payload: CreateAccountPayload) => Promise<void>;
};

export function AccountForm({ disabled, onSubmit }: Props): JSX.Element {
  const { t } = useI18n();
  const [form, setForm] = useState<CreateAccountPayload>({
    name: "",
    account_type: "cash",
    currency: "IDR",
    opening_balance_minor: 0
  });
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError(null);
    try {
      await onSubmit(form);
      setForm({
        name: "",
        account_type: "cash",
        currency: form.currency,
        opening_balance_minor: 0
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.createAccountFailed"));
    }
  }

  return (
    <form className="form-grid" onSubmit={handleSubmit}>
      <h3 className="form-title">{t("accounts.form.title")}</h3>
      <label className="form-field">
        {t("accounts.form.name")}
        <input
          className="input-control"
          value={form.name}
          onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
          required
          disabled={disabled}
        />
      </label>
      <label className="form-field">
        {t("accounts.form.type")}
        <select
          className="input-control"
          value={form.account_type}
          onChange={(e) =>
            setForm((prev) => ({
              ...prev,
              account_type: e.target.value as CreateAccountPayload["account_type"]
            }))
          }
          disabled={disabled}
        >
          <option value="cash">{t("accounts.type.cash")}</option>
          <option value="bank">{t("accounts.type.bank")}</option>
          <option value="ewallet">{t("accounts.type.ewallet")}</option>
          <option value="credit">{t("accounts.type.credit")}</option>
        </select>
      </label>
      <label className="form-field">
        {t("accounts.form.currency")}
        <input
          className="input-control"
          value={form.currency}
          onChange={(e) => setForm((prev) => ({ ...prev, currency: e.target.value.toUpperCase().slice(0, 3) }))}
          required
          disabled={disabled}
        />
      </label>
      <label className="form-field">
        {t("accounts.form.openingBalance")}
        <input
          className="input-control"
          type="number"
          value={form.opening_balance_minor}
          onChange={(e) => setForm((prev) => ({ ...prev, opening_balance_minor: Number(e.target.value) }))}
          disabled={disabled}
        />
      </label>
      {error ? <p className="alert error">{error}</p> : null}
      <button className="btn btn-primary" type="submit" disabled={disabled}>
        {t("accounts.form.save")}
      </button>
    </form>
  );
}

