export type NotificationChannelSetting = {
  id?: string;
  channel_code: "email" | "telegram" | "whatsapp_official" | "whatsapp_gowa" | "whatsapp_fonte";
  is_enabled: boolean;
  config_json: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
};

export type ReminderTemplateSetting = {
  id: string;
  template_code: string;
  channel_code: "any" | "email" | "telegram" | "whatsapp_official" | "whatsapp_gowa" | "whatsapp_fonte";
  language_code: string;
  title_template?: string | null;
  body_template: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type TenantRole = {
  id: string;
  code: string;
  name: string;
  is_system: boolean;
  permission_ids?: string[];
};

export type TenantUserRoleEntry = {
  membership_id: string;
  user_id: string;
  email: string;
  full_name: string;
  status: string;
  is_active?: boolean;
  last_login_at?: string | null;
  whatsapp_number?: string | null;
  username?: string | null;
  roles: TenantRole[];
};

export type TenantPermission = {
  id: string;
  code: string;
  name: string;
  module_code: string;
  action: string;
};

export type CreateTenantUserPayload = {
  email: string;
  full_name: string;
  username?: string;
  phone_number?: string;
  password: string;
  status: string;
  is_active: boolean;
  role_ids: string[];
};

export type UpdateTenantUserPayload = {
  email: string;
  full_name: string;
  username?: string;
  phone_number?: string;
  password?: string;
  status: string;
  is_active: boolean;
  role_ids: string[];
};

export type CreateRolePayload = {
  code: string;
  name: string;
  permission_ids: string[];
};

export type AuditLogEntry = {
  id: number;
  tenant_id?: string;
  actor_user_id?: string;
  actor_user_name?: string;
  action: string;
  resource_type: string;
  resource_id: string;
  before_json?: unknown;
  after_json?: unknown;
  request_id?: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
};
