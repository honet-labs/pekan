import { Transaction, TransactionItem } from "../api/transaction.types";
import { useI18n } from "../../../../core/i18n/i18n";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  transaction: Transaction | null;
};

export function TransactionItemsModal({ isOpen, onClose, transaction }: Props): JSX.Element | null {
  const { locale, t } = useI18n();

  if (!isOpen || !transaction) return null;

  const items = transaction.items || [];
  const currency = transaction.currency;

  const formatAmount = (amount: number) => {
    if (currency.toUpperCase() === "IDR") {
      return `Rp ${amount.toLocaleString(locale === "id" ? "id-ID" : "en-US")}`;
    }
    return `${amount.toLocaleString(locale === "id" ? "id-ID" : "en-US")} ${currency}`;
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content surface" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem', borderBottom: '1px solid var(--border)', paddingBottom: '1rem' }}>
          <h3 style={{ margin: 0 }}>{t("transactions.items.title")} - {transaction.tid}</h3>
          <button 
            className="modal-close" 
            onClick={onClose}
            style={{ 
              background: 'none', 
              border: 'none', 
              cursor: 'pointer', 
              color: 'var(--text-muted)',
              padding: '4px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'color 0.2s'
            }}
            onMouseOver={(e) => e.currentTarget.style.color = 'var(--text)'}
            onMouseOut={(e) => e.currentTarget.style.color = 'var(--text-muted)'}
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
        <div className="modal-body">
          <div className="data-table-wrap transaction-list-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t("transactions.items.name")}</th>
                  <th>{t("transactions.items.qty")}</th>
                  <th>{t("transactions.items.price")}</th>
                  <th>{t("transactions.items.discount")}</th>
                  <th>{t("transactions.items.total")}</th>
                </tr>
              </thead>
              <tbody>
                {items.length > 0 ? (
                  items.map((item, idx) => (
                    <tr key={item.id ?? idx}>
                      <td data-label={t("transactions.items.name")}>{item.item_name}</td>
                      <td data-label={t("transactions.items.qty")}>{item.quantity.toLocaleString(locale === "id" ? "id-ID" : "en-US")}</td>
                      <td data-label={t("transactions.items.price")}>{formatAmount(item.price_per_unit_minor)}</td>
                      <td data-label={t("transactions.items.discount")}>{formatAmount(item.discount_minor ?? 0)}</td>
                      <td data-label={t("transactions.items.total")}>{formatAmount(item.total_minor)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5}>{t("common.noItems")}</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {transaction.description && (
            <div className="form-field" style={{ marginTop: '1rem' }}>
              <strong>{t("transactions.table.description")}</strong>
              <p className="page-subtitle">{transaction.description}</p>
            </div>
          )}
        </div>
        <div className="modal-footer">
          <button className="btn btn-secondary" onClick={onClose}>
            {t("common.close")}
          </button>
        </div>
      </div>
    </div>
  );
}
