import { FormEvent, ReactNode, useEffect, useMemo, useState, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { FinanceAccount, FinanceCategory } from "../../masterdata/api/masterdata.types";
import { CreateTransactionPayload, TransactionItem, TransactionType } from "../api/transaction.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { Savings } from "../../savings/api/savings.types";
import { TransactionItemsForm } from "./TransactionItemsForm";

type SubmitInput = {
  payload: CreateTransactionPayload;
  files: File[];
};

type Props = {
  onSubmit: (input: SubmitInput) => Promise<void>;
  accounts: FinanceAccount[];
  categories: FinanceCategory[];
  savingsOptions: Savings[];
  initialValue?: Partial<CreateTransactionPayload>;
  initialCategoryName?: string;
  submitLabel?: string;
  includeAttachmentUpload?: boolean;
  actorLabel?: string;
  onCancel?: () => void;
  initialFiles?: File[];
};

export function TransactionForm({
  onSubmit,
  accounts,
  categories,
  savingsOptions,
  initialValue,
  initialCategoryName,
  submitLabel,
  includeAttachmentUpload = true,
  actorLabel,
  onCancel,
  initialFiles = []
}: Props): JSX.Element {
  const navigate = useNavigate();
  const { tenantCode } = useParams();
  const { locale, t } = useI18n();
  const isIgnorableFileRequirementError = (message: string): boolean => {
    const normalized = message.toLowerCase();
    return (
      /f(?:i)?le.*required|required.*f(?:i)?le/.test(normalized) ||
      /f\w*\s*(is|are)?\s*required/.test(normalized) ||
      normalized.includes("file is required") ||
      normalized.includes("file required") ||
      normalized.includes("file_required") ||
      normalized.includes("at least one file is required") ||
      normalized.includes("invalid multipart")
    );
  };
  const [form, setForm] = useState<CreateTransactionPayload>({
    account_id: initialValue?.account_id ?? "",
    category_id: initialValue?.category_id ?? null,
    category_name: initialValue?.category_name ?? null,
    savings_ids: initialValue?.savings_ids ?? [],
    type: (initialValue?.type as TransactionType | undefined) ?? "expense",
    amount_minor: Number(initialValue?.amount_minor ?? 0),
    currency: (initialValue?.currency ?? "IDR").toUpperCase().slice(0, 3),
    input_date: initialValue?.input_date ?? new Date().toISOString().slice(0, 10),
    transaction_date: initialValue?.transaction_date ?? new Date().toISOString().slice(0, 10),
    description: initialValue?.description ?? "",
    merchant_name: initialValue?.merchant_name ?? "",
    receipt_number: initialValue?.receipt_number ?? "",
    payment_method: initialValue?.payment_method ?? "",
    subtotal_minor: Number(initialValue?.subtotal_minor ?? 0),
    tax_minor: Number(initialValue?.tax_minor ?? 0),
    service_charge_minor: Number(initialValue?.service_charge_minor ?? 0),
    receipt_discount_minor: Number(initialValue?.receipt_discount_minor ?? 0),
    items: Array.isArray(initialValue?.items) ? initialValue.items : [],
    receipt_scan_id: initialValue?.receipt_scan_id ?? undefined
  });
  const [selectedCategoryID, setSelectedCategoryID] = useState<string>(initialValue?.category_id ?? "");
  const [customCategoryName, setCustomCategoryName] = useState(() => {
    if (initialValue?.category_id) {
      return "";
    }
    return initialCategoryName ?? initialValue?.category_name ?? "";
  });
  const [selectedSavingsIDs, setSelectedSavingsIDs] = useState<string[]>(() =>
    Array.isArray(initialValue?.savings_ids) ? initialValue.savings_ids.filter(Boolean) : []
  );
  const [files, setFiles] = useState<File[]>(initialFiles);
  const [error, setError] = useState<ReactNode | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const isSubmittingRef = useRef(false);

  useEffect(() => {
    if (!accounts.length || form.account_id) {
      return;
    }
    const normalizedActor = actorLabel?.trim().toLowerCase();
    const matchedAccount = normalizedActor
      ? accounts.find((account) => account.name.trim().toLowerCase() === normalizedActor) ??
        accounts.find((account) => account.name.trim().toLowerCase().includes(normalizedActor))
      : undefined;
    setForm((prev) => ({ ...prev, account_id: matchedAccount?.id ?? accounts[0].id }));
  }, [accounts, actorLabel, form.account_id]);

  const categoryOptions = useMemo(() => {
    if (form.type === "transfer" || form.type === "savings") {
      return [] as FinanceCategory[];
    }
    const normalizedType = form.type.toLowerCase();
    return categories.filter((item) => {
      const rawType = String((item as FinanceCategory & { type?: string }).category_type ?? (item as FinanceCategory & { type?: string }).type ?? "").toLowerCase();
      return rawType === normalizedType;
    });
  }, [categories, form.type]);

  useEffect(() => {
    if (form.type === "transfer" || form.type === "savings") {
      if (selectedCategoryID || customCategoryName) {
        setSelectedCategoryID("");
        setCustomCategoryName("");
      }
      return;
    }
    if (selectedCategoryID && !categoryOptions.some((item) => item.id === selectedCategoryID)) {
      setSelectedCategoryID("");
    }
  }, [categoryOptions, customCategoryName, form.type, selectedCategoryID]);

  useEffect(() => {
    if (form.type !== "expense") {
      return;
    }
    const items = Array.isArray(form.items) ? form.items : [];
    const itemsTotal = items.reduce((sum, item) => sum + Number(item.total_minor || 0), 0);
    const subtotal = Number(form.subtotal_minor || 0);
    const tax = Number(form.tax_minor || 0);
    const serviceCharge = Number(form.service_charge_minor || 0);
    const receiptDiscount = Number(form.receipt_discount_minor || 0);
    let nextAmount = form.amount_minor;
    if (subtotal > 0 || tax > 0 || serviceCharge > 0 || receiptDiscount > 0) {
      const base = subtotal > 0 ? subtotal : itemsTotal;
      nextAmount = Math.max(base + tax + serviceCharge - receiptDiscount, 0);
    } else if (items.length > 0) {
      nextAmount = itemsTotal;
    }
    if (nextAmount !== form.amount_minor) {
      setForm((prev) => ({ ...prev, amount_minor: nextAmount }));
    }
  }, [form.items, form.type, form.amount_minor, form.subtotal_minor, form.tax_minor, form.service_charge_minor, form.receipt_discount_minor]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    if (isSubmittingRef.current || isSubmitting) return;
    isSubmittingRef.current = true;
    setIsSubmitting(true);
    setError(null);

    let categoryID: string | null = null;
    let normalizedCategoryName: string | null = null;

    if (form.type !== "transfer" && form.type !== "savings") {
      const matched = categoryOptions.find((item) => item.id === selectedCategoryID);
      const normalized = customCategoryName.trim();
      if (matched) {
        categoryID = matched.id;
        normalizedCategoryName = matched.name;
      } else if (normalized) {
        normalizedCategoryName = normalized;
      }
    }

    const normalizedSavingsIDs = form.type === "savings" ? selectedSavingsIDs : [];
    if (form.type === "savings" && normalizedSavingsIDs.length === 0) {
      setError(t("transactions.form.savingsRequired"));
      isSubmittingRef.current = false;
      setIsSubmitting(false);
      return;
    }

    const normalizedItems: TransactionItem[] = form.type === "expense"
      ? (Array.isArray(form.items) ? form.items : []).filter((item) => item.item_name?.trim())
      : [];

    const finalAccountID = form.account_id || (accounts.length > 0 ? accounts[0].id : "");
    if (!finalAccountID) {
      setError(
        <span>
          {t("errors.noAccountAvailable")} 
          <br />
          <button 
            type="button" 
            className="btn-link" 
            style={{ padding: 0, marginTop: '0.5rem' }} 
            onClick={() => navigate(`/app/${tenantCode}/finance/master-data`)}
          >
            {t("nav.masterData")} &rarr;
          </button>
        </span>
      );
      isSubmittingRef.current = false;
      setIsSubmitting(false);
      return;
    }

    try {
      await onSubmit({
        payload: {
          ...form,
          account_id: finalAccountID,
          input_date: undefined,
          category_id: categoryID,
          category_name: normalizedCategoryName,
          savings_ids: normalizedSavingsIDs,
          items: normalizedItems,
          receipt_scan_id: form.receipt_scan_id
        },
        files
      });
      setFiles([]);
    } catch (err) {
      const message = err instanceof Error ? err.message : t("errors.saveTransactionFailed");
      if (files.length > 0 && isIgnorableFileRequirementError(message)) {
        setError(null);
        isSubmittingRef.current = false;
        setIsSubmitting(false);
        return;
      }
      setError(message);
    } finally {
      isSubmittingRef.current = false;
      setIsSubmitting(false);
    }
  }

  return (
    <div className="surface card">
      <form className="form-grid" onSubmit={handleSubmit}>
        <input type="hidden" value={form.account_id} />

        <label className="form-field">
          {t("transactions.form.user")}
          <input className="input-control" value={actorLabel ?? "-"} readOnly disabled />
        </label>

        <label className="form-field">
          {t("transactions.form.type")}
          <select
            className="input-control"
            value={form.type}
            onChange={(e) =>
              setForm((prev) => ({
                ...prev,
                type: e.target.value as TransactionType,
                items: e.target.value === "expense" ? prev.items : [],
                merchant_name: e.target.value === "expense" ? prev.merchant_name : "",
                receipt_number: e.target.value === "expense" ? prev.receipt_number : "",
                payment_method: e.target.value === "expense" ? prev.payment_method : "",
                subtotal_minor: e.target.value === "expense" ? prev.subtotal_minor : 0,
                tax_minor: e.target.value === "expense" ? prev.tax_minor : 0,
                service_charge_minor: e.target.value === "expense" ? prev.service_charge_minor : 0,
                receipt_discount_minor: e.target.value === "expense" ? prev.receipt_discount_minor : 0
              }))
            }
          >
            <option value="expense">{t("transactions.type.expense")}</option>
            <option value="income">{t("transactions.type.income")}</option>
            <option value="transfer">{t("transactions.type.transfer")}</option>
            <option value="savings">{t("transactions.type.savings")}</option>
          </select>
        </label>

        {form.type === "transfer" ? (
          <label className="form-field">
            {t("transactions.form.category")}
            <input className="input-control" value={t("transactions.form.notUsed")} readOnly disabled />
          </label>
        ) : form.type === "savings" ? (
          <label className="form-field">
            {t("transactions.form.category")}
            <input className="input-control" value={t("transactions.type.savings")} readOnly disabled />
          </label>
        ) : (
          <div className="form-field">
            <span>{t("transactions.form.category")}</span>
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
              {!categoryOptions.length ? <p className="page-subtitle">{t("common.noItems")}</p> : null}
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
              placeholder={t("transactions.form.categoryPlaceholder")}
            />
          </div>
        )}

        {form.type === "expense" ? (
          <TransactionItemsForm
            items={Array.isArray(form.items) ? form.items : []}
            onItemsChange={(items) => setForm((prev) => ({ ...prev, items }))}
            locale={locale}
          />
        ) : null}

        {form.type === "savings" ? (
          <fieldset className="form-field">
            <legend>{t("transactions.form.savingsTargets")}</legend>
            <div className="entity-list">
              {savingsOptions.map((goal) => (
                <label key={goal.id} className="entity-item">
                  <input
                    type="checkbox"
                    checked={selectedSavingsIDs.includes(goal.id)}
                    onChange={(event) => {
                      if (event.target.checked) {
                        setSelectedSavingsIDs((prev) => [...prev, goal.id]);
                      } else {
                        setSelectedSavingsIDs((prev) => prev.filter((id) => id !== goal.id));
                      }
                    }}
                  />
                  <span>{goal.name}</span>
                </label>
              ))}
              {!savingsOptions.length ? <p className="page-subtitle">{t("common.noItems")}</p> : null}
            </div>
          </fieldset>
        ) : null}

        {form.type === "expense" ? (
          <div className="receipt-meta-card">
            <h3 className="form-title">{t("transactions.receipt.title")}</h3>
            <div className="receipt-meta-grid">
              <label className="form-field">
                {t("transactions.receipt.merchant")}
                <input className="input-control" value={form.merchant_name ?? ""} onChange={(e) => setForm((prev) => ({ ...prev, merchant_name: e.target.value }))} placeholder={t("transactions.receipt.merchant")} />
              </label>
              <label className="form-field">
                {t("transactions.receipt.paymentMethod")}
                <select className="input-control" value={form.payment_method ?? ""} onChange={(e) => setForm((prev) => ({ ...prev, payment_method: e.target.value }))}>
                  <option value="">{t("common.selectPlaceholder")}</option>
                  <option value="cash">{t("transactions.receipt.payment.cash")}</option>
                  <option value="debit">{t("transactions.receipt.payment.debit")}</option>
                  <option value="credit">{t("transactions.receipt.payment.credit")}</option>
                  <option value="ewallet">{t("transactions.receipt.payment.ewallet")}</option>
                  <option value="bank_transfer">{t("transactions.receipt.payment.bankTransfer")}</option>
                  <option value="qr">{t("transactions.receipt.payment.qr")}</option>
                  <option value="unknown">{t("transactions.receipt.payment.unknown")}</option>
                </select>
              </label>
              <label className="form-field">
                {t("transactions.receipt.receiptNumber")}
                <input className="input-control" value={form.receipt_number ?? ""} onChange={(e) => setForm((prev) => ({ ...prev, receipt_number: e.target.value }))} placeholder={t("transactions.receipt.receiptNumber")} />
              </label>
              <label className="form-field">
                {t("transactions.receipt.subtotal")}
                <input className="input-control" type="number" min={0} value={form.subtotal_minor ?? 0} onChange={(e) => setForm((prev) => ({ ...prev, subtotal_minor: Number(e.target.value) || 0 }))} />
              </label>
              <label className="form-field">
                {t("transactions.receipt.tax")}
                <input className="input-control" type="number" min={0} value={form.tax_minor ?? 0} onChange={(e) => setForm((prev) => ({ ...prev, tax_minor: Number(e.target.value) || 0 }))} />
              </label>
              <label className="form-field">
                {t("transactions.receipt.serviceCharge")}
                <input className="input-control" type="number" min={0} value={form.service_charge_minor ?? 0} onChange={(e) => setForm((prev) => ({ ...prev, service_charge_minor: Number(e.target.value) || 0 }))} />
              </label>
              <label className="form-field">
                {t("transactions.receipt.discount")}
                <input className="input-control" type="number" min={0} value={form.receipt_discount_minor ?? 0} onChange={(e) => setForm((prev) => ({ ...prev, receipt_discount_minor: Number(e.target.value) || 0 }))} />
              </label>
            </div>
          </div>
        ) : null}

        <label className="form-field">
          {t("transactions.form.amount")}
          <input
            className="input-control"
            type="number"
            min={1}
            value={form.amount_minor}
            onChange={(e) => setForm((prev) => ({ ...prev, amount_minor: Number(e.target.value) }))}
            required
          />
        </label>

        <label className="form-field">
          {t("transactions.form.currency")}
          <input
            className="input-control"
            value={form.currency}
            onChange={(e) => setForm((prev) => ({ ...prev, currency: e.target.value.toUpperCase().slice(0, 3) }))}
            required
          />
        </label>

        <label className="form-field">
          {t("transactions.form.inputDate")}
          <input className="input-control" type="date" value={form.input_date ?? ""} readOnly disabled required />
        </label>

        <label className="form-field">
          {t("transactions.form.date")}
          <input
            className="input-control"
            type="date"
            value={form.transaction_date}
            onChange={(e) => setForm((prev) => ({ ...prev, transaction_date: e.target.value }))}
            required
          />
        </label>

        <label className="form-field">
          {t("transactions.form.description")}
          <textarea
            className="input-control textarea-control"
            value={form.description ?? ""}
            onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))}
          />
        </label>

        {includeAttachmentUpload ? (
          <label className="form-field">
            {t("transactions.form.attachments")}
            <input
              className="input-control"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              multiple
              onChange={(e) => setFiles(Array.from(e.target.files ?? []))}
            />
            {files.length > 0 ? (
              <div style={{ marginTop: "0.5rem" }}>
                <small className="page-subtitle" style={{ display: "block", marginBottom: "0.25rem" }}>
                  {files.length} {t("transactions.form.filesSelected")}
                </small>
                <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
                  {files.map((file, idx) => (
                    <div key={`${file.name}-${idx}`} className="badge-status running-flat" style={{ fontSize: "0.75rem", display: "flex", alignItems: "center", gap: "5px" }}>
                      <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="3"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                      {file.name}
                    </div>
                  ))}
                </div>
              </div>
            ) : form.receipt_scan_id ? (
              <div className="badge-status running-flat" style={{ marginTop: "0.5rem", fontSize: "0.8rem", display: "inline-flex", alignItems: "center", gap: "8px" }}>
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="3"><polyline points="20 6 9 17 4 12"/></svg>
                Struk terscan otomatis terlampir
              </div>
            ) : null}
          </label>
        ) : null}

        {error ? <p className="alert error">{error}</p> : null}
        <div className="form-actions" style={{ marginTop: "1rem" }}>
          <button className="btn btn-primary" type="submit" disabled={isSubmitting}>
            {isSubmitting ? t("common.loading") : (submitLabel ?? t("transactions.form.save"))}
          </button>
          {onCancel && (
            <button className="btn btn-secondary-outline" type="button" onClick={onCancel}>
              {t("common.cancel")}
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
