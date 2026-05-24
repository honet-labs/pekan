import { useEffect, useMemo, useState } from "react";
import { NavLink, Outlet, useNavigate, useParams } from "react-router-dom";
import { useAccessStore } from "../../core/access/access-store";
import { logout, getMeProfile } from "../../core/auth/auth-api";
import { useAuthStore } from "../../core/auth/auth-store";
import { useI18n } from "../../core/i18n/i18n";
import { useBrandingStore } from "../../core/branding/branding-store";

export function AppShell(): JSX.Element {
  const access = useAccessStore();
  const auth = useAuthStore();
  const navigate = useNavigate();
  const branding = useBrandingStore();
  const { tenantCode } = useParams();
  const { t, locale, setLocale } = useI18n();
  const { modules, permissions } = access;
  const [profileName, setProfileName] = useState(t("common.user"));
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [profileMenuOpen, setProfileMenuOpen] = useState(false);
  const [settingsExpanded, setSettingsExpanded] = useState(localStorage.getItem("pekan_settings_expanded") === "true");
  const [showBackToTop, setShowBackToTop] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setShowBackToTop(window.scrollY > 300);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const toggleSettings = () => {
    const newVal = !settingsExpanded;
    setSettingsExpanded(newVal);
    localStorage.setItem("pekan_settings_expanded", String(newVal));
  };

  const canManageCategories = modules.has("finance.masterdata") && (permissions.has("finance.categories.read") || permissions.has("finance.categories.create"));
  const canOpenSettings =
    modules.has("finance.settings") &&
    (permissions.has("finance.settings.read") || permissions.has("finance.settings.roles.manage") || permissions.has("finance.settings.audit.read")) ||
    canManageCategories;

  const defaultSettingsPath = useMemo(() => {
    if (permissions.has("finance.settings.read")) {
      return "finance/settings/notifications";
    }
    if (canManageCategories) {
      return "finance/settings/categories";
    }
    if (permissions.has("finance.settings.roles.manage")) {
      return "finance/settings/users";
    }
    return "finance/logs";
  }, [canManageCategories, permissions]);

  useEffect(() => {
    getMeProfile()
      .then((profile) => setProfileName(profile.full_name || profile.username || t("common.user")))
      .catch(() => setProfileName(t("common.user")));
  }, []);


  useEffect(() => {
    setProfileName((prev: string) => (prev === "User" || prev === "Pengguna" || !prev ? t("common.user") : prev));
  }, [t]);
  async function handleLogout(): Promise<void> {
    try {
      if (auth.refreshToken) {
        await logout(auth.refreshToken);
      }
    } finally {
      auth.clear();
      access.clearAccess();
      navigate("/login", { replace: true });
    }
  }

  // Automatic 10-minute inactivity session timeout
  useEffect(() => {
    let timeoutId: number | undefined;

    const handleTimeoutLogout = async () => {
      try {
        if (auth.refreshToken) {
          await logout(auth.refreshToken);
        }
      } catch (err) {
        console.error("Inactivity logout error:", err);
      } finally {
        auth.clear();
        access.clearAccess();
        navigate("/login", { replace: true });
      }
    };

    const resetTimer = () => {
      if (timeoutId) {
        window.clearTimeout(timeoutId);
      }
      timeoutId = window.setTimeout(() => {
        handleTimeoutLogout().catch(() => undefined);
      }, 10 * 60 * 1000); // 10 minutes
    };

    const events = ["mousemove", "mousedown", "keypress", "scroll", "touchstart"];

    // Initialize timer
    resetTimer();

    // Attach listeners
    events.forEach((event) => {
      window.addEventListener(event, resetTimer);
    });

    return () => {
      if (timeoutId) {
        window.clearTimeout(timeoutId);
      }
      events.forEach((event) => {
        window.removeEventListener(event, resetTimer);
      });
    };
  }, [auth.refreshToken, navigate]);

  return (
    <div className={`app-shell${sidebarOpen ? " sidebar-open" : ""}`}>
      <aside className="app-sidebar">
        <div className="sidebar-header" style={{ marginBottom: "2.5rem", paddingBottom: "1.2rem", borderBottom: "1px solid rgba(255,255,255,0.06)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
            {branding.logo ? (
              <img src={branding.logo} alt="Logo" style={{ width: "32px", height: "32px", borderRadius: "8px", objectFit: "contain", flexShrink: 0 }} />
            ) : (
              <svg width="32" height="32" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" style={{ flexShrink: 0 }}>
                <rect width="32" height="32" rx="8" fill="url(#logo_grad)" />
                <path d="M12 6H21C22.1046 6 23 6.89543 23 8V19C23 19.5523 22.5523 20 22 20H12C11.4477 20 11 19.5523 11 19V7C11 6.44772 11.4477 6 12 6Z" fill="#F8FAFC" />
                <line x1="14" y1="9" x2="20" y2="9" stroke="#0D9488" strokeWidth="1.5" strokeLinecap="round" />
                <line x1="14" y1="12" x2="20" y2="12" stroke="#0D9488" strokeWidth="1.5" strokeLinecap="round" />
                <line x1="14" y1="15" x2="18" y2="15" stroke="#D97706" strokeWidth="1.5" strokeLinecap="round" />
                <path d="M8 12H20C21.1046 12 22 12.8954 22 14V23C22 24.1046 21.1046 25 20 25H8C6.89543 25 6 24.1046 6 23V14C6 12.8954 6.89543 12 8 12Z" fill="#0F766E" stroke="#0D9488" strokeWidth="1" />
                <path d="M18 15.5H22V21.5H18C16.3431 21.5 15 20.1569 15 18.5C15 16.8431 16.3431 15.5 18 15.5Z" fill="#11395F" />
                <circle cx="18" cy="18.5" r="1.75" fill="#D97706" />
                <defs>
                  <linearGradient id="logo_grad" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#0f766e" />
                    <stop offset="1" stopColor="#0d9488" />
                  </linearGradient>
                </defs>
              </svg>
            )}
            <div style={{ display: "flex", flexDirection: "column", justifyContent: "center" }}>
              <h2 className="brand" style={{ margin: 0, fontSize: "1.35rem", fontWeight: 700, letterSpacing: "0.5px", lineHeight: 1.2 }}>{branding.app_name || t("app.brand")}</h2>
              <p className="sidebar-caption" style={{ margin: 0, fontSize: "0.68rem", opacity: 0.8, lineHeight: 1.2, marginTop: "2px" }}>{branding.page_title || t("app.subtitle")}</p>
            </div>
          </div>
          <button type="button" className="btn btn-ghost-inline sidebar-close-btn" onClick={() => setSidebarOpen(false)} aria-label={t("common.closeMenu")}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <nav className="app-nav">
          {modules.has("finance.dashboard") && permissions.has("finance.dashboard.read") ? (
            <NavLink to="finance/dashboard" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.dashboard")}
            </NavLink>
          ) : null}
          {modules.has("finance") ? (
            <NavLink to="finance/transactions" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.transactions")}
            </NavLink>
          ) : null}
          {modules.has("finance") && permissions.has("finance.transactions.create") ? (
            <NavLink to="finance/receipt-scan" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.receiptScan")}
            </NavLink>
          ) : null}
          {modules.has("finance.savings") && permissions.has("finance.savings.read") ? (
            <NavLink to="finance/savings" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.savings")}
            </NavLink>
          ) : null}
          {modules.has("finance.budgets") && permissions.has("finance.budgets.read") ? (
            <NavLink to="finance/budgets" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.budgets")}
            </NavLink>
          ) : null}
          {modules.has("finance.reminders") && permissions.has("finance.reminders.read") ? (
            <NavLink to="finance/reminders" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.reminders")}
            </NavLink>
          ) : null}
          {modules.has("finance.reports") && permissions.has("finance.reports.read") ? (
            <NavLink to="finance/reports" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.reports")}
            </NavLink>
          ) : null}
          {modules.has("finance.dashboard") && permissions.has("finance.dashboard.read") ? (
            <NavLink to="finance/chatbot" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.chatbot")}
            </NavLink>
          ) : null}
          {canOpenSettings ? (
            <>
              <button 
                type="button" 
                className={`app-nav-link sidebar-expand-btn ${settingsExpanded ? "is-expanded" : ""}`} 
                onClick={toggleSettings}
                style={{ width: "100%", textAlign: "left", display: "flex", alignItems: "center", justifyContent: "space-between" }}
              >
                <span>{t("nav.settings")}</span>
                <svg 
                  viewBox="0 0 24 24" 
                  width="18" height="18" 
                  fill="currentColor" 
                  style={{ transform: settingsExpanded ? "rotate(180deg)" : "none", transition: "transform 0.2s", opacity: 0.7 }}
                >
                  <path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z" />
                </svg>
              </button>
              
              {settingsExpanded && (
                <div className="sidebar-sub-nav">
                  {permissions.has("finance.settings.read") ? (
                    <NavLink to={`finance/settings/notifications`} className={({ isActive }) => `app-nav-link sub-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
                      {t("settings.nav.notifications")}
                    </NavLink>
                  ) : null}
                  {permissions.has("finance.settings.read") ? (
                    <NavLink to={`finance/settings/templates`} className={({ isActive }) => `app-nav-link sub-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
                      {t("settings.nav.templates")}
                    </NavLink>
                  ) : null}
                  {permissions.has("finance.settings.read") ? (
                    <NavLink to={`finance/settings/whatsapp`} className={({ isActive }) => `app-nav-link sub-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
                      {t("settings.nav.whatsapp")}
                    </NavLink>
                  ) : null}
                  {canManageCategories ? (
                    <NavLink to={`finance/settings/categories`} className={({ isActive }) => `app-nav-link sub-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
                      {t("settings.nav.categories")}
                    </NavLink>
                  ) : null}
                  {permissions.has("finance.settings.roles.manage") ? (
                    <NavLink to={`finance/settings/users`} className={({ isActive }) => `app-nav-link sub-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
                      {t("settings.nav.users")}
                    </NavLink>
                  ) : null}
                </div>
              )}
            </>
          ) : null}
          {modules.has("finance.settings") && permissions.has("finance.settings.audit.read") ? (
            <NavLink to="finance/logs" className={({ isActive }) => `app-nav-link${isActive ? " is-active" : ""}`} onClick={() => setSidebarOpen(false)}>
              {t("nav.logging")}
            </NavLink>
          ) : null}
        </nav>

        <div className="sidebar-footer">
          <div className="sidebar-user" onClick={() => setProfileMenuOpen(!profileMenuOpen)} style={{ cursor: "pointer" }}>
            <div className="sidebar-user-avatar">
              <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                <path d="M12 12a4 4 0 1 0-4-4 4 4 0 0 0 4 4Zm0 2c-4 0-7 2-7 4.5V20h14v-1.5C19 16 16 14 12 14Z" />
              </svg>
            </div>
            <div className="sidebar-user-info">
              <span className="sidebar-user-name">{profileName}</span>
              <span className="sidebar-user-role">{t("common.owner")}</span>
            </div>
            <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" style={{ transform: profileMenuOpen ? "rotate(180deg)" : "none", transition: "transform 0.2s" }}>
              <path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z" />
            </svg>
          </div>

          {profileMenuOpen && (
            <div className="sidebar-profile-menu">
              <NavLink to="profile" className="app-nav-link sub-link" onClick={() => { setProfileMenuOpen(false); setSidebarOpen(false); }}>
                {t("nav.profile")}
              </NavLink>
              <button className="app-nav-link sub-link danger" onClick={() => handleLogout().catch(() => undefined)} style={{ background: "none", border: "none", width: "100%", textAlign: "left", cursor: "pointer", color: "#ff8a8a" }}>
                {t("nav.logout")}
              </button>
            </div>
          )}

          <div className="sidebar-footer-actions">
            <button className="sidebar-footer-btn" onClick={() => setLocale(locale === "id" ? "en" : "id")}>
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" style={{ marginRight: "6px" }}>
                <circle cx="12" cy="12" r="10" />
                <line x1="2" y1="12" x2="22" y2="12" />
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
              </svg>
              {locale.toUpperCase()}
            </button>
          </div>
        </div>
      </aside>

      <button type="button" className={`sidebar-backdrop${sidebarOpen ? " is-visible" : ""}`} onClick={() => setSidebarOpen(false)} aria-label={t("common.closeSidebar")} />

      <header className="mobile-header surface">
        <button type="button" className="btn btn-ghost-inline sidebar-toggle" onClick={() => setSidebarOpen(true)} aria-label={t("common.toggleMenu")}>
          <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
            <path d="M3 6h18v2H3V6Zm0 5h18v2H3v-2Zm0 5h18v2H3v-2Z" />
          </svg>
        </button>
        <strong className="mobile-brand" style={{ display: "flex", alignItems: "center", gap: "8px" }}>
          {branding.logo ? (
            <img src={branding.logo} alt="Logo" style={{ width: "24px", height: "24px", borderRadius: "6px", objectFit: "contain", flexShrink: 0 }} />
          ) : (
            <svg width="24" height="24" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" style={{ flexShrink: 0 }}>
              <rect width="32" height="32" rx="8" fill="url(#logo_grad_mobile)" />
              <path d="M12 6H21C22.1046 6 23 6.89543 23 8V19C23 19.5523 22.5523 20 22 20H12C11.4477 20 11 19.5523 11 19V7C11 6.44772 11.4477 6 12 6Z" fill="#F8FAFC" />
              <line x1="14" y1="9" x2="20" y2="9" stroke="#0D9488" strokeWidth="1.5" strokeLinecap="round" />
              <line x1="14" y1="12" x2="20" y2="12" stroke="#0D9488" strokeWidth="1.5" strokeLinecap="round" />
              <line x1="14" y1="15" x2="18" y2="15" stroke="#D97706" strokeWidth="1.5" strokeLinecap="round" />
              <path d="M8 12H20C21.1046 12 22 12.8954 22 14V23C22 24.1046 21.1046 25 20 25H8C6.89543 25 6 24.1046 6 23V14C6 12.8954 6.89543 12 8 12Z" fill="#0F766E" stroke="#0D9488" strokeWidth="1" />
              <path d="M18 15.5H22V21.5H18C16.3431 21.5 15 20.1569 15 18.5C15 16.8431 16.3431 15.5 18 15.5Z" fill="#11395F" />
              <circle cx="18" cy="18.5" r="1.75" fill="#D97706" />
              <defs>
                <linearGradient id="logo_grad_mobile" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
                  <stop stopColor="#0f766e" />
                  <stop offset="1" stopColor="#0d9488" />
                </linearGradient>
              </defs>
            </svg>
          )}
          {branding.app_name || t("app.brand")}
        </strong>
      </header>

      <div className="app-main">
        <main className="app-content">
          <Outlet />
        </main>
      </div>

      <button 
        type="button" 
        className={`back-to-top ${showBackToTop ? "is-visible" : ""}`} 
        onClick={scrollToTop}
        aria-label={t("common.backToTop")}
      >
        <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
          <line x1="12" y1="19" x2="12" y2="5" />
          <polyline points="5 12 12 5 19 12" />
        </svg>
      </button>
    </div>
  );
}
