import { useEffect, useState } from "react";
import { listNotificationChannels, updateNotificationChannels } from "../api/settings.api";
import { NotificationChannelSetting } from "../api/settings.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { PageHeader } from "../../../../core/components/PageHeader";
import { useToast } from "../../../../core/hooks/useToast";
import { ToastContainer } from "../../../../core/components/Toast";

export function SettingsPage(): JSX.Element {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [channels, setChannels] = useState<NotificationChannelSetting[]>([]);
  const [savingChannels, setSavingChannels] = useState(false);
  const { toasts, success, error: showError, remove: removeToast } = useToast();

  async function loadChannels(): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const result = await listNotificationChannels();
      setChannels(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadChannels().catch(() => undefined);
  }, [t]);

  async function handleSaveChannels(): Promise<void> {
    setSavingChannels(true);
    setError(null);
    try {
      await updateNotificationChannels(channels);
      await loadChannels();
      success(t("common.saveSuccess"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("errors.saveDataFailed");
      setError(message);
      showError(message);
    } finally {
      setSavingChannels(false);
    }
  }

  const updateConfig = (channelCode: string, key: string, value: string) => {
    setChannels(prev => prev.map(ch => {
      if (ch.channel_code === channelCode) {
        return {
          ...ch,
          config_json: {
            ...(ch.config_json as any || {}),
            [key]: value
          }
        };
      }
      return ch;
    }));
  };

  return (
    <section className="page-section">
      <PageHeader 
        title={t("settings.title")} 
        description={t("settings.subtitleNotifications")} 
      />

      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      <div className="card surface">
        <h3 className="form-title">{t("settings.channels.title")}</h3>
        <div className="entity-list">
          {channels
            .filter(ch => ["email", "telegram", "whatsapp_official"].includes(ch.channel_code))
            .map((channel) => {
            const config = (channel.config_json as any) || {};
            return (
              <div key={channel.channel_code} className="entity-item">
                <div className="entity-row-flex">
                  <div className="entity-info">
                    <strong>{t(`settings.channel.${channel.channel_code}`)}</strong>
                    <small className="page-subtitle">{t("settings.channels.destinationHint")}</small>
                  </div>
                  <select
                    className="input-control"
                    style={{ width: "auto" }}
                    value={channel.is_enabled ? "1" : "0"}
                    onChange={(event) =>
                      setChannels((prev) =>
                        prev.map((item) =>
                          item.channel_code === channel.channel_code ? { ...item, is_enabled: event.target.value === "1" } : item
                        )
                      )
                    }
                  >
                    <option value="1">{t("common.enabled")}</option>
                    <option value="0">{t("common.disabled")}</option>
                  </select>
                </div>

                <div className="form-grid spacing-mt-sm">
                  {channel.channel_code === "email" && (
                    <label className="form-field">
                      {t("settings.channels.toEmail")}
                      <input 
                        className="input-control" 
                        value={config.to_email || ""} 
                        onChange={(e) => updateConfig(channel.channel_code, "to_email", e.target.value)}
                        placeholder="email1@example.com, email2@example.com"
                      />
                      <small className="page-subtitle" style={{ marginTop: "0.25rem", display: "block" }}>
                        Pisahkan dengan koma untuk banyak email.
                      </small>
                    </label>
                  )}
                  {channel.channel_code === "telegram" && (
                    <label className="form-field">
                      {t("settings.channels.telegramChatId")}
                      <input 
                        className="input-control" 
                        value={config.chat_id || ""} 
                        onChange={(e) => updateConfig(channel.channel_code, "chat_id", e.target.value)}
                        placeholder="-100..."
                      />
                    </label>
                  )}
                  {channel.channel_code === "whatsapp_official" && (
                    <label className="form-field">
                      {t("settings.channels.whatsappPhone")}
                      <input 
                        className="input-control" 
                        value={config.phone_number || ""} 
                        onChange={(e) => updateConfig(channel.channel_code, "phone_number", e.target.value)}
                        placeholder="62812..., 62813..."
                      />
                      <small className="page-subtitle" style={{ marginTop: "0.25rem", display: "block" }}>
                        Pisahkan dengan koma untuk banyak nomor.
                      </small>
                    </label>
                  )}
                </div>
              </div>
            );
          })}
        </div>
        <div className="spacing-mt-lg">
          <button className="btn btn-primary" type="button" onClick={() => handleSaveChannels().catch(() => undefined)} disabled={savingChannels}>
            {savingChannels ? t("common.loading") : t("common.saveChanges")}
          </button>
        </div>
      </div>
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}
