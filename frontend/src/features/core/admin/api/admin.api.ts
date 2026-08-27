import { apiFetch } from "../../../../core/api/client";

export type BootstrapTenantPayload = {
  tenant_code: string;
  tenant_name: string;
  admin_email: string;
  admin_name: string;
  password: string;
};

export type TenantListItem = {
  id: string;
  code: string;
  name: string;
  status: string;
  timezone: string;
  user_count: number;
  quota_users: number;
  quota_transactions: number;
};

export type AuditLog = {
  id: string;
  tenant_id: string;
  tenant_name: string;
  actor_user_id: string;
  actor_user_name: string;
  action: string;
  resource: string;
  resource_id: string;
  ip_address: string;
  created_at: string;
  details?: string;
};

export type GrowthStats = {
  tenants: { date: string; count: number }[];
  users: { date: string; count: number }[];
  total_tenants: number;
  total_users: number;
  total_transactions: number;
};

export type TenantModule = {
  module_code: string;
  is_enabled: boolean;
};

export type ServerStatus = {
  os: string;
  uptime: string;
  ip_address: string;
  port: string;
  db_status: string;
  redis_status: string;
  services: ServiceStatus[];
};

export type ServiceStatus = {
  name: string;
  status: string;
  port: number;
};

export type BackupFile = {
  name: string;
  size: number;
  created_at: string;
};

export type DatabaseTable = {
  name: string;
  rows: number;
  data_size: string;
  index_size: string;
  total_size: string;
};

export type QueryResult = {
  columns: string[];
  rows: any[];
  error?: string;
};

export type DatabaseGrowthPoint = {
  date: string;
  schema_name: string;
  total_size_bytes: number;
};


function getAdminToken(): string | null {
  return localStorage.getItem("pekan_admin_token");
}

async function adminFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getAdminToken();
  const headers = {
    ...(options.headers || {}),
    "X-Admin-Token": token || ""
  };
  try {
    return await apiFetch<T>(path, { ...options, headers });
  } catch (err) {
    // If unauthorized, clear token and reload to show login screen
    if (err instanceof Error && (err.message.includes("401") || err.message.includes("Unauthorized"))) {
      localStorage.removeItem("pekan_admin_token");
      window.location.reload();
    }
    throw err;
  }
}

export async function adminLogin(secret: string): Promise<{ token: string }> {
  const res = await apiFetch<{ token: string }>("/admin/login", {
    method: "POST",
    body: JSON.stringify({ secret })
  });
  localStorage.setItem("pekan_admin_token", res.token);
  return res;
}

