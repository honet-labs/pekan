import { useState } from "react";
import { Link } from "react-router-dom";
import { Transaction } from "../api/transaction.types";
import { openTransactionAttachment } from "../api/transaction.api";
import { useI18n } from "../../../../core/i18n/i18n";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { TransactionItemsModal } from "./TransactionItemsModal";

type Props = {
  items: Transaction[];
  onDelete: (transactionID: string) => Promise<void>;
};

export function TransactionTable({ items, onDelete }: Props): JSX.Element {
  const { locale, t } = useI18n();
  const [itemToDelete, setItemToDelete] = useState<Transaction | null>(null);
  const [itemToView, setItemToView] = useState<Transaction | null>(null);
  const [deleting, setDeleting] = useState(false);

  const formatAmount = (amount: number, currency: string) => {
    if (currency.toUpperCase() === "IDR") {
      return `Rp ${amount.toLocaleString(locale === "id" ? "id-ID" : "en-US")}`;
    }
    return `${amount.toLocaleString(locale === "id" ? "id-ID" : "en-US")} ${currency}`;
  };

  const renderCategory = (item: Transaction): string => {
    if (item.type === "savings") {
      return t("transactions.type.savings");
    }
    return item.category_name ?? item.category_id ?? "-";
  };

  const renderUser = (item: Transaction): string => item.created_by_name?.trim() || item.created_by || "-";
  
  const renderItemsSummary = (item: Transaction): JSX.Element | string => {
    if (item.type !== "expense" || !Array.isArray(item.items) || item.items.length === 0) {
      return "-";
    }
    const names = item.items.map((line) => line.item_name).filter(Boolean).slice(0, 2).join(", ");
    const hasMore = item.items.length > 2;
    
    return (
      <div style={{ display: 'flex', justifyContent: 'center' }}>
        <button 
          className="btn btn-ghost-inline btn-sm" 
          onClick={() => setItemToView(item)}
          style={{ 
            fontSize: '11px', 
            padding: '4px 8px', 
            border: '1px solid var(--border-color)',
            borderRadius: '6px',
            backgroundColor: 'rgba(255,255,255,0.5)'
          }}
        >
          {t("common.view")} {t("transactions.items.title")}
        </button>
      </div>
    );
  };

  const renderQty = (item: Transaction): string => {
    if (item.type !== "expense" || !Array.isArray(item.items) || item.items.length === 0) {
      return "-";
    }
    const qty = item.items.reduce((sum, line) => sum + Number(line.quantity || 0), 0);
    return qty.toLocaleString(locale === "id" ? "id-ID" : "en-US");
  };

  const renderDescription = (item: Transaction): string => {
    const merchant = item.merchant_name?.trim() || "";
    const desc = item.description?.trim() || "";
    if (merchant && desc) {
      return `${merchant} — ${desc}`;
    }
    return merchant || desc || "-";
  };

  const renderAttachments = (item: Transaction): JSX.Element | string => {
    const attachments = item.attachments;
    if (!Array.isArray(attachments) || attachments.length === 0) {
      return "N/A";
    }
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', alignItems: 'center' }}>
        {attachments.map((att, idx) => (
          <button
            key={att.id}
            className="btn btn-ghost-inline btn-sm"
            type="button"
            onClick={() => openTransactionAttachment(item.id, att.id).catch(() => undefined)}
            style={{
              fontSize: '11px',
              padding: '4px 8px',
              borderRadius: '6px',
              border: '1px solid var(--border)',
              backgroundColor: 'rgba(255, 255, 255, 0.05)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              maxWidth: '120px',
              cursor: 'pointer'
            }}
            title={att.original_filename}
          >
            📎 {attachments.length > 1 ? `Lihat Gambar ${idx + 1}` : "Lihat Gambar"}
          </button>
        ))}
      </div>
    );
  };

  const totalColumns = 12;

  async function handleConfirmDelete(): Promise<void> {
    if (!itemToDelete) {
      return;
    }
    setDeleting(true);
    try {
      await onDelete(itemToDelete.id);
      setItemToDelete(null);
    } finally {
      setDeleting(false);
    }
  }

  return (
    <>
      <div className="data-table-wrap table-mobile-stack">
        <table className="data-table">
          <thead>
            <tr>
              <th>TID</th>
              <th>{t("transactions.table.date")}</th>
              <th>{t("transactions.table.inputDate")}</th>
              <th>{t("transactions.table.type")}</th>
              <th>{t("transactions.table.total")}</th>
              <th>{t("transactions.table.category")}</th>
              <th>{t("transactions.items.title")}</th>
              <th>{t("transactions.items.qty")}</th>
              <th>{t("transactions.table.user")}</th>
              <th>{t("transactions.table.description")}</th>
              <th>{t("transactions.attachments.title") || "Lampiran"}</th>
              <th>{t("transactions.table.action")}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td data-label="TID">{item.tid}</td>
                <td data-label={t("transactions.table.date")}>{item.transaction_date}</td>
                <td data-label={t("transactions.table.inputDate")}>{item.input_date}</td>
                <td data-label={t("transactions.table.type")}>
                  <span className={`type-pill ${item.type}`}>{t(`transactions.type.${item.type}`)}</span>
                </td>
                <td data-label={t("transactions.table.total")}>{formatAmount(item.amount_minor, item.currency)}</td>
                <td data-label={t("transactions.table.category")}>{renderCategory(item)}</td>
                <td data-label={t("transactions.items.title")}>
                  {renderItemsSummary(item)}
                </td>
                <td data-label={t("transactions.items.qty")}>{renderQty(item)}</td>
                <td data-label={t("transactions.table.user")}>{renderUser(item)}</td>
                <td data-label={t("transactions.table.description")}>{renderDescription(item)}</td>
                <td data-label={t("transactions.attachments.title") || "Lampiran"}>
                  {renderAttachments(item)}
                </td>
                <td data-label={t("transactions.table.action")}>
                  <div className="table-actions">
                    <Link to={item.id} className="btn btn-ghost-inline">
                      {t("common.edit")}
                    </Link>
                    <button type="button" className="btn btn-ghost-inline danger" onClick={() => setItemToDelete(item)}>
                      {t("common.delete")}
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {!items.length ? (
              <tr>
                <td colSpan={totalColumns}>{t("common.noItems")}</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <TransactionItemsModal
        isOpen={!!itemToView}
        onClose={() => setItemToView(null)}
        transaction={itemToView}
      />

      <DeleteConfirmModal
        isOpen={!!itemToDelete}
        title={t("transactions.delete.title")}
        message={itemToDelete ? `${t("transactions.delete.confirm")} TID ${itemToDelete.tid}?` : t("transactions.delete.confirm")}
        isLoading={deleting}
        onConfirm={() => {
          handleConfirmDelete().catch(() => undefined);
        }}
        onCancel={() => {
          if (!deleting) {
            setItemToDelete(null);
          }
        }}
      />
    </>
  );
}
