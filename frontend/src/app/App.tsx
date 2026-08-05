import { useEffect, useState } from "react";

import { useAccessStore } from "../core/access/access-store";
import { getMeContext } from "../core/auth/auth-api";
import { useAuthStore } from "../core/auth/auth-store";
import { useI18n } from "../core/i18n/i18n";
import { useTenantStore } from "../core/tenant/tenant-store";
import { useBrandingStore } from "../core/branding/branding-store";
import { AppRouter } from "./router";

export function App(): JSX.Element {
  const auth = useAuthStore();
  const access = useAccessStore();
  const tenant = useTenantStore();
  const branding = useBrandingStore();
  const { t } = useI18n();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    // Fetch global branding first
    branding.fetchBranding().finally(() => {
      // Always try to get context on mount to recover session from cookies
      getMeContext()
        .then((ctx) => {
          auth.setAuth(ctx.user.id); // Mark as authenticated
          access.setAccess({
            modules: ctx.modules || [],
            features: ctx.features || [],
            permissions: ctx.permissions || []
          });
          if (ctx.memberships && Array.isArray(ctx.memberships)) {
            tenant.setAllowedTenants(ctx.memberships.map(m => ({ id: m.tenant_id, code: m.tenant_code || m.tenant_id })));
          }
          if (ctx.active_tenant && ctx.active_tenant.id) {
            tenant.setTenant(ctx.active_tenant.id, ctx.active_tenant.code || ctx.active_tenant.id);
          }
        })
        .catch(() => {
          // Only clear if we were previously thought to be authenticated
          if (auth.isAuthenticated) {
            auth.clear();
          }
        })
        .finally(() => {
          setReady(true);
        });
    });
  }, []); // Only run once on mount


  if (!ready) {
    return <div className="loading-screen">{t("common.loading")}</div>;
  }

  return <AppRouter />;
}
