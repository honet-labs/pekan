import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useReminders } from "../hooks/useReminders";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { PageHeader } from "../../../../core/components/PageHeader";
import { useToast } from "../../../../core/hooks/useToast";
import { Pagination } from "../../../../core/components/Pagination";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { Reminder, ReminderPayment } from "../api/reminders.types";
import { AddPaymentModal } from "../components/AddPaymentModal";
import { PaymentHistory } from "../components/PaymentHistory";
import { InfoModal } from "../../../../core/components/InfoModal";

export function RemindersPage(): JSX.Element {
  const { locale, t } = useI18n();
  const navigate = useNavigate();
  const { tenantCode } = useParams();
  const createPath = tenantCode ? `/app/${tenantCode}/finance/reminders/new` : "/app/default/finance/reminders/new";
  const { toasts, success, error: showError, remove: removeToast } = useToast();
  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );
  const { 
    items, 
    dueItems, 
    loading, 
    error, 
    page, 
    pageSize, 
    total, 
    setPage, 
    markStatus, 
    remove, 
    fetchPayments, 
    addPayment,
    updatePayment,
    removePayment
  } = useReminders();
  const [activeTab, setActiveTab] = useState<"pending" | "paid">("pending");
  const [itemToDelete, setItemToDelete] = useState<Reminder | null>(null);
  const [deletingID, setDeletingID] = useState<string | null>(null);
  const [selectedItemForDetail, setSelectedItemForDetail] = useState<Reminder | null>(null);
  const [paymentsHistory, setPaymentsHistory] = useState<ReminderPayment[]>([]);
  const [isAddingPayment, setIsAddingPayment] = useState(false);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [submittingPayment, setSubmittingPayment] = useState(false);
  const [editingPayment, setEditingPayment] = useState<ReminderPayment | null>(null);
  const [descToShow, setDescToShow] = useState<{ title: string; description: string } | null>(null);

  const unpaidItems = useMemo(() => items.filter(item => item.status !== "paid"), [items]);
  const paidItems = useMemo(() => items.filter(item => item.status === "paid"), [items]);
  const displayedItems = useMemo(() => activeTab === "pending" ? unpaidItems : paidItems, [activeTab, unpaidItems, paidItems]);

  useEffect(() => {
    if (selectedItemForDetail) {
      setLoadingHistory(true);
      fetchPayments(selectedItemForDetail.id)
        .then(setPaymentsHistory)
        .catch(() => setPaymentsHistory([]))
        .finally(() => setLoadingHistory(false));
    } else {
      setPaymentsHistory([]);
    }
  }, [selectedItemForDetail, fetchPayments]);

  const handleDelete = async (id: string) => {
    setDeletingID(id);
    try {
      await remove(id);
      success(t("common.deleteSuccess"));
      setItemToDelete(null);
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.loadRemindersFailed"));
    } finally {
      setDeletingID(null);
    }
  };

  return (
    <section className="page-section">
      <PageHeader 
        title={t("reminders.title")} 
        description={t("reminders.subtitle")}
      >
        <button className="btn btn-primary" onClick={() => navigate(createPath)}>
          {t("reminders.form.create")}
        </button>
      </PageHeader>

      {error ? <div className="alert error">{error}</div> : null}

      <div className="card surface" style={{ marginBottom: "2rem" }}>
        <h3 className="form-title">{t("reminders.dueSoon")}</h3>
        {loading && !items.length ? <p className="page-subtitle">{t("common.loading")}</p> : null}
        <ul className="entity-list">
          {dueItems.map((item) => (
            <li key={item.id} className="entity-item">
              <div style={{ flex: 1 }}>
                <strong>{item.title}</strong>
                <p className="page-subtitle" style={{ margin: "0.25rem 0" }}>
                  {t("reminders.form.dueDate")} {item.due_date}
                </p>
              </div>
              <div className="inline-metrics" style={{ flexShrink: 0 }}>
                <span className="pill expense">Rp {numberFormatter.format(item.amount_minor ?? 0)}</span>
                <button
                  className="btn btn-ghost-inline"
                  type="button"
                  onClick={() =>
                    markStatus(item.id, "paid")
                      .then(() => success(t("common.updateSuccess")))
                      .catch((err) => showError(err instanceof Error ? err.message : t("errors.loadRemindersFailed")))
                  }
                >
                  {t("reminders.markPaid")}
                </button>
              </div>
            </li>
          ))}
          {!dueItems.length && !loading ? <li className="entity-item">{t("common.noItems")}</li> : null}
        </ul>
      </div>

        <div className="card surface">
          <div style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            flexWrap: "wrap",
            gap: "1rem",
            marginBottom: "1rem"
          }}>
            <h3 className="form-title" style={{ margin: 0 }}>{t("reminders.all")}</h3>
            
            {/* Tabs */}
            <div style={{
              display: "flex",
              background: "var(--surface-soft)",
              padding: "4px",
              borderRadius: "10px",
              border: "1px solid var(--border)"
            }}>
              <button
                type="button"
                onClick={() => setActiveTab("pending")}
                style={{
                  padding: "6px 16px",
                  borderRadius: "8px",
                  border: "none",
                  background: activeTab === "pending" ? "var(--primary)" : "transparent",
                  color: activeTab === "pending" ? "#fff" : "var(--text-muted)",
                  fontWeight: 600,
                  fontSize: "0.875rem",
                  cursor: "pointer",
                  transition: "all 0.2s"
                }}
              >
                Belum Lunas ({unpaidItems.length})
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("paid")}
                style={{
                  padding: "6px 16px",
                  borderRadius: "8px",
                  border: "none",
                  background: activeTab === "paid" ? "var(--primary)" : "transparent",
                  color: activeTab === "paid" ? "#fff" : "var(--text-muted)",
                  fontWeight: 600,
                  fontSize: "0.875rem",
                  cursor: "pointer",
                  transition: "all 0.2s"
                }}
              >
                Lunas ({paidItems.length})
              </button>
            </div>
          </div>

          <div className="data-table-wrap table-mobile-stack">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t("reminders.form.title")}</th>
                  <th>{t("reminders.form.dueDate")}</th>
                  <th>{t("transactions.table.total")}</th>
                  <th>{t("reminders.form.description")}</th>
                  <th>{t("reminders.form.status")}</th>
                  <th>{t("transactions.table.action")}</th>
                </tr>
              </thead>
              <tbody>
                {displayedItems.map((item) => (
                  <tr key={item.id}>
                    <td data-label={t("reminders.form.title")}>{item.title}</td>
                    <td data-label={t("reminders.form.dueDate")}>{item.due_date}</td>
                    <td data-label={t("transactions.table.total")}>Rp {numberFormatter.format(item.amount_minor ?? 0)}</td>
                    <td data-label={t("reminders.form.description")}>
                      {item.description ? (
                        <button 
                          className="btn btn-ghost-inline" 
                          title={t("common.view")}
                          onClick={() => setDescToShow({ title: item.title, description: item.description || "" })}
                        >
                          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                            <circle cx="12" cy="12" r="3" />
                          </svg>
                        </button>
                      ) : "-"}
                    </td>
                    <td data-label={t("reminders.form.status")}>{t(`reminders.status.${item.status}`)}</td>
                    <td data-label={t("transactions.table.action")}>
                      <div className="table-actions">
                        <button className="btn btn-ghost-inline" type="button" onClick={() => setSelectedItemForDetail(item)}>
                          {t("common.detail")}
                        </button>
                        <Link className="btn btn-ghost-inline" to={item.id}>
                          {t("common.edit")}
                        </Link>
                        <button 
                          className="btn btn-ghost-inline danger" 
                          type="button" 
                          onClick={() => setItemToDelete(item)}
                        >
                          {t("common.delete")}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!displayedItems.length && !loading ? (
                  <tr>
                    <td colSpan={6}>{t("common.noItems")}</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
          <Pagination
            currentPage={page}
            pageSize={pageSize}
            totalItems={activeTab === "pending" ? unpaidItems.length : paidItems.length}
            onPageChange={setPage}
            disabled={loading}
          />
        </div>

      <DeleteConfirmModal
        isOpen={!!itemToDelete}
        title={t("common.delete")}
        message={`${t("common.delete")} "${itemToDelete?.title}"?`}
        isLoading={deletingID !== null}
        onConfirm={() => itemToDelete && handleDelete(itemToDelete.id)}
        onCancel={() => setItemToDelete(null)}
      />

      {selectedItemForDetail && (
        <div className="modal-overlay" onClick={() => setSelectedItemForDetail(null)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header" style={{ padding: "0 0 1rem", marginBottom: "1.5rem" }}>
              <h2 className="form-title" style={{ margin: 0 }}>{t("common.detail")} {t("nav.reminders")}</h2>
              <button className="modal-close" onClick={() => setSelectedItemForDetail(null)}>×</button>
            </div>
            <div className="info-grid responsive-grid">
              <div className="info-item" style={{ flexDirection: "column", alignItems: "flex-start", border: "none" }}>
                <span className="info-label">{t("reminders.form.title")}</span>
                <span className="info-value">{selectedItemForDetail.title}</span>
              </div>
              <div className="info-item" style={{ flexDirection: "column", alignItems: "flex-start", border: "none" }}>
                <span className="info-label">{t("reminders.form.dueDate")}</span>
                <span className="info-value">{selectedItemForDetail.due_date}</span>
              </div>
              <div className="info-item" style={{ flexDirection: "column", alignItems: "flex-start", border: "none" }}>
                <span className="info-label">{t("transactions.table.total")}</span>
                <span className="info-value">Rp {numberFormatter.format(selectedItemForDetail.amount_minor ?? 0)}</span>
              </div>
              <div className="info-item" style={{ flexDirection: "column", alignItems: "flex-start", border: "none" }}>
                <span className="info-label">{t("reminders.form.status")}</span>
                <span className="info-value">{t(`reminders.status.${selectedItemForDetail.status}`)}</span>
              </div>
            </div>

            <div style={{ marginTop: "1.5rem" }}>
              <span className="info-label">{t("reminders.form.description")}</span>
              <p style={{ margin: "0.4rem 0 0", fontSize: "0.9rem", color: "var(--text)", whiteSpace: "pre-wrap", wordBreak: "break-word" }}>{selectedItemForDetail.description || "-"}</p>
            </div>

            <div style={{ marginTop: "2rem" }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.75rem" }}>
                <h4 style={{ margin: 0 }}>Riwayat Pembayaran</h4>
                <button 
                  className="btn btn-ghost-inline" 
                  style={{ fontSize: "0.75rem" }}
                  onClick={() => setIsAddingPayment(true)}
                >
                  + Tambah Pembayaran
                </button>
              </div>
              {loadingHistory ? (
                <p className="page-subtitle">Memuat riwayat...</p>
              ) : (
                <PaymentHistory 
                  payments={paymentsHistory} 
                  numberFormatter={numberFormatter} 
                  onEdit={(p) => {
                    setEditingPayment(p);
                    setIsAddingPayment(true);
                  }}
                  onDelete={(pid) => {
                    if (confirm("Hapus riwayat pembayaran ini?")) {
                      removePayment(selectedItemForDetail.id, pid)
                        .then(() => {
                          success("Pembayaran dihapus");
                          return fetchPayments(selectedItemForDetail.id);
                        })
                        .then(setPaymentsHistory)
                        .catch((err: unknown) => showError(err instanceof Error ? err.message : "Gagal menghapus pembayaran"));
                    }
                  }}
                />
              )}
            </div>

            <div className="modal-footer" style={{ padding: "1.5rem 0 0", marginTop: "1.5rem", borderTop: "1px solid var(--border)", display: "flex", justifyContent: "flex-end", gap: "0.75rem" }}>
              <button className="btn btn-secondary" onClick={() => setSelectedItemForDetail(null)}>{t("common.close")}</button>
              <button className="btn btn-primary" onClick={() => setIsAddingPayment(true)}>{t("reminders.markPaid")}</button>
            </div>
          </div>
        </div>
      )}

      <AddPaymentModal 
        isOpen={isAddingPayment}
        isLoading={submittingPayment}
        title={editingPayment ? "Edit Pembayaran" : "Catat Pembayaran"}
        initialData={editingPayment ? {
          paid_at: editingPayment.paid_at,
          amount_minor: editingPayment.amount_minor,
          status: editingPayment.status,
          notes: editingPayment.notes || ""
        } : {
          paid_at: new Date().toISOString().split("T")[0],
          amount_minor: selectedItemForDetail?.amount_minor || 0,
          status: "paid",
          notes: ""
        }}
        onClose={() => {
          setIsAddingPayment(false);
          setEditingPayment(null);
        }}
        onConfirm={async (data) => {
          if (!selectedItemForDetail) return;
          setSubmittingPayment(true);
          try {
            if (editingPayment) {
              await updatePayment(selectedItemForDetail.id, editingPayment.id, {
                paid_at: data.paid_at,
                amount_minor: data.amount_minor,
                status: data.status,
                notes: data.notes,
                image: data.image
              });
              success("Pembayaran diperbarui");
            } else {
              await addPayment(selectedItemForDetail.id, {
                paid_at: data.paid_at,
                amount_minor: data.amount_minor,
                status: data.status,
                notes: data.notes,
                image: data.image
              });
              success(t("common.updateSuccess"));
            }
            setIsAddingPayment(false);
            setEditingPayment(null);
            // Refresh history
            const updated = await fetchPayments(selectedItemForDetail.id);
            setPaymentsHistory(updated);
          } catch (err) {
            showError(err instanceof Error ? err.message : "Gagal menyimpan pembayaran");
          } finally {
            setSubmittingPayment(false);
          }
        }}
      />

      <ToastContainer toasts={toasts} onRemove={removeToast} />
      <InfoModal 
        isOpen={!!descToShow}
        title={descToShow?.title ?? ""}
        description={descToShow?.description ?? ""}
        onClose={() => setDescToShow(null)}
      />
    </section>
  );
}
