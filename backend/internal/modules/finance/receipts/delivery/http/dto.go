package http

import "encoding/json"

type ProviderConfigResponse struct {
	ProviderCode string  `json:"provider_code"`
	DisplayName  string  `json:"display_name"`
	BaseURL      *string `json:"base_url,omitempty"`
	ModelName    string  `json:"model_name"`
	IsEnabled    bool    `json:"is_enabled"`
	HasAPIKey    bool    `json:"has_api_key"`
}

type ProviderModelOptionResponse struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type UpsertProviderConfigRequest struct {
	ProviderCode string  `json:"provider_code"`
	DisplayName  string  `json:"display_name"`
	BaseURL      *string `json:"base_url,omitempty"`
	ModelName    string  `json:"model_name"`
	IsEnabled    bool    `json:"is_enabled"`
	APIKey       string  `json:"api_key,omitempty"`
	ClearAPIKey  bool    `json:"clear_api_key,omitempty"`
}

type UpsertProviderConfigsRequest struct {
	Items []UpsertProviderConfigRequest `json:"items"`
}

type TestProviderConnectionRequest struct {
	ProviderCode string  `json:"provider_code"`
	BaseURL      *string `json:"base_url,omitempty"`
	APIKey       string  `json:"api_key,omitempty"`
	ModelName    string  `json:"model_name,omitempty"`
}

type TestProviderConnectionResponse struct {
	ProviderCode     string                        `json:"provider_code"`
	BaseURL          string                        `json:"base_url"`
	UsingSavedAPIKey bool                          `json:"using_saved_api_key"`
	Models           []ProviderModelOptionResponse `json:"models"`
}

type ConfigStatusResponse struct {
	HasConfiguredProvider bool     `json:"has_configured_provider"`
	ActiveProviders       []string `json:"active_providers"`
}

type ReceiptScanResponse struct {
	ID               string          `json:"id"`
	ProviderCode     string          `json:"provider_code"`
	ModelName        string          `json:"model_name"`
	Status           string          `json:"status"`
	OriginalFilename string          `json:"original_filename"`
	MimeType         string          `json:"mime_type"`
	ExtractedJSON    json.RawMessage `json:"extracted_json,omitempty"`
	ErrorMessage     *string         `json:"error_message,omitempty"`
	CreatedAt        string          `json:"created_at"`
}

type DraftTransactionItemResponse struct {
	ItemName          string  `json:"item_name"`
	Quantity          float64 `json:"quantity"`
	PriceMinor        int64   `json:"price_per_unit_minor"`
	DiscountMinor     int64   `json:"discount_minor,omitempty"`
	TotalMinor        int64   `json:"total_minor"`
	Notes             string  `json:"notes,omitempty"`
}

type DraftTransactionResponse struct {
	Type                 string                         `json:"type"`
	CategoryName         string                         `json:"category_name,omitempty"`
	AmountMinor          int64                          `json:"amount_minor"`
	Currency             string                         `json:"currency"`
	Date                 string                         `json:"transaction_date,omitempty"`
	Description          string                         `json:"description,omitempty"`
	MerchantName         string                         `json:"merchant_name,omitempty"`
	ReceiptNumber        string                         `json:"receipt_number,omitempty"`
	PaymentMethod        string                         `json:"payment_method,omitempty"`
	SubtotalMinor        int64                          `json:"subtotal_minor,omitempty"`
	TaxMinor             int64                          `json:"tax_minor,omitempty"`
	ServiceChargeMinor   int64                          `json:"service_charge_minor,omitempty"`
	ReceiptDiscountMinor int64                          `json:"receipt_discount_minor,omitempty"`
	Confidence           float64                        `json:"confidence,omitempty"`
	Items                []DraftTransactionItemResponse `json:"items,omitempty"`
}

type ScanReceiptOutputResponse struct {
	Scan  ReceiptScanResponse      `json:"scan"`
	Draft DraftTransactionResponse `json:"draft"`
}
