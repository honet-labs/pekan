import { FormEvent, useState } from "react";
import { LanguageSwitcher } from "../../../../core/components/LanguageSwitcher";
import { Link } from "react-router-dom";
import { requestForgotTenant } from "../../../../core/auth/auth-api";
import { useI18n } from "../../../../core/i18n/i18n";

export function ForgotTenantPage(): JSX.Element {
  const { t } = useI18n();
  const [emailOrPhone, setEmailOrPhone] = useState("");
  const [method, setMethod] = useState<"email" | "whatsapp">("email");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError(null);
    setSuccess(null);
    
    if (!emailOrPhone.trim()) {
      setError(t("auth.tenantRequired"));
      return;
    }

    setLoading(true);
    try {
      const result = await requestForgotTenant({
        email_or_phone: emailOrPhone.trim(),
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
        <h1 className="auth-title">{t("auth.forgotTenantTitle")}</h1>
        <p className="page-subtitle" style={{ fontSize: '0.875rem', opacity: 0.8, marginBottom: '0.5rem' }}>
          {t("auth.forgotTenantSubtitle")}
        </p>
        
        <form className="form-grid" onSubmit={handleSubmit}>
          <label className="form-field">
            {t("auth.forgotTenantLabel")}
            <input 
              className="input-control" 
              placeholder={t("auth.forgotTenantPlaceholder")}
              value={emailOrPhone} 
              onChange={(e) => setEmailOrPhone(e.target.value)} 
              required 
            />
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
            {loading ? t("auth.sendingInfo") : t("auth.sendInfo")}
          </button>
        </form>

        <div className="auth-actions auth-actions-spaced" style={{ marginTop: '2rem', textAlign: 'right' }}>
          <Link className="btn-link" style={{ fontSize: '0.875rem' }} to="/login">
            {t("auth.backToLogin")}
          </Link>
        </div>
      </div>
    </section>
  );
}
