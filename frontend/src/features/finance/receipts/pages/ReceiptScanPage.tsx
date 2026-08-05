import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useI18n } from "../../../../core/i18n/i18n";
import { PageHeader } from "../../../../core/components/PageHeader";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { getReceiptConfigStatus, listReceiptProviders, listReceiptScanHistory, scanReceipt, deleteReceiptScanHistoryItem, clearReceiptScanHistory, fetchReceiptScanImageBlob } from "../api/receipts.api";
import { ReceiptProviderConfig, ReceiptScanDraft, ReceiptScanHistoryItem, ReceiptScanResult } from "../api/receipts.types";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";

export function ReceiptScanPage(): JSX.Element {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { tenantCode } = useParams();
  const settingsPath = tenantCode ? `/app/${tenantCode}/finance/settings/receipt-scan` : "/app/default/finance/settings/receipt-scan";
  const transactionCreatePath = tenantCode ? `/app/${tenantCode}/finance/transactions/new` : "/app/default/finance/transactions/new";
  const { toasts, info, error: showError, success, remove } = useToast();
  const [providers, setProviders] = useState<ReceiptProviderConfig[]>([]);
  const [history, setHistory] = useState<ReceiptScanHistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [configured, setConfigured] = useState(false);
  const [providerCode, setProviderCode] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [scanning, setScanning] = useState(false);
  const [result, setResult] = useState<ReceiptScanResult | null>(null);
  const [clearingHistory, setClearingHistory] = useState(false);
  const [deletingScanID, setDeletingScanID] = useState<string | null>(null);
  const [viewingImageURL, setViewingImageURL] = useState<string | null>(null);
  const [loadingImage, setLoadingImage] = useState(false);
  const [isClearModalOpen, setIsClearModalOpen] = useState(false);
  const [itemToDelete, setItemToDelete] = useState<ReceiptScanHistoryItem | null>(null);
  const [filePreview, setFilePreview] = useState<string | null>(null);

  async function load(): Promise<void> {
    setLoading(true);
    try {
      const [status, providerItems, historyItems] = await Promise.all([
        getReceiptConfigStatus(),
        listReceiptProviders(),
        listReceiptScanHistory(10)
      ]);
      setConfigured(status.has_configured_provider);
      if (!status.has_configured_provider) {
        info(t("receipt.scan.needConfig"));
      }
      const enabledProviders = providerItems.filter((item) => item.is_enabled && item.has_api_key);
      setProviders(enabledProviders);
      setHistory(historyItems);
      
      // Default to gemini if no tenant-specific provider is configured
      if (enabledProviders.length > 0) {
        setProviderCode((prev) => prev || enabledProviders[0].provider_code);
      } else {
        setProviderCode("gemini");
      }
    } catch (err) {
      console.error("Failed to load receipt scan configs, providers or history:", err);
      showError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, [t]);

  async function handleScan(): Promise<void> {
    if (!providerCode || !file) {
      showError(t("receipt.scan.fileRequired"));
      return;
    }
    setScanning(true);
    setResult(null);
    try {
      const out = await scanReceipt(providerCode, file);
      setResult(out);
      success(t("receipt.scan.success"));
      const historyItems = await listReceiptScanHistory(10);
      setHistory(historyItems);
    } catch (err) {
      console.error("Failed to scan receipt image:", err);
      showError(err instanceof Error ? err.message : t("receipt.scan.failed"));
    } finally {
      setScanning(false);
    }
  }

  function handleUseDraft(draft: ReceiptScanDraft): void {
    const storeAndNavigate = (fileDataURL: string | null): void => {
      const draftData: Record<string, unknown> = {
        amount_minor: draft.amount_minor,
        currency: draft.currency,
        transaction_date: draft.transaction_date,
        description: draft.description,
        type: draft.type,
        category_name: draft.category_name,
        merchant_name: draft.merchant_name,
        receipt_number: draft.receipt_number,
        payment_method: draft.payment_method,
        subtotal_minor: draft.subtotal_minor,
        tax_minor: draft.tax_minor,
        service_charge_minor: draft.service_charge_minor,
        receipt_discount_minor: draft.receipt_discount_minor,
        items: Array.isArray(draft.items) ? draft.items : [],
        receipt_scan_id: result?.scan?.id
      };
      
      try {
        window.sessionStorage.setItem("pekan_receipt_draft", JSON.stringify(draftData));
      } catch (e) {
        console.error("Failed to store receipt draft in sessionStorage", e);
      }

      if (fileDataURL && file) {
        try {
          window.sessionStorage.setItem(
            "pekan_receipt_attach_file",
            JSON.stringify({
              name: file.name,
              type: file.type,
              dataURL: fileDataURL
            })
          );
        } catch (e) {
          console.warn("Failed to store receipt attachment in sessionStorage (exceeded quota)", e);
        }
      }
      navigate(transactionCreatePath);
    };

    if (file) {
      const reader = new FileReader();
      reader.onload = (): void => {
        storeAndNavigate(reader.result as string);
      };
      reader.onerror = (): void => {
        storeAndNavigate(null);
      };
      reader.readAsDataURL(file);
    } else {
      storeAndNavigate(null);
    }
  }

  async function handleClearHistory(): Promise<void> {
    setIsClearModalOpen(false);
    setClearingHistory(true);
    try {
      await clearReceiptScanHistory();
      setHistory([]);
      success(t("common.deleteSuccess"));
    } catch (err) {
      console.error("Failed to clear receipt scan history:", err);
      showError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setClearingHistory(false);
    }
  }

  async function handleDeleteScan(scanID: string): Promise<void> {
    setItemToDelete(null);
    setDeletingScanID(scanID);
    try {
      await deleteReceiptScanHistoryItem(scanID);
      setHistory((prev) => prev.filter((h) => h.id !== scanID));
      success(t("common.deleteSuccess"));
    } catch (err) {
      console.error("Failed to delete receipt scan item:", err);
      showError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setDeletingScanID(null);
    }
  }

  async function handleViewImage(scanID: string): Promise<void> {
    setLoadingImage(true);
    try {
      const url = await fetchReceiptScanImageBlob(scanID);
      setViewingImageURL(url);
    } catch (err) {
      console.error("Failed to view receipt scan image:", err);
      showError(err instanceof Error ? err.message : "Gagal memuat gambar");
    } finally {
      setLoadingImage(false);
    }
  }

  function closeImageModal(): void {
    if (viewingImageURL) {
      URL.revokeObjectURL(viewingImageURL);
    }
    setViewingImageURL(null);
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("receipt.scan.title")} 
        description={t("receipt.scan.subtitle")} 
      />



      <div className="card-grid two-col stack-on-mobile">
        <div className="card surface shadow-soft" style={{ padding: "1.5rem" }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "1.25rem" }}>
            <h3 className="form-title" style={{ margin: 0 }}>{t("receipt.scan.formTitle")}</h3>
            {configured ? (
              <div className="badge-status running-flat" style={{ fontSize: "0.7rem" }}>PEKAN AI ASSISTANT READY</div>
            ) : (
              <div className="badge-status draft-flat" style={{ fontSize: "0.7rem" }}>AI Not Configured</div>
            )}
          </div>
          
          <div className="upload-zone" style={{ border: "2px dashed var(--border)", borderRadius: "12px", padding: "2rem", textAlign: "center", marginBottom: "1.5rem", background: "var(--surface-sunken)", transition: "border-color 0.2s" }}>
             <input type="file" id="receipt-upload" hidden accept="image/jpeg,image/png,image/webp" onChange={(event) => {
                const f = event.target.files?.[0] ?? null;
                setFile(f);
                if (f) {
                  const url = URL.createObjectURL(f);
                  setFilePreview(url);
                } else {
                  setFilePreview(null);
                }
              }} />
             
             {!filePreview ? (
               <label htmlFor="receipt-upload" style={{ cursor: "pointer", display: "block" }}>
                 <div style={{ background: "var(--primary-light)", width: "48px", height: "48px", borderRadius: "50%", display: "flex", alignItems: "center", justifyContent: "center", margin: "0 auto 1rem", color: "var(--primary)" }}>
                   <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                 </div>
                 <p style={{ fontWeight: 600, fontSize: "0.95rem", marginBottom: "0.25rem" }}>{t("receipt.scan.file")}</p>
                 <p className="text-muted text-xs">JPG, PNG, WEBP (Max 5MB)</p>
               </label>
             ) : (
               <div className="receipt-preview-small shadow-soft" style={{ position: "relative", maxWidth: "200px", margin: "0 auto" }}>
                  <img src={filePreview} alt="Receipt Preview" style={{ width: "100%", borderRadius: "8px", border: "1px solid var(--border)" }} />
                  <button type="button" onClick={() => {setFile(null); setFilePreview(null);}} style={{ position: "absolute", top: "-10px", right: "-10px", background: "var(--danger)", color: "white", border: "none", borderRadius: "50%", width: "24px", height: "24px", cursor: "pointer", display: "flex", alignItems: "center", justifyContent: "center", boxShadow: "0 2px 8px rgba(0,0,0,0.2)" }}>✕</button>
               </div>
             )}
          </div>

          <button className="btn btn-primary btn-block" type="button" onClick={() => handleScan().catch(() => undefined)} disabled={scanning || !file} style={{ padding: "0.85rem", height: "auto" }}>
            {scanning ? (
              <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "10px" }}>
                <span className="spinner-sm" /> {t("receipt.scan.scanning")}...
              </div>
            ) : (
              <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "8px" }}>
                <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2.5"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                {t("receipt.scan.scanNow")}
              </div>
            )}
          </button>
        </div>
      </div>

      {result && (
        <div className="card surface shadow-soft" style={{ marginTop: '2rem', padding: '1.5rem', borderRadius: '16px', border: '1px solid var(--primary-light)' }}>
          <div className="result-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', paddingBottom: '1rem', borderBottom: '1px solid var(--border)' }}>
            <h3 className="form-title" style={{ margin: 0, color: 'var(--primary)', display: 'flex', alignItems: 'center', gap: '10px' }}>
              <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
              {t("receipt.scan.result")}
            </h3>
            <div className="badge-status running" style={{ borderRadius: '20px', padding: '6px 16px', fontWeight: 700 }}>
              {result.draft.confidence ? `${Math.round(result.draft.confidence * 100)}% Match` : "Success"}
            </div>
          </div>

          <div className="receipt-result-dashboard card-grid two-col stack-on-mobile" style={{ gap: "2.5rem" }}>
            <div className="result-info-side">
              <div className="info-grid-modern" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1.5rem" }}>
                <div className="info-block">
                  <span className="info-label-muted">{t("receipt.scan.merchant")}</span>
                  <div className="info-value-prominent">{result.draft.merchant_name || "-"}</div>
                </div>
                <div className="info-block">
                  <span className="info-label-muted">{t("receipt.scan.date")}</span>
                  <div className="info-value-prominent">{result.draft.transaction_date || "-"}</div>
                </div>
                <div className="info-block" style={{ gridColumn: "span 2", background: "var(--surface-sunken)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border)" }}>
                  <span className="info-label-muted">{t("receipt.scan.total")}</span>
                  <div className="info-value-hero" style={{ fontSize: "2rem", fontWeight: 800, color: "var(--primary)" }}>
                    {result.draft.currency} {result.draft.amount_minor.toLocaleString()}
                  </div>
                </div>
                <div className="info-block">
                  <span className="info-label-muted">{t("receipt.scan.category")}</span>
                  <div className="badge-status running-flat" style={{ marginTop: "4px" }}>{result.draft.category_name || "General"}</div>
                </div>
                <div className="info-block">
                  <span className="info-label-muted">{t("receipt.scan.paymentMethod")}</span>
                  <div className="info-value-small">{result.draft.payment_method || "-"}</div>
                </div>
                <div className="info-block">
                  <span className="info-label-muted">{t("receipt.scan.receiptNumber")}</span>
                  <div className="info-value-small">{result.draft.receipt_number || "-"}</div>
                </div>
                <div className="info-block">
                  <span className="info-label-muted">Tax & Service</span>
                  <div className="info-value-small">{result.draft.currency} {((result.draft.tax_minor ?? 0) + (result.draft.service_charge_minor ?? 0)).toLocaleString()}</div>
                </div>
              </div>
            </div>

            <div className="result-preview-side" style={{ minWidth: 0 }}>
              {Array.isArray(result.draft.items) && result.draft.items.length > 0 && (
                <div style={{ marginBottom: '2rem' }}>
                  <h4 className="info-label-muted" style={{ marginBottom: '1rem', display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M6 2 3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4Z"/><path d="M3 6h18"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>
                    {t("receipt.scan.itemsPreview")}
                  </h4>
                  <div className="data-table-wrap shadow-soft" style={{ borderRadius: "12px", border: "1px solid var(--border)", maxHeight: "250px", overflowY: "auto", overflowX: "auto" }}>
                    <table className="data-table text-xs">
                      <thead>
                        <tr>
                          <th>{t("transactions.items.name")}</th>
                          <th style={{ textAlign: "center" }}>Qty</th>
                          <th style={{ textAlign: "right" }}>{t("transactions.items.total")}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {result.draft.items.map((item, index) => (
                          <tr key={`${item.item_name}-${index}`}>
                            <td>{item.item_name}</td>
                            <td style={{ textAlign: "center" }}>{item.quantity}</td>
                            <td style={{ textAlign: "right", fontWeight: 600 }}>{result.draft.currency} {item.total_minor.toLocaleString()}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {filePreview && (
                <div>
                  <h4 className="info-label-muted" style={{ marginBottom: '1rem' }}>{t("common.preview")}</h4>
                  <div className="receipt-preview-result shadow-soft" style={{ overflow: "hidden", border: "1px solid var(--border)", borderRadius: "12px", background: "white" }}>
                    <img src={filePreview} alt="Receipt Preview" style={{ width: "100%", height: "auto", display: "block", maxHeight: "350px", objectFit: "contain", padding: "10px" }} />
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="action-row" style={{ marginTop: '2.5rem', display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <button className="btn btn-primary btn-block" type="button" onClick={() => handleUseDraft(result.draft)} style={{ padding: '0.85rem', fontSize: '1rem', fontWeight: 600 }}>
              {t("receipt.scan.useDraft")}
            </button>
            <button className="btn btn-secondary-outline btn-block" type="button" onClick={() => setResult(null)}>
              {t("common.cancel")}
            </button>
          </div>
        </div>
      )}

      <div className="card surface form-grid shadow-soft" style={{ marginTop: "2rem" }}>
        <div className="receipt-history-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h3 className="form-title" style={{ margin: 0 }}>{t("receipt.scan.history")}</h3>
          {history.length > 0 ? (
            <button
              className="btn btn-danger-outline"
              style={{ fontSize: '0.8rem', padding: '4px 8px' }}
              type="button"
              onClick={() => setIsClearModalOpen(true)}
              disabled={clearingHistory}
            >
              {clearingHistory ? t("common.loading") : "Hapus Semua"}
            </button>
          ) : null}
        </div>
        <div className="data-table-wrap table-mobile-stack" style={{ maxHeight: '400px', overflowY: 'auto' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("receipt.scan.table.file")}</th>
                <th>{t("receipt.scan.table.status")}</th>
                <th>{t("transactions.table.action")}</th>
              </tr>
            </thead>
            <tbody>
              {history.length === 0 ? (
                <tr><td colSpan={3}>{t("common.noItems")}</td></tr>
              ) : history.map((item) => (
                <tr key={item.id}>
                  <td data-label={t("receipt.scan.table.file")}>{item.original_filename}</td>
                  <td data-label={t("receipt.scan.table.status")}>{item.status}</td>
                  <td data-label={t("transactions.table.action")}>
                    <div className="table-actions">
                      <button
                        type="button"
                        className="btn btn-ghost-inline"
                        onClick={() => handleViewImage(item.id).catch(() => undefined)}
                        disabled={loadingImage}
                      >
                        {"Lihat"}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {viewingImageURL ? (
        <div className="image-modal-overlay" onClick={closeImageModal}>
          <div className="image-modal-content" onClick={(e) => e.stopPropagation()}>
            <button type="button" className="image-modal-close" onClick={closeImageModal}>✕</button>
            <img src={viewingImageURL} alt="Receipt scan" />
          </div>
        </div>
      ) : null}

      <DeleteConfirmModal
        isOpen={isClearModalOpen}
        title={t("common.delete")}
        message={t("receipt.scan.confirmClearAll") || "Hapus semua riwayat scan?"}
        isLoading={clearingHistory}
        onConfirm={() => handleClearHistory().catch(() => undefined)}
        onCancel={() => setIsClearModalOpen(false)}
      />

      <DeleteConfirmModal
        isOpen={!!itemToDelete}
        title={t("common.delete")}
        message={`${t("common.delete")} "${itemToDelete?.original_filename}"?`}
        isLoading={deletingScanID !== null}
        onConfirm={() => itemToDelete && handleDeleteScan(itemToDelete.id).catch(() => undefined)}
        onCancel={() => setItemToDelete(null)}
      />

      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}

