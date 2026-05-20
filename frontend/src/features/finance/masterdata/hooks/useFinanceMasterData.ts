import { useCallback, useEffect, useState } from "react";

import { createAccount, createCategory, listAccounts, listCategories, updateCategory, deleteCategory } from "../api/masterdata.api";
import { CreateAccountPayload, CreateCategoryPayload, FinanceAccount, FinanceCategory } from "../api/masterdata.types";
import { useI18n } from "../../../../core/i18n/i18n";

type State = {
  accounts: FinanceAccount[];
  categories: FinanceCategory[];
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
  addAccount: (payload: CreateAccountPayload) => Promise<void>;
  addCategory: (payload: CreateCategoryPayload) => Promise<void>;
  editCategory: (categoryID: string, payload: { name: string; category_type: string }) => Promise<void>;
  removeCategory: (categoryID: string) => Promise<void>;
};

export function useFinanceMasterData(): State {
  const { t } = useI18n();
  const [accounts, setAccounts] = useState<FinanceAccount[]>([]);
  const [categories, setCategories] = useState<FinanceCategory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [accountsRes, categoriesRes] = await Promise.all([listAccounts(), listCategories()]);
      setAccounts(accountsRes.items);
      setCategories(categoriesRes.items);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.loadMasterDataFailed"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const addAccount = useCallback(
    async (payload: CreateAccountPayload) => {
      await createAccount(payload);
      await reload();
    },
    [reload]
  );

  const addCategory = useCallback(
    async (payload: CreateCategoryPayload) => {
      await createCategory(payload);
      await reload();
    },
    [reload]
  );

  const editCategory = useCallback(
    async (categoryID: string, payload: { name: string; category_type: string }) => {
      await updateCategory(categoryID, payload);
      await reload();
    },
    [reload]
  );

  const removeCategory = useCallback(
    async (categoryID: string) => {
      await deleteCategory(categoryID);
      await reload();
    },
    [reload]
  );

  return {
    accounts,
    categories,
    loading,
    error,
    reload,
    addAccount,
    addCategory,
    editCategory,
    removeCategory
  };
}
