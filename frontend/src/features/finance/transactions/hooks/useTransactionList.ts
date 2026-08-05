import { useCallback, useEffect, useState } from "react";
import { useDebounce } from "../../../../core/hooks/useDebounce";
import { deleteTransaction, listTransactions } from "../api/transaction.api";
import { Transaction, TransactionType } from "../api/transaction.types";

type State = {
  items: Transaction[];
  loading: boolean;
  error: string | null;
  query: string;
  type: TransactionType | "";
  from: string;
  to: string;
  page: number;
  pageSize: number;
  total: number;
  setQuery: (value: string) => void;
  setType: (value: TransactionType | "") => void;
  setFrom: (value: string) => void;
  setTo: (value: string) => void;
  setPage: (value: number) => void;
  reload: () => Promise<void>;
  remove: (transactionID: string) => Promise<void>;
};

export function useTransactionList(): State {
  const [items, setItems] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 500);
  const [type, setType] = useState<TransactionType | "">("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [total, setTotal] = useState(0);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listTransactions({
        page,
        page_size: pageSize,
        q: debouncedQuery || undefined,
        type: type || undefined,
        from: from || undefined,
        to: to || undefined
      });
      setItems(res.items);
      setTotal(res.pagination.total);
    } catch (err) {
      setItems([]);
      setTotal(0);
      setError(err instanceof Error ? err.message : "Failed to load transactions");
    } finally {
      setLoading(false);
    }
  }, [from, debouncedQuery, to, type, page, pageSize]);

  const remove = useCallback(
    async (transactionID: string) => {
      try {
        await deleteTransaction(transactionID);
        setError(null);
        await reload();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to delete transaction");
        throw err;
      }
    },
    [reload]
  );

  useEffect(() => {
    reload().catch(() => undefined);
  }, [reload]);

  return {
    items,
    loading,
    error,
    query,
    type,
    from,
    to,
    page,
    pageSize,
    total,
    setQuery: (v) => { setQuery(v); setPage(1); },
    setType: (v) => { setType(v); setPage(1); },
    setFrom: (v) => { setFrom(v); setPage(1); },
    setTo: (v) => { setTo(v); setPage(1); },
    setPage,
    reload,
    remove
  };
}
