import { useEffect, useState } from "react";

import { listAccounts, listCategories } from "../../masterdata/api/masterdata.api";
import { FinanceAccount, FinanceCategory } from "../../masterdata/api/masterdata.types";
import { listSavings } from "../../savings/api/savings.api";
import { Savings } from "../../savings/api/savings.types";

type State = {
  accounts: FinanceAccount[];
  categories: FinanceCategory[];
  savings: Savings[];
  loading: boolean;
  error: string | null;
};

export function useTransactionMasterData(): State {
  const [state, setState] = useState<State>({
    accounts: [],
    categories: [],
    savings: [],
    loading: true,
    error: null
  });

  useEffect(() => {
    let mounted = true;
    Promise.all([listAccounts(), listCategories(), listSavings()])
      .then(([accountsRes, categoriesRes, savingsRes]) => {
        if (!mounted) return;
        setState({
          accounts: accountsRes.items,
          categories: categoriesRes.items,
          savings: savingsRes.items,
          loading: false,
          error: null
        });
      })
      .catch((err: Error) => {
        if (!mounted) return;
        setState({
          accounts: [],
          categories: [],
          savings: [],
          loading: false,
          error: err.message
        });
      });

    return () => {
      mounted = false;
    };
  }, []);

  return state;
}
