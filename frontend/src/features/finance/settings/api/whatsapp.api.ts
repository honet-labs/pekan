import { apiFetch } from "../../../../core/api/client";

export async function generateWhatsAppOTP(): Promise<string> {
  const res = await apiFetch<{ otp_code: string }>("/settings/whatsapp/otp", {
    method: "POST"
  });
  return res?.otp_code || "";
}

export async function connectWhatsApp(phone_number: string): Promise<void> {
  await apiFetch("/settings/whatsapp/connect", {
    method: "POST",
    body: JSON.stringify({ phone_number })
  });
}

export type WhatsAppStatus = {
  connected: boolean;
  phone_number?: string;
  last_active?: string;
};

export async function checkWhatsAppStatus(): Promise<WhatsAppStatus> {
  return apiFetch<WhatsAppStatus>("/settings/whatsapp/status");
}

export async function disconnectWhatsApp(): Promise<void> {
  await apiFetch("/settings/whatsapp/disconnect", {
    method: "DELETE"
  });
}

export async function sendChatbotMessage(message: string): Promise<string> {
  const res = await apiFetch<{ reply: string }>("/settings/whatsapp/chat", {
    method: "POST",
    body: JSON.stringify({ message })
  });
  return res?.reply || "";
}
