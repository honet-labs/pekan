import { useEffect, useState } from "react";
import { createSavings, deleteSavings, listSavings, updateSavings } from "../api/savings.api";
import { Savings, SavingsPayload } from "../api/savings.types";
import { useI18n } from "../../../../core/i18n/i18n";

export function useSavings() {
  const { t } = useI18n();
  const [items, setItems] = useState<Savings[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [total, setTotal] = useState(0);

  const refresh = () => {
    setLoading(true);
    setError(undefined);
    listSavings({ page, page_size: pageSize })
      .then((res) => {
        setItems(res.items);
        setTotal(res.pagination.total);
        setError(undefined);
      })
      .catch((err) => {
        setItems([]);
        setTotal(0);
        setError(err instanceof Error ? err.message : t("errors.loadSavingsFailed"));
      })
      .finally(() => setLoading(false));
  };

  const toErrorMessage = (err: unknown): string =>
    err instanceof Error ? err.message : t("errors.loadSavingsFailed");

  useEffect(() => {
    refresh();
  }, [t, page, pageSize]);

  const create = async (payload: SavingsPayload) => {
    try {
      const created = await createSavings(payload);
      setError(undefined);
      refresh();
      return created;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  };

  const update = async (id: string, payload: SavingsPayload) => {
    try {
      const updated = await updateSavings(id, payload);
      setError(undefined);
      refresh();
      return updated;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  };

  const remove = async (id: string) => {
    try {
      await deleteSavings(id);
      setError(undefined);
      refresh();
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  };

  return { items, loading, error, page, pageSize, total, setPage, create, update, remove, refresh };
}

