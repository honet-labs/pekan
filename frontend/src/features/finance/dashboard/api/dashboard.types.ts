export type DashboardSummary = {
  total_income_minor: number;
  total_expense_minor: number;
  total_transfer_minor: number;
  net_amount_minor: number;
  total_savings_minor: number;
  transaction_count: number;
  income_count: number;
  expense_count: number;
  transfer_count: number;
  savings_count: number;
};

export type DashboardSeriesPoint = {
  date: string;
  income_minor: number;
  expense_minor: number;
};

export type DashboardCategoryTotal = {
  category_id?: string | null;
  category_name?: string | null;
  transaction_type: string;
  total_minor: number;
  count: number;
};
