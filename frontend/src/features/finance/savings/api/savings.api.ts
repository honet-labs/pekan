import { apiFetch } from "../../../../core/api/client";
import { Savings, SavingsPayload } from "./savings.types";

export type SavingsRelatedTransaction = {
  id: string;
  tid?: string;
  transaction_date: string;
  amount_minor: number;
  currency?: string;
  category_name?: string;
  description?: string;
};

type SavingsRelatedTransactionWire = SavingsRelatedTransaction & {
  ID?: string;
  TID?: string;
  TransactionDate?: string;
  AmountMinor?: number;
  Currency?: string;
  CategoryName?: string | null;
  Description?: string | null;
};

type SavingsWire = Savings & {
  sid?: string;
};

function normalizeSavings(item: SavingsWire): Savings {
  return {
    ...item,
    sid: item.sid || `SVG-${item.id.slice(0, 8).toUpperCase()}`
  };
}

type ListSavingsParams = {
  page?: number;
  page_size?: number;
  status?: string;
};

type ListSavingsResponse = {
  items: SavingsWire[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
};

export async function listSavings(params: ListSavingsParams = {}): Promise<ListSavingsResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.page_size) query.set("page_size", String(params.page_size));
  if (params.status) query.set("status", params.status);

  const queryString = query.toString() ? `?${query.toString()}` : "";
  const data = await apiFetch<ListSavingsResponse>(`/finance/savings${queryString}`);
  
  return {
    ...data,
    items: Array.isArray(data?.items) ? data.items.map(normalizeSavings) : []
  };
}

export async function getSavings(id: string): Promise<Savings> {
  const data = await apiFetch<SavingsWire>(`/finance/savings/${id}`);
  return normalizeSavings(data);
}

export async function createSavings(payload: SavingsPayload): Promise<Savings> {
  const data = await apiFetch<SavingsWire>("/finance/savings", {
    method: "POST",
    body: JSON.stringify(payload)
  });
  return normalizeSavings(data);
}

export async function updateSavings(id: string, payload: SavingsPayload): Promise<Savings> {
  const data = await apiFetch<SavingsWire>(`/finance/savings/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
  return normalizeSavings(data);
}

function normalizeRelatedTransaction(item: SavingsRelatedTransactionWire): SavingsRelatedTransaction {
  return {
    id: item.id ?? item.ID ?? "",
    tid: item.tid ?? item.TID,
    transaction_date: item.transaction_date ?? item.TransactionDate ?? "",
    amount_minor: Number(item.amount_minor ?? item.AmountMinor ?? 0),
    currency: item.currency ?? item.Currency ?? "IDR",
    category_name: item.category_name ?? item.CategoryName ?? undefined,
    description: item.description ?? item.Description ?? undefined
  };
}

export async function listRelatedSavingsTransactions(id: string): Promise<SavingsRelatedTransaction[]> {
  const data = await apiFetch<SavingsRelatedTransactionWire[]>(`/finance/savings/${id}/transactions`);
  return Array.isArray(data) ? data.map(normalizeRelatedTransaction).filter((item) => item.id) : [];
}

export async function deleteSavings(id: string): Promise<void> {
  await apiFetch(`/finance/savings/${id}`, { method: "DELETE" });
}
