import { FormEvent, useState } from "react";
import { LanguageSwitcher } from "../../../../core/components/LanguageSwitcher";
import { Link, useSearchParams, useNavigate } from "react-router-dom";

import { apiFetch } from "../../../../core/api/client";
import { useI18n } from "../../../../core/i18n/i18n";

export function ResetPasswordPage(): JSX.Element {
  const { t } = useI18n();
  const [searchParams] = useSearchParams();
  const tenantFromUrl = searchParams.get("t") || "";
  const emailFromUrl = searchParams.get("e") || "";
  const navigate = useNavigate();

  const [email, setEmail] = useState(emailFromUrl);
  const [tenantID, setTenantID] = useState(tenantFromUrl);
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError(null);
    setSuccess(null);

    if (password !== confirmPassword) {
      setError(t("auth.passwordMismatch"));
      return;
    }

    setLoading(true);
    try {
      await apiFetch("/auth/reset-password", {
        method: "POST",
        body: JSON.stringify({
          email,
          tenant_id: tenantID,
          new_password: password
        })
      });
      setSuccess(t("auth.resetSuccess"));
      setTimeout(() => {
        navigate("/login");
      }, 5000);
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
        <h1 className="auth-title">{t("auth.resetTitle")}</h1>
        <p className="page-subtitle" style={{ fontSize: '0.875rem', opacity: 0.8, marginBottom: '1.5rem' }}>
          {t("auth.resetSubtitle")}
        </p>

        <form className="form-grid" onSubmit={handleSubmit}>
          <label className="form-field">
            {t("auth.email")}
            <input 
              className="input-control" 
              type="email" 
              value={email} 
              onChange={(e) => setEmail(e.target.value)} 
              required 
              autoComplete="username email"
            />
          </label>
          <label className="form-field">
            {t("auth.tenantId")}
            <input 
              className="input-control" 
              value={tenantID} 
              onChange={(e) => setTenantID(e.target.value)} 
              required 
              disabled={!!tenantFromUrl}
              autoComplete="off"
            />
          </label>
          <label className="form-field">
            {t("auth.password")}
            <input 
              className="input-control" 
              type="password" 
              value={password} 
              onChange={(e) => setPassword(e.target.value)} 
              required 
              autoComplete="new-password"
            />
          </label>
          <label className="form-field">
            {t("auth.confirmPassword")}
            <input 
              className="input-control" 
              type="password" 
              value={confirmPassword} 
              onChange={(e) => setConfirmPassword(e.target.value)} 
              required 
              autoComplete="new-password"
            />
          </label>

          {error ? <p className="alert error">{error}</p> : null}
          {success ? <p className="alert success">{success}</p> : null}

          <button className="btn btn-primary" type="submit" disabled={loading || !!success}>
            {loading ? t("auth.resetSubmitting") : t("auth.resetSubmit")}
          </button>
        </form>

        <div className="auth-actions auth-actions-spaced" style={{ marginTop: '1.5rem', justifyContent: 'center' }}>
          <Link className="btn-link" style={{ fontSize: '0.875rem' }} to="/login">
            {t("auth.backToLogin")}
          </Link>
        </div>
      </div>
    </section>
  );
}
