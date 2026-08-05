export type Savings = {
  id: string;
  sid: string;
  name: string;
  target_amount_minor: number;
  current_amount_minor: number;
  progress_percent?: number;
  currency: string;
  start_date?: string | null;
  target_date?: string | null;
  notes: string | null;
  status: string;
  created_at?: string;
  updated_at?: string;
};

export type SavingsPayload = {
  name: string;
  target_amount_minor: number;
  current_amount_minor: number;
  currency: string;
  start_date?: string | null;
  target_date?: string | null;
  notes?: string | null;
  status?: string;
};
