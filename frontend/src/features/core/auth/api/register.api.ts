const BASE_URL = "/api/v1";

export interface RegisterInitPayload {
  tenant_code: string;
  tenant_name: string;
  admin_email: string;
  admin_name: string;
  password: string;
  phone?: string;
  otp_method: "email" | "whatsapp";
}

export interface RegisterInitResponse {
  session_token: string;
  otp_method: string;
  message: string;
}

export interface RegisterVerifyResponse {
  success: boolean;
  message: string;
}

export async function registerInit(payload: RegisterInitPayload): Promise<RegisterInitResponse> {
  const res = await fetch(`${BASE_URL}/auth/register/init`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data?.error?.message || data?.message || "Gagal memulai registrasi");
  }
  return data.data ?? data;
}

export async function registerVerify(sessionToken: string, otpCode: string): Promise<RegisterVerifyResponse> {
  const res = await fetch(`${BASE_URL}/auth/register/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ session_token: sessionToken, otp_code: otpCode }),
  });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data?.error?.message || data?.message || "Kode OTP tidak valid");
  }
  return data.data ?? data;
}
