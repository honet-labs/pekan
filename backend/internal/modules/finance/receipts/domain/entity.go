package domain

import (
	"encoding/json"
	"time"
)

type ProviderConfig struct {
	ID               string
	TenantID         string
	ProviderCode     string
	DisplayName      string
	BaseURL          *string
	ModelName        string
	IsEnabled        bool
	APIKeyCiphertext *string
	HasAPIKey        bool
	CreatedBy        string
	UpdatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ReceiptScan struct {
	ID               string
	TenantID         string
	ProviderCode     string
	ModelName        string
	Status           string
	OriginalFilename string
	MimeType         string
	ExtractedJSON    json.RawMessage
	ErrorMessage     *string
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ReceiptExtraction struct {
	MerchantName          string            `json:"merchant_name"`
	ReceiptNumber         string            `json:"receipt_number,omitempty"`
	TransactionDate       string            `json:"transaction_date,omitempty"`
	Currency              string            `json:"currency,omitempty"`
	Subtotal              float64           `json:"subtotal,omitempty"`
	Total                 float64           `json:"total,omitempty"`
	Tax                   float64           `json:"tax,omitempty"`
	ServiceCharge         float64           `json:"service_charge,omitempty"`
	Discount              float64           `json:"discount,omitempty"`
	PaymentMethod         string            `json:"payment_method,omitempty"`
	SuggestedType         string            `json:"suggested_type,omitempty"`
	SuggestedCategoryName string            `json:"suggested_category_name,omitempty"`
	Description           string            `json:"description,omitempty"`
	Confidence            float64           `json:"confidence,omitempty"`
	Notes                 string            `json:"notes,omitempty"`
	Items                 []ReceiptLineItem `json:"items,omitempty"`
}

type ReceiptLineItem struct {
	Name         string  `json:"name"`
	Qty          float64 `json:"qty,omitempty"`
	UnitPrice    float64 `json:"unit_price,omitempty"`
	Discount     float64 `json:"discount,omitempty"`
	LineTotal    float64 `json:"line_total,omitempty"`
	CategoryHint string  `json:"category_hint,omitempty"`
}
