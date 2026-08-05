package http

type BudgetRequest struct {
	Name              string  `json:"name"`
	CategoryID        *string `json:"category_id"`
	CategoryName      *string `json:"category_name"`
	AmountLimitMinor  int64   `json:"amount_limit_minor"`
	SpentAmountMinor  int64   `json:"spent_amount_minor"`
	ProgressPercent   float64 `json:"progress_percent"`
	Currency          string  `json:"currency"`
	Period            string  `json:"period"`
	StartDate         string  `json:"start_date"`
	EndDate           *string `json:"end_date"`
	AlertThresholdPct *int    `json:"alert_threshold_pct"`
	Notes             *string `json:"notes"`
	Status            string  `json:"status"`
}

type BudgetResponse struct {
	ID                string  `json:"id"`
	IDA               string  `json:"ida"`
	IDAnggaran        string  `json:"id_anggaran"`
	Name              string  `json:"name"`
	CategoryID        *string `json:"category_id"`
	CategoryName      *string `json:"category_name"`
	AmountLimitMinor  int64   `json:"amount_limit_minor"`
	SpentAmountMinor  int64   `json:"spent_amount_minor"`
	ProgressPercent   float64 `json:"progress_percent"`
	Currency          string  `json:"currency"`
	Period            string  `json:"period"`
	StartDate         string  `json:"start_date"`
	EndDate           *string `json:"end_date"`
	AlertThresholdPct *int    `json:"alert_threshold_pct"`
	Notes             *string `json:"notes"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}
