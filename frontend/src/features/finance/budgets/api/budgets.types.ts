export type Budget = {
  id: string;
  ida: string;
  id_anggaran?: string;
  name: string;
  category_id?: string | null;
  category_name?: string | null;
  amount_limit_minor: number;
  spent_amount_minor?: number;
  progress_percent?: number;
  currency: string;
  period: string;
  start_date: string;
  end_date?: string | null;
  alert_threshold_pct: number | null;
  notes: string | null;
  status: string;
  created_at: string;
  updated_at: string;
};

export type BudgetPayload = {
  name: string;
  category_id?: string;
  category_name?: string;
  amount_limit_minor: number;
  currency: string;
  period: string;
  start_date: string;
  end_date?: string;
  alert_threshold_pct?: number;
  notes?: string;
  status?: string;
};
