package http

type CreateTransactionsReportRequest struct {
	ReportType string `json:"report_type"`
	DateFrom *string `json:"date_from"`
	DateTo   *string `json:"date_to"`
	CategoryID *string `json:"category_id"`
	Type     *string `json:"type"`
	Status   *string `json:"status"`
	Format   string  `json:"format"`
}

type ReportResponse struct {
	ID             string  `json:"id"`
	ReportType     string  `json:"report_type"`
	Format         string  `json:"format"`
	Status         string  `json:"status"`
	StorageKey     *string `json:"storage_key"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}
