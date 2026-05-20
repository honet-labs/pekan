export type FinanceAccount = {
  id: string;
  name: string;
  account_type: "cash" | "bank" | "ewallet" | "credit";
  currency: string;
  opening_balance_minor: number;
  is_active: boolean;
};

export type FinanceCategory = {
  id: string;
  name: string;
  category_type: "income" | "expense";
  parent_id?: string | null;
  is_active: boolean;
};

export type CreateAccountPayload = {
  name: string;
  account_type: "cash" | "bank" | "ewallet" | "credit";
  currency: string;
  opening_balance_minor: number;
};

export type CreateCategoryPayload = {
  name: string;
  category_type: "income" | "expense";
  parent_id?: string | null;
};

