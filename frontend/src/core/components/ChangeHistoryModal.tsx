import { useI18n } from "../i18n/i18n";

export interface ChangeHistory {
  id: string;
  field_name: string;
  old_value_text: string | null;
  new_value_text: string | null;
  change_type: "create" | "update" | "delete";
  changed_by: string;
  created_at: string;
}

interface ChangeHistoryModalProps {
  isOpen: boolean;
  history: ChangeHistory[];
  onClose: () => void;
}

export function ChangeHistoryModal({ isOpen, history, onClose }: ChangeHistoryModalProps): JSX.Element | null {
  const { t, locale } = useI18n();

  if (!isOpen) {
    return null;
  }

  const dateFormatter = new Intl.DateTimeFormat(locale === "id" ? "id-ID" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  });

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content modal-lg" onClick={(e) => e.stopPropagation()}>
        <header className="modal-header">
          <h2>{t("common.changeHistory")}</h2>
          <button type="button" className="btn-icon" onClick={onClose} title={t("common.close")}>
            ✕
          </button>
        </header>

        <div className="modal-body">
          {history.length === 0 ? (
            <p className="page-subtitle">{t("common.noData")}</p>
          ) : (
            <div className="data-table-wrap table-mobile-stack">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t("common.timestamp")}</th>
                    <th>{t("common.field")}</th>
                    <th>{t("common.changeType")}</th>
                    <th>{t("common.oldValue")}</th>
                    <th>{t("common.newValue")}</th>
                    <th>{t("common.changedBy")}</th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((change) => (
                    <tr key={change.id}>
                      <td data-label={t("common.timestamp")}>{dateFormatter.format(new Date(change.created_at))}</td>
                      <td data-label={t("common.field")}>
                        <code>{change.field_name}</code>
                      </td>
                      <td data-label={t("common.changeType")}>
                        <span className={`type-pill ${change.change_type}`}>{change.change_type}</span>
                      </td>
                      <td data-label={t("common.oldValue")} className="monospace">
                        {change.old_value_text || "-"}
                      </td>
                      <td data-label={t("common.newValue")} className="monospace">
                        {change.new_value_text || "-"}
                      </td>
                      <td data-label={t("common.changedBy")}>{change.changed_by}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <footer className="modal-footer">
          <button type="button" className="btn btn-primary" onClick={onClose}>
            {t("common.close")}
          </button>
        </footer>
      </div>
    </div>
  );
}
