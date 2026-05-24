import { apiFetch } from "../../../../core/api/client";
import {
  AuditLogEntry,
  CreateRolePayload,
  CreateTenantUserPayload,
  NotificationChannelSetting,
  ReminderTemplateSetting,
  TenantPermission,
  TenantRole,
  TenantUserRoleEntry,
  UpdateTenantUserPayload
} from "./settings.types";

const defaultChannels: NotificationChannelSetting[] = [
  { channel_code: "email", is_enabled: false, config_json: {} },
  { channel_code: "telegram", is_enabled: false, config_json: {} },
  { channel_code: "whatsapp_official", is_enabled: false, config_json: {} },
  { channel_code: "whatsapp_gowa", is_enabled: false, config_json: {} },
  { channel_code: "whatsapp_fonte", is_enabled: false, config_json: {} }
];

function isRecoverableReadError(err: unknown): boolean {
  const message = err instanceof Error ? err.message.toLowerCase() : "";
  return (
    message.includes("forbidden") ||
    message.includes("permission") ||
    message.includes("module disabled") ||
    message.includes("feature locked") ||
    message.includes("not found") ||
    message.includes("method not allowed") ||
    message.includes("api request failed")
  );
}

export async function listNotificationChannels(): Promise<NotificationChannelSetting[]> {
  try {
    const data = await apiFetch<{ items: NotificationChannelSetting[] }>("/finance/settings/channels");
    return Array.isArray(data?.items) ? data.items : [];
  } catch (err) {
    if (isRecoverableReadError(err)) {
      return defaultChannels;
    }
    throw err;
  }
}

export async function updateNotificationChannels(channels: NotificationChannelSetting[]): Promise<NotificationChannelSetting[]> {
  const data = await apiFetch<{ items: NotificationChannelSetting[] }>("/finance/settings/channels", {
    method: "PUT",
    body: JSON.stringify({ channels })
  });
  return Array.isArray(data?.items) ? data.items : [];
}

export async function listReminderTemplates(templateCode = "reminder.due"): Promise<ReminderTemplateSetting[]> {
  const params = new URLSearchParams({ template_code: templateCode });
  try {
    const data = await apiFetch<{ items: ReminderTemplateSetting[] }>(`/finance/settings/templates/reminder?${params.toString()}`);
    return Array.isArray(data?.items) ? data.items : [];
  } catch (err) {
    if (isRecoverableReadError(err)) {
      return [];
    }
    throw err;
  }
}

export async function upsertReminderTemplate(payload: {
  template_code: string;
  channel_code: string;
  language_code: string;
  title_template?: string;
  body_template: string;
  is_enabled: boolean;
}): Promise<ReminderTemplateSetting> {
  return apiFetch<ReminderTemplateSetting>("/finance/settings/templates/reminder", {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function listUsersRoles(): Promise<{ users: TenantUserRoleEntry[]; roles: TenantRole[] }> {
  try {
    const data = await apiFetch<{ users: TenantUserRoleEntry[]; roles: TenantRole[] }>("/finance/settings/users/roles");
    return {
      users: Array.isArray(data?.users) ? data.users : [],
      roles: Array.isArray(data?.roles) ? data.roles : []
    };
  } catch (err) {
    if (isRecoverableReadError(err)) {
      return { users: [], roles: [] };
    }
    throw err;
  }
}

export async function listRoleCatalog(): Promise<{ roles: TenantRole[]; permissions: TenantPermission[] }> {
  try {
    const data = await apiFetch<{ roles: TenantRole[]; permissions: TenantPermission[] }>("/finance/settings/roles");
    return {
      roles: Array.isArray(data?.roles) ? data.roles : [],
      permissions: Array.isArray(data?.permissions) ? data.permissions : []
    };
  } catch (err) {
    if (isRecoverableReadError(err)) {
      return { roles: [], permissions: [] };
    }
    throw err;
  }
}

export async function createRole(payload: CreateRolePayload): Promise<TenantRole> {
  return apiFetch<TenantRole>("/finance/settings/roles", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function updateRole(roleID: string, payload: CreateRolePayload): Promise<TenantRole> {
  return apiFetch<TenantRole>(`/finance/settings/roles/${roleID}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function deleteRole(roleID: string): Promise<void> {
  await apiFetch(`/finance/settings/roles/${roleID}`, { method: "DELETE" });
}

export async function listTenantUsers(): Promise<TenantUserRoleEntry[]> {
  try {
    const data = await apiFetch<{ items: TenantUserRoleEntry[] }>("/finance/settings/users");
    return Array.isArray(data?.items) ? data.items : [];
  } catch (err) {
    if (isRecoverableReadError(err)) {
      return [];
    }
    throw err;
  }
}

export async function createTenantUser(payload: CreateTenantUserPayload): Promise<TenantUserRoleEntry> {
  return apiFetch<TenantUserRoleEntry>("/finance/settings/users", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function updateTenantUser(membershipID: string, payload: UpdateTenantUserPayload): Promise<TenantUserRoleEntry> {
  return apiFetch<TenantUserRoleEntry>(`/finance/settings/users/${membershipID}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function deleteTenantUser(membershipID: string): Promise<void> {
  await apiFetch(`/finance/settings/users/${membershipID}`, { method: "DELETE" });
}

export async function updateMembershipRoles(membershipID: string, roleIDs: string[]): Promise<void> {
  await apiFetch(`/finance/settings/users/roles/${membershipID}`, {
    method: "PUT",
    body: JSON.stringify({ role_ids: roleIDs })
  });
}

export async function listAuditLogs(params: {
  page?: number;
  page_size?: number;
  action?: string;
  resource_type?: string;
  actor_user_id?: string;
  from?: string;
  to?: string;
}): Promise<{ items: AuditLogEntry[]; pagination: { page: number; page_size: number; total: number } }> {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.page_size) query.set("page_size", String(params.page_size));
  if (params.action) query.set("action", params.action);
  if (params.resource_type) query.set("resource_type", params.resource_type);
  if (params.actor_user_id) query.set("actor_user_id", params.actor_user_id);
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  const suffix = query.toString();
  const data = await apiFetch<{ items: AuditLogEntry[]; pagination: { page: number; page_size: number; total: number } }>(
    `/finance/settings/audit-logs${suffix ? `?${suffix}` : ""}`
  );
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    pagination: data?.pagination ?? { page: 1, page_size: params.page_size ?? 50, total: 0 }
  };
}
