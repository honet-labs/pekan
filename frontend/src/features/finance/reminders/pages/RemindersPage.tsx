import { useEffect, useMemo, useState, Fragment } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useReminders } from "../hooks/useReminders";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { PageHeader } from "../../../../core/components/PageHeader";
import { useToast } from "../../../../core/hooks/useToast";
import { Pagination } from "../../../../core/components/Pagination";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { Reminder, ReminderPayment } from "../api/reminders.types";
import { openPaymentProofImage } from "../api/reminders.api";
import { AddPaymentModal } from "../components/AddPaymentModal";
import { PaymentHistory } from "../components/PaymentHistory";
import { InfoModal } from "../../../../core/components/InfoModal";

function getDueDateForInstallment(baseDateStr: string, repeatInterval: string, installmentIndex: number): string {
  if (installmentIndex <= 1 || repeatInterval === "none" || !repeatInterval) {
    return baseDateStr;
  }
  const date = new Date(baseDateStr);
  if (isNaN(date.getTime())) {
    return baseDateStr;
  }
  
  if (repeatInterval === "daily") {
    date.setDate(date.getDate() + (installmentIndex - 1));
  } else if (repeatInterval === "weekly") {
    date.setDate(date.getDate() + 7 * (installmentIndex - 1));
  } else if (repeatInterval === "monthly") {
    date.setMonth(date.getMonth() + (installmentIndex - 1));
  }
  
  return date.toISOString().split("T")[0];
}

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
    removePayment,
    update
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
  const [expandedReminderId, setExpandedReminderId] = useState<string | null>(null);
  const [reminderPaymentsMap, setReminderPaymentsMap] = useState<Record<string, ReminderPayment[]>>({});
  const [loadingPaymentsMap, setLoadingPaymentsMap] = useState<Record<string, boolean>>({});

  const toggleExpand = async (reminderId: string) => {
    if (expandedReminderId === reminderId) {
      setExpandedReminderId(null);
      return;
    }
    setExpandedReminderId(reminderId);
    setLoadingPaymentsMap((prev) => ({ ...prev, [reminderId]: true }));
    try {
      const payments = await fetchPayments(reminderId);
      setReminderPaymentsMap((prev) => ({ ...prev, [reminderId]: payments }));
    } catch (err) {
      console.error(err);
    } finally {
      setLoadingPaymentsMap((prev) => ({ ...prev, [reminderId]: false }));
    }
  };

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
                {displayedItems.map((item) => {
                  const isExpanded = expandedReminderId === item.id;
                  const payments = reminderPaymentsMap[item.id] || [];
                  const isLoadingPayments = loadingPaymentsMap[item.id];
                  
                  return (
                    <Fragment key={item.id}>
                      <tr style={{ background: isExpanded ? "rgba(var(--primary-color-rgb), 0.02)" : "transparent" }}>
                        <td data-label={t("reminders.form.title")}>
                          <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                            <button
                              type="button"
                              onClick={() => toggleExpand(item.id)}
                              style={{
                                background: "none",
                                border: "none",
                                padding: "4px",
                                cursor: "pointer",
                                display: "inline-flex",
                                alignItems: "center",
                                color: "var(--primary)",
                                transform: isExpanded ? "rotate(90deg)" : "rotate(0deg)",
                                transition: "transform 0.2s"
                              }}
                            >
                              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                                <polyline points="9 18 15 12 9 6" />
                              </svg>
                            </button>
                            <span style={{ fontWeight: 600 }}>{item.title}</span>
                          </div>
                        </td>
                        <td data-label={t("reminders.form.dueDate")}>
                          {item.due_date}
                          {item.total_tenor && item.total_tenor > 1 ? (
                            <div style={{ fontSize: "0.75rem", color: "var(--muted)", marginTop: "2px" }}>
                              Tenor: {item.current_tenor || 0} / {item.total_tenor}
                            </div>
                          ) : null}
                        </td>
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
                        <td data-label={t("reminders.form.status")}>
                          <span className={`pill ${item.status === "paid" ? "income" : "transfer"}`} style={{ fontSize: "0.75rem", padding: "4px 10px", borderRadius: "12px" }}>
                            {t(`reminders.status.${item.status}`)}
                          </span>
                        </td>
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
                      {isExpanded && (
                        <tr>
                          <td colSpan={6} style={{ background: "var(--surface-soft)", padding: "1.5rem", borderBottom: "1px solid var(--border)" }}>
                            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
                              <h4 style={{ margin: 0, color: "var(--text)", fontSize: "1.05rem", display: "flex", alignItems: "center", gap: "8px", fontWeight: 700 }}>
                                <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{ color: "var(--primary)" }}>
                                  <rect x="3" y="4" width="18" height="18" rx="2" ry="2" /><line x1="16" y1="2" x2="16" y2="6" /><line x1="8" y1="2" x2="8" y2="6" /><line x1="3" y1="10" x2="21" y2="10" />
                                </svg>
                                Status Angsuran & Tenor Pembayaran
                              </h4>
                              {(!item.total_tenor || item.total_tenor <= 1) && (
                                <button
                                  className="btn btn-primary"
                                  style={{ fontSize: "0.8rem", padding: "6px 12px" }}
                                  onClick={() => {
                                    setSelectedItemForDetail(item);
                                    setEditingPayment(null);
                                    setIsAddingPayment(true);
                                  }}
                                >
                                  + Catat Pembayaran
                                </button>
                              )}
                            </div>
                            
                            {isLoadingPayments ? (
                              <p style={{ margin: 0, fontStyle: "italic", color: "var(--muted)", fontSize: "0.875rem" }}>Memuat data pembayaran...</p>
                            ) : (
                              <div>
                                {item.total_tenor && item.total_tenor > 1 ? (
                                  <div style={{ display: "grid", gap: "10px" }}>
                                    {Array.from({ length: item.total_tenor }).map((_, index) => {
                                      const slotIndex = index + 1;
                                      const isPaid = slotIndex <= payments.length;
                                      const payment = isPaid ? payments[slotIndex - 1] : null;
                                      const installmentDueDate = getDueDateForInstallment(item.due_date, item.repeat_interval, slotIndex);
                                      
                                      return (
                                        <div
                                          key={slotIndex}
                                          style={{
                                            display: "flex",
                                            justifyContent: "space-between",
                                            alignItems: "center",
                                            padding: "0.8rem 1.2rem",
                                            background: "var(--surface)",
                                            border: "1px solid var(--border)",
                                            borderRadius: "8px",
                                            boxShadow: "0 2px 4px rgba(0,0,0,0.02)"
                                          }}
                                        >
                                          <div style={{ display: "flex", flexDirection: "column", gap: "4px" }}>
                                            <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                              <span style={{ fontWeight: 700, fontSize: "0.9rem", color: "var(--text)" }}>
                                                Cicilan Ke-{slotIndex} dari {item.total_tenor}
                                              </span>
                                              <span className={`pill ${isPaid ? "income" : "transfer"}`} style={{ fontSize: "0.7rem", padding: "2px 8px" }}>
                                                {isPaid ? "Lunas" : "Belum Bayar"}
                                              </span>
                                            </div>
                                            <span style={{ fontSize: "0.8rem", color: "var(--muted)" }}>
                                              Jatuh Tempo: {installmentDueDate}
                                            </span>
                                          </div>
                                          
                                          <div>
                                            {isPaid && payment ? (
                                              <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
                                                <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end", gap: "4px" }}>
                                                  <span style={{ fontWeight: 600, fontSize: "0.9rem", color: "var(--primary)" }}>
                                                    Rp {numberFormatter.format(payment.amount_minor)}
                                                  </span>
                                                  <span style={{ fontSize: "0.75rem", color: "var(--muted)" }}>
                                                    Bayar: {payment.paid_at}
                                                  </span>
                                                  {payment.notes && (
                                                    <span style={{ fontSize: "0.75rem", color: "var(--text)", fontStyle: "italic" }}>
                                                      "{payment.notes}"
                                                    </span>
                                                  )}
                                                </div>
                                                
                                                <div style={{ display: "flex", gap: "8px" }}>
                                                  {payment.proof_image_url && (
                                                    <button
                                                      type="button"
                                                      className="btn btn-ghost-inline"
                                                      style={{ fontSize: "0.75rem", padding: "4px 8px", display: "inline-flex", alignItems: "center", gap: "4px" }}
                                                      onClick={() => openPaymentProofImage(item.id, payment.id).catch(console.error)}
                                                    >
                                                      <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                                                        <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/><circle cx="12" cy="13" r="4"/>
                                                      </svg>
                                                      Bukti
                                                    </button>
                                                  )}
                                                  <button
                                                    type="button"
                                                    className="btn btn-ghost-inline"
                                                    style={{ fontSize: "0.75rem", padding: "4px 8px" }}
                                                    onClick={() => {
                                                      setSelectedItemForDetail(item);
                                                      setEditingPayment(payment);
                                                      setIsAddingPayment(true);
                                                    }}
                                                  >
                                                    Edit
                                                  </button>
                                                  <button
                                                    type="button"
                                                    className="btn btn-ghost-inline danger"
                                                    style={{ fontSize: "0.75rem", padding: "4px 8px" }}
                                                    onClick={() => {
                                                      if (confirm("Hapus riwayat pembayaran ini?")) {
                                                        removePayment(item.id, payment.id)
                                                          .then(async () => {
                                                            success("Pembayaran dihapus");
                                                            const updated = await fetchPayments(item.id);
                                                            setReminderPaymentsMap(prev => ({ ...prev, [item.id]: updated }));
                                                            
                                                            // Rollback status/tenor
                                                            if (item.total_tenor && item.total_tenor > 1) {
                                                              const totalTenor = item.total_tenor;
                                                              const paidCount = updated.length;
                                                              const isFullyPaid = paidCount >= totalTenor;
                                                              await update(item.id, {
                                                                title: item.title,
                                                                description: item.description,
                                                                amount_minor: item.amount_minor,
                                                                currency: item.currency,
                                                                due_date: item.due_date,
                                                                repeat_interval: item.repeat_interval,
                                                                status: isFullyPaid ? "paid" : "pending",
                                                                total_tenor: totalTenor,
                                                                current_tenor: Math.min(paidCount, totalTenor)
                                                              });
                                                            } else {
                                                              await markStatus(item.id, "pending");
                                                            }
                                                          })
                                                          .catch((err: unknown) => showError(err instanceof Error ? err.message : "Gagal menghapus pembayaran"));
                                                      }
                                                    }}
                                                  >
                                                    Hapus
                                                  </button>
                                                </div>
                                              </div>
                                            ) : (
                                              <button
                                                className="btn btn-primary"
                                                style={{ fontSize: "0.8rem", padding: "6px 12px" }}
                                                onClick={() => {
                                                  setSelectedItemForDetail(item);
                                                  setEditingPayment(null);
                                                  setIsAddingPayment(true);
                                                }}
                                              >
                                                + Catat Pembayaran
                                              </button>
                                            )}
                                          </div>
                                        </div>
                                      );
                                    })}
                                  </div>
                                ) : (
                                  <div>
                                    <PaymentHistory 
                                      payments={payments} 
                                      numberFormatter={numberFormatter} 
                                      onEdit={(p) => {
                                        setSelectedItemForDetail(item);
                                        setEditingPayment(p);
                                        setIsAddingPayment(true);
                                      }}
                                      onDelete={(pid) => {
                                        if (confirm("Hapus riwayat pembayaran ini?")) {
                                          removePayment(item.id, pid)
                                            .then(async () => {
                                              success("Pembayaran dihapus");
                                              const updated = await fetchPayments(item.id);
                                              setReminderPaymentsMap(prev => ({ ...prev, [item.id]: updated }));
                                              
                                              // Rollback status/tenor
                                              if (item.total_tenor && item.total_tenor > 1) {
                                                const totalTenor = item.total_tenor;
                                                const paidCount = updated.length;
                                                const isFullyPaid = paidCount >= totalTenor;
                                                await update(item.id, {
                                                  title: item.title,
                                                  description: item.description,
                                                  amount_minor: item.amount_minor,
                                                  currency: item.currency,
                                                  due_date: item.due_date,
                                                  repeat_interval: item.repeat_interval,
                                                  status: isFullyPaid ? "paid" : "pending",
                                                  total_tenor: totalTenor,
                                                  current_tenor: Math.min(paidCount, totalTenor)
                                                });
                                              } else {
                                                await markStatus(item.id, "pending");
                                              }
                                            })
                                            .catch((err: unknown) => showError(err instanceof Error ? err.message : "Gagal menghapus pembayaran"));
                                        }
                                      }}
                                    />
                                  </div>
                                )}
                              </div>
                            )}
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
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
                        .then(async () => {
                          success("Pembayaran dihapus");
                          const updated = await fetchPayments(selectedItemForDetail.id);
                          setPaymentsHistory(updated);
                          setReminderPaymentsMap(prev => ({ ...prev, [selectedItemForDetail.id]: updated }));
                          
                          // Rollback status/tenor
                          if (selectedItemForDetail.total_tenor && selectedItemForDetail.total_tenor > 1) {
                            const totalTenor = selectedItemForDetail.total_tenor;
                            const paidCount = updated.length;
                            const isFullyPaid = paidCount >= totalTenor;
                            await update(selectedItemForDetail.id, {
                              title: selectedItemForDetail.title,
                              description: selectedItemForDetail.description,
                              amount_minor: selectedItemForDetail.amount_minor,
                              currency: selectedItemForDetail.currency,
                              due_date: selectedItemForDetail.due_date,
                              repeat_interval: selectedItemForDetail.repeat_interval,
                              status: isFullyPaid ? "paid" : "pending",
                              total_tenor: totalTenor,
                              current_tenor: Math.min(paidCount, totalTenor)
                            });
                          } else {
                            await markStatus(selectedItemForDetail.id, "pending");
                          }
                        })
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
            
            // Refresh history and map
            const updated = await fetchPayments(selectedItemForDetail.id);
            setPaymentsHistory(updated);
            setReminderPaymentsMap(prev => ({ ...prev, [selectedItemForDetail.id]: updated }));

            // Automatically transition/advance tenor and status
            if (selectedItemForDetail.total_tenor && selectedItemForDetail.total_tenor > 1) {
              const totalTenor = selectedItemForDetail.total_tenor;
              const paidCount = updated.length;
              const isFullyPaid = paidCount >= totalTenor;
              
              await update(selectedItemForDetail.id, {
                title: selectedItemForDetail.title,
                description: selectedItemForDetail.description,
                amount_minor: selectedItemForDetail.amount_minor,
                currency: selectedItemForDetail.currency,
                due_date: selectedItemForDetail.due_date,
                repeat_interval: selectedItemForDetail.repeat_interval,
                status: isFullyPaid ? "paid" : "pending",
                total_tenor: totalTenor,
                current_tenor: Math.min(paidCount, totalTenor)
              });
            } else {
              if (data.status === "paid") {
                await markStatus(selectedItemForDetail.id, "paid");
              }
            }
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