export async function bootstrapTenant(payload: BootstrapTenantPayload): Promise<void> {
  return adminFetch("/admin/bootstrap-tenant", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function listTenants(): Promise<TenantListItem[]> {
  const data = await adminFetch<TenantListItem[]>("/admin/tenants");
  return Array.isArray(data) ? data : [];
}

export async function listLogs(): Promise<AuditLog[]> {
  const data = await adminFetch<AuditLog[]>("/admin/logs");
  return Array.isArray(data) ? data : [];
}

export async function getGrowthStats(from?: string, to?: string): Promise<GrowthStats> {
  const query = (from && to) ? `?from=${from}&to=${to}` : from ? `?from=${from}` : "";
  return adminFetch<GrowthStats>(`/admin/stats/growth${query}`);
}

export async function updateTenantQuotas(tenantID: string, users: number, transactions: number): Promise<void> {
  return adminFetch(`/admin/tenants/${tenantID}/quotas`, {
    method: "PUT",
    body: JSON.stringify({ users, transactions })
  });
}

export async function updateTenant(tenantID: string, name: string, status: string): Promise<void> {
  return adminFetch(`/admin/tenants/${tenantID}`, {
    method: "PUT",
    body: JSON.stringify({ name, status })
  });
}

export async function deleteTenant(tenantID: string): Promise<void> {
  return adminFetch(`/admin/tenants/${tenantID}`, {
    method: "DELETE"
  });
}

export async function listTenantModules(tenantID: string): Promise<TenantModule[]> {
  const data = await adminFetch<TenantModule[]>(`/admin/tenants/${tenantID}/modules`);
  return Array.isArray(data) ? data : [];
}

export async function updateTenantModule(tenantID: string, moduleCode: string, isEnabled: boolean): Promise<void> {
  return adminFetch(`/admin/tenants/${tenantID}/modules`, {
    method: "PUT",
    body: JSON.stringify({ module_code: moduleCode, is_enabled: isEnabled })
  });
}

export type TenantUser = {
  id: string;
  email: string;
  full_name: string;
  is_active: boolean;
  status: string;
  role: string;
  created_at: string;
};

export async function listTenantUsers(tenantID: string): Promise<TenantUser[]> {
  const data = await adminFetch<TenantUser[]>(`/admin/tenants/${tenantID}/users`);
  return Array.isArray(data) ? data : [];
}

export async function listTenantBackups(tenantID: string): Promise<BackupFile[]> {
  const data = await adminFetch<{ data: BackupFile[] }>(`/admin/tenants/${tenantID}/backups`);
  return data.data || [];
}

export async function createTenantBackup(tenantID: string, type: string = "full"): Promise<void> {
  return adminFetch(`/admin/tenants/${tenantID}/backups`, {
    method: "POST",
    body: JSON.stringify({ type })
  });
}

export async function restoreTenantBackup(tenantID: string, filename: string): Promise<void> {
  return adminFetch(`/admin/tenants/${tenantID}/backups/restore`, {
    method: "POST",
    body: JSON.stringify({ filename })
  });
}

export async function adminResetUserPassword(userID: string, password: string): Promise<void> {
  return adminFetch(`/admin/users/${userID}/reset-password`, {
    method: "POST",
    body: JSON.stringify({ password })
  });
}

export async function adminUpdateUserEmail(userID: string, email: string): Promise<void> {
  return adminFetch(`/admin/users/${userID}/email`, {
    method: "PUT",
    body: JSON.stringify({ email })
  });
}

export async function adminUpdateUserPhone(userID: string, phone: string): Promise<void> {
  return adminFetch(`/admin/users/${userID}/phone`, {
    method: "PUT",
    body: JSON.stringify({ phone })
  });
}

export async function impersonateUser(userID: string, tenantID: string, email: string): Promise<{ access_token: string }> {
  return adminFetch<{ access_token: string }>("/admin/impersonate", {
    method: "POST",
    body: JSON.stringify({ user_id: userID, tenant_id: tenantID, email })
  });
}

export async function getServerStatus(): Promise<ServerStatus> {
  return adminFetch<ServerStatus>("/admin/server/status");
}

export async function getGlobalSetting(key: string): Promise<{ value: string; is_encrypted: boolean; is_masked: boolean }> {
  return adminFetch<{ value: string; is_encrypted: boolean; is_masked: boolean }>(`/admin/settings/${key}`);
}

export async function saveGlobalSetting(key: string, value: string, isEncrypted: boolean): Promise<void> {
  return adminFetch(`/admin/settings/${key}`, {
    method: "PUT",
    body: JSON.stringify({ value, is_encrypted: isEncrypted })
  });
}

export async function testNotification(provider: string, configJSON: string, destination: string): Promise<void> {
  return adminFetch("/admin/test/notification", {
    method: "POST",
    body: JSON.stringify({ provider, config_json: configJSON, destination })
  });
}

export async function testAI(provider: string, apiKey: string): Promise<{ success: boolean; models: { id: string; label: string }[] }> {
  return adminFetch<{ success: boolean; models: { id: string; label: string }[] }>("/admin/test/ai", {
    method: "POST",
    body: JSON.stringify({ provider, api_key: apiKey })
  });
}

export async function testDatabase(configJSON: string): Promise<{ success: boolean }> {
  return adminFetch<{ success: boolean }>("/admin/test/database", {
    method: "POST",
    body: JSON.stringify({ config_json: configJSON })
  });
}

export async function listBackups(): Promise<BackupFile[]> {
  const res = await adminFetch<{ data: BackupFile[] }>("/admin/backups");
  return res.data || [];
}

export async function createBackup(type: string): Promise<void> {
  return adminFetch("/admin/backups", {
    method: "POST",
    body: JSON.stringify({ type })
  });
}

export async function restoreBackup(filename: string): Promise<void> {
  return adminFetch("/admin/backups/restore", {
    method: "POST",
    body: JSON.stringify({ filename })
  });
}

export async function uploadBackup(file: File): Promise<{ success: boolean; filename: string }> {
  const token = getAdminToken();
  const formData = new FormData();
  formData.append("backup", file);

  const res = await fetch("/api/v1/admin/backups/upload", {
    method: "POST",
    headers: { "X-Admin-Token": token || "" },
    body: formData
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: "Upload failed" } }));
    throw new Error(err.error?.message || "Upload failed");
  }

  return res.json();
}

