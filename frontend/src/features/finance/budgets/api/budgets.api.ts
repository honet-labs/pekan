import { apiFetch } from "../../../../core/api/client";
import { Budget, BudgetPayload } from "./budgets.types";

type BudgetWire = Budget & {
  id_anggaran?: string;
};

function normalizeBudget(item: BudgetWire): Budget {
  const ida = item.ida || item.id_anggaran || "";
  return {
    ...item,
    ida,
    id_anggaran: item.id_anggaran ?? ida,
    spent_amount_minor: Number(item.spent_amount_minor ?? 0),
    progress_percent: Number(item.progress_percent ?? 0)
  };
}

type ListBudgetsParams = {
  page?: number;
  page_size?: number;
  status?: string;
};

type ListBudgetsResponse = {
  items: BudgetWire[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
};

export async function listBudgets(params: ListBudgetsParams = {}): Promise<ListBudgetsResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.page_size) query.set("page_size", String(params.page_size));
  if (params.status) query.set("status", params.status);

  const queryString = query.toString() ? `?${query.toString()}` : "";
  const data = await apiFetch<ListBudgetsResponse>(`/finance/budgets${queryString}`);
  
  return {
    ...data,
    items: Array.isArray(data?.items) ? data.items.map(normalizeBudget) : []
  };
}

export async function getBudget(id: string): Promise<Budget> {
  const data = await apiFetch<BudgetWire>(`/finance/budgets/${id}`);
  return normalizeBudget(data);
}

export async function createBudget(payload: BudgetPayload): Promise<Budget> {
  const data = await apiFetch<BudgetWire>("/finance/budgets", {
    method: "POST",
    body: JSON.stringify(payload)
  });
  return normalizeBudget(data);
}

export async function updateBudget(id: string, payload: BudgetPayload): Promise<Budget> {
  const data = await apiFetch<BudgetWire>(`/finance/budgets/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
  return normalizeBudget(data);
}

export async function deleteBudget(id: string): Promise<void> {
  await apiFetch(`/finance/budgets/${id}`, { method: "DELETE" });
}
