export type Reminder = {
  id: string;
  title: string;
  description?: string | null;
  amount_minor?: number | null;
  currency?: string | null;
  due_date: string;
  repeat_interval: string;
  status: string;
  total_tenor?: number | null;
  current_tenor?: number | null;
  created_at?: string;
  updated_at?: string;
};

export type ReminderPayload = {
  title: string;
  description?: string | null;
  amount_minor?: number | null;
  currency?: string | null;
  due_date: string;
  repeat_interval?: string;
  status?: string;
	total_tenor?: number | null;
	current_tenor?: number | null;
};

export type ReminderPayment = {
  id: string;
  reminder_id: string;
  paid_at: string;
  amount_minor: number;
  status: string;
  notes?: string | null;
  proof_image_url?: string | null;
  created_at?: string;
};

export type ReminderPaymentPayload = {
  paid_at: string;
  amount_minor: number;
  status: string;
  notes?: string | null;
  proof_image_url?: string | null;
};

