import { useEffect, useMemo, useState } from "react";
import { NavLink, Outlet, useNavigate, useParams } from "react-router-dom";
import { useAccessStore } from "../../core/access/access-store";
import { logout, getMeProfile } from "../../core/auth/auth-api";
import { useAuthStore } from "../../core/auth/auth-store";
import { useI18n } from "../../core/i18n/i18n";

export function AppShell(): JSX.Element {
  const access = useAccessStore();
  const auth = useAuthStore();
  const navigate = useNavigate();
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
        <div className="sidebar-header">
          <div>
            <h2 className="brand">{t("app.brand")}</h2>
            <p className="sidebar-caption">{t("app.subtitle")}</p>
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
        <strong className="mobile-brand">{t("app.brand")}</strong>
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
