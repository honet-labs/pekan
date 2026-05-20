export type TransactionType = "income" | "expense" | "transfer" | "savings";

export type TransactionItem = {
  id?: string;
  item_name: string;
  quantity: number;
  price_per_unit_minor: number;
  discount_minor?: number;
  total_minor: number;
  notes?: string;
};

export type Transaction = {
  id: string;
  tid: string;
  account_id: string;
  account_name?: string;
  category_id?: string | null;
  category_name?: string | null;
  savings_ids?: string[];
  savings_names?: string[];
  type: TransactionType;
  amount_minor: number;
  currency: string;
  input_date: string;
  transaction_date: string;
  description?: string | null;
  merchant_name?: string | null;
  receipt_number?: string | null;
  payment_method?: string | null;
  subtotal_minor?: number;
  tax_minor?: number;
  service_charge_minor?: number;
  receipt_discount_minor?: number;
  created_by: string;
  created_by_name?: string;
  created_at: string;
  updated_at: string;
  items?: TransactionItem[];
};

export type CreateTransactionPayload = {
  account_id: string;
  category_id?: string | null;
  category_name?: string | null;
  savings_ids?: string[];
  type: TransactionType;
  amount_minor: number;
  currency: string;
  input_date?: string;
  transaction_date: string;
  description?: string | null;
  merchant_name?: string | null;
  receipt_number?: string | null;
  payment_method?: string | null;
  subtotal_minor?: number;
  tax_minor?: number;
  service_charge_minor?: number;
  receipt_discount_minor?: number;
  items?: TransactionItem[];
  receipt_scan_id?: string;
};

export type UpdateTransactionPayload = CreateTransactionPayload;

export type TransactionAttachment = {
  id: string;
  transaction_id: string;
  original_filename: string;
  mime_type: string;
  scan_status: string;
  size_bytes: number;
  created_at: string;
};
