import { useEffect, useState } from "react";
import { listReminderTemplates, upsertReminderTemplate } from "../api/settings.api";
import { ReminderTemplateSetting } from "../api/settings.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { PageHeader } from "../../../../core/components/PageHeader";

const defaultTemplateForm = {
  template_code: "reminder.due",
  channel_code: "any",
  language_code: "id",
  title_template: "",
  body_template: "Pengingat: {{title}} jatuh tempo pada {{due_date}}.",
  is_enabled: true
};

export function SettingsTemplatesPage(): JSX.Element {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [templates, setTemplates] = useState<ReminderTemplateSetting[]>([]);
  const [templateForm, setTemplateForm] = useState(defaultTemplateForm);
  const [savingTemplate, setSavingTemplate] = useState(false);

  async function loadTemplates(): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const result = await listReminderTemplates(defaultTemplateForm.template_code);
      setTemplates(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadTemplates().catch(() => undefined);
  }, [t]);

  async function handleSaveTemplate(): Promise<void> {
    setSavingTemplate(true);
    setError(null);
    try {
      await upsertReminderTemplate({
        template_code: templateForm.template_code,
        channel_code: templateForm.channel_code,
        language_code: templateForm.language_code,
        title_template: templateForm.title_template || undefined,
        body_template: templateForm.body_template,
        is_enabled: templateForm.is_enabled
      });
      const refreshed = await listReminderTemplates(templateForm.template_code);
      setTemplates(refreshed);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setSavingTemplate(false);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("settings.title")} 
        description={t("settings.subtitleTemplates")} 
      />


      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      <div className="card surface">
        <h3 className="form-title">{t("settings.templates.title")}</h3>
        <form
          className="form-grid"
          onSubmit={(event) => {
            event.preventDefault();
            handleSaveTemplate().catch(() => undefined);
          }}
        >
          <label className="form-field">
            {t("settings.templates.code")}
            <input
              className="input-control"
              value={templateForm.template_code}
              onChange={(event) => setTemplateForm({ ...templateForm, template_code: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("settings.templates.channel")}
            <select
              className="input-control"
              value={templateForm.channel_code}
              onChange={(event) => setTemplateForm({ ...templateForm, channel_code: event.target.value })}
            >
              <option value="any">{t("settings.channel.any")}</option>
              <option value="email">{t("settings.channel.email")}</option>
              <option value="telegram">{t("settings.channel.telegram")}</option>
              <option value="whatsapp_official">{t("settings.channel.whatsapp_official")}</option>
              <option value="whatsapp_gowa">{t("settings.channel.whatsapp_gowa")}</option>
              <option value="whatsapp_fonte">{t("settings.channel.whatsapp_fonte")}</option>
            </select>
          </label>
          <label className="form-field">
            {t("settings.templates.language")}
            <input
              className="input-control"
              value={templateForm.language_code}
              onChange={(event) => setTemplateForm({ ...templateForm, language_code: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("settings.templates.titleField")}
            <input
              className="input-control"
              value={templateForm.title_template}
              onChange={(event) => setTemplateForm({ ...templateForm, title_template: event.target.value })}
            />
          </label>
          <label className="form-field">
            {t("settings.templates.body")}
            <textarea
              className="input-control textarea-control"
              value={templateForm.body_template}
              onChange={(event) => setTemplateForm({ ...templateForm, body_template: event.target.value })}
              required
            />
          </label>
          <label className="form-field">
            {t("settings.templates.enabled")}
            <select
              className="input-control"
              value={templateForm.is_enabled ? "1" : "0"}
              onChange={(event) => setTemplateForm({ ...templateForm, is_enabled: event.target.value === "1" })}
            >
              <option value="1">{t("common.enabled")}</option>
              <option value="0">{t("common.disabled")}</option>
            </select>
          </label>
          <button className="btn btn-primary" type="submit" disabled={savingTemplate}>
            {savingTemplate ? t("common.loading") : t("settings.templates.save")}
          </button>
        </form>
        <div className="data-table-wrap table-mobile-stack">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("settings.templates.code")}</th>
                <th>{t("settings.templates.channel")}</th>
                <th>{t("settings.templates.language")}</th>
                <th>{t("settings.templates.enabled")}</th>
              </tr>
            </thead>
            <tbody>
              {templates.map((item) => (
                <tr key={item.id}>
                  <td data-label={t("settings.templates.code")}>{item.template_code}</td>
                  <td data-label={t("settings.templates.channel")}>{item.channel_code}</td>
                  <td data-label={t("settings.templates.language")}>{item.language_code}</td>
                  <td data-label={t("settings.templates.enabled")}>{item.is_enabled ? t("common.enabled") : t("common.disabled")}</td>
                </tr>
              ))}
              {!templates.length ? (
                <tr>
                  <td colSpan={4}>{t("common.noItems")}</td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}
