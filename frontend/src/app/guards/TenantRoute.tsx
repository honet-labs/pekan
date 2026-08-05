import { useParams } from "react-router-dom";
import { useTenantStore } from "../../core/tenant/tenant-store";
import NotFoundPage from "../pages/NotFoundPage";

type Props = { children: JSX.Element };

export function TenantRoute({ children }: Props): JSX.Element {
  const { tenantCode } = useParams();
  const tenant = useTenantStore();

  if (!tenantCode || tenantCode.trim() === "") {
    return <NotFoundPage />;
  }

  // Verify membership case-insensitively
  const isAllowed = tenant.allowedTenants.some(
    t => t.code.toLowerCase() === tenantCode.toLowerCase() || t.id.toLowerCase() === tenantCode.toLowerCase()
  );
  
  if (!isAllowed) {
    // Show 404 page directly instead of automatic redirect
    return <NotFoundPage />;
  }

  // Ensure active tenant is synced properly with original membership code/id
  const matched = tenant.allowedTenants.find(
    t => t.code.toLowerCase() === tenantCode.toLowerCase() || t.id.toLowerCase() === tenantCode.toLowerCase()
  );

  if (matched && (matched.id !== tenant.activeTenantID || matched.code !== tenant.activeTenantCode)) {
    tenant.setTenant(matched.id, matched.code);
  } else if (!matched && tenantCode !== tenant.activeTenantCode) {
    tenant.setTenant(tenantCode, tenantCode);
  }

  return children;
}
