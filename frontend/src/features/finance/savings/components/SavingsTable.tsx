import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Savings } from "../api/savings.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { InfoModal } from "../../../../core/components/InfoModal";

type Props = {
  items: Savings[];
  onDelete: (id: string) => Promise<void>;
};

function resolveProgressPercent(item: Savings): number {
  if (typeof item.progress_percent === "number" && Number.isFinite(item.progress_percent)) {
    return item.progress_percent;
  }
  if (item.target_amount_minor <= 0) {
    return 0;
  }
  return (item.current_amount_minor / item.target_amount_minor) * 100;
}

export function SavingsTable({ items, onDelete }: Props): JSX.Element {
  const { tenantCode } = useParams();
  const { locale, t } = useI18n();
  const [itemToDelete, setItemToDelete] = useState<Savings | null>(null);
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
            <th>{t("savings.table.id")}</th>
            <th>{t("savings.table.name")}</th>
            <th>{t("savings.table.target")}</th>
            <th>{t("savings.table.current")}</th>
            <th>{t("savings.table.progress")}</th>
            <th>{t("savings.table.notes")}</th>
            <th>{t("savings.table.status")}</th>
            <th>{t("savings.table.action")}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const progressPercent = resolveProgressPercent(item);
            return (
              <tr key={item.id}>
                <td data-label={t("savings.table.id")}>{item.sid}</td>
                <td data-label={t("savings.table.name")}>{item.name}</td>
                <td data-label={t("savings.table.target")}>Rp {numberFormatter.format(item.target_amount_minor)}</td>
                <td data-label={t("savings.table.current")}>Rp {numberFormatter.format(item.current_amount_minor)}</td>
                <td data-label={t("savings.table.progress")}>
                  <div className="progress-cell">
                    <span>{progressPercent.toFixed(2)}%</span>
                    <div className="progress-track">
                      <span className="progress-fill" style={{ width: `${Math.max(0, Math.min(progressPercent, 100))}%` }} />
                    </div>
                  </div>
                </td>
                <td data-label={t("savings.table.notes")}>
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
                <td data-label={t("savings.table.status")}>{t(`savings.status.${item.status}`)}</td>
                <td data-label={t("savings.table.action")}>
                  <div className="table-actions">
                    <Link className="btn btn-ghost-inline" to={`/app/${tenantCode ?? "default"}/finance/savings/${item.id}`}>
                      {t("common.edit")}
                    </Link>
                    <button className="btn btn-ghost-inline danger" type="button" onClick={() => setItemToDelete(item)}>
                      {t("common.delete")}
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
          {!items.length ? (
            <tr>
              <td colSpan={8}>{t("common.noItems")}</td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
      <DeleteConfirmModal
        isOpen={!!itemToDelete}
        title={t("savings.delete.title")}
        message={itemToDelete ? `${t("savings.delete.confirm")} "${itemToDelete.name}"?` : t("savings.delete.confirm")}
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
