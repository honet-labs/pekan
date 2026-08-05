import { useState } from "react";
import { useI18n } from "../../../../core/i18n/i18n";
import { TransactionItem } from "../api/transaction.types";

interface TransactionItemsFormProps {
  items: TransactionItem[];
  onItemsChange: (items: TransactionItem[]) => void;
  locale: string;
}

export function TransactionItemsForm({ items, onItemsChange, locale }: TransactionItemsFormProps): JSX.Element {
  const { t } = useI18n();
  const numberFormatter = new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US");
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const recalcTotal = (quantity: number, unitPrice: number, discountMinor?: number): number => {
    const total = Math.round(Number(quantity || 0) * Number(unitPrice || 0)) - Number(discountMinor || 0);
    return total > 0 ? total : 0;
  };

  const addItem = () => {
    const newItem: TransactionItem = {
      id: `temp-${Date.now()}`,
      item_name: "",
      quantity: 1,
      price_per_unit_minor: 0,
      discount_minor: 0,
      total_minor: 0,
      notes: ""
    };
    onItemsChange([...items, newItem]);
    setEditingIndex(items.length);
  };

  const updateItem = (index: number, field: keyof TransactionItem, value: string | number) => {
    const updated = items.map((item, i) => {
      if (i !== index) {
        return item;
      }
      const next: TransactionItem = { ...item, [field]: value };
      if (field === "quantity" || field === "price_per_unit_minor" || field === "discount_minor") {
        next.total_minor = recalcTotal(next.quantity, next.price_per_unit_minor, next.discount_minor);
      }
      return next;
    });
    onItemsChange(updated);
  };

  const removeItem = (index: number) => {
    onItemsChange(items.filter((_, i) => i !== index));
  };

  return (
    <div className="form-field form-section">
      <span>{t("transactions.items.title")}</span>

      {items.length > 0 ? (
        <div className="data-table-wrap table-mobile-stack">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("transactions.items.name")}</th>
                <th>{t("transactions.items.qty")}</th>
                <th>{t("transactions.items.price")}</th>
                <th>{t("transactions.items.discount")}</th>
                <th>{t("transactions.items.total")}</th>
                <th>{t("transactions.items.notes")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item, index) => (
                <tr key={item.id || `item-${index}`}>
                  <td data-label={t("transactions.items.name")}>
                    {editingIndex === index ? (
                      <input
                        type="text"
                        className="input-control"
                        value={item.item_name}
                        onChange={(e) => updateItem(index, "item_name", e.target.value)}
                        placeholder={t("transactions.items.name")}
                      />
                    ) : (
                      item.item_name || "-"
                    )}
                  </td>
                  <td data-label={t("transactions.items.qty")}>
                    {editingIndex === index ? (
                      <input
                        type="number"
                        className="input-control"
                        value={item.quantity}
                        onChange={(e) => updateItem(index, "quantity", parseFloat(e.target.value) || 0)}
                        step="0.01"
                        min="0"
                      />
                    ) : (
                      numberFormatter.format(item.quantity)
                    )}
                  </td>
                  <td data-label={t("transactions.items.price")}>
                    {editingIndex === index ? (
                      <input
                        type="number"
                        className="input-control"
                        value={item.price_per_unit_minor}
                        onChange={(e) => updateItem(index, "price_per_unit_minor", parseInt(e.target.value, 10) || 0)}
                        min="0"
                      />
                    ) : (
                      `Rp ${numberFormatter.format(item.price_per_unit_minor)}`
                    )}
                  </td>
                  <td data-label={t("transactions.items.discount")}>
                    {editingIndex === index ? (
                      <input
                        type="number"
                        className="input-control"
                        value={item.discount_minor ?? 0}
                        onChange={(e) => updateItem(index, "discount_minor", parseInt(e.target.value, 10) || 0)}
                        min="0"
                      />
                    ) : (
                      `Rp ${numberFormatter.format(item.discount_minor ?? 0)}`
                    )}
                  </td>
                  <td data-label={t("transactions.items.total")}>
                    {`Rp ${numberFormatter.format(item.total_minor)}`}
                  </td>
                  <td data-label={t("transactions.items.notes")}>
                    {editingIndex === index ? (
                      <input
                        type="text"
                        className="input-control"
                        value={item.notes || ""}
                        onChange={(e) => updateItem(index, "notes", e.target.value)}
                        placeholder={t("transactions.items.notes")}
                      />
                    ) : (
                      item.notes || "-"
                    )}
                  </td>
                  <td data-label={t("common.action")}>
                    <div className="inline-buttons">
                      {editingIndex === index ? (
                        <button type="button" className="btn btn-sm btn-primary" onClick={() => setEditingIndex(null)}>
                          Done
                        </button>
                      ) : (
                        <button type="button" className="btn btn-sm btn-secondary" onClick={() => setEditingIndex(index)}>
                          {t("common.edit")}
                        </button>
                      )}
                      <button type="button" className="btn btn-sm btn-danger" onClick={() => removeItem(index)}>
                        {t("transactions.items.remove")}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      <button type="button" className="btn btn-secondary" onClick={addItem}>
        {t("transactions.items.addItem")}
      </button>
    </div>
  );
}
