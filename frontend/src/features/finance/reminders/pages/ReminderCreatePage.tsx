import { useState, useRef } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useReminders } from "../hooks/useReminders";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { PageHeader } from "../../../../core/components/PageHeader";

const initialForm = {
  title: "",
  description: "",
  amount_minor: 0,
  currency: "IDR",
  due_date: "",
  repeat_interval: "none",
  status: "pending"
};

export function ReminderCreatePage(): JSX.Element {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { tenantCode } = useParams();
  const listPath = tenantCode ? `/app/${tenantCode}/finance/reminders` : "/app/default/finance/reminders";
  const { toasts, success, error: showError, remove: removeToast } = useToast();
  const { create } = useReminders();
  const [form, setForm] = useState(initialForm);
  const [saving, setSaving] = useState(false);
  const savingRef = useRef(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (savingRef.current || saving) return;
    savingRef.current = true;
    setSaving(true);
    try {
      await create({
        title: form.title,
        description: form.description || undefined,
        amount_minor: form.amount_minor || undefined,
        currency: form.currency || undefined,
        due_date: form.due_date,
        repeat_interval: form.repeat_interval,
        status: form.status
      });
      success(t("common.saveSuccess"));
      setTimeout(() => {
        navigate(listPath);
      }, 1000);
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadRemindersFailed"));
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  };

  return (
    <section className="page-section">
      <PageHeader 
        title={t("reminders.form.create")} 
        description={t("reminders.subtitle")}
      />

      <div className="card surface">
        <form className="form-grid" onSubmit={handleSubmit}>
          <label className="form-field">
            {t("reminders.form.title")}
            <input 
              className="input-control" 
              value={form.title} 
              onChange={(event) => setForm({ ...form, title: event.target.value })} 
              required 
              disabled={saving}
            />
          </label>
          <label className="form-field">
            {t("reminders.form.description")}
            <textarea 
              className="input-control textarea-control" 
              value={form.description} 
              onChange={(event) => setForm({ ...form, description: event.target.value })} 
              disabled={saving}
            />
          </label>
          <div className="form-grid two-col">
            <label className="form-field">
              {t("reminders.form.amount")}
              <input 
                className="input-control" 
                type="number" 
                value={form.amount_minor} 
                onChange={(event) => setForm({ ...form, amount_minor: Number(event.target.value) })} 
                disabled={saving}
              />
            </label>
            <label className="form-field">
              {t("reminders.form.currency")}
              <input 
                className="input-control" 
                value={form.currency} 
                onChange={(event) => setForm({ ...form, currency: event.target.value })} 
                disabled={saving}
              />
            </label>
          </div>
          <div className="form-grid three-col">
            <label className="form-field">
              {t("reminders.form.dueDate")}
              <input 
                className="input-control" 
                type="date" 
                value={form.due_date} 
                onChange={(event) => setForm({ ...form, due_date: event.target.value })} 
                required 
                disabled={saving}
              />
            </label>
            <label className="form-field">
              {t("reminders.form.repeat")}
              <select 
                className="input-control" 
                value={form.repeat_interval} 
                onChange={(event) => setForm({ ...form, repeat_interval: event.target.value })}
                disabled={saving}
              >
                <option value="none">{t("reminders.repeat.none")}</option>
                <option value="daily">{t("reminders.repeat.daily")}</option>
                <option value="weekly">{t("reminders.repeat.weekly")}</option>
                <option value="monthly">{t("reminders.repeat.monthly")}</option>
              </select>
            </label>
            <label className="form-field">
              {t("reminders.form.status")}
              <select 
                className="input-control" 
                value={form.status} 
                onChange={(event) => setForm({ ...form, status: event.target.value })}
                disabled={saving}
              >
                <option value="pending">{t("reminders.status.pending")}</option>
                <option value="paid">{t("reminders.status.paid")}</option>
                <option value="cancelled">{t("reminders.status.cancelled")}</option>
              </select>
            </label>
          </div>
          <div className="form-actions" style={{ marginTop: "1rem" }}>
            <button className="btn btn-primary" type="submit" disabled={saving}>
              {saving ? t("common.loading") : t("reminders.form.createBtn")}
            </button>
            <button 
              className="btn btn-secondary" 
              type="button" 
              onClick={() => navigate(listPath)}
              disabled={saving}
            >
              {t("common.cancel")}
            </button>
          </div>
        </form>
      </div>

      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}

