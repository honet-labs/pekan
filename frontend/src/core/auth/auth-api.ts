import { apiFetch } from "../api/client";

type LoginRequest = {
  email: string;
  password: string;
  tenant_id: string;
};

export type LoginResponse = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
  user_id: string;
  email: string;
  active_tenant_id: string;
  permissions: string[];
  features: string[];
  modules: string[];
  must_change_password: boolean;
};

export type MeContextResponse = {
  user: {
    id: string;
    email: string;
    name: string;
  };
  active_tenant: {
    id: string;
    code: string;
  };
  memberships: Array<{
    id: string;
    tenant_id: string;
    tenant_code: string;
    user_id: string;
    status: string;
  }>;
  permissions: string[];
  features: string[];
  modules: string[];
};

export type MeProfileResponse = {
  user_id: string;
  full_name: string;
  username: string;
  email: string;
  phone?: string | null;
  address?: string | null;
  updated_at: string;
};

export function login(payload: LoginRequest): Promise<LoginResponse> {
  return apiFetch<LoginResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function refresh(refreshToken: string): Promise<LoginResponse> {
  return apiFetch<LoginResponse>("/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refresh_token: refreshToken })
  });
}

export function logout(refreshToken: string): Promise<{ logged_out: boolean }> {
  return apiFetch<{ logged_out: boolean }>("/auth/logout", {
    method: "POST",
    body: JSON.stringify({ refresh_token: refreshToken })
  });
}

export function getMeContext(): Promise<MeContextResponse> {
  return apiFetch<MeContextResponse>("/me/context");
}

export function getMeProfile(): Promise<MeProfileResponse> {
  return apiFetch<MeProfileResponse>("/me/profile");
}

export function updateMeProfile(payload: {
  full_name: string;
  username: string;
  email: string;
  phone?: string | null;
  address?: string | null;
}): Promise<MeProfileResponse> {
  return apiFetch<MeProfileResponse>("/me/profile", {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function requestPasswordReset(payload: {
  email: string;
  tenant_id: string;
  method: "email" | "whatsapp";
}): Promise<{ message: string }> {
  return apiFetch<{ message: string }>("/auth/forgot-password", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function requestForgotTenant(payload: {
  email_or_phone: string;
  method: "email" | "whatsapp";
}): Promise<{ message: string }> {
  return apiFetch<{ message: string }>("/auth/forgot-tenant", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
export function changePassword(payload: {
  new_password: string;
}): Promise<{ success: boolean; message: string }> {
  return apiFetch<{ success: boolean; message: string }>("/me/change-password", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
