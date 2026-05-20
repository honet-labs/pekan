import { useEffect, useState } from "react";
import { createBudget, deleteBudget, listBudgets, updateBudget } from "../api/budgets.api";
import { Budget, BudgetPayload } from "../api/budgets.types";
import { useI18n } from "../../../../core/i18n/i18n";

export function useBudgets() {
  const { t } = useI18n();
  const [items, setItems] = useState<Budget[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [total, setTotal] = useState(0);

  const refresh = () => {
    setLoading(true);
    setError(undefined);
    listBudgets({ page, page_size: pageSize })
      .then((res) => {
        setItems(res.items);
        setTotal(res.pagination.total);
        setError(undefined);
      })
      .catch((err) => {
        setItems([]);
        setTotal(0);
        setError(err instanceof Error ? err.message : t("errors.loadBudgetsFailed"));
      })
      .finally(() => setLoading(false));
  };

  const toErrorMessage = (err: unknown): string =>
    err instanceof Error ? err.message : t("errors.loadBudgetsFailed");

  useEffect(() => {
    refresh();
  }, [t, page, pageSize]);

  const create = async (payload: BudgetPayload) => {
    try {
      const created = await createBudget(payload);
      setError(undefined);
      refresh();
      return created;
    } catch (err) {
      const message = toErrorMessage(err);
      setError(message);
      throw new Error(message);
    }
  };

  const update = async (id: string, payload: BudgetPayload) => {
    try {
      const updated = await updateBudget(id, payload);
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
      await deleteBudget(id);
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

