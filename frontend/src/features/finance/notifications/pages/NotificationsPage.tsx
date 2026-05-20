import { useNotifications } from "../hooks/useNotifications";
import { useI18n } from "../../../../core/i18n/i18n";

export function NotificationsPage(): JSX.Element {
  const { items, loading, error, markRead } = useNotifications();
  const { t } = useI18n();

  return (
    <section className="page-section">
      <header className="page-header">
        <div>
          <h1 className="page-title">{t("notifications.title")}</h1>
          <div className="tagline-info" title={t("notifications.subtitle")}>
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
          </div>
        </div>
      </header>

      {error ? <div className="alert error">{error}</div> : null}

      <div className="card surface">
        <h3 className="form-title">{t("notifications.list.title")}</h3>
        {loading ? <p className="page-subtitle">{t("common.loading")}</p> : null}
        <ul className="entity-list">
          {items.map((item) => (
            <li key={item.id} className="entity-item">
              <strong>{item.title}</strong>
              <p className="page-subtitle">{item.message}</p>
              <div className="inline-metrics">
                <span className={`pill ${item.status === "read" ? "info" : "warning"}`}>{item.status}</span>
                {item.status !== "read" ? (
                  <button className="btn btn-ghost-inline" type="button" onClick={() => markRead(item.id).catch(() => undefined)}>
                    {t("notifications.markRead")}
                  </button>
                ) : null}
              </div>
            </li>
          ))}
          {!items.length && !loading ? <li className="entity-item">{t("common.noItems")}</li> : null}
        </ul>
      </div>
    </section>
  );
}

