export type Report = {
  id: string;
  report_type: string;
  format: string;
  status: string;
  storage_key?: string | null;
  created_at: string;
  updated_at: string;
};

export type CreateTransactionsReportPayload = {
  report_type?: "transactions" | "savings" | "budgets" | "reminders";
  date_from?: string;
  date_to?: string;
  category_id?: string;
  type?: "income" | "expense" | "transfer" | "savings";
  status?: string;
  format: string;
};
