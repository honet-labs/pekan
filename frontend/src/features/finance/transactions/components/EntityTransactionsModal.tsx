import { useEffect, useState } from "react";
import { useI18n } from "../../../../core/i18n/i18n";
import { Transaction } from "../api/transaction.types";
import { listTransactions, listTransactionsBySavings } from "../api/transaction.api";

export interface EntityTransactionsModalProps {
  isOpen: boolean;
  title: string;
  type: "savings" | "budget";
  entityId: string;
  startDate?: string | null;
  endDate?: string | null;
  onClose: () => void;
}

export function EntityTransactionsModal({
  isOpen,
  title,
  type,
  entityId,
  startDate,
  endDate,
  onClose
}: EntityTransactionsModalProps): JSX.Element | null {
  const { locale, t } = useI18n();
  const [items, setItems] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen && entityId) {
      loadTransactions();
    }
  }, [isOpen, entityId, type, startDate, endDate]);

  async function loadTransactions() {
    setLoading(true);
    try {
      let data: Transaction[] = [];
      if (type === "savings") {
        data = await listTransactionsBySavings(entityId);
      } else {
        // For budgets, we filter by category_id and budget's start & end dates
        const resp = await listTransactions({ 
          category_id: entityId, 
          page_size: 100,
          from: startDate || undefined,
          to: endDate || undefined
        });
        data = resp.items;
      }
      setItems(data);
    } catch (err) {
      console.error("Failed to load transactions:", err);
    } finally {
      setLoading(false);
    }
  }

  if (!isOpen) return null;

  const formatAmount = (amount: number, currency: string) => {
    if (currency.toUpperCase() === "IDR") {
      return `Rp ${amount.toLocaleString(locale === "id" ? "id-ID" : "en-US")}`;
    }
    return `${amount.toLocaleString(locale === "id" ? "id-ID" : "en-US")} ${currency}`;
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content modal-lg" onClick={(e) => e.stopPropagation()} style={{ maxWidth: '900px' }}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
             <h2 className="modal-title">{title}</h2>
          </div>
          <button type="button" className="modal-close" onClick={onClose}>✕</button>
        </div>
        
        <div className="modal-body" style={{ maxHeight: '70vh', overflowY: 'auto', padding: '0' }}>
          {loading ? (
            <div style={{ padding: '3rem', textAlign: 'center' }}>
               <div className="spinner" style={{ marginBottom: '1rem' }}></div>
               <p className="text-muted">{t("common.loading") || "Loading..."}</p>
            </div>
          ) : items.length === 0 ? (
            <div style={{ padding: '3rem', textAlign: 'center' }}>
               <p className="text-muted" style={{ fontSize: '1.1rem' }}>{t("common.noItems") || "Tidak ada data transaksi"}</p>
            </div>
          ) : (
            <div className="data-table-wrap table-mobile-stack entity-transactions-table-wrap" style={{ border: 'none', borderRadius: '0' }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>TID</th>
                    <th>{t("transactions.table.date")}</th>
                    <th>{t("transactions.table.type")}</th>
                    <th>{t("transactions.table.total")}</th>
                    <th>{t("transactions.table.description")}</th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((item) => (
                    <tr key={item.id}>
                      <td data-label="TID">
                        <span style={{ fontWeight: 600, color: 'var(--primary-color)', fontSize: '0.85rem' }}>{item.tid}</span>
                      </td>
                      <td data-label={t("transactions.table.date")} style={{ whiteSpace: 'nowrap' }}>{item.transaction_date}</td>
                      <td data-label={t("transactions.table.type")}>
                        <span className={`type-pill ${item.type}`} style={{ fontSize: '10px', padding: '2px 8px' }}>
                          {t(`transactions.type.${item.type}`)}
                        </span>
                      </td>
                      <td data-label={t("transactions.table.total")} style={{ fontWeight: 600 }}>{formatAmount(item.amount_minor, item.currency)}</td>
                      <td data-label={t("transactions.table.description")} style={{ fontSize: '0.9rem' }}>
                        {item.description || item.merchant_name || "-"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        
        <div className="modal-footer" style={{ padding: '1rem 1.5rem' }}>
          <button type="button" className="btn btn-secondary" onClick={onClose}>
            {t("common.close") || "Tutup"}
          </button>
        </div>
      </div>
    </div>
  );
}
