export type ReceiptProviderCode = "openai" | "gemini" | "anthropic" | "openai_compatible";

export type ReceiptProviderConfig = {
  provider_code: ReceiptProviderCode;
  display_name: string;
  base_url?: string | null;
  model_name: string;
  is_enabled: boolean;
  has_api_key: boolean;
};

export type ReceiptProviderModelOption = {
  id: string;
  label?: string;
};

export type ReceiptProviderTestResult = {
  provider_code: ReceiptProviderCode;
  base_url: string;
  using_saved_api_key: boolean;
  models: ReceiptProviderModelOption[];
};

export type ReceiptConfigStatus = {
  has_configured_provider: boolean;
  active_providers: string[];
};

export type ReceiptDraftItem = {
  item_name: string;
  quantity: number;
  price_per_unit_minor: number;
  discount_minor?: number;
  total_minor: number;
  notes?: string;
};

export type ReceiptScanDraft = {
  type: "expense" | "income" | "transfer" | "savings";
  category_name?: string;
  amount_minor: number;
  currency: string;
  transaction_date?: string;
  description?: string;
  merchant_name?: string;
  receipt_number?: string;
  payment_method?: string;
  subtotal_minor?: number;
  tax_minor?: number;
  service_charge_minor?: number;
  receipt_discount_minor?: number;
  confidence?: number;
  items?: ReceiptDraftItem[];
};

export type ReceiptScanHistoryItem = {
  id: string;
  provider_code: string;
  model_name: string;
  status: string;
  original_filename: string;
  mime_type: string;
  extracted_json?: Record<string, unknown>;
  error_message?: string | null;
  created_at: string;
};

export type ReceiptScanResult = {
  scan: ReceiptScanHistoryItem;
  draft: ReceiptScanDraft;
};