export function getBackupDownloadUrl(filename: string): string {
  const token = getAdminToken() || "";
  // In a real scenario, downloading via GET with token in header is preferred, 
  // but for simple anchor tag download, we might need a short-lived token or cookies.
  // For admin dashboard, we can just fetch and trigger blob download.
  return `/api/v1/admin/backups/download/${filename}`;
}

export async function downloadBackupBlob(filename: string): Promise<Blob> {
  const token = getAdminToken();
  const res = await fetch(`/api/v1/admin/backups/download/${filename}`, {
    headers: { "X-Admin-Token": token || "" }
  });
  if (!res.ok) throw new Error("Failed to download backup");
  return res.blob();
}

export async function downloadTenantBackupBlob(tenantID: string, filename: string): Promise<Blob> {
  const token = getAdminToken();
  const res = await fetch(`/api/v1/admin/tenants/${tenantID}/backups/download/${filename}`, {
    headers: { "X-Admin-Token": token || "" }
  });
  if (!res.ok) throw new Error("Failed to download tenant backup");
  return res.blob();
}

export async function executeQuery(query: string): Promise<QueryResult> {
  return adminFetch<QueryResult>("/admin/database/query", {
    method: "POST",
    body: JSON.stringify({ query })
  });
}

export async function getDatabaseStats(): Promise<DatabaseTable[]> {
  const data = await adminFetch<DatabaseTable[]>("/admin/database/stats");
  return Array.isArray(data) ? data : [];
}

export async function getDatabaseGrowth(): Promise<DatabaseGrowthPoint[]> {
  const data = await adminFetch<DatabaseGrowthPoint[]>("/admin/database/growth");
  return Array.isArray(data) ? data : [];
}


export function adminLogout(): void {
  localStorage.removeItem("pekan_admin_token");
}

export type WhatsAppQueueStats = {
  total_processed: number;
  total_pending: number;
  total_processing: number;
  total_success: number;
  total_failed: number;
  average_latency_ms: number;
};

export type WhatsAppQueueItem = {
  id: string;
  phone_number: string;
  message: string;
  reply_message: string | null;
  status: 'pending' | 'processing' | 'success' | 'failed';
  error_message: string | null;
  processing_time_ms: number | null;
  tenant_id: string | null;
  tenant_code: string | null;
  user_id: string | null;
  user_email: string | null;
  received_at: string;
  processed_at: string | null;
};

export async function getWhatsAppQueueStats(): Promise<WhatsAppQueueStats> {
  return adminFetch<WhatsAppQueueStats>("/admin/whatsapp/queue/stats");
}

export async function getWhatsAppQueueHistory(limit: number, offset: number, search: string): Promise<{ items: WhatsAppQueueItem[]; total: number }> {
  const queryParams = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
    search
  });
  return adminFetch<{ items: WhatsAppQueueItem[]; total: number }>(`/admin/whatsapp/queue/history?${queryParams.toString()}`);
}

export async function retryWhatsAppQueueMessage(id: string): Promise<{ success: boolean }> {
  return adminFetch<{ success: boolean }>(`/admin/whatsapp/queue/retry/${id}`, {
    method: "POST"
  });
}

export type UpdateStatusInfo = {
  current_commit: string;
  current_date: string;
  latest_commit: string;
  latest_date: string;
  latest_message: string;
  update_available: boolean;
  is_git_repo: boolean;
};

export type UpdateProgress = {
  status: "idle" | "running" | "success" | "failed";
  started_at: string;
  ended_at: string;
  error: string;
  logs: string;
};

export async function checkUpdate(): Promise<UpdateStatusInfo> {
  return adminFetch<UpdateStatusInfo>("/admin/updates/check");
}

export async function applyUpdate(): Promise<{ status: string }> {
  return adminFetch<{ status: string }>("/admin/updates/apply", {
    method: "POST"
  });
}

export async function getUpdateStatus(): Promise<UpdateProgress> {
  return adminFetch<UpdateProgress>("/admin/updates/status");
}

export async function getSystemLogs(service: string = "pekan-api", lines: number = 200): Promise<{ logs: string }> {
  return adminFetch<{ logs: string }>(`/admin/system-logs?service=${service}&lines=${lines}`);
}

