type TenantState = {
  activeTenantID: string;
  activeTenantCode: string;
  allowedTenants: Array<{ id: string; code: string }>;
  setTenant: (tenantID: string, tenantCode: string) => void;
  setAllowedTenants: (tenants: Array<{ id: string; code: string }>) => void;
  clearTenant: () => void;
};

type Listener = () => void;
const listeners = new Set<Listener>();

function emitChange() {
  for (const listener of listeners) {
    listener();
  }
}

const STORAGE_ID_KEY = "pekan_tenant_id";
const STORAGE_CODE_KEY = "pekan_tenant_code";

const storedTenantID = localStorage.getItem(STORAGE_ID_KEY);
const storedTenantCode = localStorage.getItem(STORAGE_CODE_KEY);

const tenantActions = {
  setTenant(tenantID: string, tenantCode: string) {
    state = {
      ...state,
      activeTenantID: tenantID,
      activeTenantCode: tenantCode,
    };
    localStorage.setItem(STORAGE_ID_KEY, tenantID);
    localStorage.setItem(STORAGE_CODE_KEY, tenantCode);
    emitChange();
  },
  setAllowedTenants(tenants: Array<{ id: string; code: string }>) {
    state = {
      ...state,
      allowedTenants: tenants,
    };
    emitChange();
  },
  clearTenant() {
    state = {
      ...state,
      activeTenantID: "default-tenant-id",
      activeTenantCode: "default",
    };
    localStorage.removeItem(STORAGE_ID_KEY);
    localStorage.removeItem(STORAGE_CODE_KEY);
    emitChange();
  }
};

let state: TenantState = {
  activeTenantID: storedTenantID ?? "default-tenant-id",
  activeTenantCode: storedTenantCode ?? "default",
  allowedTenants: [],
  ...tenantActions
};

export const tenantStore = {
  getState() {
    return state;
  },
  subscribe(listener: Listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  }
};

import { useSyncExternalStore } from "react";

export function useTenantStore() {
  return useSyncExternalStore(tenantStore.subscribe, tenantStore.getState);
}
