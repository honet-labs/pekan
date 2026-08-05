import { apiFetch, API_BASE_URL } from "../../../../core/api/client";
import { Reminder, ReminderPayload, ReminderPayment } from "./reminders.types";

export type ReminderPaymentPayload = {
  paid_at: string;
  amount_minor: number;
  status: string;
  notes?: string | null;
  proof_image_url?: string | null;
  image?: File | null;
};

type ListRemindersParams = {
  page?: number;
  page_size?: number;
  status?: string;
  from?: string;
  to?: string;
};

type ListRemindersResponse = {
  items: Reminder[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
};

export async function listReminders(params: ListRemindersParams = {}): Promise<ListRemindersResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.page_size) query.set("page_size", String(params.page_size));
  if (params.status) query.set("status", params.status);
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);

  const queryString = query.toString() ? `?${query.toString()}` : "";
  const data = await apiFetch<ListRemindersResponse>(`/finance/reminders${queryString}`);
  
  return {
    ...data,
    items: Array.isArray(data?.items) ? data.items : []
  };
}

export async function listDueReminders(): Promise<Reminder[]> {
  const data = await apiFetch<{ items: Reminder[] }>("/finance/reminders/due");
  return Array.isArray(data?.items) ? data.items : [];
}

export async function createReminder(payload: ReminderPayload): Promise<Reminder> {
  return apiFetch<Reminder>("/finance/reminders", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function updateReminder(id: string, payload: ReminderPayload): Promise<Reminder> {
  return apiFetch<Reminder>(`/finance/reminders/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function markReminderStatus(id: string, status: string): Promise<Reminder> {
  return apiFetch<Reminder>(`/finance/reminders/${id}/status`, {
    method: "POST",
    body: JSON.stringify({ status })
  });
}

export async function deleteReminder(id: string): Promise<void> {
  await apiFetch(`/finance/reminders/${id}`, { method: "DELETE" });
}

export async function getReminder(id: string): Promise<Reminder> {
  return apiFetch<Reminder>(`/finance/reminders/${id}`);
}

export async function listReminderPayments(reminderId: string): Promise<ReminderPayment[]> {
  const data = await apiFetch<{ items: ReminderPayment[] }>(`/finance/reminders/${reminderId}/payments`);
  return Array.isArray(data?.items) ? data.items : [];
}

export async function addReminderPayment(reminderId: string, payload: ReminderPaymentPayload): Promise<ReminderPayment> {
  const body = payload.image ? new FormData() : JSON.stringify(payload);
  if (payload.image) {
    const fd = body as FormData;
    fd.append("paid_at", payload.paid_at);
    fd.append("amount_minor", String(payload.amount_minor));
    fd.append("status", payload.status);
    if (payload.notes) fd.append("notes", payload.notes);
    fd.append("image", payload.image);
  }

  return apiFetch<ReminderPayment>(`/finance/reminders/${reminderId}/payments`, {
    method: "POST",
    body,
    // apiFetch automatically handles content-type for FormData if body is FormData
  });
}

export async function updateReminderPayment(reminderId: string, paymentId: string, payload: ReminderPaymentPayload): Promise<ReminderPayment> {
  const body = payload.image ? new FormData() : JSON.stringify(payload);
  if (payload.image) {
    const fd = body as FormData;
    fd.append("paid_at", payload.paid_at);
    fd.append("amount_minor", String(payload.amount_minor));
    fd.append("status", payload.status);
    if (payload.notes) fd.append("notes", payload.notes);
    fd.append("image", payload.image);
  }

  return apiFetch<ReminderPayment>(`/finance/reminders/${reminderId}/payments/${paymentId}`, {
    method: "PUT",
    body,
  });
}

export async function deleteReminderPayment(reminderId: string, paymentId: string): Promise<void> {
  await apiFetch(`/finance/reminders/${reminderId}/payments/${paymentId}`, {
    method: "DELETE"
  });
}

export function getPaymentProofImageUrl(reminderId: string, paymentId: string): string {
  return `${API_BASE_URL}/finance/reminders/${reminderId}/payments/${paymentId}/proof`;
}

export async function openPaymentProofImage(reminderId: string, paymentId: string): Promise<void> {
  const response = await fetch(getPaymentProofImageUrl(reminderId, paymentId), {
    credentials: "include"
  });
  if (!response.ok) {
    throw new Error("Failed to open proof image");
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank", "noopener,noreferrer");
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}
