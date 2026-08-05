package http

type SavingsRequest struct {
	Name               string  `json:"name"`
	TargetAmountMinor  int64   `json:"target_amount_minor"`
	CurrentAmountMinor int64   `json:"current_amount_minor"`
	Currency           string  `json:"currency"`
	StartDate          *string `json:"start_date"`
	TargetDate         *string `json:"target_date"`
	Notes              *string `json:"notes"`
	Status             string  `json:"status"`
}

type SavingsResponse struct {
	ID                 string  `json:"id"`
	SID                string  `json:"sid"`
	IDSavings          string  `json:"id_savings"`
	Name               string  `json:"name"`
	TargetAmountMinor  int64   `json:"target_amount_minor"`
	CurrentAmountMinor int64   `json:"current_amount_minor"`
	ProgressPercent    float64 `json:"progress_percent"`
	Currency           string  `json:"currency"`
	StartDate          *string `json:"start_date"`
	TargetDate         *string `json:"target_date"`
	Notes              *string `json:"notes"`
	Status             string  `json:"status"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type RelatedTransactionResponse struct {
	ID              string  `json:"id"`
	TID             string  `json:"tid"`
	CategoryName    *string `json:"category_name"`
	AmountMinor     int64   `json:"amount_minor"`
	Currency        string  `json:"currency"`
	TransactionDate string  `json:"transaction_date"`
	Description     *string `json:"description"`
}
