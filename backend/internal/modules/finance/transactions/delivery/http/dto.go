package http

import "time"

type TransactionItemRequest struct {
	ID                string  `json:"id,omitempty"`
	ItemName          string  `json:"item_name"`
	Quantity          float64 `json:"quantity"`
	PriceMinor int64   `json:"price_per_unit_minor"`
	DiscountMinor     int64   `json:"discount_minor,omitempty"`
	TotalMinor        int64   `json:"total_minor"`
	Notes             *string `json:"notes,omitempty"`
}

type CreateTransactionRequest struct {
	AccountID            string                   `json:"account_id"`
	CategoryID           *string                  `json:"category_id"`
	CategoryName         *string                  `json:"category_name"`
	SavingsIDs           []string                 `json:"savings_ids"`
	Type                 string                   `json:"type"`
	AmountMinor          int64                    `json:"amount_minor"`
	Currency             string                   `json:"currency"`
	InputDate            *string                  `json:"input_date"`
	TransactionDate      string                   `json:"transaction_date"`
	Description          *string                  `json:"description"`
	MerchantName         *string                  `json:"merchant_name"`
	ReceiptNumber        *string                  `json:"receipt_number"`
	PaymentMethod        *string                  `json:"payment_method"`
	SubtotalMinor        int64                    `json:"subtotal_minor"`
	TaxMinor             int64                    `json:"tax_minor"`
	ServiceChargeMinor   int64                    `json:"service_charge_minor"`
	ReceiptDiscountMinor int64                    `json:"receipt_discount_minor"`
	Items                []TransactionItemRequest `json:"items"`
	ReceiptScanID        *string                  `json:"receipt_scan_id,omitempty"`
}

type UpdateTransactionRequest struct {
	AccountID            string                   `json:"account_id"`
	CategoryID           *string                  `json:"category_id"`
	CategoryName         *string                  `json:"category_name"`
	SavingsIDs           []string                 `json:"savings_ids"`
	Type                 string                   `json:"type"`
	AmountMinor          int64                    `json:"amount_minor"`
	Currency             string                   `json:"currency"`
	InputDate            *string                  `json:"input_date"`
	TransactionDate      string                   `json:"transaction_date"`
	Description          *string                  `json:"description"`
	MerchantName         *string                  `json:"merchant_name"`
	ReceiptNumber        *string                  `json:"receipt_number"`
	PaymentMethod        *string                  `json:"payment_method"`
	SubtotalMinor        int64                    `json:"subtotal_minor"`
	TaxMinor             int64                    `json:"tax_minor"`
	ServiceChargeMinor   int64                    `json:"service_charge_minor"`
	ReceiptDiscountMinor int64                    `json:"receipt_discount_minor"`
	Items                []TransactionItemRequest `json:"items"`
}

type TransactionItemResponse struct {
	ID                string  `json:"id,omitempty"`
	ItemName          string  `json:"item_name"`
	Quantity          float64 `json:"quantity"`
	PriceMinor int64   `json:"price_per_unit_minor"`
	DiscountMinor     int64   `json:"discount_minor,omitempty"`
	TotalMinor        int64   `json:"total_minor"`
	Notes             *string `json:"notes,omitempty"`
}

type TransactionResponse struct {
	ID                   string                    `json:"id"`
	TID                  string                    `json:"tid"`
	AccountID            string                    `json:"account_id"`
	AccountName          string                    `json:"account_name"`
	CategoryID           *string                   `json:"category_id"`
	CategoryName         *string                   `json:"category_name"`
	SavingsIDs           []string                  `json:"savings_ids"`
	SavingsNames         []string                  `json:"savings_names"`
	Type                 string                    `json:"type"`
	AmountMinor          int64                     `json:"amount_minor"`
	Currency             string                    `json:"currency"`
	InputDate            string                    `json:"input_date"`
	TransactionDate      string                    `json:"transaction_date"`
	Description          *string                   `json:"description"`
	MerchantName         *string                   `json:"merchant_name,omitempty"`
	ReceiptNumber        *string                   `json:"receipt_number,omitempty"`
	PaymentMethod        *string                   `json:"payment_method,omitempty"`
	SubtotalMinor        int64                     `json:"subtotal_minor,omitempty"`
	TaxMinor             int64                     `json:"tax_minor,omitempty"`
	ServiceChargeMinor   int64                     `json:"service_charge_minor,omitempty"`
	ReceiptDiscountMinor int64                     `json:"receipt_discount_minor,omitempty"`
	CreatedBy            string                    `json:"created_by"`
	CreatedByName        string                    `json:"created_by_name"`
	CreatedAt            string                    `json:"created_at"`
	UpdatedAt            string                    `json:"updated_at"`
	Items                []TransactionItemResponse `json:"items,omitempty"`
	Attachments          []AttachmentResponse      `json:"attachments,omitempty"`
}

type AttachmentResponse struct {
	ID               string `json:"id"`
	TransactionID    string `json:"transaction_id"`
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	ScanStatus       string `json:"scan_status"`
	SizeBytes        int64  `json:"size_bytes"`
	CreatedAt        string `json:"created_at"`
}

func formatDateOnly(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
