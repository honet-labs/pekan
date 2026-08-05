import { FormEvent, useState } from "react";
import { LanguageSwitcher } from "../../../../core/components/LanguageSwitcher";
import { Link } from "react-router-dom";

import { requestPasswordReset } from "../../../../core/auth/auth-api";
import { useI18n } from "../../../../core/i18n/i18n";

export function ForgotPasswordPage(): JSX.Element {
  const { t } = useI18n();
  const [email, setEmail] = useState("");
  const [tenantID, setTenantID] = useState("");
  const [method, setMethod] = useState<"email" | "whatsapp">("email");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError(null);
    setSuccess(null);
    const trimmedTenantID = tenantID.trim();
    if (!trimmedTenantID) {
      setError(t("auth.tenantRequired"));
      return;
    }
    setLoading(true);
    try {
      const result = await requestPasswordReset({
        email,
        tenant_id: trimmedTenantID,
        method
      });
      setSuccess(result.message || t("auth.forgotSuccess"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="auth-wrap">
      <div className="lang-switcher-wrap">
        <LanguageSwitcher />
      </div>
      <div className="auth-card">
        <p className="auth-kicker">{t("auth.kicker")}</p>
        <h1 className="auth-title">{t("auth.forgotTitle")}</h1>
        <p className="page-subtitle" style={{ fontSize: '0.875rem', opacity: 0.8, marginBottom: '0.5rem' }}>{t("auth.forgotSubtitle")}</p>
        <p className="auth-helper" style={{ fontSize: '0.75rem', opacity: 0.7, marginBottom: '1.5rem' }}>{t("auth.forgotHint")}</p>
        <form className="form-grid" onSubmit={handleSubmit}>
          <label className="form-field">
            {t("auth.email")}
            <input className="input-control" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="username email" />
          </label>
          <label className="form-field">
            {t("auth.tenantId")}
            <input className="input-control" value={tenantID} onChange={(e) => setTenantID(e.target.value)} required autoComplete="off" />
          </label>
          <div className="form-field">
            <span style={{ fontSize: '0.875rem', marginBottom: '0.5rem', display: 'block' }}>{t("auth.sendMethod")}</span>
            <div style={{ display: 'flex', gap: '1rem' }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', fontSize: '0.875rem' }}>
                <input type="radio" checked={method === 'email'} onChange={() => setMethod('email')} />
                {t("settings.channel.email")}
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', fontSize: '0.875rem' }}>
                <input type="radio" checked={method === 'whatsapp'} onChange={() => setMethod('whatsapp')} />
                {t("settings.channel.whatsapp")}
              </label>
            </div>
          </div>
          {error ? <p className="alert error">{error}</p> : null}
          {success ? <p className="alert success">{success}</p> : null}
          <button className="btn btn-primary" type="submit" disabled={loading}>
            {loading ? t("auth.forgotSubmitting") : t("auth.forgotSubmit")}
          </button>
        </form>
        <div className="auth-actions" style={{ display: 'flex', justifyContent: 'space-between', gap: '1.25rem', marginTop: '2rem', alignItems: 'center' }}>
          <Link className="btn-link" style={{ fontSize: '0.875rem' }} to="/forgot-tenant">
            {t("auth.forgotTenantLink")}
          </Link>
          <Link className="btn-link" style={{ fontSize: '0.875rem' }} to="/login">
            {t("auth.backToLogin")}
          </Link>
        </div>
      </div>
    </section>
  );
}
