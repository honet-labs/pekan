import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { AttachmentPanel } from "../../attachments/components/AttachmentPanel";
import { getReminder, updateReminder, deleteReminder } from "../api/reminders.api";
import { Reminder, ReminderPayload } from "../api/reminders.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { PageHeader } from "../../../../core/components/PageHeader";

const initialForm = {
  title: "",
  description: "",
  amount_minor: 0,
  currency: "IDR",
  due_date: "",
  repeat_interval: "none",
  status: "pending",
  total_tenor: 0,
  current_tenor: 0
};

export function ReminderDetailPage(): JSX.Element {
  const { reminderID, tenantCode } = useParams();
  const navigate = useNavigate();
  const { locale, t } = useI18n();
  const { toasts, success, error: showError, remove: removeToast } = useToast();
  const [item, setItem] = useState<Reminder | null>(null);
  const [form, setForm] = useState(initialForm);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );

  async function loadReminder(): Promise<void> {
    if (!reminderID) {
      setError(t("reminders.error.missingId"));
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const reminder = await getReminder(reminderID);
      setItem(reminder);
      setForm({
        title: reminder.title,
        description: reminder.description ?? "",
        amount_minor: reminder.amount_minor ?? 0,
        currency: reminder.currency ?? "IDR",
        due_date: reminder.due_date,
        repeat_interval: reminder.repeat_interval,
        status: reminder.status,
        total_tenor: reminder.total_tenor ?? 0,
        current_tenor: reminder.current_tenor ?? 0
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.loadRemindersFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadReminder().catch(() => undefined);
  }, [reminderID, t]);

  async function handleSubmit(event: React.FormEvent): Promise<void> {
    event.preventDefault();
    if (!reminderID) {
      return;
    }
    if (saving) return;
    setSaving(true);
    setError(null);
    try {
      const payload: ReminderPayload = {
        title: form.title,
        description: form.description || undefined,
        amount_minor: form.amount_minor || undefined,
        currency: form.currency || undefined,
        due_date: form.due_date,
        repeat_interval: form.repeat_interval,
        status: form.status,
        total_tenor: form.total_tenor,
        current_tenor: form.current_tenor
      };
      const updated = await updateReminder(reminderID, payload);
      setItem(updated);
      success(t("common.updateSuccess"));
      setTimeout(() => {
        navigate(tenantCode ? `/app/${tenantCode}/finance/reminders` : "..");
      }, 1000);
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : t("errors.loadRemindersFailed");
      setError(errMsg);
      showError(errMsg);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(): Promise<void> {
    if (!reminderID) {
      return;
    }
    setIsDeleting(true);
    try {
      await deleteReminder(reminderID);
      navigate(tenantCode ? `/app/${tenantCode}/finance/reminders` : "..");
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadRemindersFailed"));
    } finally {
      setIsDeleting(false);
      setIsDeleteModalOpen(false);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("reminders.form.update")} 
        description={t("reminders.subtitle")}
      >
        {item ? (
          <button className="btn-icon-danger" type="button" onClick={() => setIsDeleteModalOpen(true)} disabled={saving} title={t("common.delete")}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
          </button>
        ) : null}
      </PageHeader>

      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      {!loading && item ? (
        <div className="card-grid two-col tight stack-on-mobile" style={{ maxWidth: '1200px' }}>
          <form className="card surface form-grid" onSubmit={(event) => handleSubmit(event).catch(() => undefined)}>
            <h3 className="form-title">{t("reminders.form.update")}</h3>
            <label className="form-field">
              {t("reminders.form.title")}
              <input className="input-control" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} required />
            </label>
            <label className="form-field">
              {t("reminders.form.description")}
              <textarea className="input-control textarea-control" value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} />
            </label>
            <div className="form-grid two-col">
              <label className="form-field">
                {t("reminders.form.amount")}
                <input className="input-control" type="number" value={form.amount_minor} onChange={(event) => setForm({ ...form, amount_minor: Number(event.target.value) })} />
              </label>
              <label className="form-field">
                {t("reminders.form.currency")}
                <input className="input-control" value={form.currency} onChange={(event) => setForm({ ...form, currency: event.target.value })} />
              </label>
            </div>
            <label className="form-field">
              {t("reminders.form.dueDate")}
              <input className="input-control" type="date" value={form.due_date} onChange={(event) => setForm({ ...form, due_date: event.target.value })} required />
            </label>
            <label className="form-field">
              {t("reminders.form.repeat")}
              <select className="input-control" value={form.repeat_interval} onChange={(event) => setForm({ ...form, repeat_interval: event.target.value })}>
                <option value="none">{t("reminders.repeat.none")}</option>
                <option value="daily">{t("reminders.repeat.daily")}</option>
                <option value="weekly">{t("reminders.repeat.weekly")}</option>
                <option value="monthly">{t("reminders.repeat.monthly")}</option>
              </select>
            </label>
            <label className="form-field">
              {t("reminders.form.status")}
              <select className="input-control" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
                <option value="pending">{t("reminders.status.pending")}</option>
                <option value="paid">{t("reminders.status.paid")}</option>
                <option value="cancelled">{t("reminders.status.cancelled")}</option>
              </select>
            </label>

            <div className="form-grid two-col tight" style={{ gridGap: '1rem' }}>
              <label className="form-field">
                {t("reminders.form.totalTenor")}
                <input className="input-control" type="number" min="0" value={form.total_tenor ?? 0} onChange={(event) => setForm({ ...form, total_tenor: Number(event.target.value) })} />
              </label>
              <label className="form-field">
                {t("reminders.form.currentTenor")}
                <input className="input-control" type="number" min="0" value={form.current_tenor ?? 0} onChange={(event) => setForm({ ...form, current_tenor: Number(event.target.value) })} />
              </label>
            </div>

            {item.updated_at ? (
              <p className="page-subtitle">
                {t("transactions.detail.updatedAt")}: {new Date(item.updated_at).toLocaleString(locale === "id" ? "id-ID" : "en-US")}
              </p>
            ) : null}
            <div className="form-actions" style={{ marginTop: "1rem" }}>
              <button className="btn btn-primary" type="submit" disabled={saving}>
                {saving ? t("common.loading") : t("reminders.form.updateBtn")}
              </button>
              
              {(form.total_tenor ?? 0) > 0 && (form.current_tenor ?? 0) < (form.total_tenor ?? 0) && (
                <button 
                  className="btn btn-success" 
                  type="button" 
                  disabled={saving}
                  onClick={async () => {
                    const nextTenor = (form.current_tenor ?? 0) + 1;
                    const d = new Date(form.due_date);
                    d.setMonth(d.getMonth() + 1);
                    const nextDate = d.toISOString().split('T')[0];
                    
                    const payload = {
                      ...form,
                      current_tenor: nextTenor,
                      due_date: nextDate,
                      status: "pending"
                    };
                    
                    try {
                      setSaving(true);
                      await updateReminder(reminderID!, payload);
                      await loadReminder();
                      success(t("common.updateSuccess"));
                    } catch (err) {
                      showError(err instanceof Error ? err.message : "Failed to advance tenor");
                    } finally {
                      setSaving(false);
                    }
                  }}
                >
                  {t("reminders.markPaidAdvance")}
                </button>
              )}

              <button className="btn btn-secondary" type="button" onClick={() => navigate("..")} disabled={saving}>
                {t("common.cancel")}
              </button>
            </div>
          </form>

          <div className="stack-col" style={{ gap: '1.5rem' }}>
            <div className="card surface">
              <h3 className="form-title">{t("reminders.title")}</h3>
              <p className="page-subtitle">{t("reminders.form.dueDate")}: {item.due_date}</p>
              <p className="page-subtitle">{t("transactions.table.total")}: Rp {numberFormatter.format(item.amount_minor ?? 0)}</p>
              {(item.total_tenor ?? 0) > 0 && (
                <p className="page-subtitle" style={{ marginTop: '0.5rem', fontWeight: 'bold' }}>
                  {t("reminders.tenor")}: {item.current_tenor ?? 0} / {item.total_tenor}
                </p>
              )}
            </div>
            <AttachmentPanel ownerType="reminders" ownerID={item.id} title={`${t("nav.reminders")} ${t("transactions.attachments.title")}`} />
          </div>
        </div>
      ) : null}

      <DeleteConfirmModal
        isOpen={isDeleteModalOpen}
        title={t("common.delete")}
        message={`${t("common.delete")} "${item?.title}"?`}
        isLoading={isDeleting}
        onConfirm={() => handleDelete().catch(() => undefined)}
        onCancel={() => setIsDeleteModalOpen(false)}
      />

      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}

