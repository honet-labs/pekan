import { apiFetch } from "../../../../core/api/client";
import { CreateAccountPayload, CreateCategoryPayload, FinanceAccount, FinanceCategory } from "./masterdata.types";

export function listAccounts(): Promise<{ items: FinanceAccount[] }> {
  return apiFetch<{ items: FinanceAccount[] }>("/finance/accounts");
}

export function createAccount(payload: CreateAccountPayload): Promise<FinanceAccount> {
  return apiFetch<FinanceAccount>("/finance/accounts", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listCategories(): Promise<{ items: FinanceCategory[] }> {
  return apiFetch<{ items: FinanceCategory[] }>("/finance/categories");
}

export function createCategory(payload: CreateCategoryPayload): Promise<FinanceCategory> {
  return apiFetch<FinanceCategory>("/finance/categories", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateCategory(categoryID: string, payload: { name: string; category_type: string }): Promise<FinanceCategory> {
  return apiFetch<FinanceCategory>(`/finance/categories/${categoryID}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export async function deleteCategory(categoryID: string): Promise<void> {
  await apiFetch(`/finance/categories/${categoryID}`, { method: "DELETE" });
}
