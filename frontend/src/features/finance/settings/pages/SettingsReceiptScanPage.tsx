import { useEffect, useMemo, useState } from "react";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { listReceiptProviders, testReceiptProviderConnection, updateReceiptProviders } from "../../receipts/api/receipts.api";
import { ReceiptProviderConfig, ReceiptProviderModelOption } from "../../receipts/api/receipts.types";
import { PasswordInput } from "../../../../core/components/PasswordInput";
import { PageHeader } from "../../../../core/components/PageHeader";

type ProviderFormState = ReceiptProviderConfig & {
  api_key?: string;
  clear_api_key?: boolean;
};

type ProviderModelState = Record<string, ReceiptProviderModelOption[]>;
type ProviderMessageState = Record<string, string | undefined>;

export function SettingsReceiptScanPage(): JSX.Element {
  const { t } = useI18n();
  const { toasts, success, error: showError, info, remove } = useToast();
  const [items, setItems] = useState<ProviderFormState[]>([]);
  const [modelOptions, setModelOptions] = useState<ProviderModelState>({});
  const [providerMessages, setProviderMessages] = useState<ProviderMessageState>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testingCode, setTestingCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const providerByCode = useMemo(() => {
    const map: Record<string, ProviderFormState> = {};
    for (const item of items) {
      map[item.provider_code] = item;
    }
    return map;
  }, [items]);

  function patchItem(providerCode: string, patch: Partial<ProviderFormState>): void {
    setItems((prev) => prev.map((entry) => (entry.provider_code === providerCode ? { ...entry, ...patch } : entry)));
  }

  async function load(): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const data = await listReceiptProviders();
      setItems(data.map((item) => ({ ...item, api_key: "", clear_api_key: false })));
      setModelOptions({});
      setProviderMessages({});
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load().catch(() => undefined);
  }, [t]);

  async function handleTestConnection(providerCode: string): Promise<void> {
    const item = providerByCode[providerCode];
    if (!item) {
      return;
    }
    if (!item.api_key?.trim() && !item.has_api_key) {
      showError(t("receipt.settings.testNeedApiKey"));
      return;
    }
    setTestingCode(providerCode);
    setProviderMessages((prev) => ({ ...prev, [providerCode]: undefined }));
    try {
      const out = await testReceiptProviderConnection({
        provider_code: item.provider_code,
        base_url: item.base_url || undefined,
        api_key: item.api_key?.trim() || undefined,
        model_name: item.model_name
      });
      const options = Array.isArray(out.models) ? out.models : [];
      setModelOptions((prev) => ({ ...prev, [providerCode]: options }));
      if (options.length > 0) {
        const hasCurrent = options.some((entry) => entry.id === item.model_name);
        if (!hasCurrent) {
          patchItem(providerCode, { model_name: options[0].id });
        }
      }
      const message = `${out.using_saved_api_key ? t("receipt.settings.testSuccessSaved") : t("receipt.settings.testSuccess")} (${options.length})`;
      setProviderMessages((prev) => ({ ...prev, [providerCode]: message }));
      if (out.base_url) {
        patchItem(providerCode, { base_url: out.base_url });
      }
      success(message);
    } catch (err) {
      let message = err instanceof Error ? err.message : t("errors.saveDataFailed");
      if (message.toLowerCase().includes("cannot be decrypted") || message.toLowerCase().includes("save the api key again")) {
        message = t("receipt.settings.testNeedApiKey") + ". " + t("receipt.settings.saveHint");
        patchItem(providerCode, { api_key: "", clear_api_key: false });
      }
      setProviderMessages((prev) => ({ ...prev, [providerCode]: message }));
      showError(message);
    } finally {
      setTestingCode(null);
    }
  }

  async function handleSave(): Promise<void> {
    setSaving(true);
    setError(null);
    try {
      const payload = items.map((item) => ({
        provider_code: item.provider_code,
        display_name: item.display_name,
        base_url: item.base_url || undefined,
        model_name: item.model_name,
        is_enabled: item.is_enabled,
        api_key: item.api_key?.trim() || undefined,
        clear_api_key: item.clear_api_key || false
      }));
      const updated = await updateReceiptProviders(payload);
      setItems(updated.map((item) => ({ ...item, api_key: "", clear_api_key: false })));
      info(t("receipt.settings.saveHint"));
      success(t("common.saveSuccess"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("errors.saveDataFailed");
      setError(message);
      showError(message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("receipt.settings.title")} 
        description={t("receipt.settings.subtitle")} 
      />


      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      {!loading ? (
        <div className="card-grid two-col">
          {items.map((item) => {
            const options = modelOptions[item.provider_code] ?? [];
            const hasModelOptions = options.length > 0;
            const helperMessage = providerMessages[item.provider_code];
            return (
              <div key={item.provider_code} className="card surface form-grid">
                <div className="form-field">
                  <strong>{item.display_name}</strong>
                  <small className="page-subtitle">{t(`receipt.provider.${item.provider_code}`)}</small>
                </div>
                <label className="form-field checkbox-inline">
                  <input
                    type="checkbox"
                    checked={item.is_enabled}
                    onChange={(event) => patchItem(item.provider_code, { is_enabled: event.target.checked })}
                  />
                  <span>{t("receipt.settings.enabled")}</span>
                </label>
                {item.provider_code === "openai_compatible" ? (
                  <label className="form-field">
                    {t("receipt.settings.baseUrl")}
                    <input
                      className="input-control"
                      value={item.base_url ?? ""}
                      placeholder="https://api.sumopod.com/v1"
                      onChange={(event) => patchItem(item.provider_code, { base_url: event.target.value })}
                    />
                  </label>
                ) : null}
                <label className="form-field">
                  {t("receipt.settings.apiKey")}
                  <input
                    className="input-control"
                    type="password"
                    value={item.api_key ?? ""}
                    placeholder={item.has_api_key ? t("receipt.settings.apiKeyConfigured") : t("receipt.settings.apiKeyPlaceholder")}
                    onChange={(event) => patchItem(item.provider_code, { api_key: event.target.value, clear_api_key: false })}
                  />
                  {item.has_api_key ? <small className="page-subtitle">{t("receipt.settings.apiKeyHidden")}</small> : null}
                </label>
                <div className="receipt-provider-actions">
                  <button
                    className="btn btn-secondary"
                    type="button"
                    onClick={() => handleTestConnection(item.provider_code).catch(() => undefined)}
                    disabled={testingCode === item.provider_code}
                  >
                    {testingCode === item.provider_code ? t("common.loading") : t("receipt.settings.testConnection")}
                  </button>
                  {helperMessage ? <small className="page-subtitle">{helperMessage}</small> : null}
                </div>
                <label className="form-field">
                  {t("receipt.settings.model")}
                  {hasModelOptions ? (
                    <select
                      className="input-control"
                      value={item.model_name}
                      onChange={(event) => patchItem(item.provider_code, { model_name: event.target.value })}
                    >
                      {options.map((option) => (
                        <option key={option.id} value={option.id}>
                          {option.label || option.id}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <input
                      className="input-control"
                      value={item.model_name}
                      placeholder={t("receipt.settings.modelPlaceholder")}
                      onChange={(event) => patchItem(item.provider_code, { model_name: event.target.value })}
                    />
                  )}
                </label>
                <label className="form-field checkbox-inline">
                  <input
                    type="checkbox"
                    checked={!!item.clear_api_key}
                    onChange={(event) => patchItem(item.provider_code, { clear_api_key: event.target.checked, api_key: event.target.checked ? "" : item.api_key })}
                  />
                  <span>{t("receipt.settings.clearApiKey")}</span>
                </label>
              </div>
            );
          })}
        </div>
      ) : null}

      {!loading ? (
        <div className="sticky-actions">
          <button className="btn btn-primary" type="button" onClick={() => handleSave().catch(() => undefined)} disabled={saving}>
            {saving ? t("common.loading") : t("receipt.settings.save")}
          </button>
        </div>
      ) : null}
      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}
