import { useEffect, useMemo, useState } from "react";
import { useDebounce } from "../../../../core/hooks/useDebounce";
import { listAuditLogs } from "../api/settings.api";
import { AuditLogEntry } from "../api/settings.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { PageHeader } from "../../../../core/components/PageHeader";

function humanizeWords(value: string): string {
  return value
    .replace(/[_\.]+/g, " ")
    .replace(/\b\w/g, (match) => match.toUpperCase())
    .trim();
}

function toSentenceCase(value: string): string {
  if (!value) {
    return value;
  }
  return value.charAt(0).toUpperCase() + value.slice(1).toLowerCase();
}


function asObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function readString(record: Record<string, unknown> | null, keys: string[]): string | null {
  if (!record) {
    return null;
  }
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string" && value.trim() !== "") {
      return value.trim();
    }
  }
  return null;
}

function toTransactionDisplayID(value: string): string {
  const normalized = value.trim();
  if (!normalized) {
    return normalized;
  }
  return normalized.length >= 8 ? normalized.slice(0, 8).toUpperCase() : normalized.toUpperCase();
}

export function AuditLogsPage(): JSX.Element {
  const { locale, t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [items, setItems] = useState<AuditLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [filter, setFilter] = useState({
    action: "",
    resource_type: "",
    actor_user_id: ""
  });
  const debouncedFilter = useDebounce(filter, 500);

  const actionLabels = useMemo(
    () => ({
      "finance.report.delete": locale === "id" ? "Hapus laporan" : "Delete report",
      "finance.report.create": locale === "id" ? "Buat laporan" : "Create report",
      "finance.transaction.delete": locale === "id" ? "Hapus transaksi" : "Delete transaction",
      "finance.transaction.create": locale === "id" ? "Buat transaksi" : "Create transaction",
      "finance.transaction.update": locale === "id" ? "Ubah transaksi" : "Update transaction",
      "finance.transaction.attachment.download": locale === "id" ? "Unduh lampiran transaksi" : "Download transaction attachment",
      "finance.budget.delete": locale === "id" ? "Hapus anggaran" : "Delete budget",
      "finance.budget.create": locale === "id" ? "Buat anggaran" : "Create budget",
      "finance.budget.update": locale === "id" ? "Ubah anggaran" : "Update budget",
      "finance.savings.delete": locale === "id" ? "Hapus target tabungan" : "Delete savings goal",
      "finance.savings.create": locale === "id" ? "Buat target tabungan" : "Create savings goal",
      "finance.savings.update": locale === "id" ? "Ubah target tabungan" : "Update savings goal",
      "finance.reminder.delete": locale === "id" ? "Hapus pengingat" : "Delete reminder",
      "finance.reminder.create": locale === "id" ? "Buat pengingat" : "Create reminder",
      "finance.reminder.update": locale === "id" ? "Ubah pengingat" : "Update reminder",
      "finance.settings.user.delete": locale === "id" ? "Hapus pengguna" : "Delete user",
      "finance.settings.notification.channels.update": locale === "id" ? "Ubah kanal notifikasi" : "Update notification channels",
      "finance.settings.template.update": locale === "id" ? "Ubah template pesan" : "Update message template",
      "auth.login.success": locale === "id" ? "Login berhasil" : "Login success"
    }),
    [locale]
  );

  const resourceLabels = useMemo(
    () => ({
      finance_report: locale === "id" ? "laporan" : "report",
      finance_transaction: locale === "id" ? "transaksi" : "transaction",
      finance_budget: locale === "id" ? "anggaran" : "budget",
      finance_savings: locale === "id" ? "tabungan" : "savings",
      finance_reminder: locale === "id" ? "pengingat" : "reminder",
      finance_transaction_attachment: locale === "id" ? "lampiran transaksi" : "transaction attachment",
      finance_notification_channels: locale === "id" ? "kanal notifikasi" : "notification channels",
      finance_message_template: locale === "id" ? "template pesan" : "message template",
      tenant_membership: locale === "id" ? "pengguna workspace" : "workspace user",
      auth_login: locale === "id" ? "login" : "login"
    }),
    [locale]
  );

  const formatAction = (action: string): string => actionLabels[action as keyof typeof actionLabels] ?? humanizeWords(action);

  const getResourceDisplayID = (item: AuditLogEntry): string => {
    const before = asObject(item.before_json);
    const after = asObject(item.after_json);
    switch (item.resource_type) {
      case "finance_transaction":
        return readString(after, ["tid"]) ?? readString(before, ["tid"]) ?? toTransactionDisplayID(item.resource_id);
      case "finance_budget":
        return (
          readString(after, ["ida", "id_anggaran"]) ??
          readString(before, ["ida", "id_anggaran"]) ??
          item.resource_id
        );
      case "finance_savings":
        return readString(after, ["sid"]) ?? readString(before, ["sid"]) ?? item.resource_id;
      case "finance_report":
        return readString(after, ["id"]) ?? readString(before, ["id"]) ?? item.resource_id;
      case "finance_reminder":
        return readString(after, ["id", "name"]) ?? readString(before, ["id", "name"]) ?? item.resource_id;
      case "finance_notification_channels":
        return readString(after, ["channel_code", "id"]) ?? readString(before, ["channel_code", "id"]) ?? item.resource_id;
      case "finance_message_template":
        return readString(after, ["template_code", "id"]) ?? readString(before, ["template_code", "id"]) ?? item.resource_id;
      case "tenant_membership":
        return readString(after, ["email", "membership_id", "id"]) ?? readString(before, ["email", "membership_id", "id"]) ?? item.resource_id;
      default:
        return item.resource_id;
    }
  };

  const formatResource = (item: AuditLogEntry): string => {
    const label = resourceLabels[item.resource_type as keyof typeof resourceLabels] ?? humanizeWords(item.resource_type).toLowerCase();
    const displayID = getResourceDisplayID(item);
    // If displayID is a long UUID-like string, shorten it
    const shortID = displayID && displayID.length > 40 ? displayID.slice(0, 16) + "..." : displayID;
    return displayID ? `${label} - ${shortID}` : label;
  };

  async function loadAuditLogs(): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const data = await listAuditLogs({
        page: 1,
        page_size: 50,
        action: debouncedFilter.action || undefined,
        resource_type: debouncedFilter.resource_type || undefined,
        actor_user_id: debouncedFilter.actor_user_id || undefined
      });
      setItems(data.items);
      setTotal(data.pagination.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAuditLogs().catch(() => undefined);
  }, [t, debouncedFilter]);

  return (
    <section className="page-section">
      <PageHeader 
        title={t("settings.audit.title")} 
        description={t("settings.audit.subtitle")} 
      />

      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      <div className="card surface">
        <div className="filter-row-flex" style={{ marginBottom: "1.5rem" }}>
          <label className="form-field">
            {t("settings.audit.action")}
            <input
              className="input-control"
              value={filter.action}
              onChange={(event) => setFilter({ ...filter, action: event.target.value })}
              placeholder={locale === "id" ? "Mis. hapus laporan" : "E.g. delete report"}
            />
          </label>
          <label className="form-field">
            {t("settings.audit.resourceType")}
            <input
              className="input-control"
              value={filter.resource_type}
              onChange={(event) => setFilter({ ...filter, resource_type: event.target.value })}
              placeholder={locale === "id" ? "Mis. transaksi" : "E.g. transaction"}
            />
          </label>
          <label className="form-field">
            {t("settings.audit.actorUser")}
            <input
              className="input-control"
              value={filter.actor_user_id}
              onChange={(event) => setFilter({ ...filter, actor_user_id: event.target.value })}
              placeholder={locale === "id" ? "Cari nama pengguna" : "Search user name"}
            />
          </label>
        </div>

        <p className="page-subtitle">{`${t("settings.audit.total")}: ${total}`}</p>
        <div className="data-table-wrap table-mobile-stack">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("settings.audit.when")}</th>
                <th>{t("settings.audit.action")}</th>
                <th>{t("settings.audit.resource")}</th>
                <th>{t("settings.audit.actorUser")}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id}>
                  <td data-label={t("settings.audit.when")} className="no-wrap opacity-90">
                    {new Date(item.created_at).toLocaleString(locale === "id" ? "id-ID" : "en-US", { dateStyle: "short", timeStyle: "short" })}
                  </td>
                  <td data-label={t("settings.audit.action")}>
                    <span className="badge-status running-flat" >{formatAction(item.action)}</span>
                  </td>
                  <td data-label={t("settings.audit.resource")} className="opacity-90" style={{ wordBreak: "break-word" }}>{formatResource(item)}</td>
                  <td data-label={t("settings.audit.actorUser")} >{item.actor_user_name ?? item.actor_user_id ?? "-"}</td>
                </tr>
              ))}
              {!items.length ? (
                <tr>
                  <td colSpan={5}>{t("common.noItems")}</td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}
