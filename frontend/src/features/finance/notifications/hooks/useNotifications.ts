import { useEffect, useState } from "react";
import { createNotification, listNotifications, markNotificationRead } from "../api/notifications.api";
import { Notification, NotificationPayload } from "../api/notifications.types";
import { useI18n } from "../../../../core/i18n/i18n";

export function useNotifications() {
  const { t } = useI18n();
  const [items, setItems] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();

  const refresh = () => {
    setLoading(true);
    setError(undefined);
    listNotifications()
      .then((data) => {
        setItems(data);
        setError(undefined);
      })
      .catch((err) => setError(err instanceof Error ? err.message : t("errors.loadNotificationsFailed")))
      .finally(() => setLoading(false));
  };

  const toErrorMessage = (err: unknown): string =>
    err instanceof Error ? err.message : t("errors.loadNotificationsFailed");

  useEffect(() => {
    refresh();
  }, [t]);

  const create = async (payload: NotificationPayload) => {
    try {
      const created = await createNotification(payload);
      setItems((prev) => [created, ...prev]);
      setError(undefined);
      return created;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  };

  const markRead = async (id: string) => {
    try {
      const updated = await markNotificationRead(id);
      setItems((prev) => prev.map((item) => (item.id === id ? updated : item)));
      setError(undefined);
      return updated;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  };

  return { items, loading, error, create, markRead, refresh };
}

