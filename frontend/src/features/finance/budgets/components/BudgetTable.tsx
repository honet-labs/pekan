import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Budget } from "../api/budgets.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { InfoModal } from "../../../../core/components/InfoModal";

type Props = {
  items: Budget[];
  onDelete: (id: string) => Promise<void>;
  onViewTransactions?: (item: Budget) => void;
};

export function BudgetTable({ items, onDelete, onViewTransactions }: Props): JSX.Element {
  const { tenantCode } = useParams();
  const { locale, t } = useI18n();
  const [itemToDelete, setItemToDelete] = useState<Budget | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [noteToShow, setNoteToShow] = useState<{ title: string; notes: string } | null>(null);
  const numberFormatter = new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US");

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
            <th>IDA</th>
            <th>{t("budgets.table.name")}</th>
            <th>{t("budgets.form.category")}</th>
            <th>{t("budgets.table.limit")}</th>
            <th>{t("budgets.table.progress")}</th>
            <th>{t("budgets.table.period")}</th>
            <th>{t("budgets.table.notes")}</th>
            <th>{t("budgets.table.status")}</th>
            <th>{t("budgets.table.action")}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id}>
              <td data-label="IDA">{item.ida}</td>
              <td data-label={t("budgets.table.name")}>{item.name}</td>
              <td data-label={t("budgets.form.category")}>{item.category_name ?? t("budgets.form.allCategories")}</td>
              <td data-label={t("budgets.table.limit")}>Rp {numberFormatter.format(item.amount_limit_minor)}</td>
              <td data-label={t("budgets.table.progress")}>
                <div className="progress-cell">
                  <span>{Number(item.progress_percent ?? 0).toFixed(2)}%</span>
                  <div className="progress-track">
                    <span
                      className="progress-fill"
                      style={{ width: `${Math.max(0, Math.min(Number(item.progress_percent ?? 0), 100))}%` }}
                    />
                  </div>
                </div>
              </td>
              <td data-label={t("budgets.table.period")}>{t(`budgets.period.${item.period}`)}</td>
              <td data-label={t("budgets.table.notes")}>
                {item.notes ? (
                  <button 
                    className="btn btn-ghost-inline" 
                    title={t("common.view")}
                    onClick={() => setNoteToShow({ title: item.name, notes: item.notes || "" })}
                  >
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                      <circle cx="12" cy="12" r="3" />
                    </svg>
                  </button>
                ) : "-"}
              </td>
              <td data-label={t("budgets.table.status")}>{t(`budgets.status.${item.status}`)}</td>
              <td data-label={t("budgets.table.action")}>
                <div className="table-actions">
                  {onViewTransactions && (
                    <button
                      className="btn btn-ghost-inline"
                      type="button"
                      title={t("savings.viewTransactions")}
                      onClick={() => onViewTransactions(item)}
                      style={{
                        color: "var(--primary-color)",
                        fontWeight: 600,
                        display: "inline-flex",
                        alignItems: "center",
                        gap: "4px"
                      }}
                    >
                      <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                        <line x1="8" y1="6" x2="21" y2="6" /><line x1="8" y1="12" x2="21" y2="12" /><line x1="8" y1="18" x2="21" y2="18" />
                        <line x1="3" y1="6" x2="3.01" y2="6" /><line x1="3" y1="12" x2="3.01" y2="12" /><line x1="3" y1="18" x2="3.01" y2="18" />
                      </svg>
                      {t("savings.viewTransactions")}
                    </button>
                  )}
                  <Link className="btn btn-ghost-inline" to={`/app/${tenantCode ?? "default"}/finance/budgets/${item.id}`}>
                    {t("common.edit")}
                  </Link>
                  <button className="btn btn-ghost-inline danger" type="button" onClick={() => setItemToDelete(item)}>
                    {t("common.delete")}
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {!items.length ? (
            <tr>
              <td colSpan={9}>{t("common.noItems")}</td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
      <DeleteConfirmModal
        isOpen={!!itemToDelete}
        title={t("budgets.delete.title")}
        message={itemToDelete ? `${t("budgets.delete.confirm")} "${itemToDelete.name}"?` : t("budgets.delete.confirm")}
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
      <InfoModal 
        isOpen={!!noteToShow}
        title={noteToShow?.title ?? ""}
        description={noteToShow?.notes ?? ""}
        onClose={() => setNoteToShow(null)}
      />
    </>
  );
}
