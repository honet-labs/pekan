import { useState } from "react";
import { downloadReport, getReportBlob } from "../api/reports.api";
import { Report } from "../api/reports.types";
import { useReports } from "../hooks/useReports";
import { useI18n } from "../../../../core/i18n/i18n";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { PageHeader } from "../../../../core/components/PageHeader";
import { useTransactionMasterData } from "../../transactions/hooks/useTransactionMasterData";
import { Pagination } from "../../../../core/components/Pagination";
import Skeleton from "../../../../core/components/Skeleton";

const initialForm = {
  report_type: "transactions",
  date_from: "",
  date_to: "",
  category_id: "",
  type: "",
  status: "",
  format: "csv"
};

export function ReportsPage(): JSX.Element {
  const { items, loading, error, page, pageSize, total, setPage, create, remove } = useReports();
  const { categories } = useTransactionMasterData();
  const { toasts, success, error: errorToast, remove: removeToast } = useToast();
  const [form, setForm] = useState(initialForm);
  const [downloading, setDownloading] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [itemToDelete, setItemToDelete] = useState<Report | null>(null);
  const { t } = useI18n();

  // Preview States
  const [previewingReport, setPreviewingReport] = useState<Report | null>(null);
  const [previewData, setPreviewData] = useState<{ headers: string[]; rows: string[][] } | null>(null);
  const [previewPdfUrl, setPreviewPdfUrl] = useState<string | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    try {
      await create({
        date_from: form.date_from || undefined,
        date_to: form.date_to || undefined,
        report_type: form.report_type as "transactions" | "savings" | "budgets" | "reminders",
        category_id: form.category_id || undefined,
        type: (form.type || undefined) as "income" | "expense" | "transfer" | "savings" | undefined,
        status: form.status || undefined,
        format: form.format
      });
      setForm(initialForm);
      success(t("common.saveSuccess"));
    } catch (err) {
      errorToast(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    }
  };

  const handleDownload = async (report: Report) => {
    setDownloading(report.id);
    setDownloadError(null);
    try {
      await downloadReport(report);
    } catch (err) {
      if (err instanceof Error) {
        setDownloadError(err.message === "Failed to download report" ? t("errors.downloadReportFailed") : err.message);
      } else {
        setDownloadError(t("errors.downloadReportFailed"));
      }
    } finally {
      setDownloading(null);
    }
  };

  const handlePreview = async (report: Report) => {
    setPreviewingReport(report);
    setPreviewLoading(true);
    setPreviewError(null);
    setPreviewData(null);
    setPreviewPdfUrl(null);

    try {
      const { blob, url } = await getReportBlob(report);

      if (report.format === "pdf") {
        setPreviewPdfUrl(url);
      } else if (report.format === "csv") {
        const text = await blob.text();
        const lines = text.split("\n");
        const parsedRows = lines
          .map(line => {
            const result: string[] = [];
            let current = "";
            let inQuotes = false;
            for (let i = 0; i < line.length; i++) {
              const char = line[i];
              if (char === '"') {
                inQuotes = !inQuotes;
              } else if (char === ',' && !inQuotes) {
                result.push(current.trim().replace(/^"|"$/g, ""));
                current = "";
              } else {
                current += char;
              }
            }
            result.push(current.trim().replace(/^"|"$/g, ""));
            return result;
          })
          .filter(row => row.length > 0 && row.some(cell => cell !== ""));

        if (parsedRows.length > 0) {
          const headers = parsedRows[0];
          const rows = parsedRows.slice(1);
          setPreviewData({ headers, rows });
        } else {
          setPreviewError("File CSV kosong.");
        }
      } else {
        setPreviewError("Format file tidak didukung untuk pratinjau.");
      }
    } catch (err) {
      setPreviewError(err instanceof Error ? err.message : "Gagal memuat pratinjau");
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleClosePreview = () => {
    if (previewPdfUrl) {
      URL.revokeObjectURL(previewPdfUrl);
    }
    setPreviewingReport(null);
    setPreviewData(null);
    setPreviewPdfUrl(null);
    setPreviewError(null);
  };

  const handleDeleteClick = (report: Report) => {
    setItemToDelete(report);
    setDeleteConfirmOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!itemToDelete) return;
    
    setDeleting(itemToDelete.id);
    setDownloadError(null);
    try {
      await remove(itemToDelete.id);
      setDeleteConfirmOpen(false);
      setItemToDelete(null);
      success(t("common.deleteSuccess") || "Data berhasil dihapus");
    } catch (err) {
      let errMsg: string;
      if (err instanceof Error) {
        errMsg = err.message;
      } else {
        errMsg = t("errors.saveDataFailed");
      }
      setDownloadError(errMsg);
      errorToast(errMsg);
    } finally {
      setDeleting(null);
    }
  };

  const handleCancelDelete = () => {
    setDeleteConfirmOpen(false);
    setItemToDelete(null);
  };

  return (
    <section className="page-section">
      <PageHeader 
        title={t("reports.title")} 
        description={t("reports.subtitle")} 
      />

      {error ? <div className="alert error">{error}</div> : null}
      {downloadError ? <div className="alert error">{downloadError}</div> : null}

      <div className="card-grid two-col tight stack-on-mobile">
        <form className="card surface form-grid" onSubmit={handleSubmit}>
          <h3 className="form-title">{t("reports.form.title")}</h3>
          <label className="form-field">
            {t("reports.form.reportType")}
            <select
              className="input-control"
              value={form.report_type}
              onChange={(event) => setForm({ ...form, report_type: event.target.value })}
            >
              <option value="transactions">{t("reports.type.transactions")}</option>
              <option value="savings">{t("reports.type.savings")}</option>
              <option value="budgets">{t("reports.type.budgets")}</option>
              <option value="reminders">{t("reports.type.reminders")}</option>
            </select>
          </label>
          <label className="form-field">
            {t("reports.form.from")}
            <input
              className="input-control"
              type="date"
              value={form.date_from}
              onChange={(event) => setForm({ ...form, date_from: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("reports.form.to")}
            <input
              className="input-control"
              type="date"
              value={form.date_to}
              onChange={(event) => setForm({ ...form, date_to: event.target.value })}
            />
          </label>
          {form.report_type === "transactions" ? (
            <>
              <label className="form-field">
                {t("reports.form.typeFilter")}
                <select
                  className="input-control"
                  value={form.type}
                  onChange={(event) => setForm({ ...form, type: event.target.value })}
                >
                  <option value="">{t("reports.form.allTypes")}</option>
                  <option value="expense">{t("transactions.type.expense")}</option>
                  <option value="income">{t("transactions.type.income")}</option>
                  <option value="transfer">{t("transactions.type.transfer")}</option>
                  <option value="savings">{t("transactions.type.savings")}</option>
                </select>
              </label>
              <label className="form-field">
                {t("reports.form.categoryFilter")}
                <select
                  className="input-control"
                  value={form.category_id}
                  onChange={(event) => setForm({ ...form, category_id: event.target.value })}
                >
                  <option value="">{t("reports.form.allCategories")}</option>
                  {categories.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                </select>
              </label>
            </>
          ) : (
            <label className="form-field">
              {t("reports.form.statusFilter")}
              <select
                className="input-control"
                value={form.status}
                onChange={(event) => setForm({ ...form, status: event.target.value })}
              >
                <option value="">{t("reports.form.allStatus")}</option>
                {form.report_type === "savings" ? (
                  <>
                    <option value="active">{t("savings.status.active")}</option>
                    <option value="completed">{t("savings.status.completed")}</option>
                    <option value="cancelled">{t("savings.status.cancelled")}</option>
                  </>
                ) : null}
                {form.report_type === "budgets" ? (
                  <>
                    <option value="active">{t("budgets.status.active")}</option>
                    <option value="paused">{t("budgets.status.paused")}</option>
                    <option value="ended">{t("budgets.status.ended")}</option>
                  </>
                ) : null}
                {form.report_type === "reminders" ? (
                  <>
                    <option value="pending">{t("reminders.status.pending")}</option>
                    <option value="paid">{t("reminders.status.paid")}</option>
                    <option value="cancelled">{t("reminders.status.cancelled")}</option>
                  </>
                ) : null}
              </select>
            </label>
          )}
          <label className="form-field">
            {t("reports.form.format")}
            <select
              className="input-control"
              value={form.format}
              onChange={(event) => setForm({ ...form, format: event.target.value })}
            >
              <option value="csv">CSV</option>
              <option value="pdf">PDF</option>
            </select>
          </label>
          <button className="btn btn-primary" type="submit">
            {t("reports.form.generate")}
          </button>
        </form>

        <div className="card surface">
          <h3 className="form-title">{t("reports.history")}</h3>
          <div className="data-table-wrap table-mobile-stack">
            <table className="data-table">
              <thead>
                <tr>
                  <th style={{ width: "180px" }}>{t("reports.table.id")}</th>
                  <th>{t("reports.table.type")}</th>
                  <th>{t("reports.table.format")}</th>
                  <th>{t("reports.table.status")}</th>
                  <th>{t("reports.table.created")}</th>
                  <th>{t("reports.table.action")}</th>
                </tr>
              </thead>
              <tbody>
                {loading && items.length === 0 ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <tr key={i}>
                      <td><Skeleton width="100%" height="1rem" /></td>
                      <td><Skeleton width="100%" height="1rem" /></td>
                      <td><Skeleton width="100%" height="1rem" /></td>
                      <td><Skeleton width="100%" height="1rem" /></td>
                      <td><Skeleton width="100%" height="1rem" /></td>
                      <td><Skeleton width="100%" height="1rem" /></td>
                    </tr>
                  ))
                ) : (
                  items.map((item) => (
                    <tr key={item.id}>
                      <td data-label={t("reports.table.id")} style={{ maxWidth: "180px" }}><span className="report-id-text" style={{ wordBreak: "break-all", fontSize: "0.75rem", color: "var(--muted)", opacity: 0.8 }}>{item.id}</span></td>
                      <td data-label={t("reports.table.type")}>{t(`reports.type.${item.report_type}`)}</td>
                      <td data-label={t("reports.table.format")}><span className="type-pill" style={{ textTransform: "uppercase", padding: "2px 8px", borderRadius: "4px", fontSize: "0.8rem", background: "var(--surface-soft)", color: "var(--text)", border: "1px solid var(--border)" }}>{item.format}</span></td>
                      <td data-label={t("reports.table.status")}>{item.status}</td>
                      <td data-label={t("reports.table.created")}>{new Date(item.created_at).toLocaleString()}</td>
                      <td data-label={t("reports.table.action")}>
                        <div className="inline-buttons" style={{ display: "flex", gap: "0.25rem" }}>
                          <button
                            className="btn btn-ghost-inline"
                            type="button"
                            onClick={() => handlePreview(item)}
                            disabled={item.status !== "ready" || downloading === item.id}
                            title={t("reports.preview")}
                          >
                            {t("reports.preview")}
                          </button>
                          <button
                            className="btn btn-ghost-inline"
                            type="button"
                            onClick={() => handleDownload(item)}
                            disabled={item.status !== "ready" || downloading === item.id}
                            title={t("reports.download")}
                          >
                            {downloading === item.id ? t("reports.downloading") : t("reports.download")}
                          </button>
                          <button
                            className="btn btn-ghost-inline danger"
                            type="button"
                            onClick={() => handleDeleteClick(item)}
                            disabled={deleting === item.id}
                            title={t("common.delete")}
                          >
                            {deleting === item.id ? t("common.loading") : t("common.delete")}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
                {!items.length && !loading ? (
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
            totalItems={total}
            onPageChange={setPage}
            disabled={loading}
          />
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      <DeleteConfirmModal
        isOpen={deleteConfirmOpen}
        title={t("reports.delete.title")}
        message={`${t("reports.delete.confirm")} ${itemToDelete?.report_type}?`}
        isLoading={!!deleting}
        onConfirm={handleConfirmDelete}
        onCancel={handleCancelDelete}
      />

      {/* Report Preview Modal */}
      {previewingReport && (
        <div className="modal-overlay" style={{ display: "flex", alignItems: "center", justifyContent: "center", zIndex: 1000 }} onClick={handleClosePreview}>
          <div className="modal-content" style={{ maxWidth: "1000px", width: "95%", maxHeight: "90vh", display: "flex", flexDirection: "column" }} onClick={(e) => e.stopPropagation()}>
            <header className="modal-header">
              <h2>
                {t("reports.preview.title")} - {t(`reports.type.${previewingReport.report_type}`)} ({previewingReport.format.toUpperCase()})
              </h2>
              <button type="button" className="btn-icon" onClick={handleClosePreview} title={t("common.close")}>
                ✕
              </button>
            </header>

            <div className="modal-body" style={{ overflowY: "auto", flex: 1, padding: "1.5rem" }}>
              {previewLoading ? (
                <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", minHeight: "300px" }}>
                  <style>{`
                    @keyframes spin {
                      0% { transform: rotate(0deg); }
                      100% { transform: rotate(360deg); }
                    }
                  `}</style>
                  <div className="spinner" style={{ border: "4px solid var(--surface-soft)", width: "40px", height: "40px", borderRadius: "50%", borderLeftColor: "var(--primary)", animation: "spin 1s linear infinite" }}></div>
                  <p style={{ marginTop: "1rem", color: "var(--muted)", fontWeight: "500" }}>{t("common.loading")}</p>
                </div>
              ) : previewError ? (
                <div className="alert error">{previewError}</div>
              ) : previewingReport.format === "csv" && previewData ? (
                <div className="data-table-wrap" style={{ overflowX: "auto", border: "1px solid var(--border)", borderRadius: "6px" }}>
                  <table className="data-table" style={{ fontSize: "0.85rem", width: "100%" }}>
                    <thead>
                      <tr>
                        {previewData.headers.map((h, i) => (
                          <th key={i} style={{ background: "var(--surface-soft)", position: "sticky", top: 0 }}>{h.replace(/_/g, " ").toUpperCase()}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {previewData.rows.map((row, rowIndex) => (
                        <tr key={rowIndex}>
                          {row.map((cell, cellIndex) => {
                            const header = previewData.headers[cellIndex];
                            const isAmount = header.includes("amount_minor") || header.includes("limit_minor");
                            let cellVal = cell;
                            if (isAmount && !isNaN(Number(cell))) {
                              cellVal = (Number(cell) / 100).toLocaleString(undefined, { minimumFractionDigits: 2 });
                            }
                            return <td key={cellIndex}>{cellVal}</td>;
                          })}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : previewingReport.format === "pdf" && previewPdfUrl ? (
                <div style={{ width: "100%", height: "550px", display: "flex", flexDirection: "column" }}>
                  <iframe
                    src={previewPdfUrl}
                    title="PDF Preview"
                    width="100%"
                    height="100%"
                    style={{ border: "1px solid var(--border)", borderRadius: "6px", flex: 1 }}
                  />
                  <div style={{ display: "flex", justifyContent: "center", marginTop: "1rem" }}>
                    <a
                      href={previewPdfUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="btn btn-ghost"
                      style={{ display: "inline-flex", alignItems: "center", gap: "0.5rem", padding: "0.5rem 1rem", border: "1px solid var(--border)", borderRadius: "6px", textDecoration: "none" }}
                    >
                      {t("reports.preview.openNewTab")}
                    </a>
                  </div>
                </div>
              ) : (
                <div className="alert error">Format tidak didukung untuk pratinjau.</div>
              )}
            </div>

            <footer className="modal-footer" style={{ borderTop: "1px solid var(--border)", display: "flex", justifyContent: "flex-end", gap: "0.75rem", padding: "1rem" }}>
              <button type="button" className="btn btn-secondary" onClick={handleClosePreview}>
                {t("common.close")}
              </button>
              <button type="button" className="btn btn-primary" onClick={() => { handleDownload(previewingReport); handleClosePreview(); }}>
                {t("reports.download")}
              </button>
            </footer>
          </div>
        </div>
      )}

      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}
