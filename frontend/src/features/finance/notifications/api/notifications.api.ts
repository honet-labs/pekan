import { apiFetch } from "../../../../core/api/client";
import { Notification, NotificationPayload } from "./notifications.types";

export async function listNotifications(): Promise<Notification[]> {
  const data = await apiFetch<{ items: Notification[] }>("/finance/notifications");
  return Array.isArray(data?.items) ? data.items : [];
}

export async function createNotification(payload: NotificationPayload): Promise<Notification> {
  return apiFetch<Notification>("/finance/notifications", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export async function markNotificationRead(id: string): Promise<Notification> {
  return apiFetch<Notification>(`/finance/notifications/${id}/read`, { method: "POST" });
}
