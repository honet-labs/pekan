import { NavLink, useParams } from "react-router-dom";
import { useAccessStore } from "../../../../core/access/access-store";
import { useI18n } from "../../../../core/i18n/i18n";

export function SettingsSubnav(): JSX.Element {
  const { tenantCode } = useParams();
  const access = useAccessStore();
  const { t } = useI18n();
  const base = tenantCode ? `/app/${tenantCode}/finance/settings` : "/app/default/finance/settings";

  return (
    <nav className="settings-subnav">
      {access.permissions.has("finance.settings.read") ? (
        <NavLink to={`${base}/notifications`} className={({ isActive }) => `btn btn-ghost-inline${isActive ? " is-active" : ""}`}>
          {t("settings.nav.notifications")}
        </NavLink>
      ) : null}
      {access.permissions.has("finance.settings.read") ? (
        <NavLink to={`${base}/templates`} className={({ isActive }) => `btn btn-ghost-inline${isActive ? " is-active" : ""}`}>
          {t("settings.nav.templates")}
        </NavLink>
      ) : null}
      {access.permissions.has("finance.categories.read") || access.permissions.has("finance.categories.create") ? (
        <NavLink to={`${base}/categories`} className={({ isActive }) => `btn btn-ghost-inline${isActive ? " is-active" : ""}`}>
          {t("settings.nav.categories")}
        </NavLink>
      ) : null}
      {access.permissions.has("finance.settings.roles.manage") ? (
        <NavLink to={`${base}/users`} className={({ isActive }) => `btn btn-ghost-inline${isActive ? " is-active" : ""}`}>
          {t("settings.nav.users")}
        </NavLink>
      ) : null}
    </nav>
  );
}
