import { PasswordInput } from "../../../../core/components/PasswordInput";
import { LanguageSwitcher } from "../../../../core/components/LanguageSwitcher";
import { FormEvent, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { useAccessStore } from "../../../../core/access/access-store";
import { login } from "../../../../core/auth/auth-api";
import { useAuthStore } from "../../../../core/auth/auth-store";
import { useTenantStore } from "../../../../core/tenant/tenant-store";
import { useI18n } from "../../../../core/i18n/i18n";
import { registerInit, registerVerify } from "../api/register.api";
import "../styles/auth-modern.css";

type Mode = "login" | "register-form" | "register-otp" | "register-done" | "change-password";

function generateCode(name: string): string {
  return name
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, "")
    .slice(0, 10) || "WORKSPACE";
}

export function LoginPage(): JSX.Element {
  const navigate = useNavigate();
  const auth = useAuthStore();
  const access = useAccessStore();
  const tenant = useTenantStore();
  const { t } = useI18n();

  // --- Login state ---
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [tenantID, setTenantID] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  // --- Registration state ---
  const [mode, setMode] = useState<Mode>("login");
  const [regTenantCode, setRegTenantCode] = useState("");
  const [regTenantName, setRegTenantName] = useState("");
  const [regAdminName, setRegAdminName] = useState("");
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [regPhone, setRegPhone] = useState("");
  const [regOTPMethod, setRegOTPMethod] = useState<"email" | "whatsapp">("email");
  const [regError, setRegError] = useState<string | null>(null);
  const [regLoading, setRegLoading] = useState(false);

  // --- OTP state ---
  const [sessionToken, setSessionToken] = useState("");
  const [otpDigits, setOtpDigits] = useState(["", "", "", "", "", ""]);
  const [otpError, setOtpError] = useState<string | null>(null);
  const [otpLoading, setOtpLoading] = useState(false);
  const [countdown, setCountdown] = useState(600); // 10 minutes
  const otpRefs = useRef<(HTMLInputElement | null)[]>([]);

  // Countdown timer for OTP
  useEffect(() => {
    if (mode !== "register-otp") return;
    setCountdown(600);
    const timer = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) { clearInterval(timer); return 0; }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [mode, sessionToken]);

  // Auto-generate tenant code from name
  useEffect(() => {
    if (regTenantName && !regTenantCode) {
      setRegTenantCode(generateCode(regTenantName));
    }
  }, [regTenantName]);

  // --- Login submit ---
  async function handleLogin(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const trimmedTenantID = tenantID.trim();
      if (!trimmedTenantID) throw new Error(t("auth.tenantRequired"));
      const result = await login({ email: email.trim(), password, tenant_id: trimmedTenantID });
      auth.setTokens(result.access_token, result.refresh_token);
      
      if (result.must_change_password) {
        setMode("change-password");
        setLoading(false);
        return;
      }

      // Fetch context to get memberships
      const { getMeContext } = await import("../../../../core/auth/auth-api");
      const ctx = await getMeContext();
      
      auth.setAuth(ctx.user.id);
      access.setAccess({ modules: ctx.modules, features: ctx.features, permissions: ctx.permissions });
      
      if (ctx.memberships && Array.isArray(ctx.memberships)) {
        tenant.setAllowedTenants(ctx.memberships.map(m => ({ id: m.tenant_id, code: m.tenant_code || m.tenant_id })));
      }
      
      tenant.setTenant(ctx.active_tenant.id, ctx.active_tenant.code || ctx.active_tenant.id);
      navigate(`/app/${ctx.active_tenant.code || ctx.active_tenant.id}/finance/dashboard`, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.loginFailed"));
    } finally {
      setLoading(false);
    }
  }

  // --- Register Step 1 submit ---
  async function handleRegisterInit(e: FormEvent) {
    e.preventDefault();
    setRegError(null);
    setRegLoading(true);
    try {
      const result = await registerInit({
        tenant_code: regTenantCode.toUpperCase().trim(),
        tenant_name: regTenantName.trim(),
        admin_email: regEmail.trim(),
        admin_name: regAdminName.trim(),
        password: regPassword,
        phone: regOTPMethod === "whatsapp" ? regPhone.trim() : undefined,
        otp_method: regOTPMethod,
      });
      setSessionToken(result.session_token);
      setOtpDigits(["", "", "", "", "", ""]);
      setMode("register-otp");
      setTimeout(() => otpRefs.current[0]?.focus(), 100);
    } catch (err) {
      setRegError(err instanceof Error ? err.message : "Gagal memulai registrasi");
    } finally {
      setRegLoading(false);
    }
  }

  // --- OTP digit input handler ---
  function handleOtpInput(idx: number, val: string) {
    if (!/^\d*$/.test(val)) return;
    const next = [...otpDigits];
    next[idx] = val.slice(-1);
    setOtpDigits(next);
    if (val && idx < 5) otpRefs.current[idx + 1]?.focus();
    // Auto-submit if all filled
    if (next.every(d => d !== "") && next.join("").length === 6) {
      handleOtpVerify(next.join(""));
    }
  }

  function handleOtpKeyDown(idx: number, e: React.KeyboardEvent) {
    if (e.key === "Backspace" && !otpDigits[idx] && idx > 0) {
      otpRefs.current[idx - 1]?.focus();
    }
  }

  // --- OTP verify ---
  async function handleOtpVerify(code?: string) {
    const finalCode = code || otpDigits.join("");
    if (finalCode.length !== 6) { setOtpError("Masukkan 6 digit kode OTP"); return; }
    setOtpError(null);
    setOtpLoading(true);
    try {
      await registerVerify(sessionToken, finalCode);
      setMode("register-done");
    } catch (err) {
      setOtpError(err instanceof Error ? err.message : "Kode OTP tidak valid");
      setOtpDigits(["", "", "", "", "", ""]);
      setTimeout(() => otpRefs.current[0]?.focus(), 100);
    } finally {
      setOtpLoading(false);
    }
  }

  const formatCountdown = (s: number) => `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;

  async function handleForceChangePassword(e: FormEvent) {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setError("Konfirmasi password tidak cocok");
      return;
    }
    if (newPassword.length < 8) {
      setError("Password minimal 8 karakter");
      return;
    }
    setLoading(true);
    try {
      // In this mode, auth tokens are already set in authStore (from handleLogin)
      // and used by apiFetch automatically.
      const { changePassword } = await import("../../../../core/auth/auth-api");
      await changePassword({ new_password: newPassword });
      
      // Password changed! Now fetch context and go to dashboard
      const { getMeContext } = await import("../../../../core/auth/auth-api");
      const ctx = await getMeContext();
      
      auth.setAuth(ctx.user.id);
      access.setAccess({ modules: ctx.modules, features: ctx.features, permissions: ctx.permissions });
      
      if (ctx.memberships && Array.isArray(ctx.memberships)) {
        tenant.setAllowedTenants(ctx.memberships.map(m => ({ id: m.tenant_id, code: m.tenant_code || m.tenant_id })));
      }
      
      tenant.setTenant(ctx.active_tenant.id, ctx.active_tenant.code || ctx.active_tenant.id);
      
      navigate(`/app/${ctx.active_tenant.code || ctx.active_tenant.id}/finance/dashboard`, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal mengganti password");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="auth-wrap">
      <div className="lang-switcher-wrap">
        <LanguageSwitcher />
      </div>
      {/* ============ LOGIN ============ */}
      {mode === "login" && (
        <div className="auth-card">
          <p className="auth-kicker">{t("auth.kicker")}</p>
          <h1 className="auth-title">{t("auth.title")}</h1>
          <p className="page-subtitle" style={{ fontSize: "0.85rem", opacity: 0.8, marginBottom: "1.5rem" }}>{t("auth.subtitle")}</p>
          <form className="form-grid" onSubmit={handleLogin}>
            <label className="form-field">
              {t("auth.email")}
              <input className="input-control" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="username email" />
            </label>
            <label className="form-field">
              {t("auth.password")}
              <PasswordInput value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" />
            </label>
            <label className="form-field">
              {t("auth.tenantId")}
              <input className="input-control" value={tenantID} onChange={(e) => setTenantID(e.target.value)} required autoComplete="off" />
            </label>
            <div className="auth-actions" style={{ display: 'flex', justifyContent: 'space-between', gap: '1.25rem', marginTop: '1.75rem', marginBottom: '0.5rem' }}>
              <Link className="btn-link" style={{ fontSize: '0.75rem' }} to="/forgot-tenant">{t("auth.forgotTenantLink")}</Link>
              <Link className="btn-link" style={{ fontSize: '0.75rem' }} to="/forgot-password">{t("auth.forgotPassword")}</Link>
            </div>
            {error ? <p className="alert error">{error}</p> : null}
            <button className="btn btn-primary" type="submit" disabled={loading}>
              {loading ? t("auth.signingIn") : t("auth.signIn")}
            </button>
          </form>
          <div style={{ marginTop: "1.25rem", textAlign: "center", borderTop: "1px solid var(--border)", paddingTop: "1.25rem" }}>
            <p style={{ fontSize: "0.875rem", color: "var(--text-muted)", marginBottom: "0.75rem" }}>
              {t("auth.noAccount")}
            </p>
            <button
              type="button"
              className="btn btn-secondary-outline"
              style={{ width: "100%" }}
              onClick={() => { setMode("register-form"); setRegError(null); }}
            >
              {t("auth.registerNow")}
            </button>
          </div>
        </div>
      )}

      {/* ============ REGISTER STEP 1: FORM ============ */}
      {mode === "register-form" && (
        <div className="auth-modern-card">
          <button type="button" className="btn-link" style={{ marginBottom: "1.5rem" }} onClick={() => setMode("login")}>
            ← {t("auth.registerBack")}
          </button>
          
          <div style={{ marginBottom: "2rem" }}>
            <h1 className="auth-title" style={{ fontSize: "1.75rem", marginBottom: "0.5rem" }}>{t("auth.registerTitle")}</h1>
            <p className="page-subtitle">{t("auth.registerSubtitle")}</p>
          </div>

          <form onSubmit={handleRegisterInit}>
            <div className="auth-grid-2">
              <div className="auth-input-group">
                <label className="auth-input-label">{t("auth.workspaceName")}</label>
                <input
                  className="input-control"
                  placeholder="Pekan DEMO"
                  value={regTenantName}
                  onChange={e => { setRegTenantName(e.target.value); setRegTenantCode(generateCode(e.target.value)); }}
                  required
                />
              </div>
              <div className="auth-input-group">
                <label className="auth-input-label">{t("auth.workspaceCode")}</label>
                <input
                  className="input-control"
                  placeholder="PEKANDEMO"
                  value={regTenantCode}
                  onChange={e => setRegTenantCode(e.target.value.toUpperCase().replace(/[^A-Z0-9_-]/g, ""))}
                  maxLength={20}
                  required
                />
                <span className="auth-input-hint">{t("auth.workspaceCodeHint")}</span>
              </div>
            </div>

            <div className="auth-input-group">
              <label className="auth-input-label">{t("auth.adminFullName")}</label>
              <input className="input-control" placeholder="John Doe" value={regAdminName} onChange={e => setRegAdminName(e.target.value)} required />
            </div>

            <div className="auth-input-group">
              <label className="auth-input-label">{t("auth.adminEmail")}</label>
              <input className="input-control" type="email" placeholder="admin@example.com" value={regEmail} onChange={e => setRegEmail(e.target.value)} required />
            </div>

            <div className="auth-input-group">
              <label className="auth-input-label">{t("auth.password")}</label>
              <PasswordInput placeholder="Min. 8 characters" value={regPassword} onChange={e => setRegPassword(e.target.value)} required />
            </div>

            <div className="auth-input-group">
              <label className="auth-input-label">
                {t("auth.phoneWA")}
              </label>
              <input
                className="input-control"
                placeholder="628123456789"
                value={regPhone}
                onChange={e => setRegPhone(e.target.value.replace(/\D/g, ""))}
                required={regOTPMethod === "whatsapp"}
              />
              <span className="auth-input-hint">{t("auth.phoneWAHint")}</span>
            </div>

            <div className="auth-input-group">
              <label className="auth-input-label">{t("auth.otpMethod")}</label>
              <div className="otp-method-selector">
                <label className={`otp-method-card ${regOTPMethod === "email" ? "is-active" : ""}`}>
                  <input type="radio" name="otp_method" value="email" checked={regOTPMethod === "email"} onChange={() => setRegOTPMethod("email")} />
                  <span className="otp-method-icon">📧</span>
                  <span className="otp-method-name">Email</span>
                </label>
                <label className={`otp-method-card ${regOTPMethod === "whatsapp" ? "is-active" : ""}`}>
                  <input type="radio" name="otp_method" value="whatsapp" checked={regOTPMethod === "whatsapp"} onChange={() => setRegOTPMethod("whatsapp")} />
                  <span className="otp-method-icon">💬</span>
                  <span className="otp-method-name">WhatsApp</span>
                </label>
              </div>
            </div>

            {regError && (
              <div className="auth-alert auth-alert-error" style={{ marginTop: "1.5rem" }}>
                <span>⚠️</span>
                <span>{regError}</span>
              </div>
            )}

            <button className="btn btn-primary" type="submit" disabled={regLoading} style={{ width: "100%", marginTop: "1.5rem", height: "52px", fontSize: "1rem" }}>
              {regLoading ? t("auth.sendingOTP") : t("auth.sendOTP") + " →"}
            </button>
          </form>
        </div>
      )}

      {/* ============ REGISTER STEP 2: OTP ============ */}
      {mode === "register-otp" && (
        <div className="auth-modern-card" style={{ textAlign: "center" }}>
          <div style={{ fontSize: "3.5rem", marginBottom: "1rem" }}>
            {regOTPMethod === "whatsapp" ? "💬" : "📧"}
          </div>
          <h1 className="auth-title" style={{ fontSize: "1.75rem", marginBottom: "0.5rem" }}>{t("auth.otpTitle")}</h1>
          <p className="page-subtitle" style={{ marginBottom: "0.5rem" }}>
            {t("auth.otpSubtitle")} <br/> 
            <strong style={{ color: "var(--primary)", fontSize: "1.1rem" }}>{regOTPMethod === "whatsapp" ? regPhone : regEmail}</strong>
          </p>
          <p style={{ fontSize: "0.875rem", color: countdown > 60 ? "var(--muted)" : "var(--danger)", marginBottom: "2rem" }}>
            {t("auth.otpExpiry")} <strong>{formatCountdown(countdown)}</strong>
          </p>

          {/* OTP digit boxes */}
          <div style={{ display: "flex", gap: "12px", justifyContent: "center", marginBottom: "2rem" }}>
            {otpDigits.map((digit, idx) => (
              <input
                key={idx}
                ref={el => { otpRefs.current[idx] = el; }}
                type="text"
                inputMode="numeric"
                maxLength={1}
                value={digit}
                onChange={e => handleOtpInput(idx, e.target.value)}
                onKeyDown={e => handleOtpKeyDown(idx, e)}
                style={{
                  width: "52px", height: "64px",
                  textAlign: "center", fontSize: "1.75rem", fontWeight: 700,
                  border: `2px solid ${digit ? "var(--primary)" : "var(--border)"}`,
                  borderRadius: "12px", background: "var(--surface)",
                  color: "var(--text)", outline: "none",
                  transition: "all 0.2s",
                  boxShadow: digit ? "0 4px 12px rgba(15, 118, 110, 0.15)" : "none"
                }}
              />
            ))}
          </div>

          {otpError && (
            <div className="auth-alert auth-alert-error" style={{ marginBottom: "1.5rem" }}>
              <span>⚠️</span>
              <span>{otpError}</span>
            </div>
          )}

          <button
            className="btn btn-primary"
            style={{ width: "100%", height: "52px", fontSize: "1rem" }}
            disabled={otpLoading || countdown === 0 || otpDigits.some(d => !d)}
            onClick={() => handleOtpVerify()}
          >
            {otpLoading ? t("auth.otpVerifying") : t("auth.otpVerify")}
          </button>

          {countdown === 0 && (
            <div style={{ marginTop: "1.5rem", fontSize: "0.875rem", color: "var(--danger)" }}>
              {t("auth.otpExpired")}{" "}
              <button type="button" className="btn-link" style={{ fontWeight: 700 }} onClick={() => setMode("register-form")}>
                {t("auth.otpRetry")}
              </button>
            </div>
          )}

          <button
            type="button"
            className="btn-link"
            style={{ marginTop: "2rem", fontSize: "0.9rem", display: "inline-block" }}
            onClick={() => setMode("register-form")}
          >
            ← {t("auth.otpChangeData")}
          </button>
        </div>
      )}

      {/* ============ REGISTER DONE ============ */}
      {mode === "register-done" && (
        <div className="auth-modern-card" style={{ textAlign: "center" }}>
          <div style={{ fontSize: "5rem", marginBottom: "1.5rem" }}>🎉</div>
          <h1 className="auth-title" style={{ fontSize: "2rem", marginBottom: "0.5rem" }}>{t("auth.regDoneSubtitle")}</h1>
          <p className="page-subtitle" style={{ marginBottom: "1.5rem" }}>
            {t("auth.regDoneDescPrefix")} <strong style={{ color: "var(--text)" }}>{regTenantName}</strong> {t("auth.regDoneDescSuffix")}
          </p>
          
          <div style={{ background: "rgba(15, 118, 110, 0.04)", borderRadius: "16px", padding: "1.5rem", margin: "2rem 0", border: "1px dashed var(--primary)" }}>
            <p style={{ fontSize: "0.875rem", color: "var(--muted)", marginBottom: "0.5rem" }}>{t("auth.regDoneHint")}</p>
            <code style={{ fontSize: "1.75rem", fontWeight: 800, color: "var(--primary)", letterSpacing: "0.15em" }}>
              {regTenantCode}
            </code>
          </div>

          <p style={{ fontSize: "0.9rem", color: "var(--muted)", marginBottom: "2rem", lineHeight: 1.6 }}>
            {t("auth.regDoneLoginHint")}
          </p>

          <button
            className="btn btn-primary"
            style={{ width: "100%", height: "56px", fontSize: "1.1rem", fontWeight: 700 }}
            onClick={() => {
              setTenantID(regTenantCode);
              setEmail(regEmail);
              setPassword("");
              setMode("login");
            }}
          >
            {t("auth.regDoneLogin")} →
          </button>
        </div>
      )}

      {/* ============ CHANGE PASSWORD (MANDATORY) ============ */}
      {mode === "change-password" && (
        <div className="auth-card">
          <div style={{ fontSize: "3.5rem", marginBottom: "1rem", textAlign: "center" }}>🔐</div>
          <h1 className="auth-title" style={{ textAlign: "center" }}>{t("auth.changePasswordTitle")}</h1>
          <p className="page-subtitle" style={{ fontSize: "0.85rem", opacity: 0.8, marginBottom: "1.5rem", textAlign: "center" }}>
            {t("auth.mustChangePasswordDesc")}
          </p>
          <form className="form-grid" onSubmit={handleForceChangePassword}>
            <label className="form-field">
              {t("auth.newPassword")}
              <PasswordInput 
                value={newPassword} 
                onChange={(e) => setNewPassword(e.target.value)} 
                required 
                autoComplete="new-password" 
                placeholder="Minimal 8 karakter"
              />
            </label>
            <label className="form-field">
              {t("auth.confirmPassword")}
              <PasswordInput 
                value={confirmPassword} 
                onChange={(e) => setConfirmPassword(e.target.value)} 
                required 
                autoComplete="new-password"
                placeholder="Ulangi password baru"
              />
            </label>
            
            {error ? <p className="alert error">{error}</p> : null}
            
            <button className="btn btn-primary" type="submit" disabled={loading} style={{ marginTop: "1rem" }}>
              {loading ? t("common.processing") : t("auth.saveAndContinue")}
            </button>
            
            <button 
              type="button" 
              className="btn-link" 
              style={{ marginTop: "1rem", fontSize: "0.85rem", textAlign: "center", width: "100%" }}
              onClick={() => {
                auth.setTokens("", "");
                setMode("login");
                setNewPassword("");
                setConfirmPassword("");
                setError(null);
              }}
            >
              ← {t("auth.backToLogin")}
            </button>
          </form>
        </div>
      )}
    </section>
  );
}
