import { Navigate, useParams } from "react-router-dom";
import { useTenantStore } from "../../core/tenant/tenant-store";

type Props = { children: JSX.Element };

export function TenantRoute({ children }: Props): JSX.Element {
  const { tenantCode } = useParams();
  const tenant = useTenantStore();

  if (!tenantCode || tenantCode.trim() === "") {
    return <Navigate to={`/app/${tenant.activeTenantCode}/finance/dashboard`} replace />;
  }

  // Verify membership
  const isAllowed = tenant.allowedTenants.some(t => t.code === tenantCode || t.id === tenantCode);
  
  if (!isAllowed) {
    // If user tries to access a tenant they don't belong to, force back to their active one
    return <Navigate to={`/app/${tenant.activeTenantCode}/finance/dashboard`} replace />;
  }

  if (tenantCode !== tenant.activeTenantCode) {
    tenant.setTenant(tenantCode, tenantCode);
  }
  return children;
}
