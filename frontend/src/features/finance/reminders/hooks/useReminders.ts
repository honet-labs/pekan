import { useCallback, useEffect, useState } from "react";
import {
  createReminder,
  deleteReminder,
  listDueReminders,
  listReminders,
  markReminderStatus,
  updateReminder,
  addReminderPayment,
  listReminderPayments,
  updateReminderPayment,
  deleteReminderPayment
} from "../api/reminders.api";
import { Reminder, ReminderPayload } from "../api/reminders.types";
import { useI18n } from "../../../../core/i18n/i18n";

export function useReminders() {
  const { t } = useI18n();
  const [items, setItems] = useState<Reminder[]>([]);
  const [dueItems, setDueItems] = useState<Reminder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [total, setTotal] = useState(0);

  const refresh = useCallback(() => {
    setLoading(true);
    setError(undefined);
    Promise.all([
      listReminders({ page, page_size: pageSize }),
      listDueReminders()
    ])
      .then(([allRes, due]) => {
        setItems(allRes.items);
        setTotal(allRes.pagination.total);
        setDueItems(Array.isArray(due) ? due : (due as any).items || []);
        setError(undefined);
      })
      .catch((err) => {
        setItems([]);
        setTotal(0);
        setError(err instanceof Error ? err.message : t("errors.loadRemindersFailed"));
      })
      .finally(() => setLoading(false));
  }, [page, pageSize, t]);

  const toErrorMessage = (err: unknown): string =>
    err instanceof Error ? err.message : t("errors.loadRemindersFailed");

  useEffect(() => {
    refresh();
  }, [t, page, pageSize]);

  const create = useCallback(async (payload: ReminderPayload) => {
    try {
      const created = await createReminder(payload);
      setError(undefined);
      refresh();
      return created;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [refresh, t]);

  const update = useCallback(async (id: string, payload: ReminderPayload) => {
    try {
      const updated = await updateReminder(id, payload);
      setError(undefined);
      refresh();
      return updated;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [refresh, t]);

  const markStatus = useCallback(async (id: string, status: string) => {
    try {
      const updated = await markReminderStatus(id, status);
      setError(undefined);
      refresh();
      return updated;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [refresh, t]);

  const remove = useCallback(async (id: string) => {
    try {
      await deleteReminder(id);
      setError(undefined);
      refresh();
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [refresh, t]);

  const addPayment = useCallback(async (reminderId: string, payload: any) => {
    try {
      const created = await addReminderPayment(reminderId, payload);
      setError(undefined);
      refresh();
      return created;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [refresh, t]);

  const updatePayment = useCallback(async (reminderId: string, paymentId: string, payload: any) => {
    try {
      const updated = await updateReminderPayment(reminderId, paymentId, payload);
      setError(undefined);
      refresh();
      return updated;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [refresh, t]);

  const removePayment = useCallback(async (reminderId: string, paymentId: string) => {
    try {
      await deleteReminderPayment(reminderId, paymentId);
      setError(undefined);
      refresh();
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [refresh, t]);

  const fetchPayments = useCallback(async (reminderId: string) => {
    try {
      return await listReminderPayments(reminderId);
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  }, [t]);

  return { items, dueItems, loading, error, page, pageSize, total, setPage, create, update, markStatus, remove, refresh, addPayment, updatePayment, removePayment, fetchPayments };
}

